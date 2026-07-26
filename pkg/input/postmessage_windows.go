//go:build windows

package input

import (
	"errors"
	"fmt"
	"maps"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/lxn/win"
)

// PostMessageBackend wraps existing pkg/input package-level functions, adds stateful
// tracking of held keys/buttons for ReleaseAll.
//
// 不重写底层 Win32 调用 — 复用 input.go 里 ClickButton/KeyDown/KeyUp/MouseMoveRel/...
// 的成熟实现. 本 struct 只加 state 跟踪 + interface 适配.
type PostMessageBackend struct {
	mu        sync.Mutex
	heldKeys  map[uint32]win.HWND           // virtual key → exact target window
	heldBtns  map[win.HWND]map[string]point // hwnd → button name → 按下时客户区坐标
	activated map[win.HWND]bool             // hwnd → 是否已 FakeActivate 过
	cursor    map[win.HWND]point            // exact target client position; never moves the global cursor
}

func newPostMessageBackend() *PostMessageBackend {
	return &PostMessageBackend{
		heldKeys:  map[uint32]win.HWND{},
		heldBtns:  map[win.HWND]map[string]point{},
		activated: map[win.HWND]bool{},
		cursor:    map[win.HWND]point{},
	}
}

func (b *PostMessageBackend) Name() string { return "postmessage" }

func (b *PostMessageBackend) Capabilities() Capabilities {
	return Capabilities{
		BackgroundInput: true,  // PostMessage 不需要前台
		RelativeMouse:   false, // PostMessage 没有原生相对移动
		GlobalInput:     false, // 只发给指定 hwnd
	}
}

// 默认 timing — caller 传 durMs=0 时用这些
const (
	defaultActivateDelay = 30 * time.Millisecond
	defaultCursorSettle  = 30 * time.Millisecond
	defaultClickHoldMs   = 50
)

// pickButton 字符串 → MouseButton 枚举 (MouseButton 在 input.go 已定义)
func pickButton(name string) MouseButton {
	switch strings.ToLower(name) {
	case "right":
		return MouseRight
	case "middle":
		return MouseMiddle
	default:
		return MouseLeft
	}
}

func (b *PostMessageBackend) ensureActivated(hwnd win.HWND) {
	b.mu.Lock()
	already := b.activated[hwnd]
	if !already {
		b.activated[hwnd] = true
	}
	b.mu.Unlock()
	if !already {
		FakeActivate(hwnd) // input.go 现有函数
		// 首次激活后等 Slate 在下一 UE tick 翻 IsActive=true 再放行后续 PostMessage —— 否则首个
		// keydown/down 在 IsActive 仍 false 时被部分游戏的 IMC 丢弃 (FakeActivate 是 SendMessage, 同步返回
		// 不代表 Slate 已处理). 复原 #4 前 Tap/Click 自带的 activateDelay: #4 把 KeyPress/ClickAt 改
		// 节点层 KeyDown/MouseDown 后, 这条 settle 只剩这里兜. 仅首次 per-hwnd → 分帧 MoveTo 不重复付.
		time.Sleep(defaultActivateDelay)
	}
}

func (b *PostMessageBackend) Click(hwnd win.HWND, xRatio, yRatio float64, button string, durMs int) error {
	if durMs <= 0 {
		durMs = defaultClickHoldMs
	}
	if err := b.MouseDown(hwnd, xRatio, yRatio, button); err != nil {
		return err
	}
	time.Sleep(time.Duration(durMs) * time.Millisecond)
	return b.MouseUp(hwnd, button)
}

func (b *PostMessageBackend) KeyDown(hwnd win.HWND, vk string) error {
	code := VK(vk)
	if code == 0 {
		return fmt.Errorf("postmessage KeyDown: unknown vk %q", vk)
	}
	return b.KeyDownCode(hwnd, code)
}

func (b *PostMessageBackend) KeyDownCode(hwnd win.HWND, code uint32) error {
	if code == 0 || code > 255 {
		return fmt.Errorf("postmessage KeyDown: invalid virtual key %d", code)
	}
	b.ensureActivated(hwnd)
	if err := postMessageChecked(hwnd, WM_KEYDOWN, uintptr(code), keyLParam(code, false)); err != nil {
		return err
	}
	b.mu.Lock()
	b.heldKeys[code] = hwnd
	b.mu.Unlock()
	return nil
}

func (b *PostMessageBackend) KeyUp(hwnd win.HWND, vk string) error {
	code := VK(vk)
	if code == 0 {
		return fmt.Errorf("postmessage KeyUp: unknown vk %q", vk)
	}
	return b.KeyUpCode(hwnd, code)
}

func (b *PostMessageBackend) KeyUpCode(hwnd win.HWND, code uint32) error {
	if code == 0 || code > 255 {
		return fmt.Errorf("postmessage KeyUp: invalid virtual key %d", code)
	}
	if err := postMessageChecked(hwnd, WM_KEYUP, uintptr(code), keyLParam(code, true)); err != nil {
		return err
	}
	b.mu.Lock()
	delete(b.heldKeys, code)
	b.mu.Unlock()
	return nil
}

