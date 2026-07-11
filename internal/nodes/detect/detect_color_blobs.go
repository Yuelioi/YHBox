// internal/nodes/detect/detect_color_blobs.go
// DetectColorBlobs — 颜色连通域(blob)定位: flood-fill 找所有色块, 返回归一化 center/bbox/area + 排序/截断/Primary.
package detect

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/yottaapp/yotta/internal/node"
)

func init() { node.Register(&DetectColorBlobs{}) }

type DetectColorBlobs struct{}

const (
	dcbInExec           = "In"
	dcbInROI            = "ROI"
	dcbInMode           = "Mode"
	dcbInRange          = "Range"
	dcbInMinArea        = "MinArea"
	dcbInMaxBlobs       = "MaxBlobs"
	dcbInSort           = "Sort"
	dcbInRefPoint       = "RefPoint"
	dcbInPollIntervalMs = "PollIntervalMs"
	dcbInTimeoutMs      = "TimeoutMs"

	dcbOutFound    = "Found"
	dcbOutNotFound = "NotFound"
	dcbOutTimeout  = "Timeout"

	dcbDataBlobs         = "Blobs"
	dcbDataBlobCount     = "BlobCount"
	dcbDataPrimaryCenter = "PrimaryCenter"
	dcbDataPrimaryArea   = "PrimaryArea"

	dcbSortAreaDesc   = "area_desc"
	dcbSortDistScreen = "dist_screen_center"
	dcbSortDistPoint  = "dist_point"
)

func (DetectColorBlobs) Spec() node.Spec {
	return node.Spec{
		Kind:                "DetectColorBlobs",
		Category:            "Detect",
		NeedsTarget:         true,
		RuntimeCapabilities: []node.RuntimeCapability{node.RuntimeCapabilityVision},
		TargetCapabilities: []node.TargetCapability{
			node.TargetCapabilityScreenshot,
		},
		Inputs: append([]node.InputSpec{
			{Name: dcbInExec, Type: "Exec"},
			{Name: dcbInROI, Type: "Geometry", Schema: node.GeometrySchema()},
			{Name: dcbInMode, Type: "String", Default: "hsv",
				Widget: node.WidgetSpec{Kind: "dropdown",
					Props: node.MarshalProps(node.DropdownProps{
						Options: []node.EnumOption{{Value: "hsv"}, {Value: "rgb"}}})}},
			{Name: dcbInRange, Type: "JSON",
				Widget: node.WidgetSpec{Kind: "json",
					Props: node.MarshalProps(node.JSONProps{Rows: 2})},
				Schema: dcRangeSchema},
			{Name: dcbInMinArea, Type: "Number", Default: json.Number("20"),
				Widget: node.WidgetSpec{Kind: "number"}},
			{Name: dcbInMaxBlobs, Type: "Number", Default: json.Number("0"),
				Widget: node.WidgetSpec{Kind: "number"}},
			{Name: dcbInSort, Type: "String", Default: dcbSortAreaDesc,
				Widget: node.WidgetSpec{Kind: "dropdown",
					Props: node.MarshalProps(node.DropdownProps{
						Options: []node.EnumOption{
							{Value: dcbSortAreaDesc},
							{Value: dcbSortDistScreen},
							{Value: dcbSortDistPoint},
						}})}},
			{Name: dcbInRefPoint, Type: "Point", Advanced: true},
			{Name: dcbInPollIntervalMs, Type: "Number", Default: json.Number("100"),
				Widget: node.WidgetSpec{Kind: "number"}},
			{Name: dcbInTimeoutMs, Type: "Number", Default: json.Number("0"),
				Widget: node.WidgetSpec{Kind: "number"}},
		}, node.WindowInputSpec()),
		Outputs: []node.OutputSpec{
			{Name: dcbOutFound, Type: "Exec",
				Data: []node.DataField{
					{Name: dcbDataBlobs, Type: "JSON"},
					{Name: dcbDataBlobCount, Type: "Number"},
					{Name: dcbDataPrimaryCenter, Type: "Point"},
					{Name: dcbDataPrimaryArea, Type: "Number"},
				}},
			{Name: dcbOutNotFound, Type: "Exec",
				Data: []node.DataField{{Name: dcbDataBlobCount, Type: "Number"}}},
			{Name: dcbOutTimeout, Type: "Exec",
				Data: []node.DataField{{Name: dcbDataBlobCount, Type: "Number"}}},
		},
	}
}

