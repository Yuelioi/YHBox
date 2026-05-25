// internal/nodes/detect/roi_color_scan.go
// ROIColorScan — ROI 内沿 axis 找连续 HSV 命中像素段 (cluster) + 轮询.
//
// 老 runtime: internal/services/container/runtime/roi_scan.go::execROIColorScan.
package detect

import (
	"encoding/json"
	"fmt"
	"time"

	"yhbox/internal/node"
)

func init() { node.Register(&ROIColorScan{}) }

type ROIColorScan struct{}

const (
	rcsInExec            = "in"
	rcsInROI             = "ROI"
	rcsInHSV             = "HSV"
	rcsInAxis            = "Axis"
	rcsInMinClusterPx    = "MinClusterPx"
	rcsInMaxClusterPx    = "MaxClusterPx"
	rcsInMinClusterCount = "MinClusterCount"
	rcsInPollIntervalMs  = "PollIntervalMs"
	rcsInTimeoutMs       = "TimeoutMs"
	rcsOutFound          = "Found"
	rcsOutNotFound       = "NotFound"
	rcsOutTimeout        = "Timeout"
	rcsDataClusters      = "Clusters"
	rcsDataClusterCount  = "ClusterCount"
)

func (ROIColorScan) Spec() node.Spec {
	return node.Spec{
		Kind:        "ROIColorScan",
		Category:    "Detect",
		DisplayName: "ROI 颜色 cluster 扫描",
		Description: "沿 axis (x/y) 扫 ROI 内 HSV 命中像素, 合并连续段为 cluster. 命中 >= minClusterCount → Found. timeoutMs<=0 + 首扫不足 → NotFound.",
		Inputs: []node.InputSpec{
			{Name: rcsInExec, Type: "Exec"},
			{Name: rcsInROI, Type: "JSON", DisplayName: "ROI (像素)",
				Doc: `{"x":0,"y":0,"w":100,"h":100}`,
				Widget: node.WidgetSpec{Kind: "json",
					Props: node.MarshalProps(node.JSONProps{Rows: 2})}},
			{Name: rcsInHSV, Type: "JSON", DisplayName: "HSV 范围",
				Doc: `{"hMin":0,"hMax":180,"sMin":0,"sMax":255,"vMin":0,"vMax":255}`,
				Widget: node.WidgetSpec{Kind: "json",
					Props: node.MarshalProps(node.JSONProps{Rows: 2})}},
			{Name: rcsInAxis, Type: "String", Default: "x", DisplayName: "扫描轴",
				Widget: node.WidgetSpec{Kind: "dropdown",
					Props: node.MarshalProps(node.DropdownProps{
						Options: []node.EnumOption{
							{Value: "x", Label: "水平 (x)"},
							{Value: "y", Label: "垂直 (y)"},
						}})}},
			{Name: rcsInMinClusterPx, Type: "Number", Default: json.Number("2"),
				DisplayName: "最小段长 (px)",
				Widget:      node.WidgetSpec{Kind: "number"}},
			{Name: rcsInMaxClusterPx, Type: "Number", Default: json.Number("0"),
				DisplayName: "最大段长 (px)", Doc: "<=0 默认 ROI 大小 / 3",
				Widget: node.WidgetSpec{Kind: "number"}},
			{Name: rcsInMinClusterCount, Type: "Number", Default: json.Number("1"),
				DisplayName: "最少 cluster 数",
				Widget:      node.WidgetSpec{Kind: "number"}},
			{Name: rcsInPollIntervalMs, Type: "Number", Default: json.Number("100"),
				DisplayName: "轮询间隔 (ms)",
				Widget:      node.WidgetSpec{Kind: "number"}},
			{Name: rcsInTimeoutMs, Type: "Number", Default: json.Number("5000"),
				DisplayName: "超时 (ms)", Doc: "<=0 单次扫描",
				Widget: node.WidgetSpec{Kind: "number"}},
		},
		Outputs: []node.OutputSpec{
			{Name: rcsOutFound, Type: "Exec", DisplayName: "命中",
				Data: []node.DataField{
					{Name: rcsDataClusters, Type: "JSON"},
					{Name: rcsDataClusterCount, Type: "Number"},
				}},
			{Name: rcsOutNotFound, Type: "Exec", DisplayName: "未命中",
				Data: []node.DataField{
					{Name: rcsDataClusterCount, Type: "Number"},
				}},
			{Name: rcsOutTimeout, Type: "Exec", DisplayName: "超时",
				Data: []node.DataField{
					{Name: rcsDataClusterCount, Type: "Number"},
				}},
		},
	}
}

