// internal/nodes/detect/dual_color_bar_track.go
// DualColorBarTrack — 按 Geometry ROI 裁子帧, 双色 HSV cluster 检测算 inner 在 outer 区域里
// 的位置. 通用双色条算法 (血条/进度条/QTE 双色条/钓鱼 cursor-target).
package detect

import (
	"encoding/json"

	"yotta/internal/node"
)

func init() { node.Register(&DualColorBarTrack{}) }

type DualColorBarTrack struct{}

const (
	dcbtInExec       = "In"
	dcbtInRoi        = "Roi"
	dcbtInInnerColor = "InnerColor"
	dcbtInOuterColor = "OuterColor"
	dcbtInOptions    = "Options"
	dcbtOutFound     = "Found"
	dcbtOutMissing   = "Missing"
	dcbtDataInnerX   = "InnerX"
	dcbtDataOuterX   = "OuterX"
	dcbtDataOuterW   = "OuterWidth"
	dcbtDataConf     = "Confidence"
	dcbtDataInnerPx  = "InnerPx"
	dcbtDataOuterPx  = "OuterPx"
	dcbtCapInnerX    = "CaptureInnerX"
	dcbtCapOuterX    = "CaptureOuterX"
	dcbtCapOuterW    = "CaptureOuterWidth"
	dcbtCapConf      = "CaptureConfidence"
	dcbtCapInnerPx   = "CaptureInnerPx"
	dcbtCapOuterPx   = "CaptureOuterPx"
)

// dualBarOptionsSchema Options 字段结构化 schema — 所有 parseDualBarOptions 实际读取的键.
var dualBarOptionsSchema = node.ObjSchema(
	node.Field("innerMinPx", node.NumberSchema(), false),
	node.Field("innerMaxPx", node.NumberSchema(), false),
	node.Field("outerMinPx", node.NumberSchema(), false),
	node.Field("bandRatioH", node.NumberSchema(), false),
	node.Field("bandRatioInner", node.NumberSchema(), false),
	node.Field("confInnerWeight", node.NumberSchema(), false),
	node.Field("confOuterWeight", node.NumberSchema(), false),
)

func (DualColorBarTrack) Spec() node.Spec {
	return node.Spec{
		Kind:        "DualColorBarTrack",
		Category:    "Detect",
		NeedsWindow: true,
		Inputs: []node.InputSpec{
			{Name: dcbtInExec, Type: "Exec"},
			{Name: dcbtInRoi, Type: "Geometry", Required: true,
				Schema: node.GeometrySchema()},
			{Name: dcbtInInnerColor, Type: "JSON",
				Widget: node.WidgetSpec{Kind: "json", Props: node.MarshalProps(node.JSONProps{Rows: 2})},
				Schema: hsvObjSchema},
			{Name: dcbtInOuterColor, Type: "JSON",
				Widget: node.WidgetSpec{Kind: "json", Props: node.MarshalProps(node.JSONProps{Rows: 2})},
				Schema: hsvObjSchema},
			{Name: dcbtInOptions, Type: "JSON",
				Widget: node.WidgetSpec{Kind: "json", Props: node.MarshalProps(node.JSONProps{Rows: 2})},
				Schema: dualBarOptionsSchema},
			{Name: dcbtCapInnerX, Type: "String", Advanced: true, Semantic: "capture",
				CaptureType: "number", Widget: node.WidgetSpec{Kind: "text"}},
			{Name: dcbtCapOuterX, Type: "String", Advanced: true, Semantic: "capture",
				CaptureType: "number", Widget: node.WidgetSpec{Kind: "text"}},
			{Name: dcbtCapOuterW, Type: "String", Advanced: true, Semantic: "capture",
				CaptureType: "number", Widget: node.WidgetSpec{Kind: "text"}},
			{Name: dcbtCapConf, Type: "String", Advanced: true, Semantic: "capture",
				CaptureType: "number", Widget: node.WidgetSpec{Kind: "text"}},
			{Name: dcbtCapInnerPx, Type: "String", Advanced: true, Semantic: "capture",
				CaptureType: "number", Widget: node.WidgetSpec{Kind: "text"}},
			{Name: dcbtCapOuterPx, Type: "String", Advanced: true, Semantic: "capture",
				CaptureType: "number", Widget: node.WidgetSpec{Kind: "text"}},
		},
		Outputs: []node.OutputSpec{
			{Name: dcbtOutFound, Type: "Exec",
				Data: []node.DataField{
					{Name: dcbtDataInnerX, Type: "Number"},
					{Name: dcbtDataOuterX, Type: "Number"},
					{Name: dcbtDataOuterW, Type: "Number"},
					{Name: dcbtDataConf, Type: "Number"},
					{Name: dcbtDataInnerPx, Type: "Number"},
					{Name: dcbtDataOuterPx, Type: "Number"},
				}},
			{Name: dcbtOutMissing, Type: "Exec",
				Data: []node.DataField{
					{Name: dcbtDataConf, Type: "Number", Optional: true},
				}},
		},
	}
}

