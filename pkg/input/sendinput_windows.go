//go:build windows

package input

import (
	"errors"
	"fmt"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/lxn/win"
)

// sendInputBackend 全局前台 OS 级注入后端 (user32.SendInput). 不挑 hwnd —— 适合只认真实前台
// 输入 (RawInput / DirectInput) 收不到 PostMessage 的游戏 (FPS / 相机类). 代价: 全局抢真实
// 光标/键盘, 跑时这台机器专给脚本用. Capabilities.GlobalInput=true 标明这点给上层判断.
//
// 复用 input.go 现成的 procSendInput / procMapVirtualKeyW / mouseInput / sendInputBlock /
// sendInputMouseRel / VK / mapVKToVSC; 本文件只补键盘 INPUT + 按钮/滚轮/absolute flag + 封装.
type sendInputBackend struct {
	mu       sync.Mutex
	heldKeys map[uint32]struct{} // vk 码 → 已按下
	heldBtns map[uint32]struct{} // VK_LBUTTON(1)/RBUTTON(2)/MBUTTON(4) → 已按下
}

func newSendInputBackend() *sendInputBackend {
	return &sendInputBackend{
		heldKeys: map[uint32]struct{}{},
		heldBtns: map[uint32]struct{}{},
	}
}

func (b *sendInputBackend) Name() string { return "sendinput" }

func (b *sendInputBackend) Capabilities() Capabilities {
	return Capabilities{BackgroundInput: false, RelativeMouse: true, GlobalInput: true}
}

// --- Win32 INPUT 封装 (键盘/按钮/滚轮/absolute move; 相对 move 复用 input.go sendInputMouseRel) ---

const (
	siMouseLeftDown   uint32 = 0x0002
	siMouseLeftUp     uint32 = 0x0004
	siMouseRightDown  uint32 = 0x0008
	siMouseRightUp    uint32 = 0x0010
	siMouseMiddleDown uint32 = 0x0020
	siMouseMiddleUp   uint32 = 0x0040
	siMouseWheel      uint32 = 0x0800
	siMouseHWheel     uint32 = 0x1000
	siMouseAbsolute   uint32 = 0x8000

	siKeyKeyUp    uint32 = 0x0002
	siKeyScanCode uint32 = 0x0008

	siInputKeyboard uint32 = 1
)

// keybdInput + key block padded 到 40B (= sizeof INPUT, SendInput cbSize 必须等于它).
type keybdInput struct {
	WVk         uint16
	WScan       uint16
	DwFlags     uint32
	Time        uint32
	DwExtraInfo uintptr
}

type sendInputKeyBlock struct {
	Type uint32
	_    uint32
	Ki   keybdInput
	_    [8]byte
}

func sendKeyEvent(vk uint32, keyUp bool) error {
	r, _, _ := procMapVirtualKeyW.Call(uintptr(vk), mapVKToVSC)
	scan := uint32(r) & 0xFFFF
	flags := siKeyScanCode
	if keyUp {
		flags |= siKeyKeyUp
	}
	in := sendInputKeyBlock{Type: siInputKeyboard, Ki: keybdInput{WVk: uint16(vk), WScan: uint16(scan), DwFlags: flags}}
	sent, _, errno := procSendInput.Call(1, uintptr(unsafe.Pointer(&in)), unsafe.Sizeof(in))
	if sent != 1 {
		if errno != syscall.Errno(0) {
			return fmt.Errorf("SendInput keyboard vk=0x%X keyUp=%v failed: %w", vk, keyUp, errno)
		}
		return fmt.Errorf("SendInput keyboard vk=0x%X keyUp=%v sent %d events, want 1", vk, keyUp, sent)
	}
	return nil
}

func sendMouseBtnEvent(flags uint32) error {
	in := sendInputBlock{Type: 0, Mi: mouseInput{Flags: flags}}
	return sendMouseEvent(&in, "button")
}

// ratioToAbs ratio(0-1) → 0..65535 (Win32 ABSOLUTE 约定: 整屏归一化, 与分辨率无关), clamp.
func ratioToAbs(ratio float64) int32 {
	v := ratio * 65535.0
	if v < 0 {
		v = 0
	}
	if v > 65535 {
		v = 65535
	}
	return int32(v)
}

// sendAbsMove ratio(0-1, 相对全屏) → absolute move.
func sendAbsMove(xRatio, yRatio float64) error {
	in := sendInputBlock{Type: 0, Mi: mouseInput{
		Dx:    ratioToAbs(xRatio),
		Dy:    ratioToAbs(yRatio),
		Flags: mouseEventFMove | siMouseAbsolute,
	}}
	return sendMouseEvent(&in, "absolute move")
}