func (ROIColorScan) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	roi, err := parseROIRect(in.JSON(rcsInROI))
	if err != nil {
		return nil, fmt.Errorf("ROIColorScan roi: %w", err)
	}
	hsv, err := parseHSVRange(in.JSON(rcsInHSV))
	if err != nil {
		return nil, fmt.Errorf("ROIColorScan hsv: %w", err)
	}
	axis := in.String(rcsInAxis)
	if axis != "x" && axis != "y" {
		return nil, fmt.Errorf("ROIColorScan axis must be x or y, got %q", axis)
	}
	minPx := in.Int(rcsInMinClusterPx)
	maxPx := in.Int(rcsInMaxClusterPx)
	if maxPx <= 0 {
		if axis == "x" {
			maxPx = int(roi.W) / 3
		} else {
			maxPx = int(roi.H) / 3
		}
	}
	minCount := in.Int(rcsInMinClusterCount)
	pollMs := in.Int(rcsInPollIntervalMs)
	if pollMs < dchPollClampMs {
		pollMs = dchPollClampMs
	}
	timeoutMs := in.Int(rcsInTimeoutMs)

	var deadline time.Time
	if timeoutMs > 0 {
		deadline = time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	}
	pollDur := time.Duration(pollMs) * time.Millisecond

	var lastCount int
	firstScan := true
	for {
		select {
		case <-ctx.Context().Done():
			return nil, ctx.Context().Err()
		default:
		}
		clusters, serr := ctx.Vision().ROIColorScan(roi, hsv, axis, minPx, maxPx)
		if serr != nil {
			return nil, fmt.Errorf("ROIColorScan: %w", serr)
		}
		lastCount = len(clusters)
		if lastCount >= minCount {
			return ctx.Out(rcsOutFound).
				Set(rcsDataClusters, clusters).
				Set(rcsDataClusterCount, lastCount).Fire(), nil
		}
		if timeoutMs <= 0 && firstScan {
			return ctx.Out(rcsOutNotFound).Set(rcsDataClusterCount, lastCount).Fire(), nil
		}
		firstScan = false
		if !deadline.IsZero() && time.Now().After(deadline) {
			return ctx.Out(rcsOutTimeout).Set(rcsDataClusterCount, lastCount).Fire(), nil
		}
		select {
		case <-ctx.Context().Done():
			return nil, ctx.Context().Err()
		case <-time.After(pollDur):
		}
	}
}

func (ROIColorScan) Display(in node.Inputs, exitName string, out node.OutputData) string {
	return fmt.Sprintf("[%s] clusters=%d axis=%s", exitName, out.Int(rcsDataClusterCount), in.String(rcsInAxis))
}

func (ROIColorScan) Validate(in node.Inputs) []node.ValidationError {
	var errs []node.ValidationError
	axis := in.String(rcsInAxis)
	if axis != "" && axis != "x" && axis != "y" {
		errs = append(errs, node.ValidationError{
			Code:    "INVALID_SCAN_AXIS",
			Message: fmt.Sprintf("scanAxis must be x or y, got %q", axis),
			Field:   rcsInAxis,
		})
	}
	minC := in.Int(rcsInMinClusterPx)
	maxC := in.Int(rcsInMaxClusterPx)
	if maxC > 0 && minC > maxC {
		errs = append(errs, node.ValidationError{
			Code:    "INVALID_CLUSTER_RANGE",
			Message: fmt.Sprintf("minClusterPx %d > maxClusterPx %d", minC, maxC),
			Field:   rcsInMinClusterPx,
		})
	}
	return errs
}
