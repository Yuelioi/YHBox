package input

import (
	"encoding/json"

	"yotta/internal/node"
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
	mmrInJitterPct  = "JitterPct"
	mmrOutDone      = "Done"
)

func (MouseMoveRel) Spec() node.Spec {
	return node.Spec{
		Kind:        "MouseMoveRel",
		Category:    "Input",
		NeedsWindow: true,
		Inputs: []node.InputSpec{
			{Name: mmrInExec, Type: "Exec"},
			{Name: mmrInDx, Type: "Number", Default: json.Number("0"),
				Widget: node.WidgetSpec{Kind: "number"}},
			{Name: mmrInDy, Type: "Number", Default: json.Number("0"),
				Widget: node.WidgetSpec{Kind: "number"}},
			{Name: mmrInDurationMs, Type: "Number", Default: json.Number("200"),
				Widget: node.WidgetSpec{Kind: "number"}},
			{Name: mmrInJitterPct, Type: "Number", Default: json.Number("0"),
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
	pct := in.Int(mmrInJitterPct)
	dx = node.JitterInt(dx, pct) // ±% 抖移动距离 (pct=0 → 原值)
	dy = node.JitterInt(dy, pct)
	if err := ctx.Input().MouseMoveRel(dx, dy, dur); err != nil {
		return nil, node.Failf(node.CodeSendFailed, err, "MouseMoveRel dx=%d dy=%d: %v", dx, dy, err)
	}
	return ctx.Out(mmrOutDone).Fire(), nil
}
