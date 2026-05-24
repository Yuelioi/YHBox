package input

import (
	"fmt"

	"yhbox/internal/node"
	pkginput "yhbox/pkg/input"
)

func init() {
	node.Register(&KeyHoldStart{})
	node.Register(&KeyHoldStop{})
}

// KeyHoldStart 按下虚拟键 (不松开). 配对 KeyHoldStop. 老 runtime: hold_nodes.go::execKeyHoldStart.
//
// 跟 KeyPress 不同, KeyHoldStart 不在节点内闭环 — 允许在 Start/Stop 之间插任意流程
// (Sleep / WaitTemplate / 子图调用 / ...). Backend.KeyDown/KeyUp 是 stateful, 配 ReleaseAll
// 兜底, 所以 runner cancel/panic 不会留残留按键 — 详见 pkg/input/backend.go.
type KeyHoldStart struct{}

const (
	khStartInExec = "in"
	khStartInVK   = "VK"
	khStartOutOut = "Out"
)

func (KeyHoldStart) Spec() node.Spec {
	return node.Spec{
		Kind:        "KeyHoldStart",
		Version:     1,
		Category:    "Input",
		DisplayName: "按住按键",
		Description: "按下虚拟键 (不松开). 配对 KeyHoldStop 节点释放. 之间可插任意流程.",
		Inputs: []node.InputSpec{
			{Name: khStartInExec, Type: "Exec"},
			{Name: khStartInVK, Type: "String", Required: true, Default: "A",
				DisplayName: "按键",
				Doc:         "虚拟键名 (e.g. A / W / shift)",
				Widget:      node.WidgetSpec{Kind: "text"}},
		},
		Outputs: []node.OutputSpec{
			{Name: khStartOutOut, Type: "Exec", DisplayName: "已按下"},
		},
	}
}

func (KeyHoldStart) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	vk := in.String(khStartInVK)
	if err := ctx.Input().KeyDown(vk); err != nil {
		return nil, fmt.Errorf("KeyHoldStart vk=%q: %w", vk, err)
	}
	return ctx.Out(khStartOutOut).Fire(), nil
}

func (KeyHoldStart) Display(in node.Inputs, exitName string, out node.OutputData) string {
	return fmt.Sprintf("hold ▼ %s", in.String(khStartInVK))
}

func (KeyHoldStart) Validate(in node.Inputs) []node.ValidationError {
	return validateVKField(in.String(khStartInVK), khStartInVK)
}

// KeyHoldStop 松开虚拟键 (跟 KeyHoldStart 配对). 老 runtime: hold_nodes.go::execKeyHoldStop.
type KeyHoldStop struct{}

const (
	khStopInExec = "in"
	khStopInVK   = "VK"
	khStopOutOut = "Out"
)

func (KeyHoldStop) Spec() node.Spec {
	return node.Spec{
		Kind:        "KeyHoldStop",
		Version:     1,
		Category:    "Input",
		DisplayName: "松开按键",
		Description: "松开虚拟键, 配对 KeyHoldStart.",
		Inputs: []node.InputSpec{
			{Name: khStopInExec, Type: "Exec"},
			{Name: khStopInVK, Type: "String", Required: true, Default: "A",
				DisplayName: "按键",
				Doc:         "虚拟键名 — 跟之前 KeyHoldStart 同一个",
				Widget:      node.WidgetSpec{Kind: "text"}},
		},
		Outputs: []node.OutputSpec{
			{Name: khStopOutOut, Type: "Exec", DisplayName: "已松开"},
		},
	}
}

func (KeyHoldStop) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	vk := in.String(khStopInVK)
	if err := ctx.Input().KeyUp(vk); err != nil {
		return nil, fmt.Errorf("KeyHoldStop vk=%q: %w", vk, err)
	}
	return ctx.Out(khStopOutOut).Fire(), nil
}

func (KeyHoldStop) Display(in node.Inputs, exitName string, out node.OutputData) string {
	return fmt.Sprintf("hold ▲ %s", in.String(khStopInVK))
}

func (KeyHoldStop) Validate(in node.Inputs) []node.ValidationError {
	return validateVKField(in.String(khStopInVK), khStopInVK)
}

// validateVKField 复刻老 validator.go::validateKeyHold + hold_nodes.go::resolveVK:
//   - 空 → INVALID_VK
//   - 未知 keyname (pkg/input.VK(name)==0) → INVALID_VK
//
// 本 helper 仅本文件内 Start/Stop 复用 (1 个 Validate 字段对 2 个节点), 不算"abstract".
func validateVKField(vk, field string) []node.ValidationError {
	if vk == "" {
		return []node.ValidationError{{
			Code:    "INVALID_VK",
			Message: "vk required (string, e.g. 'A' / 'space' / 'F9')",
			Field:   field,
		}}
	}
	if pkginput.VK(vk) == 0 {
		return []node.ValidationError{{
			Code:    "INVALID_VK",
			Message: fmt.Sprintf("vk: unknown keyname %q", vk),
			Field:   field,
		}}
	}
	return nil
}