func sendWheel(notches int) error {
	in := sendInputBlock{Type: 0, Mi: mouseInput{MouseData: uint32(int32(notches) * 120), Flags: siMouseWheel}}
	return sendMouseEvent(&in, "vertical wheel")
}

func sendHWheel(notches int) error {
	in := sendInputBlock{Type: 0, Mi: mouseInput{MouseData: uint32(int32(notches) * 120), Flags: siMouseHWheel}}
	return sendMouseEvent(&in, "horizontal wheel")
}

func sendMouseEvent(input *sendInputBlock, operation string) error {
	sent, _, errno := procSendInput.Call(1, uintptr(unsafe.Pointer(input)), unsafe.Sizeof(*input))
	if sent == 1 {
		return nil
	}
	if errno != syscall.Errno(0) {
		return fmt.Errorf("SendInput %s failed: %w", operation, errno)
	}
	return fmt.Errorf("SendInput %s sent %d events, want 1", operation, sent)
}

func siButtonFlags(button string, up bool) uint32 {
	switch button {
	case "right":
		if up {
			return siMouseRightUp
		}
		return siMouseRightDown
	case "middle":
		if up {
			return siMouseMiddleUp
		}
		return siMouseMiddleDown
	default:
		if up {
			return siMouseLeftUp
		}
		return siMouseLeftDown
	}
}

func siButtonVK(button string) uint32 {
	switch button {
	case "right":
		return 2 // VK_RBUTTON
	case "middle":
		return 4 // VK_MBUTTON
	default:
		return 1 // VK_LBUTTON
	}
}

// --- Backend 接口实现 (hwnd 一律忽略: 全局注入, 无单窗口概念) ---

func (b *sendInputBackend) KeyDown(_ win.HWND, vk string) error {
	code := VK(vk)
	if code == 0 {
		return fmt.Errorf("sendinput KeyDown: unknown vk %q", vk)
	}
	return b.KeyDownCode(0, code)
}

func (b *sendInputBackend) KeyDownCode(_ win.HWND, code uint32) error {
	if code == 0 || code > 255 {
		return fmt.Errorf("sendinput KeyDown: invalid virtual key %d", code)
	}
	if err := sendKeyEvent(code, false); err != nil {
		return err
	}
	b.mu.Lock()
	b.heldKeys[code] = struct{}{}
	b.mu.Unlock()
	return nil
}

func (b *sendInputBackend) KeyUp(_ win.HWND, vk string) error {
	code := VK(vk)
	if code == 0 {
		return fmt.Errorf("sendinput KeyUp: unknown vk %q", vk)
	}
	return b.KeyUpCode(0, code)
}

func (b *sendInputBackend) KeyUpCode(_ win.HWND, code uint32) error {
	if code == 0 || code > 255 {
		return fmt.Errorf("sendinput KeyUp: invalid virtual key %d", code)
	}
	if err := sendKeyEvent(code, true); err != nil {
		return err
	}
	b.mu.Lock()
	delete(b.heldKeys, code)
	b.mu.Unlock()
	return nil
}

func (b *sendInputBackend) MouseDown(hwnd win.HWND, xRatio, yRatio float64, button string) error {
	sx, sy := clientRatioToScreenRatio(hwnd, xRatio, yRatio)
	if err := sendAbsMove(sx, sy); err != nil {
		return err
	}
	if err := sendMouseBtnEvent(siButtonFlags(button, false)); err != nil {
		return err
	}
	b.mu.Lock()
	b.heldBtns[siButtonVK(button)] = struct{}{}
	b.mu.Unlock()
	return nil
}

func (b *sendInputBackend) MouseUp(_ win.HWND, button string) error {
	if err := sendMouseBtnEvent(siButtonFlags(button, true)); err != nil {
		return err
	}
	b.mu.Lock()
	delete(b.heldBtns, siButtonVK(button))
	b.mu.Unlock()
	return nil
}

func (b *sendInputBackend) Click(hwnd win.HWND, xRatio, yRatio float64, button string, durMs int) error {
	if err := b.MouseDown(hwnd, xRatio, yRatio, button); err != nil {
		return err
	}
	if durMs <= 0 {
		durMs = 50
	}
	time.Sleep(time.Duration(durMs) * time.Millisecond)
	return b.MouseUp(hwnd, button)
}