func (DualColorBarTrack) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	roi := in.Geometry(dcbtInRoi)
	inner := parseDualBarHSV(in.JSON(dcbtInInnerColor), defaultInnerHSV)
	outer := parseDualBarHSV(in.JSON(dcbtInOuterColor), defaultOuterHSV)
	opts := parseDualBarOptions(in.JSON(dcbtInOptions))

	result, err := ctx.Vision().DualBarTrack(roi, inner, outer, opts)
	if err != nil {
		return nil, node.Failf(node.CodeCaptureFailed, err, "DualColorBarTrack: %v", err)
	}
	if !result.Found || result.InnerX < 0 || result.OuterX < 0 {
		node.Capture(ctx, in, dcbtCapConf, result.Confidence) // missing 仅带 Conf
		return ctx.Out(dcbtOutMissing).Set(dcbtDataConf, result.Confidence).Fire(), nil
	}
	node.Capture(ctx, in, dcbtCapInnerX, result.InnerX)
	node.Capture(ctx, in, dcbtCapOuterX, result.OuterX)
	node.Capture(ctx, in, dcbtCapOuterW, result.OuterWidth)
	node.Capture(ctx, in, dcbtCapConf, result.Confidence)
	node.Capture(ctx, in, dcbtCapInnerPx, result.InnerPx)
	node.Capture(ctx, in, dcbtCapOuterPx, result.OuterPx)
	return ctx.Out(dcbtOutFound).
		Set(dcbtDataInnerX, result.InnerX).
		Set(dcbtDataOuterX, result.OuterX).
		Set(dcbtDataOuterW, result.OuterWidth).
		Set(dcbtDataConf, result.Confidence).
		Set(dcbtDataInnerPx, result.InnerPx).
		Set(dcbtDataOuterPx, result.OuterPx).
		Fire(), nil
}

// 默认 HSV 阈值 (示例, 实测调出): inner=cursor 浅黄, outer=target 高饱和青.
// 通用 case 用户填自己的, 这是一组示例默认.
var (
	defaultInnerHSV = node.HSVRange{HMin: 45, HMax: 70, SMin: 16, SMax: 100, VMin: 78, VMax: 100}
	defaultOuterHSV = node.HSVRange{HMin: 160, HMax: 180, SMin: 55, SMax: 100, VMin: 39, VMax: 100}
)

func parseDualBarHSV(m map[string]any, fallback node.HSVRange) node.HSVRange {
	if m == nil {
		return fallback
	}
	get := func(k string, def int) int {
		v, ok := m[k]
		if !ok {
			return def
		}
		switch x := v.(type) {
		case float64:
			return int(x)
		case int:
			return x
		case int64:
			return int(x)
		case json.Number:
			n, _ := x.Int64()
			return int(n)
		}
		return def
	}
	return node.HSVRange{
		HMin: get("hMin", fallback.HMin),
		HMax: get("hMax", fallback.HMax),
		SMin: get("sMin", fallback.SMin),
		SMax: get("sMax", fallback.SMax),
		VMin: get("vMin", fallback.VMin),
		VMax: get("vMax", fallback.VMax),
	}
}

func parseDualBarOptions(m map[string]any) node.DualBarOptions {
	if m == nil {
		return node.DualBarOptions{}
	}
	getI := func(k string) int {
		v, ok := m[k]
		if !ok {
			return 0
		}
		switch x := v.(type) {
		case float64:
			return int(x)
		case int:
			return x
		case int64:
			return int(x)
		case json.Number:
			n, _ := x.Int64()
			return int(n)
		}
		return 0
	}
	getF := func(k string) float64 {
		v, ok := m[k]
		if !ok {
			return 0
		}
		switch x := v.(type) {
		case float64:
			return x
		case int:
			return float64(x)
		case int64:
			return float64(x)
		case json.Number:
			f, _ := x.Float64()
			return f
		}
		return 0
	}
	return node.DualBarOptions{
		InnerMinPx:      getI("innerMinPx"),
		InnerMaxPx:      getI("innerMaxPx"),
		OuterMinPx:      getI("outerMinPx"),
		BandRatioH:      getF("bandRatioH"),
		BandRatioInner:  getF("bandRatioInner"),
		ConfInnerWeight: getF("confInnerWeight"),
		ConfOuterWeight: getF("confOuterWeight"),
	}
}

func (DualColorBarTrack) Validate(in node.Inputs) []node.ValidationError {
	// Geometry 输入由 framework 的 Required 校验兜 (Required: true in Spec).
	// 无需再手动校验; 具体 Geometry 值合法性由 ResolveGeometry 在 adapter 侧处理.
	return nil
}
