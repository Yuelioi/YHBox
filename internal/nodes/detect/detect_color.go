// internal/nodes/detect/detect_color.go
// DetectColor — 在 region 内统计落在颜色范围内的像素数 (RGB 或 HSV 模式).
// 跟 DetectColorHSV 区别: DetectColor 用 ratio region + CSV-like range, 单次扫描无轮询,
// 主要服务老钓鱼 "看状态栏有没有 X 颜色" 场景.
//
// 老 runtime: internal/services/container/runtime/nodes.go::execDetectColor.
package detect

import (
	"encoding/json"
	"fmt"

	"yhbox/internal/node"
)

func init() { node.Register(&DetectColor{}) }

type DetectColor struct{}

const (
	dcInExec      = "in"
	dcInRegion    = "Region"
	dcInMode      = "Mode"
	dcInRange     = "Range"
	dcInMinPixels = "MinPixels"
	dcOutYes      = "Yes"
	dcOutNo       = "No"
	dcDataCount   = "Count"
	dcDataCenter  = "Center"
)

func (DetectColor) Spec() node.Spec {
	return node.Spec{
		Kind:        "DetectColor",
		Category:    "Detect",
		DisplayName: "颜色检测",
		Description: "在 region (ratio) 内统计落在颜色范围内的像素. count >= minPixels → Yes.",
		Inputs: []node.InputSpec{
			{Name: dcInExec, Type: "Exec"},
			{Name: dcInRegion, Type: "Rect", DisplayName: "区域 (ratio)",
				Doc:    "客户区 ratio 矩形, 全 0 = 全屏",
				Widget: node.WidgetSpec{Kind: "rect-editor"}},
			{Name: dcInMode, Type: "String", Default: "hsv", DisplayName: "模式",
				Widget: node.WidgetSpec{Kind: "dropdown",
					Props: node.MarshalProps(node.DropdownProps{
						Options: []node.EnumOption{
							{Value: "hsv", Label: "HSV"},
							{Value: "rgb", Label: "RGB"},
						}})}},
			{Name: dcInRange, Type: "JSON", DisplayName: "范围 [aMin..vMax]",
				Doc: "6 元素: hsv=[hMin,hMax,sMin,sMax,vMin,vMax] / rgb=[rMin..bMax]",
				Widget: node.WidgetSpec{Kind: "json",
					Props: node.MarshalProps(node.JSONProps{Rows: 2})}},
			{Name: dcInMinPixels, Type: "Number", Default: json.Number("5"),
				DisplayName: "最小像素", Widget: node.WidgetSpec{Kind: "number"}},
		},
		Outputs: []node.OutputSpec{
			{Name: dcOutYes, Type: "Exec", DisplayName: "命中",
				Data: []node.DataField{
					{Name: dcDataCount, Type: "Number", Doc: "命中像素数"},
					{Name: dcDataCenter, Type: "Point", Doc: "命中中心 (ratio)"},
				}},
			{Name: dcOutNo, Type: "Exec", DisplayName: "未命中",
				Data: []node.DataField{
					{Name: dcDataCount, Type: "Number", Doc: "命中像素数"},
				}},
		},
	}
}

func (DetectColor) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	region := in.Rect(dcInRegion)
	mode := in.String(dcInMode)
	if mode == "" {
		mode = "hsv"
	}
	rngArr, err := parseRange6(in.Raw(dcInRange))
	if err != nil {
		return nil, fmt.Errorf("DetectColor range: %w", err)
	}
	minPx := in.Int(dcInMinPixels)

	regionArr := [4]float64{region.X, region.Y, region.W, region.H}
	count, cx, cy, err := ctx.Vision().DetectColor(regionArr, mode, rngArr)
	if err != nil {
		return nil, fmt.Errorf("DetectColor: %w", err)
	}
	if count >= minPx {
		return ctx.Out(dcOutYes).
			Set(dcDataCount, count).
			Set(dcDataCenter, node.Point{X: cx, Y: cy}).
			Fire(), nil
	}
	return ctx.Out(dcOutNo).Set(dcDataCount, count).Fire(), nil
}

// parseRange6 把 Raw any (JSON [Number,...] / [int...] / [float64...]) 转 [6]int.
// 长度不足填 0, 多余截断. 非数字位置返 error.
func parseRange6(raw any) ([6]int, error) {
	var out [6]int
	if raw == nil {
		return out, nil
	}
	arr, ok := raw.([]any)
	if !ok {
		return out, fmt.Errorf("range must be array of 6 numbers, got %T", raw)
	}
	for i := 0; i < 6 && i < len(arr); i++ {
		switch v := arr[i].(type) {
		case float64:
			out[i] = int(v)
		case int:
			out[i] = v
		case json.Number:
			n, err := v.Int64()
			if err != nil {
				f, ferr := v.Float64()
				if ferr != nil {
					return out, fmt.Errorf("range[%d]: %v", i, err)
				}
				out[i] = int(f)
			} else {
				out[i] = int(n)
			}
		default:
			return out, fmt.Errorf("range[%d]: not a number (%T)", i, arr[i])
		}
	}
	return out, nil
}

func (DetectColor) Display(in node.Inputs, exitName string, out node.OutputData) string {
	switch exitName {
	case dcOutYes:
		c := out.Point(dcDataCenter)
		return fmt.Sprintf("✓ %s count=%d @ (%.2f,%.2f)",
			in.String(dcInMode), out.Int(dcDataCount), c.X, c.Y)
	case dcOutNo:
		return fmt.Sprintf("· %s count=%d", in.String(dcInMode), out.Int(dcDataCount))
	}
	return ""
}

func (DetectColor) Validate(in node.Inputs) []node.ValidationError {
	mode := in.String(dcInMode)
	if mode != "" && mode != "hsv" && mode != "rgb" {
		return []node.ValidationError{{
			Code:    "INVALID_COLOR_MODE",
			Message: fmt.Sprintf("mode %q not in hsv/rgb", mode),
			Field:   dcInMode,
		}}
	}
	return nil
}
