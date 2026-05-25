package input

import (
	"encoding/json"
	"fmt"

	"yhbox/internal/node"
)

func init() { node.Register(&MouseMoveRel{}) }

// MouseMoveRel 相对当前光标位置移动 (dx, dy) 像素, 在 durationMs 内插值.
// 老 runtime: internal/services/container/runtime/nodes.go::execMouseMoveRel.
//
// 注意: 老节点在 runner 层做 MouseCalibration scale (target/source counts360),
// 该 scaling 是 ContainerRunner 持有 state 的语义, 节点本身不感知 — Phase 5 wire
// 时若需要保留该 calibration, 由 InputService 适配层在 backend wire 处处理.
type MouseMoveRel struct{}

const (
	mmrInExec       = "in"
	mmrInDx         = "Dx"
	mmrInDy         = "Dy"
	mmrInDurationMs = "DurationMs"
	mmrOutDone      = "Done"
)

func (MouseMoveRel) Spec() node.Spec {
	return node.Spec{
		Kind:        "MouseMoveRel",
		Category:    "Input",
		DisplayName: "鼠标相对移动",
		Description: "相对当前光标位置移动 (dx, dy) 像素, 在 DurationMs 内插值.",
		Inputs: []node.InputSpec{
			{Name: mmrInExec, Type: "Exec"},
			{Name: mmrInDx, Type: "Number", Default: json.Number("0"),
				DisplayName: "Δx (px)",
				Widget:      node.WidgetSpec{Kind: "number"}},
			{Name: mmrInDy, Type: "Number", Default: json.Number("0"),
				DisplayName: "Δy (px)",
				Widget:      node.WidgetSpec{Kind: "number"}},
			{Name: mmrInDurationMs, Type: "Number", Default: json.Number("200"),
				DisplayName: "时长 (ms)",
				Widget:      node.WidgetSpec{Kind: "number"}},
		},
		Outputs: []node.OutputSpec{
			{Name: mmrOutDone, Type: "Exec", DisplayName: "完成"},
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
