// internal/nodes/detect/detect_color.go
// DetectColor — 在 region 内统计落在颜色范围内的像素数 (RGB 或 HSV 模式).
// 跟 DetectColorHSV 区别: DetectColor 用 ratio region + CSV-like range, 单次扫描无轮询.
package detect

import (
	"encoding/json"
	"fmt"

	"yotta/internal/node"
)

func init() { node.Register(&DetectColor{}) }

type DetectColor struct{}

const (
	dcInExec      = "In"
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
		NeedsWindow: true,
		Inputs: []node.InputSpec{
			{Name: dcInExec, Type: "Exec"},
			{Name: dcInRegion, Type: "Geometry", Schema: node.GeometrySchema()},
			{Name: dcInMode, Type: "String", Default: "hsv",
				Widget: node.WidgetSpec{Kind: "dropdown",
					Props: node.MarshalProps(node.DropdownProps{
						Options: []node.EnumOption{
							{Value: "hsv"},
							{Value: "rgb"},
						}})}},
			{Name: dcInRange, Type: "JSON",
				Widget: node.WidgetSpec{Kind: "json",
					Props: node.MarshalProps(node.JSONProps{Rows: 2})}},
			{Name: dcInMinPixels, Type: "Number", Default: json.Number("5"),
				Widget: node.WidgetSpec{Kind: "number"}},
		},
		Outputs: []node.OutputSpec{
			{Name: dcOutYes, Type: "Exec",
				Data: []node.DataField{
					{Name: dcDataCount, Type: "Number"},
					{Name: dcDataCenter, Type: "Point"},
				}},
			{Name: dcOutNo, Type: "Exec",
				Data: []node.DataField{
					{Name: dcDataCount, Type: "Number"},
				}},
		},
	}
}

func (DetectColor) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	geo := in.Geometry(dcInRegion)
	mode := in.String(dcInMode)
	if mode == "" {
		mode = "hsv"
	}
	rngArr, err := parseRange6(in.Raw(dcInRange))
	if err != nil {
		return nil, fmt.Errorf("DetectColor range: %w", err)
	}
	minPx := in.Int(dcInMinPixels)

	count, cx, cy, err := ctx.Vision().DetectColor(geo, mode, rngArr)
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
