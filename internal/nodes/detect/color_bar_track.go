// internal/nodes/detect/color_bar_track.go
// ColorBarTrack — 在多分辨率 ROI 数组中按当前 client size 选 ROI, 抓帧 → 双 HSV cluster
// 检测算 cursor / target 位置. 单次抓帧 + 分析.
package detect

import (
	"encoding/json"
	"fmt"

	"yhbox/internal/node"
)

func init() { node.Register(&ColorBarTrack{}) }

type ColorBarTrack struct{}

const (
	cbtInExec        = "in"
	cbtInRois        = "Rois"
	cbtOutFound      = "Found"
	cbtOutMissing    = "Missing"
	cbtDataCursorX   = "CursorX"
	cbtDataTargetX   = "TargetX"
	cbtDataTargetW   = "TargetW"
	cbtDataConf      = "Confidence"
	cbtDataYellowPx  = "YellowPx"
	cbtDataGreenPx   = "GreenPx"

	// confBarV2 — vision.AnalyzeBar 算法置信度阈值, 复用 runtime/color_bar_track.go::confBarV2.
	cbtConfBarThreshold = 0.50
)

func (ColorBarTrack) Spec() node.Spec {
	return node.Spec{
		Kind:        "ColorBarTrack",
		Category:    "Detect",
		DisplayName: "颜色条追踪",
		Description: "多分辨率 ROI 内 HSV cluster 算 cursor + target 位置 (钓鱼溜鱼专用). rois=[{resolution:[W,H], x,y,w,h}, ...]. 当前 client size 没匹配项走 Missing.",
		Inputs: []node.InputSpec{
			{Name: cbtInExec, Type: "Exec"},
			{Name: cbtInRois, Type: "JSON", Required: true, DisplayName: "ROI 数组 (多分辨率)",
				Doc: `[{"resolution":[1920,1080],"x":576,"y":594,"w":768,"h":54}, ...]`,
				Widget: node.WidgetSpec{Kind: "json",
					Props: node.MarshalProps(node.JSONProps{Rows: 4})}},
		},
		Outputs: []node.OutputSpec{
			{Name: cbtOutFound, Type: "Exec", DisplayName: "命中",
				Data: []node.DataField{
					{Name: cbtDataCursorX, Type: "Number"},
					{Name: cbtDataTargetX, Type: "Number"},
					{Name: cbtDataTargetW, Type: "Number"},
					{Name: cbtDataConf, Type: "Number"},
					{Name: cbtDataYellowPx, Type: "Number"},
					{Name: cbtDataGreenPx, Type: "Number"},
				}},
			{Name: cbtOutMissing, Type: "Exec", DisplayName: "未命中",
				Data: []node.DataField{
					{Name: cbtDataConf, Type: "Number", Optional: true},
				}},
		},
	}
}

func (ColorBarTrack) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	clientW, clientH, _ := ctx.Window().ClientSize()
	roi, ok := pickColorBarROI(in.Raw(cbtInRois), clientW, clientH)
	if !ok {
		// rois 没匹配当前 frame size → Missing. Warning 节流 emit 留给 framework log adapter (Phase 5).
		ctx.Log().Warn("ColorBarTrack: no rois entry matches client %dx%d", clientW, clientH)
		return ctx.Out(cbtOutMissing).Fire(), nil
	}
	result, err := ctx.Vision().BarTrack(roi)
	if err != nil {
		return nil, fmt.Errorf("ColorBarTrack: %w", err)
	}
	if !result.Found || result.CursorX < 0 || result.TargetX < 0 || result.Confidence < cbtConfBarThreshold {
		return ctx.Out(cbtOutMissing).Set(cbtDataConf, result.Confidence).Fire(), nil
	}
	return ctx.Out(cbtOutFound).
		Set(cbtDataCursorX, result.CursorX).
		Set(cbtDataTargetX, result.TargetX).
		Set(cbtDataTargetW, result.TargetW).
		Set(cbtDataConf, result.Confidence).
		Set(cbtDataYellowPx, result.YellowPx).
		Set(cbtDataGreenPx, result.GreenPx).
		Fire(), nil
}

