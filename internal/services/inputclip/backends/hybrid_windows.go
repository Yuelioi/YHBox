//go:build windows

// Package backends — HybridBackend: clip 回放的混合后端.
//
// 有目标 hwnd 时键盘/鼠标按钮/移动/滚轮全走 PostMessage (精确投递给目标窗口, 后台也能收);
// 相机转向 (RawDelta) 无条件走 SendInput —— 某些游戏的窗口消息只接 PostMessage 键鼠, 但
// 相机/视角转向只认 SendInput 的 OS raw input. 无 hwnd 时整体降级走 SendInput (录到桌面也能凑合).
//
// heldKeys / heldButtons 状态机: 每次 KeyDown/MouseBtnDown 加入, KeyUp/MouseBtnUp 移除.
// Cancel 时 ReleaseHeld 遍历发对应 Up, 防止键卡住.
package backends

import (
	"fmt"
	"sync"
	"syscall"
	"unsafe"

	"github.com/lxn/win"

	"yhbox/internal/services/inputclip"
	"yhbox/pkg/input"
)

// Win32 INPUT 常量.
const (
	inputMouse    uint32 = 0
	inputKeyboard uint32 = 1

	// Mouse flags
	mouseEventfMove       uint32 = 0x0001
	mouseEventfLeftDown   uint32 = 0x0002
	mouseEventfLeftUp     uint32 = 0x0004
	mouseEventfRightDown  uint32 = 0x0008
	mouseEventfRightUp    uint32 = 0x0010
	mouseEventfMiddleDown uint32 = 0x0020
	mouseEventfMiddleUp   uint32 = 0x0040
	mouseEventfXDown      uint32 = 0x0080
	mouseEventfXUp        uint32 = 0x0100
	mouseEventfWheel      uint32 = 0x0800
	mouseEventfAbsolute   uint32 = 0x8000

	// Keyboard flags
	keyEventfExtendedKey uint32 = 0x0001
	keyEventfKeyUp       uint32 = 0x0002
	keyEventfScanCode    uint32 = 0x0008

	xButton1 uint32 = 1
	xButton2 uint32 = 2

	mapVKToVSC uint32 = 0
)

// MOUSEINPUT (24 bytes).
type mouseInput struct {
	Dx          int32
	Dy          int32
	MouseData   uint32
	DwFlags     uint32
	Time        uint32
	DwExtraInfo uintptr
}

// KEYBDINPUT (24 bytes on x64 due to extraInfo uintptr 8B).
type keybdInput struct {
	WVk         uint16
	WScan       uint16
	DwFlags     uint32
	Time        uint32
	DwExtraInfo uintptr
	_           [8]byte // pad 让 union 对齐到 mouseInput (24B) + 8 reserved = match
}

// INPUT union — x64 上 type(4) + pad(4) + 28 字节 union 实际占 32 字节 (sizeof INPUT 在 user32 是 40).
// 这里跟 pkg/input 一致 用 sendInputBlock 持有 mouseInput; key 复用同 buffer (24 字节够装下 keybdInput).
type sendInputMouseBlock struct {
	Type uint32
	_    uint32 // pad on amd64
	Mi   mouseInput
	_    [8]byte // pad to 40 总长 (sizeof INPUT)
}

type sendInputKeyBlock struct {
	Type uint32
	_    uint32 // pad on amd64
	Ki   keybdInput
	_    [8]byte
}

var (
	user32            = syscall.NewLazyDLL("user32.dll")
	procSendInput     = user32.NewProc("SendInput")
	procMapVirtualKey = user32.NewProc("MapVirtualKeyW")
)

// HybridBackend 混合后端: PostMessage 路径 (键盘/鼠标按钮/移动/滚轮) + SendInput 路径 (RawDelta 相机).
//
// 关键: 某些游戏的窗口消息处理只接 PostMessage 的 WM_KEYDOWN/WM_LBUTTONDOWN, 不接 SendInput 键鼠;
// 但相机/视角转向只认 SendInput 的 OS raw input pipeline (RawDelta 唯一路径).
// ClipPlayer 调度后只是把每个 event 路由到对应 PostMessage / SendInput.
type HybridBackend struct {
	hwndGetter  func() uintptr      // 拿当前游戏 hwnd. 0 = SendInput fallback (录到桌面也能凑合).
	mu          sync.Mutex
	heldKeys    map[uint32]struct{} // vk → present
	heldButtons map[uint32]struct{} // btn (1L 2R 3M 4X1 5X2) → present
}

// NewHybridBackend 构造. hwndGetter 拿游戏窗口 hwnd, nil = 永远走 SendInput.
func NewHybridBackend(hwndGetter func() uintptr) IInputBackend {
	return &HybridBackend{
		hwndGetter:  hwndGetter,
		heldKeys:    make(map[uint32]struct{}),
		heldButtons: make(map[uint32]struct{}),
	}
}

