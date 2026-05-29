// Package input 输入节点 — 模拟键鼠操作 (KeyPress / ClickAt / KeyHold / MouseHold / Scroll / etc).
// 每节点 1 文件 1 const block, 无共享 helper (跟 detect/control 风格一致).
package input

import (
	"encoding/json"
	"fmt"

	"yhbox/internal/node"
	pkginput "yhbox/pkg/input"
)

func init() { node.Register(&KeyPress{}) }

// KeyPress 按下并松开一个虚拟键. duration 控制按下到松开的间隔.
type KeyPress struct{}

const (
	kpInExec       = "In"
	kpInVK         = "VK"
	kpInDurationMs = "DurationMs"
	kpOutDone      = "Done"
)

func (KeyPress) Spec() node.Spec {
	return node.Spec{
		Kind:     "KeyPress",
		Category: "Input",
		Inputs: []node.InputSpec{
			{Name: kpInExec, Type: "Exec"},
			{Name: kpInVK, Type: "String", Required: true, Default: "W",
				Widget: node.WidgetSpec{Kind: "text"}},
			{Name: kpInDurationMs, Type: "Number", Default: json.Number("50"),
				Widget: node.WidgetSpec{Kind: "number"}},
		},
		Outputs: []node.OutputSpec{
			{Name: kpOutDone, Type: "Exec"},
		},
	}
}

func (KeyPress) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	vk := in.String(kpInVK)
	dur := in.Int(kpInDurationMs)
	if err := ctx.Input().KeyPress(vk, dur); err != nil {
		return nil, fmt.Errorf("KeyPress vk=%q: %w", vk, err)
	}
	return ctx.Out(kpOutDone).Fire(), nil
}

func (KeyPress) Display(in node.Inputs, exitName string, out node.OutputData) string {
	return fmt.Sprintf("key %s (%dms)", in.String(kpInVK), in.Int(kpInDurationMs))
}

// Validate vk 必须非空, 且为已知 keyname (复刻老 resolveVK 校验).
func (KeyPress) Validate(in node.Inputs) []node.ValidationError {
	vk := in.String(kpInVK)
	if vk == "" {
		return []node.ValidationError{{
			Code:    "INVALID_VK",
			Message: "vk required (string, e.g. 'A' / 'space' / 'F9')",
			Field:   kpInVK,
		}}
	}
	if pkginput.VK(vk) == 0 {
		return []node.ValidationError{{
			Code:    "INVALID_VK",
			Message: fmt.Sprintf("vk: unknown keyname %q", vk),
			Field:   kpInVK,
		}}
	}
	return nil
}
