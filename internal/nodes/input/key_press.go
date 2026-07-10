// Package input 输入节点 — 模拟键鼠操作 (KeyPress / ClickAt / KeyHold / MouseHold / Scroll / etc).
// 每节点 1 文件 1 const block, 无共享 helper (跟 detect/control 风格一致).
package input

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/yottaapp/yotta/internal/node"
	pkginput "github.com/yottaapp/yotta/pkg/input"
)

func init() { node.Register(&KeyPress{}) }

// KeyPress 按下并松开一个虚拟键. duration 控制按下到松开的间隔.
type KeyPress struct{}

const (
	kpInExec       = "In"
	kpInVK         = "VK"
	kpInDurationMs = "DurationMs"
	kpInJitterPct  = "JitterPct"
	kpOutDone      = "Done"
)

func (KeyPress) Spec() node.Spec {
	return node.Spec{
		Kind:        "KeyPress",
		Category:    "Input",
		NeedsTarget: true,
		TargetCapabilities: []node.TargetCapability{
			node.TargetCapabilityKeyState,
		},
		NeedsForeground: true,
		Inputs: append([]node.InputSpec{
			{Name: kpInExec, Type: "Exec"},
			{Name: kpInVK, Type: "String", Required: true, Default: "W",
				Widget: node.WidgetSpec{Kind: "key-capture"}},
			{Name: kpInDurationMs, Type: "Number", Default: json.Number("50"),
				Widget: node.WidgetSpec{Kind: "number"}},
			{Name: kpInJitterPct, Type: "Number", Default: json.Number("0"),
				Widget: node.WidgetSpec{Kind: "number"}},
		}, node.WindowInputSpec()),
		Outputs: []node.OutputSpec{
			{Name: kpOutDone, Type: "Exec"},
		},
	}
}

func (KeyPress) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	vk := in.String(kpInVK)
	dur := in.Int(kpInDurationMs)
	if dur <= 0 {
		dur = 50 // 默认 50ms 按压
	}
	dur = node.JitterInt(dur, in.Int(kpInJitterPct)) // ±% 近正态抖动 (pct=0 → 原值)
	if err := ctx.Input().KeyDown(vk); err != nil {
		return nil, node.Failf(node.CodeSendFailed, err, "KeyPress keydown vk=%q: %v", vk, err)
	}
	// 可取消按住: stop/强停 cancel ctx 时立即松键返回, 不死等 dur
	// (修「长按 999999ms 停不下来」—— 裸 time.Sleep 不看 ctx 的旧 bug)。
	select {
	case <-ctx.Context().Done():
		_ = ctx.Input().KeyUp(vk) // 释放, 别留残留按键
		return nil, ctx.Context().Err()
	case <-time.After(time.Duration(dur) * time.Millisecond):
	}
	if err := ctx.Input().KeyUp(vk); err != nil {
		return nil, node.Failf(node.CodeSendFailed, err, "KeyPress keyup vk=%q: %v", vk, err)
	}
	return ctx.Out(kpOutDone).Fire(), nil
}

// Validate vk 必须非空, 且为已知 keyname.
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