// Name 实现 IInputBackend.
func (b *HybridBackend) Name() string { return "postmessage+sendinput" }

// Send 投递单 event. 同步.
func (b *HybridBackend) Send(ev inputclip.Event) error {
	var hwnd win.HWND
	if b.hwndGetter != nil {
		hwnd = win.HWND(b.hwndGetter())
	}
	switch ev.Type {
	case inputclip.EventTypeKeyDown:
		b.trackKey(uint32(ev.A), false)
		if hwnd != 0 {
			input.PostKeyDownVK(hwnd, uint32(ev.A))
			return nil
		}
		return b.sendKeyRaw(uint32(ev.A), uint32(ev.B), uint32(ev.C), false)
	case inputclip.EventTypeKeyUp:
		b.trackKey(uint32(ev.A), true)
		if hwnd != 0 {
			input.PostKeyUpVK(hwnd, uint32(ev.A))
			return nil
		}
		return b.sendKeyRaw(uint32(ev.A), uint32(ev.B), uint32(ev.C), true)
	case inputclip.EventTypeMouseBtnDown:
		b.trackBtn(uint32(ev.A), false)
		if hwnd != 0 {
			input.MouseBtnDown(hwnd, int(ev.B), int(ev.C), mapBtn(uint32(ev.A)))
			return nil
		}
		return b.sendMouseBtnRaw(uint32(ev.A), ev.B, ev.C, false)
	case inputclip.EventTypeMouseBtnUp:
		b.trackBtn(uint32(ev.A), true)
		if hwnd != 0 {
			input.MouseBtnUp(hwnd, int(ev.B), int(ev.C), mapBtn(uint32(ev.A)))
			return nil
		}
		return b.sendMouseBtnRaw(uint32(ev.A), ev.B, ev.C, true)
	case inputclip.EventTypeMouseMove:
		// mode A=0 absolute / 1 relative; recorder 当前走 absolute 写 client px.
		// 有 hwnd 走 cursor 移动 + WM_MOUSEMOVE PostMessage (UI / Slate hover).
		if hwnd != 0 {
			input.MouseMove(hwnd, int(ev.B), int(ev.C), 0, 0)
			return nil
		}
		return b.sendMouseMove(uint32(ev.A), ev.B, ev.C)
	case inputclip.EventTypeRawDelta:
		// 相机转向 — 永远 SendInput, 跟有无 hwnd 无关 (OS raw input pipeline).
		input.SendInputMouseRel(ev.B, ev.C)
		return nil
	case inputclip.EventTypeScroll:
		if hwnd != 0 {
			input.MouseScroll(hwnd, int(ev.A), 0)
			return nil
		}
		return b.sendScroll(ev.A, ev.B, ev.C)
	default:
		return nil
	}
}

// trackKey / trackBtn 状态机更新, Send 路径用 (不发实际 input — 由路由决定 Post / SendInput).
func (b *HybridBackend) trackKey(vk uint32, keyUp bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if keyUp {
		delete(b.heldKeys, vk)
	} else {
		b.heldKeys[vk] = struct{}{}
	}
}

func (b *HybridBackend) trackBtn(btn uint32, btnUp bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if btnUp {
		delete(b.heldButtons, btn)
	} else {
		b.heldButtons[btn] = struct{}{}
	}
}

// mapBtn recorder btn 编码 1=L 2=R 3=M → pkg/input.MouseButton.
func mapBtn(btn uint32) input.MouseButton {
	switch btn {
	case 2:
		return input.MouseRight
	case 3:
		return input.MouseMiddle
	default:
		return input.MouseLeft
	}
}

// SendBatch foreach 调 Send, 不真合批成单次 SendInput.
func (b *HybridBackend) SendBatch(evs []inputclip.Event) error {
	for _, ev := range evs {
		if err := b.Send(ev); err != nil {
			return err
		}
	}
	return nil
}

// ReleaseHeld 遍历 held 状态发对应 Up. 防止键卡住.
func (b *HybridBackend) ReleaseHeld() error {
	b.mu.Lock()
	keys := make([]uint32, 0, len(b.heldKeys))
	for k := range b.heldKeys {
		keys = append(keys, k)
	}
	btns := make([]uint32, 0, len(b.heldButtons))
	for k := range b.heldButtons {
		btns = append(btns, k)
	}
	b.heldKeys = make(map[uint32]struct{})
	b.heldButtons = make(map[uint32]struct{})
	b.mu.Unlock()

	for _, vk := range keys {
		// 0 scan / 0 flag, sendKeyRaw 内 scan=0 会自动补; 状态机已清空所以这里直接调底层.
		_ = b.sendKeyRaw(vk, 0, 0, true)
	}
	for _, btn := range btns {
		_ = b.sendMouseBtnRaw(btn, -1, -1, true)
	}
	return nil
}

// --- 内部实现 ---

