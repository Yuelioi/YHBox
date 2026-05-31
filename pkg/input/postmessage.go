package input

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/lxn/win"
)

// PostMessageBackend wraps existing pkg/input package-level functions, adds stateful
// tracking of held keys/buttons for ReleaseAll.
//
// 不重写底层 Win32 调用 — 复用 input.go 里 ClickButton/KeyDown/KeyUp/MouseMoveRel/...
// 的成熟实现. 本 struct 只加 state 跟踪 + interface 适配.
type PostMessageBackend struct {
	mu        sync.Mutex
	heldKeys  map[string]struct{}              // vk name → tracked
	heldBtns  map[win.HWND]map[string]struct{} // hwnd → button name → tracked
	activated map[win.HWND]bool                // hwnd → 是否已 FakeActivate 过
}

func newPostMessageBackend() *PostMessageBackend {
	return &PostMessageBackend{
		heldKeys:  map[string]struct{}{},
		heldBtns:  map[win.HWND]map[string]struct{}{},
		activated: map[win.HWND]bool{},
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
	defaultKeyPressMs    = 50
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
	}
}

// pixelCoords ratio → 客户区像素 (用 win.GetClientRect, 跟 input.go 同模式)
func (b *PostMessageBackend) pixelCoords(hwnd win.HWND, xRatio, yRatio float64) (int, int) {
	var rect win.RECT
	win.GetClientRect(hwnd, &rect)
	w := int(rect.Right - rect.Left)
	h := int(rect.Bottom - rect.Top)
	return int(float64(w) * xRatio), int(float64(h) * yRatio)
}

func (b *PostMessageBackend) Click(hwnd win.HWND, xRatio, yRatio float64, button string, durMs int) error {
	b.ensureActivated(hwnd)
	if durMs <= 0 {
		durMs = defaultClickHoldMs
	}
	x, y := b.pixelCoords(hwnd, xRatio, yRatio)
	ClickButton(hwnd, x, y, pickButton(button), time.Duration(durMs)*time.Millisecond, defaultActivateDelay, defaultCursorSettle)
	return nil
}

func (b *PostMessageBackend) KeyPress(hwnd win.HWND, vk string, durMs int) error {
	b.ensureActivated(hwnd)
	if durMs <= 0 {
		durMs = defaultKeyPressMs
	}
	Tap(hwnd, vk, time.Duration(durMs)*time.Millisecond, defaultActivateDelay)
	return nil
}

func (b *PostMessageBackend) KeyDown(hwnd win.HWND, vk string) error {
	b.ensureActivated(hwnd)
	KeyDown(hwnd, vk) // input.go 现有, 返 bool, 这里忽略 (按下即记录)
	b.mu.Lock()
	b.heldKeys[vk] = struct{}{}
	b.mu.Unlock()
	return nil
}

func (b *PostMessageBackend) KeyUp(hwnd win.HWND, vk string) error {
	KeyUp(hwnd, vk)
	b.mu.Lock()
	delete(b.heldKeys, vk)
	b.mu.Unlock()
	return nil
}

func (b *PostMessageBackend) MouseDown(hwnd win.HWND, xRatio, yRatio float64, button string) error {
	b.ensureActivated(hwnd)
	x, y := b.pixelCoords(hwnd, xRatio, yRatio)
	MouseBtnDown(hwnd, x, y, pickButton(button))
	b.mu.Lock()
	if b.heldBtns[hwnd] == nil {
		b.heldBtns[hwnd] = map[string]struct{}{}
	}
	b.heldBtns[hwnd][button] = struct{}{}
	b.mu.Unlock()
	return nil
}

func (b *PostMessageBackend) MouseUp(hwnd win.HWND, button string) error {
	MouseBtnUp(hwnd, 0, 0, pickButton(button)) // 坐标对 Up 基本无关紧要
	b.mu.Lock()
	if set, ok := b.heldBtns[hwnd]; ok {
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
	if durMs <= 0 {
		durMs = 200
	}
	MouseMoveRel(hwnd, dx, dy, time.Duration(durMs)*time.Millisecond, defaultActivateDelay)
	return nil
}

func (b *PostMessageBackend) MoveTo(hwnd win.HWND, xRatio, yRatio float64) error {
	b.ensureActivated(hwnd)
	x, y := b.pixelCoords(hwnd, xRatio, yRatio)
	MoveToClient(hwnd, x, y) // input.go: setCursorPos + WM_MOUSEMOVE, 无 sleep
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

func (b *PostMessageBackend) Scroll(hwnd win.HWND, xRatio, yRatio float64, notches int) error {
	b.ensureActivated(hwnd)
	_ = xRatio
	_ = yRatio
	// 现有 MouseScroll 不接 xy
	MouseScroll(hwnd, notches, defaultActivateDelay)
	return nil
}

// ReleaseAll 放所有 held key + button. backend stateful 设计的核心.
func (b *PostMessageBackend) ReleaseAll() error {
	b.mu.Lock()
	keys := make([]string, 0, len(b.heldKeys))
	for k := range b.heldKeys {
		keys = append(keys, k)
	}
	hwndBtns := make(map[win.HWND][]string, len(b.heldBtns))
	for h, set := range b.heldBtns {
		btns := make([]string, 0, len(set))
		for bb := range set {
			btns = append(btns, bb)
		}
		hwndBtns[h] = btns
	}
	activated := make([]win.HWND, 0, len(b.activated))
	for h := range b.activated {
		activated = append(activated, h)
	}
	b.heldKeys = map[string]struct{}{}
	b.heldBtns = map[win.HWND]map[string]struct{}{}
	b.mu.Unlock()

	// 单 container 一个 backend = 一个 hwnd. 优先用有 mouse button held 的 hwnd,
	// fallback 用 activated map (KeyDown 时 ensureActivated 一定写过 — 只用 KeyHold
	// 不点鼠标的场景, 之前 anyHwnd==0 → KeyUp 不发, container stop 后游戏端按键残留).
	var anyHwnd win.HWND
	for h := range hwndBtns {
		anyHwnd = h
		break
	}
	if anyHwnd == 0 {
		for _, h := range activated {
			anyHwnd = h
			break
		}
	}
	if anyHwnd != 0 {
		for _, vk := range keys {
			KeyUp(anyHwnd, vk)
		}
	}
	for h, btns := range hwndBtns {
		for _, bb := range btns {
			MouseBtnUp(h, 0, 0, pickButton(bb))
		}
		// input.ReleaseAll 是 hwnd-scoped 大杀器 — 清残留
		ReleaseAll(h)
	}
	return nil
}

func (b *PostMessageBackend) Close() error { return nil }
