package input

import (
	"encoding/json"
	"fmt"

	"yhbox/internal/node"
)

func init() { node.Register(&Scroll{}) }

// Scroll 在客户区坐标 (xRatio, yRatio) 处发送鼠标滚轮事件, delta=notches (正向上 / 负向下).
// 老 runtime: internal/services/container/runtime/nodes.go::execScroll.
type Scroll struct{}

const (
	scInExec   = "in"
	scInXRatio = "XRatio"
	scInYRatio = "YRatio"
	scInDelta  = "Delta"
	scOutDone  = "Done"
)

func (Scroll) Spec() node.Spec {
	return node.Spec{
		Kind:        "Scroll",
		Version:     1,
		Category:    "Input",
		DisplayName: "鼠标滚轮",
		Description: "在 (xRatio, yRatio) 客户区坐标发送鼠标滚轮事件. Delta = notches, 正向上 / 负向下.",
		Inputs: []node.InputSpec{
			{Name: scInExec, Type: "Exec"},
			{Name: scInXRatio, Type: "Number", Default: json.Number("0.5"),
				DisplayName: "X 比例",
				Widget: node.WidgetSpec{Kind: "slider",
					Props: node.MarshalProps(node.SliderProps{Min: 0, Max: 1, Step: 0.01})}},
			{Name: scInYRatio, Type: "Number", Default: json.Number("0.5"),
				DisplayName: "Y 比例",
				Widget: node.WidgetSpec{Kind: "slider",
					Props: node.MarshalProps(node.SliderProps{Min: 0, Max: 1, Step: 0.01})}},
			{Name: scInDelta, Type: "Number", Default: json.Number("3"),
				DisplayName: "滚动量 (notches)",
				Doc:         "正值向上滚, 负值向下滚",
				Widget:      node.WidgetSpec{Kind: "number"}},
		},
		Outputs: []node.OutputSpec{
			{Name: scOutDone, Type: "Exec", DisplayName: "完成"},
		},
	}
}

func (Scroll) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	x := in.Float64(scInXRatio)
	y := in.Float64(scInYRatio)
	delta := in.Int(scInDelta)
	if err := ctx.Input().Scroll(x, y, delta); err != nil {
		return nil, fmt.Errorf("Scroll (%.3f,%.3f) Δ=%d: %w", x, y, delta, err)
	}
	return ctx.Out(scOutDone).Fire(), nil
}

func (Scroll) Display(in node.Inputs, exitName string, out node.OutputData) string {
	return fmt.Sprintf("scroll Δ=%d @ (%.2f,%.2f)",
		in.Int(scInDelta), in.Float64(scInXRatio), in.Float64(scInYRatio))
}