func (b *PostMessageBackend) MouseDown(hwnd win.HWND, xRatio, yRatio float64, button string) error {
	b.ensureActivated(hwnd)
	pt, err := checkedClientPoint(hwnd, xRatio, yRatio)
	if err != nil {
		return err
	}
	// 跟 ClickButton 同序: 先 hover (setCursorPos + WM_MOUSEMOVE) + settle 让 Slate 在它的
	// tick 更新 hover 元素, 再 DOWN —— 否则按下可能落不到目标控件 (跟 ClickAt 旧路径同源).
	// 不松开 / 不复位光标 (hold 语义: 光标留在按下点直到 MouseHoldStop).
	if err := postMessageChecked(hwnd, WM_MOUSEMOVE, 0, makeLParam(pt.X, pt.Y)); err != nil {
		return err
	}
	time.Sleep(defaultCursorSettle)
	downMessage, downFlags := postMessageButton(button, false)
	if err := postMessageChecked(hwnd, downMessage, downFlags, makeLParam(pt.X, pt.Y)); err != nil {
		return err
	}
	b.mu.Lock()
	if b.heldBtns[hwnd] == nil {
		b.heldBtns[hwnd] = map[string]point{}
	}
	b.heldBtns[hwnd][button] = pt
	b.cursor[hwnd] = pt
	b.mu.Unlock()
	return nil
}

func (b *PostMessageBackend) MouseUp(hwnd win.HWND, button string) error {
	b.mu.Lock()
	pt, ok := b.heldBtns[hwnd][button]
	if !ok {
		pt, ok = b.cursor[hwnd]
	}
	b.mu.Unlock()
	if !ok {
		var err error
		pt, err = checkedClientPoint(hwnd, 0.5, 0.5)
		if err != nil {
			return err
		}
	}
	upMessage, upFlags := postMessageButton(button, true)
	if err := postMessageChecked(hwnd, upMessage, upFlags, makeLParam(pt.X, pt.Y)); err != nil {
		return err
	}
	b.mu.Lock()
	if set := b.heldBtns[hwnd]; set != nil {
		delete(set, button)
		if len(set) == 0 {
			delete(b.heldBtns, hwnd)
		}
	}
	b.mu.Unlock()
	return nil
}

func (b *PostMessageBackend) MouseMoveRel(hwnd win.HWND, dx, dy, durMs int) error {
	b.ensureActivated(hwnd)
	b.mu.Lock()
	current, ok := b.cursor[hwnd]
	b.mu.Unlock()
	if !ok {
		var err error
		current, err = checkedClientPoint(hwnd, 0.5, 0.5)
		if err != nil {
			return err
		}
	}
	width, height, err := checkedClientSize(hwnd)
	if err != nil {
		return err
	}
	next := point{X: min(max(current.X+int32(dx), 0), int32(width-1)), Y: min(max(current.Y+int32(dy), 0), int32(height-1))}
	if err := postMessageChecked(hwnd, WM_MOUSEMOVE, 0, makeLParam(next.X, next.Y)); err != nil {
		return err
	}
	b.mu.Lock()
	b.cursor[hwnd] = next
	b.mu.Unlock()
	return nil
}

func (b *PostMessageBackend) Drag(hwnd win.HWND, x1Ratio, y1Ratio, x2Ratio, y2Ratio float64, button string, durationMs int) error {
	if durationMs <= 0 {
		durationMs = 200
	}
	if err := b.MouseDown(hwnd, x1Ratio, y1Ratio, button); err != nil {
		return err
	}
	steps := max(1, durationMs/16)
	for step := 1; step <= steps; step++ {
		progress := float64(step) / float64(steps)
		if err := b.MoveTo(hwnd, x1Ratio+(x2Ratio-x1Ratio)*progress, y1Ratio+(y2Ratio-y1Ratio)*progress); err != nil {
			return err
		}
		time.Sleep(time.Duration(durationMs/steps) * time.Millisecond)
	}
	return b.MouseUp(hwnd, button)
}

func (b *PostMessageBackend) MoveTo(hwnd win.HWND, xRatio, yRatio float64) error {
	b.ensureActivated(hwnd)
	pt, err := checkedClientPoint(hwnd, xRatio, yRatio)
	if err != nil {
		return err
	}
	if err := postMessageChecked(hwnd, WM_MOUSEMOVE, 0, makeLParam(pt.X, pt.Y)); err != nil {
		return err
	}
	b.mu.Lock()
	b.cursor[hwnd] = pt
	b.mu.Unlock()
	return nil
}

func (b *PostMessageBackend) CursorRatio(hwnd win.HWND) (float64, float64, error) {
	var rect win.RECT
	win.GetClientRect(hwnd, &rect)
	w := float64(rect.Right - rect.Left)
	h := float64(rect.Bottom - rect.Top)
	if w <= 0 || h <= 0 {
		return 0, 0, fmt.Errorf("CursorRatio: client rect 为空 (hwnd=%v)", hwnd)
	}
	sx, sy := getCursorPos()
	cx, cy := screenToClient(hwnd, sx, sy)
	return float64(cx) / w, float64(cy) / h, nil
}