// pickColorBarROI 复刻老 runtime pickROIByResolution. rois 是 []any (JSON 反序列化),
// 每项 map[string]any 含 resolution=[W,H] + x/y/w/h. 返 node.Rect (pixel 坐标转 float64).
func pickColorBarROI(raw any, frameW, frameH int) (node.Rect, bool) {
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

func (ColorBarTrack) Display(in node.Inputs, exitName string, out node.OutputData) string {
	switch exitName {
	case cbtOutFound:
		return fmt.Sprintf("✓ cursor=%d target=%d±%d conf=%.2f",
			out.Int(cbtDataCursorX), out.Int(cbtDataTargetX), out.Int(cbtDataTargetW)/2,
			out.Float64(cbtDataConf))
	case cbtOutMissing:
		c := out.Float64(cbtDataConf)
		if c > 0 {
			return fmt.Sprintf("· bar miss conf=%.2f", c)
		}
	}
	return ""
}

func (ColorBarTrack) Validate(in node.Inputs) []node.ValidationError {
	raw := in.Raw(cbtInRois)
	if raw == nil {
		return []node.ValidationError{{
			Code:    "INVALID_COLORBAR_ROIS",
			Message: "rois 必须非空数组 [{resolution:[W,H], x,y,w,h}, ...]",
			Field:   cbtInRois,
		}}
	}
	arr, ok := raw.([]any)
	if !ok || len(arr) == 0 {
		return []node.ValidationError{{
			Code:    "INVALID_COLORBAR_ROIS",
			Message: "rois 必须是非空数组",
			Field:   cbtInRois,
		}}
	}
	var errs []node.ValidationError
	seen := map[[2]int]bool{}
	for i, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			errs = append(errs, node.ValidationError{
				Code:    "INVALID_COLORBAR_ROIS",
				Message: fmt.Sprintf("rois[%d] 不是对象", i),
				Field:   cbtInRois,
			})
			continue
		}
		res, ok := m["resolution"].([]any)
		if !ok || len(res) != 2 {
			errs = append(errs, node.ValidationError{
				Code:    "INVALID_COLORBAR_ROIS",
				Message: fmt.Sprintf("rois[%d].resolution 必须 [W,H]", i),
				Field:   cbtInRois,
			})
			continue
		}
		resW := numAsInt(res[0])
		resH := numAsInt(res[1])
		if resW <= 0 || resH <= 0 {
			errs = append(errs, node.ValidationError{
				Code:    "INVALID_COLORBAR_ROIS",
				Message: fmt.Sprintf("rois[%d].resolution 必须正数 (got %dx%d)", i, resW, resH),
				Field:   cbtInRois,
			})
			continue
		}
		rw := numAsInt(m["w"])
		rh := numAsInt(m["h"])
		if rw < 1 || rh < 1 {
			errs = append(errs, node.ValidationError{
				Code:    "INVALID_COLORBAR_ROIS",
				Message: fmt.Sprintf("rois[%d] w/h 必须 >= 1 (got %dx%d)", i, rw, rh),
				Field:   cbtInRois,
			})
			continue
		}
		x := numAsInt(m["x"])
		y := numAsInt(m["y"])
		if x+rw > resW || y+rh > resH {
			errs = append(errs, node.ValidationError{
				Code:    "INVALID_COLORBAR_ROIS",
				Message: fmt.Sprintf("rois[%d] (%d,%d)+%dx%d 超出 %dx%d", i, x, y, rw, rh, resW, resH),
				Field:   cbtInRois,
			})
			continue
		}
		key := [2]int{resW, resH}
		if seen[key] {
			errs = append(errs, node.ValidationError{
				Code:    "DUPLICATE_COLORBAR_ROI",
				Message: fmt.Sprintf("rois[%d] resolution %dx%d 重复声明", i, resW, resH),
				Field:   cbtInRois,
			})
		}
		seen[key] = true
	}
	return errs
}
