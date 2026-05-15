// Package runtime 是 actions 执行层：state machine + InputDriver + cleanup contract。
// 跟 actions definition 层（model/store/validate/service）分离，前者关心"长什么样"，
// 后者关心"怎么跑"。
package runtime

import (
	"fmt"
	"time"

	"github.com/lxn/win"

	"yhbox/pkg/input"
)

// InputDriver 抽象按键 / 鼠标后端。v1 只有 Win32 message driver；
// v2 加 SendInput driver 兼容某些 anti-cheat 屏蔽 injected message 的游戏。
// 单测用 MockDriver 验证 step 顺序而不真碰 Win32。
type InputDriver interface {
	Click(hwnd win.HWND, x, y int, button input.MouseButton, holdMs int) error
	KeyDown(hwnd win.HWND, vk string) error
	KeyUp(hwnd win.HWND, vk string) error
	MouseMove(hwnd win.HWND, x, y int) error
	MouseDrag(hwnd win.HWND, x1, y1, x2, y2 int, button input.MouseButton, durationMs int) error
	Scroll(hwnd win.HWND, notches int) error
	// MouseMoveRel 相对位移注入 Raw Input —— camera/视角转向走这条。
	MouseMoveRel(hwnd win.HWND, dx, dy int, durationMs int) error
}

// Win32Driver 包 pkg/input —— 跟 cook / fish 用同一套 PostMessage 路径。
type Win32Driver struct {
	ActivateDelay     time.Duration
	CursorSettleDelay time.Duration
}

func (d *Win32Driver) Click(hwnd win.HWND, x, y int, button input.MouseButton, holdMs int) error {
	// 走 NoRestore：action 回放时游戏在前台，光标弹回原位会"飘"。
	// 让光标留在 click 处，跟用户录制时的轨迹一致。
	input.ClickButtonNoRestore(hwnd, x, y, button, time.Duration(holdMs)*time.Millisecond, d.ActivateDelay, d.CursorSettleDelay)
	return nil
}

// KeyDown / KeyUp 用 pkg/input 返回 bool 表示 vk 名是否识别；这里转 error。
// 不识别的 vk 是 caller 配错 action，应该早发现 → 返 error 让 runner 立刻报。
func (d *Win32Driver) KeyDown(hwnd win.HWND, vk string) error {
	if !input.KeyDown(hwnd, vk) {
		return fmt.Errorf("unknown vk %q", vk)
	}
	return nil
}

func (d *Win32Driver) KeyUp(hwnd win.HWND, vk string) error {
	if !input.KeyUp(hwnd, vk) {
		return fmt.Errorf("unknown vk %q", vk)
	}
	return nil
}

func (d *Win32Driver) MouseMove(hwnd win.HWND, x, y int) error {
	input.MouseMove(hwnd, x, y, d.ActivateDelay, d.CursorSettleDelay)
	return nil
}

func (d *Win32Driver) MouseDrag(hwnd win.HWND, x1, y1, x2, y2 int, button input.MouseButton, durationMs int) error {
	input.MouseDrag(hwnd, x1, y1, x2, y2, button,
		time.Duration(durationMs)*time.Millisecond, d.ActivateDelay, d.CursorSettleDelay)
	return nil
}

func (d *Win32Driver) Scroll(hwnd win.HWND, notches int) error {
	input.MouseScroll(hwnd, notches, d.ActivateDelay)
	return nil
}

func (d *Win32Driver) MouseMoveRel(hwnd win.HWND, dx, dy int, durationMs int) error {
	input.MouseMoveRel(hwnd, dx, dy, time.Duration(durationMs)*time.Millisecond, d.ActivateDelay)
	return nil
}
