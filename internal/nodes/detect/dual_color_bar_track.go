// internal/nodes/detect/dual_color_bar_track.go
// DualColorBarTrack — 在多分辨率 ROI 数组中按当前 client size 选 ROI, 抓帧 → 双色 HSV
// cluster 检测算 inner 在 outer 区域里的位置. 通用双色条算法 (血条/进度条/QTE 双色条
// /钓鱼 cursor-target).
package detect

import (
	"encoding/json"
	"fmt"

	"yhbox/internal/node"
)

func init() { node.Register(&DualColorBarTrack{}) }

type DualColorBarTrack struct{}

const (
	dcbtInExec       = "In"
	dcbtInRois       = "Rois"
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
)

func (DualColorBarTrack) Spec() node.Spec {
	return node.Spec{
		Kind:     "DualColorBarTrack",
		Category: "Detect",
		Inputs: []node.InputSpec{
			{Name: dcbtInExec, Type: "Exec"},
			{Name: dcbtInRois, Type: "JSON", Required: true,
				Widget: node.WidgetSpec{Kind: "json",
					Props: node.MarshalProps(node.JSONProps{Rows: 4})}},
			{Name: dcbtInInnerColor, Type: "JSON",
				Widget: node.WidgetSpec{Kind: "json", Props: node.MarshalProps(node.JSONProps{Rows: 2})}},
			{Name: dcbtInOuterColor, Type: "JSON",
				Widget: node.WidgetSpec{Kind: "json", Props: node.MarshalProps(node.JSONProps{Rows: 2})}},
			{Name: dcbtInOptions, Type: "JSON",
				Widget: node.WidgetSpec{Kind: "json", Props: node.MarshalProps(node.JSONProps{Rows: 2})}},
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
	clientW, clientH, _ := ctx.Window().ClientSize()
	roi, ok := pickDualBarROI(in.Raw(dcbtInRois), clientW, clientH)
	if !ok {
		ctx.Log().Warn("DualColorBarTrack: no rois entry matches client %dx%d", clientW, clientH)
		return ctx.Out(dcbtOutMissing).Fire(), nil
	}
	inner := parseDualBarHSV(in.JSON(dcbtInInnerColor), defaultInnerHSV)
	outer := parseDualBarHSV(in.JSON(dcbtInOuterColor), defaultOuterHSV)
	opts := parseDualBarOptions(in.JSON(dcbtInOptions))

	result, err := ctx.Vision().DualBarTrack(roi, inner, outer, opts)
	if err != nil {
		return nil, fmt.Errorf("DualColorBarTrack: %w", err)
	}
	if !result.Found || result.InnerX < 0 || result.OuterX < 0 {
		return ctx.Out(dcbtOutMissing).Set(dcbtDataConf, result.Confidence).Fire(), nil
	}
	return ctx.Out(dcbtOutFound).
		Set(dcbtDataInnerX, result.InnerX).
		Set(dcbtDataOuterX, result.OuterX).
		Set(dcbtDataOuterW, result.OuterWidth).
		Set(dcbtDataConf, result.Confidence).
		Set(dcbtDataInnerPx, result.InnerPx).
		Set(dcbtDataOuterPx, result.OuterPx).
		Fire(), nil
}

// 默认 HSV 阈值 (fishing v1 实测): inner=cursor 浅黄, outer=target 高饱和青.
// 通用 case 用户填自己的, 这是 fishing-friendly 默认.
var (
	defaultInnerHSV = node.HSVRange{HMin: 45, HMax: 70, SMin: 40, SMax: 255, VMin: 200, VMax: 255}
	defaultOuterHSV = node.HSVRange{HMin: 160, HMax: 180, SMin: 140, SMax: 255, VMin: 100, VMax: 255}
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

// pickDualBarROI 按 client size 选 ROI. rois 是 []any (JSON 反序列化), 每项
// map[string]any 含 resolution=[W,H] + x/y/w/h. 返 node.Rect.
func pickDualBarROI(raw any, frameW, frameH int) (node.Rect, bool) {
	arr, ok := raw.([]any)
	if !ok {
		return node.Rect{}, false
	}
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		res, ok := m["resolution"].([]any)
		if !ok || len(res) != 2 {
			continue
		}
		w := numAsInt(res[0])
		h := numAsInt(res[1])
		if w != frameW || h != frameH {
			continue
		}
		return node.Rect{
			X: float64(numAsInt(m["x"])),
			Y: float64(numAsInt(m["y"])),
			W: float64(numAsInt(m["w"])),
			H: float64(numAsInt(m["h"])),
		}, true
	}
	return node.Rect{}, false
}

func numAsInt(v any) int {
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

func (DualColorBarTrack) Display(in node.Inputs, exitName string, out node.OutputData) string {
	switch exitName {
	case dcbtOutFound:
		return fmt.Sprintf("✓ inner=%d outer=%d±%d conf=%.2f",
			out.Int(dcbtDataInnerX), out.Int(dcbtDataOuterX), out.Int(dcbtDataOuterW)/2,
			out.Float64(dcbtDataConf))
	case dcbtOutMissing:
		c := out.Float64(dcbtDataConf)
		if c > 0 {
			return fmt.Sprintf("· bar miss conf=%.2f", c)
		}
	}
	return ""
}

func (DualColorBarTrack) Validate(in node.Inputs) []node.ValidationError {
	raw := in.Raw(dcbtInRois)
	if raw == nil {
		return []node.ValidationError{{
			Code:    "INVALID_DUALBAR_ROIS",
			Message: "rois 必须非空数组 [{resolution:[W,H], x,y,w,h}, ...]",
			Field:   dcbtInRois,
		}}
	}
	arr, ok := raw.([]any)
	if !ok || len(arr) == 0 {
		return []node.ValidationError{{
			Code:    "INVALID_DUALBAR_ROIS",
			Message: "rois 必须是非空数组",
			Field:   dcbtInRois,
		}}
	}
	var errs []node.ValidationError
	seen := map[[2]int]bool{}
	for i, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			errs = append(errs, node.ValidationError{
				Code:    "INVALID_DUALBAR_ROIS",
				Message: fmt.Sprintf("rois[%d] 不是对象", i),
				Field:   dcbtInRois,
			})
			continue
		}
		res, ok := m["resolution"].([]any)
		if !ok || len(res) != 2 {
			errs = append(errs, node.ValidationError{
				Code:    "INVALID_DUALBAR_ROIS",
				Message: fmt.Sprintf("rois[%d].resolution 必须 [W,H]", i),
				Field:   dcbtInRois,
			})
			continue
		}
		resW := numAsInt(res[0])
		resH := numAsInt(res[1])
		if resW <= 0 || resH <= 0 {
			errs = append(errs, node.ValidationError{
				Code:    "INVALID_DUALBAR_ROIS",
				Message: fmt.Sprintf("rois[%d].resolution 必须正数 (got %dx%d)", i, resW, resH),
				Field:   dcbtInRois,
			})
			continue
		}
		rw := numAsInt(m["w"])
		rh := numAsInt(m["h"])
		if rw < 1 || rh < 1 {
			errs = append(errs, node.ValidationError{
				Code:    "INVALID_DUALBAR_ROIS",
				Message: fmt.Sprintf("rois[%d] w/h 必须 >= 1 (got %dx%d)", i, rw, rh),
				Field:   dcbtInRois,
			})
			continue
		}
		x := numAsInt(m["x"])
		y := numAsInt(m["y"])
		if x+rw > resW || y+rh > resH {
			errs = append(errs, node.ValidationError{
				Code:    "INVALID_DUALBAR_ROIS",
				Message: fmt.Sprintf("rois[%d] (%d,%d)+%dx%d 超出 %dx%d", i, x, y, rw, rh, resW, resH),
				Field:   dcbtInRois,
			})
			continue
		}
		key := [2]int{resW, resH}
		if seen[key] {
			errs = append(errs, node.ValidationError{
				Code:    "DUPLICATE_DUALBAR_ROI",
				Message: fmt.Sprintf("rois[%d] resolution %dx%d 重复声明", i, resW, resH),
				Field:   dcbtInRois,
			})
		}
		seen[key] = true
	}
	return errs
}
