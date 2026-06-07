package input

import (
	"encoding/json"

	"yotta/internal/node"
)

func init() { node.Register(&Scroll{}) }

// Scroll 在客户区坐标 (xRatio, yRatio) 处发送鼠标滚轮事件, delta=notches (正向上 / 负向下).
type Scroll struct{}

const (
	scInExec      = "In"
	scInXRatio    = "XRatio"
	scInYRatio    = "YRatio"
	scInDelta     = "Delta"
	scInJitterPct = "JitterPct"
	scOutDone     = "Done"
)

func (Scroll) Spec() node.Spec {
	return node.Spec{
		Kind:        "Scroll",
		Category:    "Input",
		NeedsWindow: true,
		Inputs: []node.InputSpec{
			{Name: scInExec, Type: "Exec"},
			{Name: scInXRatio, Type: "Number", Default: json.Number("0.5"),
				Widget: node.WidgetSpec{Kind: "slider",
					Props: node.MarshalProps(node.SliderProps{Min: 0, Max: 1, Step: 0.01})}},
			{Name: scInYRatio, Type: "Number", Default: json.Number("0.5"),
				Widget: node.WidgetSpec{Kind: "slider",
					Props: node.MarshalProps(node.SliderProps{Min: 0, Max: 1, Step: 0.01})}},
			{Name: scInDelta, Type: "Number", Default: json.Number("3"),
				Widget: node.WidgetSpec{Kind: "number"}},
			{Name: scInJitterPct, Type: "Number", Default: json.Number("0"),
				Widget: node.WidgetSpec{Kind: "number"}},
		},
		Outputs: []node.OutputSpec{
			{Name: scOutDone, Type: "Exec"},
		},
	}
}

func (Scroll) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	x := in.Float64(scInXRatio)
	y := in.Float64(scInYRatio)
	delta := in.Int(scInDelta)
	delta = node.JitterInt(delta, in.Int(scInJitterPct)) // ±% 抖滚动量 (pct=0 → 原值)
	if err := ctx.Input().Scroll(x, y, delta); err != nil {
		return nil, node.Failf(node.CodeSendFailed, err, "Scroll (%.3f,%.3f) Δ=%d: %v", x, y, delta, err)
	}
	return ctx.Out(scOutDone).Fire(), nil
}
