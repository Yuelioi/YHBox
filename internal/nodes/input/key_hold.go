package input

import (
	"fmt"

	"github.com/yottaapp/yotta/internal/node"
	pkginput "github.com/yottaapp/yotta/pkg/input"
)

func init() {
	node.Register(&KeyHoldStart{})
	node.Register(&KeyHoldStop{})
}

// KeyHoldStart 按下虚拟键 (不松开). 配对 KeyHoldStop.
//
// 跟 KeyPress 不同, KeyHoldStart 不在节点内闭环 — 允许在 Start/Stop 之间插任意流程
// (Sleep / WaitTemplate / 子图调用 / ...). Backend.KeyDown/KeyUp 是 stateful, 配 ReleaseAll
// 兜底, 所以 runner cancel/panic 不会留残留按键.
type KeyHoldStart struct{}

const (
	khStartInExec = "In"
	khStartInVK   = "VK"
	khStartOutOut = "Done"
)

func (KeyHoldStart) Spec() node.Spec {
	return node.Spec{
		Kind:                "KeyHoldStart",
		RuntimeCapabilities: []node.RuntimeCapability{node.RuntimeCapabilityInput},
		Category:            "Input",
		NeedsTarget:         true,
		TargetCapabilities: []node.TargetCapability{
			node.TargetCapabilityKeyState,
		},
		NeedsForeground: true,
		Inputs: append([]node.InputSpec{
			{Name: khStartInExec, Type: "Exec"},
			{Name: khStartInVK, Type: "String", Required: true, Default: "A",
				Widget: node.WidgetSpec{Kind: "key-capture"}},
		}, node.WindowInputSpec()),
		Outputs: []node.OutputSpec{
			{Name: khStartOutOut, Type: "Exec"},
		},
	}
}

func (KeyHoldStart) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	vk := in.String(khStartInVK)
	if err := ctx.Services().Input.KeyDown(vk); err != nil {
		return nil, node.Failf(node.CodeSendFailed, err, "KeyHoldStart vk=%q: %v", vk, err)
	}
	return ctx.Out(khStartOutOut).Fire(), nil
}

func (KeyHoldStart) Validate(in node.Inputs) []node.ValidationError {
	return validateVKField(in.String(khStartInVK), khStartInVK)
}

// KeyHoldStop 松开虚拟键 (跟 KeyHoldStart 配对).
type KeyHoldStop struct{}

const (
	khStopInExec = "In"
	khStopInVK   = "VK"
	khStopOutOut = "Done"
)

func (KeyHoldStop) Spec() node.Spec {
	return node.Spec{
		Kind:                "KeyHoldStop",
		RuntimeCapabilities: []node.RuntimeCapability{node.RuntimeCapabilityInput},
		Category:            "Input",
		NeedsTarget:         true,
		TargetCapabilities: []node.TargetCapability{
			node.TargetCapabilityKeyState,
		},
		NeedsForeground: true,
		Inputs: append([]node.InputSpec{
			{Name: khStopInExec, Type: "Exec"},
			{Name: khStopInVK, Type: "String", Required: true, Default: "A",
				Widget: node.WidgetSpec{Kind: "key-capture"}},
		}, node.WindowInputSpec()),
		Outputs: []node.OutputSpec{
			{Name: khStopOutOut, Type: "Exec"},
		},
	}
}

func (KeyHoldStop) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	vk := in.String(khStopInVK)
	if err := ctx.Services().Input.KeyUp(vk); err != nil {
		return nil, node.Failf(node.CodeSendFailed, err, "KeyHoldStop vk=%q: %v", vk, err)
	}
	return ctx.Out(khStopOutOut).Fire(), nil
}

func (KeyHoldStop) Validate(in node.Inputs) []node.ValidationError {
	return validateVKField(in.String(khStopInVK), khStopInVK)
}

// validateVKField vk 字段校验:
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
