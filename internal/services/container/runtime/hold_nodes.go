package runtime

import (
	"context"
	"fmt"

	"github.com/lxn/win"

	"yhbox/internal/services/container"
	pkginput "yhbox/pkg/input"
)

// ---- KeyHold / MouseHold (v3 Phase C) ----
//
// 4 节点: KeyHoldStart / KeyHoldStop / MouseHoldStart / MouseHoldStop.
// 解耦 down/up — 跟 KeyPress 不同, 不在单节点内闭环, 允许用户在 down/up 之间插
// Sleep / WaitTemplate / 任意流程. Backend.KeyDown/Up + MouseDown/Up 是 stateful
// (ReleaseAll 兜底), 所以 runner cancel/panic 不会留残留 — 详见 pkg/input/backend.go.
//
// vk 配置同 KeyPress (string "A"/"space"/"F9"), 但 resolveVK 兼容 numeric
// 输入 (65 → "A") 方便 power user / programmatic config.

// resolveVK 把 Config["vk"] 标准化成 Backend.KeyDown/Up 吃的 string keyname.
// 接受 string passthrough (validator + VK() 双校验) 或 numeric vk code (反查).
func resolveVK(raw any) (string, error) {
	switch v := raw.(type) {
	case string:
		if v == "" {
			return "", fmt.Errorf("vk: empty string")
		}
		if pkginput.VK(v) == 0 {
			return "", fmt.Errorf("vk: unknown keyname %q", v)
		}
		return v, nil
	case float64:
		return vkCodeToName(int(v))
	case int:
		return vkCodeToName(v)
	case nil:
		return "", fmt.Errorf("vk: nil")
	default:
		return "", fmt.Errorf("vk: unsupported type %T", raw)
	}
}

// vkCodeToName 把 numeric vk code 转 string keyname (Backend 吃 string).
// 覆盖: A-Z / 0-9 / F1-F12 / 常见 special (esc/space/enter/...).
// 不在覆盖范围的 code 直接 error — 保持显式, 不静默丢失.
func vkCodeToName(code int) (string, error) {
	if code < 1 || code > 254 {
		return "", fmt.Errorf("vk: %d out of [1,254]", code)
	}
	// ASCII letters (VK_A=0x41 ... VK_Z=0x5A)
	if code >= 'A' && code <= 'Z' {
		return string(rune(code)), nil
	}
	// ASCII digits (VK_0=0x30 ... VK_9=0x39)
	if code >= '0' && code <= '9' {
		return string(rune(code)), nil
	}
	// F1-F12 (VK_F1=0x70 ... VK_F12=0x7B)
	if code >= 0x70 && code <= 0x7B {
		return fmt.Sprintf("F%d", code-0x70+1), nil
	}
	// 反查 pkg/input vkMap 里的 special keys. 重复一份避免 export pkg/input 内部表.
	switch code {
	case 0x1B:
		return "esc", nil
	case 0x20:
		return "space", nil
	case 0x0D:
		return "enter", nil
	case 0x10:
		return "shift", nil
	case 0x11:
		return "ctrl", nil
	case 0x12:
		return "alt", nil
	case 0x09:
		return "tab", nil
	case 0x08:
		return "backspace", nil
	case 0x2E:
		return "delete", nil
	case 0x2D:
		return "insert", nil
	case 0x24:
		return "home", nil
	case 0x23:
		return "end", nil
	case 0x21:
		return "pgup", nil
	case 0x22:
		return "pgdn", nil
	case 0x26:
		return "up", nil
	case 0x28:
		return "down", nil
	case 0x25:
		return "left", nil
	case 0x27:
		return "right", nil
	case 0xBC:
		return ",", nil
	case 0xBE:
		return ".", nil
	case 0x14:
		return "caps", nil
	}
	return "", fmt.Errorf("vk: code %d (0x%X) not in supported set", code, code)
}

func (r *ContainerRunner) execKeyHoldStart(ctx context.Context, n *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	vk, err := resolveVK(n.Config["vk"])
	if err != nil {
		return nil, fmt.Errorf("KeyHoldStart %s: %w", n.ID, err)
	}
	r.rt.InputBus.Lock()
	err = r.rt.Input.KeyDown(win.HWND(r.rt.Window.HWND), vk)
	r.rt.InputBus.Unlock()
	if err != nil {
		return nil, fmt.Errorf("KeyHoldStart %s: %w", n.ID, err)
	}
	return r.edges.next(n.ID+".out", tok.LoopStack), nil
}

func (r *ContainerRunner) execKeyHoldStop(ctx context.Context, n *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	vk, err := resolveVK(n.Config["vk"])
	if err != nil {
		return nil, fmt.Errorf("KeyHoldStop %s: %w", n.ID, err)
	}
	r.rt.InputBus.Lock()
	err = r.rt.Input.KeyUp(win.HWND(r.rt.Window.HWND), vk)
	r.rt.InputBus.Unlock()
	if err != nil {
		return nil, fmt.Errorf("KeyHoldStop %s: %w", n.ID, err)
	}
	return r.edges.next(n.ID+".out", tok.LoopStack), nil
}

func (r *ContainerRunner) execMouseHoldStart(ctx context.Context, n *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	button := configString(n, "button")
	if button == "" {
		button = "left"
	}
	if button != "left" && button != "right" && button != "middle" {
		return nil, fmt.Errorf("MouseHoldStart %s: invalid button %q", n.ID, button)
	}
	// v4: xRatio/yRatio via data-in pin.
	xRatio := r.pullNumber(n, "xRatio", 0.5)
	yRatio := r.pullNumber(n, "yRatio", 0.5)
	r.rt.InputBus.Lock()
	err := r.rt.Input.MouseDown(win.HWND(r.rt.Window.HWND), xRatio, yRatio, button)
	r.rt.InputBus.Unlock()
	if err != nil {
		return nil, fmt.Errorf("MouseHoldStart %s: %w", n.ID, err)
	}
	return r.edges.next(n.ID+".out", tok.LoopStack), nil
}

func (r *ContainerRunner) execMouseHoldStop(ctx context.Context, n *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	button := configString(n, "button")
	if button == "" {
		button = "left"
	}
	if button != "left" && button != "right" && button != "middle" {
		return nil, fmt.Errorf("MouseHoldStop %s: invalid button %q", n.ID, button)
	}
	r.rt.InputBus.Lock()
	err := r.rt.Input.MouseUp(win.HWND(r.rt.Window.HWND), button)
	r.rt.InputBus.Unlock()
	if err != nil {
		return nil, fmt.Errorf("MouseHoldStop %s: %w", n.ID, err)
	}
	return r.edges.next(n.ID+".out", tok.LoopStack), nil
}
