package runtime

import (
	"sync"

	"github.com/lxn/win"

	"yhbox/pkg/input"
)

// MockCall 记录 driver 收到的一次调用，给单测 assert step 顺序用。
type MockCall struct {
	Op      string // "click_left" / "key_down" / "mouse_move" / "mouse_drag" / "scroll" / "mouse_move_rel" 等
	X, Y    int
	X2, Y2  int // mouse_drag 终点；mouse_move_rel 复用 X/Y 存 dx/dy
	HoldMs  int // click_* / mouse_drag / mouse_move_rel duration
	Vk      string
	Notches int // scroll 用
}

// MockDriver 录每个调用，runner 单测/集成测时替代真 Win32 driver。
type MockDriver struct {
	mu    sync.Mutex
	Calls []MockCall

	// 可注入失败：设了 NextErr 则下一次任意 op 返这个 error 然后清空。
	NextErr error
}

func (d *MockDriver) Click(_ win.HWND, x, y int, b input.MouseButton, holdMs int) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.takeErr(); err != nil {
		return err
	}
	var op string
	switch b {
	case input.MouseLeft:
		op = "click_left"
	case input.MouseMiddle:
		op = "click_mid"
	case input.MouseRight:
		op = "click_right"
	}
	d.Calls = append(d.Calls, MockCall{Op: op, X: x, Y: y, HoldMs: holdMs})
	return nil
}

func (d *MockDriver) KeyDown(_ win.HWND, vk string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.takeErr(); err != nil {
		return err
	}
	d.Calls = append(d.Calls, MockCall{Op: "key_down", Vk: vk})
	return nil
}

func (d *MockDriver) KeyUp(_ win.HWND, vk string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.takeErr(); err != nil {
		return err
	}
	d.Calls = append(d.Calls, MockCall{Op: "key_up", Vk: vk})
	return nil
}

func (d *MockDriver) MouseMove(_ win.HWND, x, y int) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.takeErr(); err != nil {
		return err
	}
	d.Calls = append(d.Calls, MockCall{Op: "mouse_move", X: x, Y: y})
	return nil
}

func (d *MockDriver) MouseDrag(_ win.HWND, x1, y1, x2, y2 int, b input.MouseButton, holdMs int) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.takeErr(); err != nil {
		return err
	}
	op := "mouse_drag_left"
	switch b {
	case input.MouseMiddle:
		op = "mouse_drag_mid"
	case input.MouseRight:
		op = "mouse_drag_right"
	}
	d.Calls = append(d.Calls, MockCall{Op: op, X: x1, Y: y1, X2: x2, Y2: y2, HoldMs: holdMs})
	return nil
}

func (d *MockDriver) Scroll(_ win.HWND, notches int) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.takeErr(); err != nil {
		return err
	}
	d.Calls = append(d.Calls, MockCall{Op: "scroll", Notches: notches})
	return nil
}

func (d *MockDriver) MouseMoveRel(_ win.HWND, dx, dy, durationMs int) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.takeErr(); err != nil {
		return err
	}
	d.Calls = append(d.Calls, MockCall{Op: "mouse_move_rel", X: dx, Y: dy, HoldMs: durationMs})
	return nil
}

func (d *MockDriver) takeErr() error {
	err := d.NextErr
	d.NextErr = nil
	return err
}