func (b *PostMessageBackend) Scroll(hwnd win.HWND, xRatio, yRatio float64, notches int, horizontal bool) error {
	b.ensureActivated(hwnd)
	pt, err := checkedClientPoint(hwnd, xRatio, yRatio)
	if err != nil {
		return err
	}
	if err := checkedClientToScreen(hwnd, &pt); err != nil {
		return err
	}
	delta := int16(notches * WheelDelta)
	wp := uintptr(uint32(uint16(delta))) << 16
	lp := makeLParam(pt.X, pt.Y)
	if horizontal {
		return postMessageChecked(hwnd, 0x020E, wp, lp)
	}
	return postMessageChecked(hwnd, WM_MOUSEWHEEL, wp, lp)
}

// ReleaseAll 放所有 held key + button. backend stateful 设计的核心.
func (b *PostMessageBackend) ReleaseAll() error {
	b.mu.Lock()
	keys := make(map[uint32]win.HWND, len(b.heldKeys))
	for key, hwnd := range b.heldKeys {
		keys[key] = hwnd
	}
	hwndBtns := make(map[win.HWND]map[string]point, len(b.heldBtns))
	for h, set := range b.heldBtns {
		btns := make(map[string]point, len(set))
		maps.Copy(btns, set)
		hwndBtns[h] = btns
	}
	b.mu.Unlock()
	var result error
	for key, hwnd := range keys {
		if err := postMessageChecked(hwnd, WM_KEYUP, uintptr(key), keyLParam(key, true)); err != nil {
			result = errors.Join(result, err)
			continue
		}
		b.mu.Lock()
		delete(b.heldKeys, key)
		b.mu.Unlock()
	}
	for h, btns := range hwndBtns {
		for bb, pt := range btns {
			message, flags := postMessageButton(bb, true)
			if err := postMessageChecked(h, message, flags, makeLParam(pt.X, pt.Y)); err != nil {
				result = errors.Join(result, err)
				continue
			}
			b.mu.Lock()
			delete(b.heldBtns[h], bb)
			if len(b.heldBtns[h]) == 0 {
				delete(b.heldBtns, h)
			}
			b.mu.Unlock()
		}
	}
	return result
}

func (b *PostMessageBackend) Close() error { return nil }

// TypeText 注入文本字符串。走 PostMessage WM_CHAR (PostText) 投递到目标 hwnd —— targeted、
// 后台可用、不抢前台, 与本 backend 的 KeyDown/Click 同语义。先 ensureActivated 翻 IsActive
// 兜 UE/Slate 类窗口在失焦时丢消息 (同其他操作)。
// (不走全局 SendInput 的 pkg-level TypeText: 那条注入到真实前台焦点窗口, 后台目标窗口收不到。)
func (b *PostMessageBackend) TypeText(hwnd win.HWND, s string) error {
	b.ensureActivated(hwnd)
	return PostText(hwnd, s)
}

func checkedClientSize(hwnd win.HWND) (int, int, error) {
	var rectangle win.RECT
	ok, _, errno := procGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rectangle)))
	width, height := int(rectangle.Right-rectangle.Left), int(rectangle.Bottom-rectangle.Top)
	if ok != 0 && width > 0 && height > 0 {
		return width, height, nil
	}
	if errno != syscall.Errno(0) {
		return 0, 0, fmt.Errorf("GetClientRect failed: %w", errno)
	}
	return 0, 0, errors.New("GetClientRect returned an empty target")
}

func checkedClientPoint(hwnd win.HWND, xRatio, yRatio float64) (point, error) {
	width, height, err := checkedClientSize(hwnd)
	if err != nil {
		return point{}, err
	}
	return point{X: int32(xRatio * float64(width-1)), Y: int32(yRatio * float64(height-1))}, nil
}

func checkedClientToScreen(hwnd win.HWND, value *point) error {
	ok, _, errno := procClientToScreen.Call(uintptr(hwnd), uintptr(unsafe.Pointer(value)))
	if ok != 0 {
		return nil
	}
	if errno != syscall.Errno(0) {
		return fmt.Errorf("ClientToScreen failed: %w", errno)
	}
	return errors.New("ClientToScreen rejected the target")
}

func postMessageButton(button string, up bool) (uint32, uintptr) {
	switch button {
	case "right":
		if up {
			return WM_RBUTTONUP, 0
		}
		return WM_RBUTTONDOWN, MK_RBUTTON
	case "middle":
		if up {
			return WM_MBUTTONUP, 0
		}
		return WM_MBUTTONDOWN, MK_MBUTTON
	default:
		if up {
			return WM_LBUTTONUP, 0
		}
		return WM_LBUTTONDOWN, MK_LBUTTON
	}
}
