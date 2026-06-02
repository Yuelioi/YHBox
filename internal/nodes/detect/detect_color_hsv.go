// internal/nodes/detect/detect_color_hsv.go
// DetectColorHSV — ROI 内 HSV 像素比例阈值 + 轮询直到命中或超时.
// 跟 DetectColor 区别: ROI 像素坐标 + HSV-only + 比例阈值 + 轮询 (3 出口 yes/no/timeout).
package detect

import (
	"encoding/json"
	"fmt"
	"time"

	"yotta/internal/node"
)

// hsvObjSchema HSV 阈值 6 字段的结构化 schema, 供 FE StructuredInput 渲染.
var hsvObjSchema = node.ObjSchema(
	node.Field("hMin", node.NumberSchema(), false),
	node.Field("hMax", node.NumberSchema(), false),
	node.Field("sMin", node.NumberSchema(), false),
	node.Field("sMax", node.NumberSchema(), false),
	node.Field("vMin", node.NumberSchema(), false),
	node.Field("vMax", node.NumberSchema(), false),
)

func init() { node.Register(&DetectColorHSV{}) }

type DetectColorHSV struct{}

const (
	dchInExec           = "In"
	dchInROI            = "ROI"
	dchInHSV            = "HSV"
	dchInMinPixelRatio  = "MinPixelRatio"
	dchInPollIntervalMs = "PollIntervalMs"
	dchInTimeoutMs      = "TimeoutMs"
	dchOutYes           = "Yes"
	dchOutNo            = "No"
	dchOutTimeout       = "Timeout"
	dchDataCount        = "PixelCount"
	dchDataRatio        = "PixelRatio"

	// poll 间隔下限.
	dchPollClampMs   = 10
	dchDefaultPollMs = 100
	dchDefaultToMs   = 5000
)

func (DetectColorHSV) Spec() node.Spec {
	return node.Spec{
		Kind:        "DetectColorHSV",
		Category:    "Detect",
		NeedsWindow: true,
		Inputs: []node.InputSpec{
			{Name: dchInExec, Type: "Exec"},
			{Name: dchInROI, Type: "Geometry", Schema: node.GeometrySchema()},
			{Name: dchInHSV, Type: "JSON",
				Widget: node.WidgetSpec{Kind: "json",
					Props: node.MarshalProps(node.JSONProps{Rows: 2})},
				Schema: hsvObjSchema},
			{Name: dchInMinPixelRatio, Type: "Number", Default: json.Number("0.05"),
				Widget: node.WidgetSpec{Kind: "slider",
					Props: node.MarshalProps(node.SliderProps{Min: 0, Max: 1, Step: 0.01})}},
			{Name: dchInPollIntervalMs, Type: "Number", Default: json.Number("100"),
				Widget: node.WidgetSpec{Kind: "number"}},
			{Name: dchInTimeoutMs, Type: "Number", Default: json.Number("5000"),
				Widget: node.WidgetSpec{Kind: "number"}},
		},
		Outputs: []node.OutputSpec{
			{Name: dchOutYes, Type: "Exec",
				Data: []node.DataField{
					{Name: dchDataCount, Type: "Number"},
					{Name: dchDataRatio, Type: "Number"},
				}},
			{Name: dchOutNo, Type: "Exec",
				Data: []node.DataField{
					{Name: dchDataCount, Type: "Number"},
					{Name: dchDataRatio, Type: "Number"},
				}},
			{Name: dchOutTimeout, Type: "Exec",
				Data: []node.DataField{
					{Name: dchDataCount, Type: "Number"},
					{Name: dchDataRatio, Type: "Number"},
				}},
		},
	}
}

func (DetectColorHSV) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	roi := in.Geometry(dchInROI)
	hsv, err := parseHSVRange(in.JSON(dchInHSV))
	if err != nil {
		return nil, fmt.Errorf("DetectColorHSV hsv: %w", err)
	}
	minRatio := in.Float64(dchInMinPixelRatio)
	pollMs := in.Int(dchInPollIntervalMs)
	if pollMs < dchPollClampMs {
		pollMs = dchPollClampMs
	}
	timeoutMs := in.Int(dchInTimeoutMs)

	var deadline time.Time
	if timeoutMs > 0 {
		deadline = time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	}

	pollDur := time.Duration(pollMs) * time.Millisecond
	var lastCount int
	var lastRatio float64
	firstScan := true
	for {
		select {
		case <-ctx.Context().Done():
			return nil, ctx.Context().Err()
		default:
		}
		count, ratio, derr := ctx.Vision().DetectColorHSV(roi, hsv)
		if derr != nil {
			return nil, fmt.Errorf("DetectColorHSV: %w", derr)
		}
		lastCount = count
		lastRatio = ratio
		if ratio >= minRatio {
			return ctx.Out(dchOutYes).
				Set(dchDataCount, count).Set(dchDataRatio, ratio).Fire(), nil
		}
		// timeoutMs<=0 且首次未命中 → No (跟 ROIColorScan 同款语义).
		if timeoutMs <= 0 && firstScan {
			return ctx.Out(dchOutNo).
				Set(dchDataCount, count).Set(dchDataRatio, ratio).Fire(), nil
		}
		firstScan = false
		if !deadline.IsZero() && time.Now().After(deadline) {
			return ctx.Out(dchOutTimeout).
				Set(dchDataCount, lastCount).Set(dchDataRatio, lastRatio).Fire(), nil
		}
		select {
		case <-ctx.Context().Done():
			return nil, ctx.Context().Err()
		case <-time.After(pollDur):
		}
	}
}

// parseHSVRange 把 inputs JSON map 转 node.HSVRange. min > max 报错.
func parseHSVRange(m map[string]any) (node.HSVRange, error) {
	if m == nil {
		return node.HSVRange{}, fmt.Errorf("missing")
	}
	f := func(k string) int {
		switch v := m[k].(type) {
		case float64:
			return int(v)
		case int:
			return v
		case json.Number:
			n, _ := v.Int64()
			return int(n)
		}
		return 0
	}
	h := node.HSVRange{
		HMin: f("hMin"), HMax: f("hMax"),
		SMin: f("sMin"), SMax: f("sMax"),
		VMin: f("vMin"), VMax: f("vMax"),
	}
	if h.HMin > h.HMax || h.SMin > h.SMax || h.VMin > h.VMax {
		return h, fmt.Errorf("inverted: H=[%d,%d] S=[%d,%d] V=[%d,%d]",
			h.HMin, h.HMax, h.SMin, h.SMax, h.VMin, h.VMax)
	}
	return h, nil
}

func (DetectColorHSV) Display(in node.Inputs, exitName string, out node.OutputData) string {
	return fmt.Sprintf("[%s] count=%d ratio=%.3f", exitName, out.Int(dchDataCount), out.Float64(dchDataRatio))
}

// Validate 在编辑期就抓 HSV 范围倒置 (min>max), 不再等 Run 时炸.
// 跟 DetectColor / ROIColorScan / DualColorBarTrack 的 Validate 对齐.
func (DetectColorHSV) Validate(in node.Inputs) []node.ValidationError {
	m := in.JSON(dchInHSV)
	if m == nil {
		return nil // 未填 → 由 Run 默认处理, 不在编辑期报
	}
	if _, err := parseHSVRange(m); err != nil {
		return []node.ValidationError{{
			Code:    "INVALID_HSV_RANGE",
			Message: fmt.Sprintf("HSV range invalid: %v", err),
			Field:   dchInHSV,
		}}
	}
	return nil
}