func (b *sendInputBackend) Drag(hwnd win.HWND, x1Ratio, y1Ratio, x2Ratio, y2Ratio float64, button string, durationMs int) (result error) {
	if err := b.MouseDown(hwnd, x1Ratio, y1Ratio, button); err != nil {
		return err
	}
	released := false
	defer func() {
		if !released {
			result = errors.Join(result, b.MouseUp(hwnd, button))
		}
	}()
	if durationMs <= 0 {
		durationMs = 200
	}
	frames := max(1, durationMs/16)
	frameDuration := time.Duration(durationMs) * time.Millisecond / time.Duration(frames)
	for frame := 1; frame <= frames; frame++ {
		progress := float64(frame) / float64(frames)
		x := x1Ratio + progress*(x2Ratio-x1Ratio)
		y := y1Ratio + progress*(y2Ratio-y1Ratio)
		sx, sy := clientRatioToScreenRatio(hwnd, x, y)
		if err := sendAbsMove(sx, sy); err != nil {
			return err
		}
		if frame < frames {
			time.Sleep(frameDuration)
		}
	}
	if err := b.MouseUp(hwnd, button); err != nil {
		return err
	}
	released = true
	return nil
}

func (b *sendInputBackend) MoveTo(hwnd win.HWND, xRatio, yRatio float64) error {
	sx, sy := clientRatioToScreenRatio(hwnd, xRatio, yRatio)
	return sendAbsMove(sx, sy)
}

func (b *sendInputBackend) MouseMoveRel(_ win.HWND, dx, dy, _ int) error {
	return sendInputMouseRel(int32(dx), int32(dy))
}

func (b *sendInputBackend) Scroll(_ win.HWND, _, _ float64, notches int, horizontal bool) error {
	if horizontal {
		return sendHWheel(notches)
	}
	return sendWheel(notches)
}

// CursorRatio 读 OS 光标 → 相对主屏比例 (全局注入无单窗口概念, 基准取主屏).
func (b *sendInputBackend) CursorRatio(hwnd win.HWND) (float64, float64, error) {
	var rect win.RECT
	win.GetClientRect(hwnd, &rect)
	w := float64(rect.Right - rect.Left)
	h := float64(rect.Bottom - rect.Top)
	if w <= 0 || h <= 0 {
		return 0, 0, fmt.Errorf("sendinput CursorRatio: client rect 为空")
	}
	sx, sy := getCursorPos()
	cx, cy := screenToClient(hwnd, sx, sy) // 屏幕像素 → 客户区像素
	return float64(cx) / w, float64(cy) / h, nil
}

func (b *sendInputBackend) ReleaseAll() error {
	return b.releaseAllWith(sendKeyEvent, sendMouseBtnEvent)
}

func (b *sendInputBackend) releaseAllWith(
	releaseKey func(uint32, bool) error,
	releaseMouseButton func(uint32) error,
) error {
	b.mu.Lock()
	keys := make([]uint32, 0, len(b.heldKeys))
	for k := range b.heldKeys {
		keys = append(keys, k)
	}
	btns := make([]string, 0, len(b.heldBtns))
	for vk := range b.heldBtns {
		switch vk {
		case 2:
			btns = append(btns, "right")
		case 4:
			btns = append(btns, "middle")
		default:
			btns = append(btns, "left")
		}
	}
	b.mu.Unlock()
	var result error
	for _, k := range keys {
		if err := releaseKey(k, true); err != nil {
			result = errors.Join(result, err)
			continue
		}
		b.mu.Lock()
		delete(b.heldKeys, k)
		b.mu.Unlock()
	}
	for _, btn := range btns {
		if err := releaseMouseButton(siButtonFlags(btn, true)); err != nil {
			result = errors.Join(result, err)
			continue
		}
		b.mu.Lock()
		delete(b.heldBtns, siButtonVK(btn))
		b.mu.Unlock()
	}
	return result
}

func (b *sendInputBackend) Close() error { return nil }

func (b *sendInputBackend) TypeText(hwnd win.HWND, s string) error {
	return TypeText(hwnd, s)
}

// clientRatioToScreenRatio 把窗口客户区比例坐标转换成 SendInput 需要的全屏绝对比例坐标.
// ClickAt / MoveTo 传入的 xRatio/yRatio 是相对窗口客户区的, sendAbsMove 需要的是相对整个屏幕的.
func clientRatioToScreenRatio(hwnd win.HWND, xRatio, yRatio float64) (float64, float64) {
	var rect win.RECT
	win.GetClientRect(hwnd, &rect)
	clientX := int32(xRatio * float64(rect.Right-rect.Left))
	clientY := int32(yRatio * float64(rect.Bottom-rect.Top))
	sx, sy := clientToScreen(hwnd, clientX, clientY) // 客户区像素 → 屏幕像素
	sw := float64(win.GetSystemMetrics(win.SM_CXSCREEN))
	sh := float64(win.GetSystemMetrics(win.SM_CYSCREEN))
	return float64(sx) / sw, float64(sy) / sh
}