// sortBlobs 原地排序：area_desc 按面积降序；dist_* 按到参考点距离升序。
func sortBlobs(blobs []node.BlobEntry, mode string, ref node.Point) {
	switch mode {
	case dcbSortDistScreen:
		sortByDist(blobs, 0.5, 0.5)
	case dcbSortDistPoint:
		sortByDist(blobs, ref.X, ref.Y) // RefPoint 未设 → (0,0)
	default: // area_desc
		sort.Slice(blobs, func(i, j int) bool { return blobs[i].Area > blobs[j].Area })
	}
}

func sortByDist(blobs []node.BlobEntry, cx, cy float64) {
	d := func(b node.BlobEntry) float64 {
		dx, dy := b.CenterX-cx, b.CenterY-cy
		return dx*dx + dy*dy
	}
	sort.Slice(blobs, func(i, j int) bool { return d(blobs[i]) < d(blobs[j]) })
}

func (DetectColorBlobs) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	roi := in.Geometry(dcbInROI)
	mode := in.String(dcbInMode)
	if mode == "" {
		mode = "hsv"
	}
	rngArr, err := parseRange6(in.Raw(dcbInRange))
	if err != nil {
		return nil, fmt.Errorf("DetectColorBlobs range: %w", err)
	}
	minArea := in.Int(dcbInMinArea)
	maxBlobs := in.Int(dcbInMaxBlobs)
	sortMode := in.String(dcbInSort)
	if sortMode == "" {
		sortMode = dcbSortAreaDesc
	}
	ref := in.Point(dcbInRefPoint)
	pollMs := in.Int(dcbInPollIntervalMs)
	if pollMs < dchPollClampMs {
		pollMs = dchPollClampMs
	}
	timeoutMs := in.Int(dcbInTimeoutMs)

	var deadline time.Time
	if timeoutMs > 0 {
		deadline = time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	}
	pollDur := time.Duration(pollMs) * time.Millisecond
	firstScan := true
	var lastCount int
	for {
		select {
		case <-ctx.Context().Done():
			return nil, ctx.Context().Err()
		default:
		}
		blobs, derr := ctx.Services().Vision.DetectColorBlobs(roi, mode, rngArr, minArea)
		if derr != nil {
			return nil, node.Failf(node.CodeCaptureFailed, derr, "DetectColorBlobs: %v", derr)
		}
		lastCount = len(blobs) // 合格块总数, 不受 MaxBlobs 影响
		if lastCount > 0 {
			sortBlobs(blobs, sortMode, ref)
			out := blobs
			if maxBlobs > 0 && len(out) > maxBlobs {
				out = out[:maxBlobs]
			}
			primary := out[0]
			pc := node.Point{X: primary.CenterX, Y: primary.CenterY}
			return ctx.Out(dcbOutFound).
				Set(dcbDataBlobs, out).
				Set(dcbDataBlobCount, lastCount).
				Set(dcbDataPrimaryCenter, pc).
				Set(dcbDataPrimaryArea, primary.Area).
				Fire(), nil
		}
		if timeoutMs <= 0 && firstScan {
			return ctx.Out(dcbOutNotFound).Set(dcbDataBlobCount, 0).Fire(), nil
		}
		firstScan = false
		if !deadline.IsZero() && time.Now().After(deadline) {
			return ctx.Out(dcbOutTimeout).Set(dcbDataBlobCount, lastCount).Fire(), nil
		}
		select {
		case <-ctx.Context().Done():
			return nil, ctx.Context().Err()
		case <-time.After(pollDur):
		}
	}
}