// sendKeyRaw 不动状态机, 真发 SendInput.
//
// scan=0 时调 MapVirtualKey 自动填. flags 是来自 Event.C 的 bitmask
// (e.g. KEYEVENTF_EXTENDEDKEY); KEYUP 由 keyUp 参数补.
func (b *HybridBackend) sendKeyRaw(vk, scan, flags uint32, keyUp bool) error {
	if scan == 0 {
		r, _, _ := procMapVirtualKey.Call(uintptr(vk), uintptr(mapVKToVSC))
		scan = uint32(r) & 0xFFFF
	}
	dw := flags
	if keyUp {
		dw |= keyEventfKeyUp
	}
	in := sendInputKeyBlock{
		Type: inputKeyboard,
		Ki: keybdInput{
			WVk:     uint16(vk),
			WScan:   uint16(scan),
			DwFlags: dw,
		},
	}
	r, _, lastErr := procSendInput.Call(1, uintptr(unsafe.Pointer(&in)), unsafe.Sizeof(in))
	if r == 0 {
		return fmt.Errorf("SendInput failed: %v", lastErr)
	}
	return nil
}

// sendMouseBtnRaw 不动状态机.
//
// btn 1=L 2=R 3=M 4=X1 5=X2. x/y screen px (-1 = 不移动, 当前光标位置).
func (b *HybridBackend) sendMouseBtnRaw(btn uint32, x, y int32, btnUp bool) error {
	var flags uint32
	var mouseData uint32
	switch btn {
	case 1:
		if btnUp {
			flags = mouseEventfLeftUp
		} else {
			flags = mouseEventfLeftDown
		}
	case 2:
		if btnUp {
			flags = mouseEventfRightUp
		} else {
			flags = mouseEventfRightDown
		}
	case 3:
		if btnUp {
			flags = mouseEventfMiddleUp
		} else {
			flags = mouseEventfMiddleDown
		}
	case 4:
		if btnUp {
			flags = mouseEventfXUp
		} else {
			flags = mouseEventfXDown
		}
		mouseData = xButton1
	case 5:
		if btnUp {
			flags = mouseEventfXUp
		} else {
			flags = mouseEventfXDown
		}
		mouseData = xButton2
	default:
		return nil
	}
	in := sendInputMouseBlock{
		Type: inputMouse,
		Mi: mouseInput{
			Dx:        x,
			Dy:        y,
			MouseData: mouseData,
			DwFlags:   flags,
		},
	}
	r, _, lastErr := procSendInput.Call(1, uintptr(unsafe.Pointer(&in)), unsafe.Sizeof(in))
	if r == 0 {
		return fmt.Errorf("SendInput failed: %v", lastErr)
	}
	return nil
}

// sendMouseMove A=mode (0 absolute, 1 relative), B/C = x/y (client px).
//
// 只在无 hwnd 降级路径触发 (正常有 hwnd 走 PostMessage MouseMove, 不到这里)。
// absolute 模式: 无目标窗口客户区可用, 按主屏尺寸把 client px 归一化到 0..65535
// (Win32 MOUSEEVENTF_ABSOLUTE 约定; 不归一化的话 client px 会被当全屏绝对值 → 光标贴左上角)。
func (b *HybridBackend) sendMouseMove(mode uint32, x, y int32) error {
	flags := mouseEventfMove
	if mode == 0 {
		flags |= mouseEventfAbsolute
		sw := win.GetSystemMetrics(win.SM_CXSCREEN)
		sh := win.GetSystemMetrics(win.SM_CYSCREEN)
		if sw > 1 {
			x = clampAbs(x * 65535 / (sw - 1))
		}
		if sh > 1 {
			y = clampAbs(y * 65535 / (sh - 1))
		}
	}
	in := sendInputMouseBlock{
		Type: inputMouse,
		Mi: mouseInput{
			Dx:      x,
			Dy:      y,
			DwFlags: flags,
		},
	}
	r, _, lastErr := procSendInput.Call(1, uintptr(unsafe.Pointer(&in)), unsafe.Sizeof(in))
	if r == 0 {
		return fmt.Errorf("SendInput failed: %v", lastErr)
	}
	return nil
}

// clampAbs 夹到 Win32 ABSOLUTE 合法范围 0..65535.
func clampAbs(v int32) int32 {
	if v < 0 {
		return 0
	}
	if v > 65535 {
		return 65535
	}
	return v
}

// sendScroll notches * 120 = wheel delta.
func (b *HybridBackend) sendScroll(notches, x, y int32) error {
	delta := uint32(int32(notches) * 120)
	in := sendInputMouseBlock{
		Type: inputMouse,
		Mi: mouseInput{
			Dx:        x,
			Dy:        y,
			MouseData: delta,
			DwFlags:   mouseEventfWheel,
		},
	}
	r, _, lastErr := procSendInput.Call(1, uintptr(unsafe.Pointer(&in)), unsafe.Sizeof(in))
	if r == 0 {
		return fmt.Errorf("SendInput failed: %v", lastErr)
	}
	return nil
}
