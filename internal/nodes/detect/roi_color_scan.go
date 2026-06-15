// internal/nodes/detect/roi_color_scan.go
// ROIColorScan — ROI 内沿 axis 找连续 HSV 命中像素段 (cluster) + 轮询.
package detect

import (
	"encoding/json"
	"fmt"
	"time"

	"yotta/internal/node"
)

func init() { node.Register(&ROIColorScan{}) }

type ROIColorScan struct{}

const (
	rcsInExec            = "In"
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
		NeedsWindow: true,
		Inputs: []node.InputSpec{
			{Name: rcsInExec, Type: "Exec"},
			{Name: rcsInROI, Type: "Geometry", Schema: node.GeometrySchema()},
			{Name: rcsInHSV, Type: "JSON",
				Widget: node.WidgetSpec{Kind: "json",
					Props: node.MarshalProps(node.JSONProps{Rows: 2})},
				Schema: hsvObjSchema},
			{Name: rcsInAxis, Type: "String", Default: "x",
				Widget: node.WidgetSpec{Kind: "dropdown",
					Props: node.MarshalProps(node.DropdownProps{
						Options: []node.EnumOption{
							{Value: "x"},
							{Value: "y"},
						}})}},
			{Name: rcsInMinClusterPx, Type: "Number", Default: json.Number("2"),
				Widget: node.WidgetSpec{Kind: "number"}},
			{Name: rcsInMaxClusterPx, Type: "Number", Default: json.Number("0"),
				Widget: node.WidgetSpec{Kind: "number"}},
			{Name: rcsInMinClusterCount, Type: "Number", Default: json.Number("1"),
				Widget: node.WidgetSpec{Kind: "number"}},
			{Name: rcsInPollIntervalMs, Type: "Number", Default: json.Number("100"),
				Widget: node.WidgetSpec{Kind: "number"}},
			{Name: rcsInTimeoutMs, Type: "Number", Default: json.Number("5000"),
				Widget: node.WidgetSpec{Kind: "number"}},
		},
		Outputs: []node.OutputSpec{
			{Name: rcsOutFound, Type: "Exec",
				Data: []node.DataField{
					{Name: rcsDataClusters, Type: "JSON"},
					{Name: rcsDataClusterCount, Type: "Number"},
				}},
			{Name: rcsOutNotFound, Type: "Exec",
				Data: []node.DataField{
					{Name: rcsDataClusterCount, Type: "Number"},
				}},
			{Name: rcsOutTimeout, Type: "Exec",
				Data: []node.DataField{
					{Name: rcsDataClusterCount, Type: "Number"},
				}},
		},
	}
}

func (ROIColorScan) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	roi := in.Geometry(rcsInROI)
	hsv, err := parseHSVRange(in.JSON(rcsInHSV))
	if err != nil {
		return nil, fmt.Errorf("ROIColorScan hsv: %w", err)
	}
	axis := in.String(rcsInAxis)
	if axis != "x" && axis != "y" {
		return nil, fmt.Errorf("ROIColorScan axis must be x or y, got %q", axis)
	}
	minPx := in.Int(rcsInMinClusterPx)
	// maxPx<=0 → adapter 按裁后子帧尺寸算默认 (宽/3 或 高/3), 节点直传不覆盖.
	maxPx := in.Int(rcsInMaxClusterPx)
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
			return nil, node.Failf(node.CodeCaptureFailed, serr, "ROIColorScan: %v", serr)
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
