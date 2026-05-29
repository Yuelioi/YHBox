package input

import (
	"encoding/json"
	"fmt"

	"yhbox/internal/node"
)

func init() { node.Register(&MouseMoveRel{}) }

// MouseMoveRel 相对当前光标位置移动 (dx, dy) 像素, 在 durationMs 内插值.
// MouseCalibration 缩放不在节点内做 — 由 InputService 适配层处理.
type MouseMoveRel struct{}

const (
	mmrInExec       = "In"
	mmrInDx         = "Dx"
	mmrInDy         = "Dy"
	mmrInDurationMs = "DurationMs"
	mmrOutDone      = "Done"
)

func (MouseMoveRel) Spec() node.Spec {
	return node.Spec{
		Kind:     "MouseMoveRel",
		Category: "Input",
		Inputs: []node.InputSpec{
			{Name: mmrInExec, Type: "Exec"},
			{Name: mmrInDx, Type: "Number", Default: json.Number("0"),
				Widget: node.WidgetSpec{Kind: "number"}},
			{Name: mmrInDy, Type: "Number", Default: json.Number("0"),
				Widget: node.WidgetSpec{Kind: "number"}},
			{Name: mmrInDurationMs, Type: "Number", Default: json.Number("200"),
				Widget: node.WidgetSpec{Kind: "number"}},
		},
		Outputs: []node.OutputSpec{
			{Name: mmrOutDone, Type: "Exec"},
		},
	}
}

func (MouseMoveRel) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	dx := in.Int(mmrInDx)
	dy := in.Int(mmrInDy)
	dur := in.Int(mmrInDurationMs)
	if err := ctx.Input().MouseMoveRel(dx, dy, dur); err != nil {
		return nil, fmt.Errorf("MouseMoveRel dx=%d dy=%d: %w", dx, dy, err)
	}
	return ctx.Out(mmrOutDone).Fire(), nil
}

func (MouseMoveRel) Display(in node.Inputs, exitName string, out node.OutputData) string {
	return fmt.Sprintf("move Δ(%d,%d) %dms",
		in.Int(mmrInDx), in.Int(mmrInDy), in.Int(mmrInDurationMs))
}
