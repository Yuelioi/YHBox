package input

import (
	"fmt"
	"maps"
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
	heldKeys  map[string]struct{}           // vk name → tracked
	heldBtns  map[win.HWND]map[string]point // hwnd → button name → 按下时客户区坐标
	activated map[win.HWND]bool             // hwnd → 是否已 FakeActivate 过
}

func newPostMessageBackend() *PostMessageBackend {
	return &PostMessageBackend{
		heldKeys:  map[string]struct{}{},
		heldBtns:  map[win.HWND]map[string]point{},
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
	// 跟 ClickButton 同序: 先 hover (setCursorPos + WM_MOUSEMOVE) + settle 让 Slate 在它的
	// tick 更新 hover 元素, 再 DOWN —— 否则按下可能落不到目标控件 (跟 ClickAt 旧路径同源).
	// 不松开 / 不复位光标 (hold 语义: 光标留在按下点直到 MouseHoldStop).
	MoveToClient(hwnd, x, y)
	time.Sleep(defaultCursorSettle)
	MouseBtnDown(hwnd, x, y, pickButton(button))
	b.mu.Lock()
	if b.heldBtns[hwnd] == nil {
		b.heldBtns[hwnd] = map[string]point{}
	}
	b.heldBtns[hwnd][button] = point{X: int32(x), Y: int32(y)}
	b.mu.Unlock()
	return nil
}

func (b *PostMessageBackend) MouseUp(hwnd win.HWND, button string) error {
	b.mu.Lock()
	pt, ok := b.heldBtns[hwnd][button]
	if set := b.heldBtns[hwnd]; set != nil {
		delete(set, button)
		if len(set) == 0 {
			delete(b.heldBtns, hwnd)
		}
	}
	b.mu.Unlock()
	if !ok {
		// 没记到按下坐标 (理论上不该发生) → 退回当前光标客户区位置, 而不是 (0,0).
		sx, sy := getCursorPos()
		cx, cy := screenToClient(hwnd, sx, sy)
		pt = point{X: cx, Y: cy}
	}
	// 在按下坐标松键 — UE Slate 在 BUTTONUP 上判 click 且看坐标, (0,0) 会被当成控件外松手 → 不触发.
	MouseBtnUp(hwnd, int(pt.X), int(pt.Y), pickButton(button))
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

func (b *PostMessageBackend) Drag(hwnd win.HWND, x1Ratio, y1Ratio, x2Ratio, y2Ratio float64, button string, durationMs int) error {
	b.ensureActivated(hwnd)
	x1, y1 := b.pixelCoords(hwnd, x1Ratio, y1Ratio)
	x2, y2 := b.pixelCoords(hwnd, x2Ratio, y2Ratio)
	if durationMs <= 0 {
		durationMs = 200
	}
	MouseDrag(hwnd, x1, y1, x2, y2, pickButton(button), time.Duration(durationMs)*time.Millisecond, defaultActivateDelay, defaultCursorSettle)
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

func (b *PostMessageBackend) Scroll(hwnd win.HWND, xRatio, yRatio float64, notches int, horizontal bool) error {
	b.ensureActivated(hwnd)
	_ = xRatio
	_ = yRatio
	if horizontal {
		MouseScrollH(hwnd, notches, defaultActivateDelay)
	} else {
		MouseScroll(hwnd, notches, defaultActivateDelay)
	}
	return nil
}

// ReleaseAll 放所有 held key + button. backend stateful 设计的核心.
func (b *PostMessageBackend) ReleaseAll() error {
	b.mu.Lock()
	keys := make([]string, 0, len(b.heldKeys))
	for k := range b.heldKeys {
		keys = append(keys, k)
	}
	hwndBtns := make(map[win.HWND]map[string]point, len(b.heldBtns))
	for h, set := range b.heldBtns {
		btns := make(map[string]point, len(set))
		maps.Copy(btns, set)
		hwndBtns[h] = btns
	}
	activated := make([]win.HWND, 0, len(b.activated))
	for h := range b.activated {
		activated = append(activated, h)
	}
	b.heldKeys = map[string]struct{}{}
	b.heldBtns = map[win.HWND]map[string]point{}
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
		for bb, pt := range btns {
			MouseBtnUp(h, int(pt.X), int(pt.Y), pickButton(bb))
		}
	}
	return nil
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
