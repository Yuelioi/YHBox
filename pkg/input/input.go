// Package input 后台键盘 / 鼠标控制。
//
// 关键 trick：
//   - 异环 IMC 在窗口内部 IsActive=false 时丢弃 PostMessage 键盘消息。
//     必须先 SendMessage(WM_ACTIVATE, WA_ACTIVE) 翻 IsActive=true，再 PostMessage。
//   - Slate UI 按钮必须先 SetCursorPos 移真实光标，再 PostMessage DOWN/UP。
package input

import (
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/lxn/win"
)

const (
	WM_KEYDOWN     = 0x0100
	WM_KEYUP       = 0x0101
	WM_ACTIVATE    = 0x0006
	WA_ACTIVE      = 1
	WM_MOUSEMOVE   = 0x0200
	WM_LBUTTONDOWN = 0x0201
	WM_LBUTTONUP   = 0x0202
	MK_LBUTTON     = 0x0001
)

var (
	user32             = syscall.NewLazyDLL("user32.dll")
	procPostMessageW   = user32.NewProc("PostMessageW")
	procSendMessageW   = user32.NewProc("SendMessageW")
	procGetCursorPos   = user32.NewProc("GetCursorPos")
	procSetCursorPos   = user32.NewProc("SetCursorPos")
	procClientToScreen = user32.NewProc("ClientToScreen")
	procMapVirtualKeyW = user32.NewProc("MapVirtualKeyW")
)

const mapVKToVSC = 0

type point struct {
	X, Y int32
}

func postMessage(hwnd win.HWND, msg uint32, wp, lp uintptr) {
	procPostMessageW.Call(uintptr(hwnd), uintptr(msg), wp, lp)
}

func sendMessage(hwnd win.HWND, msg uint32, wp, lp uintptr) {
	procSendMessageW.Call(uintptr(hwnd), uintptr(msg), wp, lp)
}

func getCursorPos() (int32, int32) {
	var p point
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&p)))
	return p.X, p.Y
}

func setCursorPos(x, y int32) {
	procSetCursorPos.Call(uintptr(x), uintptr(y))
}

func clientToScreen(hwnd win.HWND, x, y int32) (int32, int32) {
	p := point{X: x, Y: y}
	procClientToScreen.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&p)))
	return p.X, p.Y
}

// FakeActivate 把窗口内部 IsActive 翻成 true，不抢前台焦点。
// SendMessage 同步返回不代表 Slate 真处理完——它通常下一 UE tick 才翻 IsActive。
// 瞬时 PostMessage 之前需要 sleep ~30ms 让 Slate 真生效。
func FakeActivate(hwnd win.HWND) {
	sendMessage(hwnd, WM_ACTIVATE, WA_ACTIVE, 0)
}

// 虚拟键码表
var vkMap = map[string]uint32{
	"esc": 0x1B, "escape": 0x1B,
	"space": 0x20, "enter": 0x0D,
	"shift": 0x10, "ctrl": 0x11, "alt": 0x12,
}

// VK 解析键名（不区分大小写），返回 0 表示未知。
func VK(name string) uint32 {
	n := strings.ToLower(name)
	if v, ok := vkMap[n]; ok {
		return v
	}
	if len(n) == 1 {
		c := n[0]
		if c >= 'a' && c <= 'z' {
			return uint32(c - 'a' + 'A')
		}
		if c >= '0' && c <= '9' {
			return uint32(c)
		}
	}
	return 0
}

// keyLParam 拼 WM_KEYDOWN/UP 的 lParam：
//   - bits 0-15:  repeat=1
//   - bits 16-23: scancode（UE InputComponent 用 scancode 查 ProfileMap，不能为 0）
//   - bit 30:     previous state（KEYUP 置 1）
//   - bit 31:     transition state（KEYUP 置 1）
func keyLParam(vk uint32, keyUp bool) uintptr {
	r, _, _ := procMapVirtualKeyW.Call(uintptr(vk), mapVKToVSC)
	sc := uintptr(r) & 0xFF
	lp := uintptr(1) | (sc << 16)
	if keyUp {
		lp |= (1 << 30) | (1 << 31)
	}
	return lp
}

// Tap 瞬时按键（抛竿 F、收线 F、Esc 等）。
//   - activateDelay: FakeActivate 后等 Slate 翻 IsActive 的时间
//   - hold:          DOWN 到 UP 的间隔
func Tap(hwnd win.HWND, key string, hold, activateDelay time.Duration) bool {
	vk := VK(key)
	if vk == 0 {
		return false
	}
	FakeActivate(hwnd)
	if activateDelay > 0 {
		time.Sleep(activateDelay)
	}
	postMessage(hwnd, WM_KEYDOWN, uintptr(vk), keyLParam(vk, false))
	if hold > 0 {
		time.Sleep(hold)
	}
	postMessage(hwnd, WM_KEYUP, uintptr(vk), keyLParam(vk, true))
	return true
}

// KeyDown 按下键不松（溜鱼长按 A/D）。
func KeyDown(hwnd win.HWND, key string) bool {
	vk := VK(key)
	if vk == 0 {
		return false
	}
	postMessage(hwnd, WM_KEYDOWN, uintptr(vk), keyLParam(vk, false))
	return true
}

// KeyUp 松开键。
func KeyUp(hwnd win.HWND, key string) bool {
	vk := VK(key)
	if vk == 0 {
		return false
	}
	postMessage(hwnd, WM_KEYUP, uintptr(vk), keyLParam(vk, true))
	return true
}

// ReleaseAll 释放溜鱼用到的 A/D 键。
func ReleaseAll(hwnd win.HWND) {
	KeyUp(hwnd, "a")
	KeyUp(hwnd, "d")
}

// makeLParam 把客户区坐标编码进 LPARAM：低 16 位 = x，高 16 位 = y
func makeLParam(x, y int32) uintptr {
	return uintptr(uint32(uint16(x)) | uint32(uint16(y))<<16)
}

// Click 单击 Slate UI 按钮。坐标为客户区像素坐标。
//
// 关键修复：显式 PostMessage WM_MOUSEMOVE，不再依赖 SetCursorPos 的 OS 自然 MOUSEMOVE。
//
// 之前的 bug：SetCursorPos 让 OS 发 MOUSEMOVE 走系统消息层有延迟；而 DOWN/UP 走
// PostMessage 直接进游戏窗口队列。在 UI 动画期/紧帧下游戏队列顺序可能变成
//
//	DOWN → UP → MOUSEMOVE（MOUSEMOVE 迟到），Slate 处理 DOWN/UP 时 hover state
//
// 还指着上一个位置 → click 落空。表现就是「检测到了、点了，没生效」。
//
// 现在显式 PostMessage WM_MOUSEMOVE 进队列，保证 Slate 在 DOWN/UP 之前已收到
// hover 转移事件。SetCursorPos 仍保留，用途：1) 给依赖 GetCursorPos 的代码用
// 2) 让用户能看到光标移动（好调试）。
//
//   - activateDelay: FakeActivate 后等 Slate 翻 IsActive 的时间
//   - cursorSettle:  WM_MOUSEMOVE 入队后等 Slate 在它的 tick 处理 hover（≥ 16ms@60fps）
//   - hold:          DOWN+UP 之后等游戏处理事件的时间
func Click(hwnd win.HWND, clientX, clientY int, hold, activateDelay, cursorSettle time.Duration) {
	FakeActivate(hwnd)
	if activateDelay > 0 {
		time.Sleep(activateDelay)
	}

	origX, origY := getCursorPos()
	sx, sy := clientToScreen(hwnd, int32(clientX), int32(clientY))
	setCursorPos(sx, sy)

	lp := makeLParam(int32(clientX), int32(clientY))

	// 显式 hover 事件先入队（wParam=0：未按任何按键，纯 hover）。
	// Slate 至少需要一个 tick 处理 MOUSEMOVE 更新 hover 元素，所以 cursorSettle ≥ 16ms。
	postMessage(hwnd, WM_MOUSEMOVE, 0, lp)
	if cursorSettle > 0 {
		time.Sleep(cursorSettle)
	}

	// DOWN 和 UP 紧贴 — Slate 长 DOWN 会当成"按住"不触发 click 事件
	postMessage(hwnd, WM_LBUTTONDOWN, MK_LBUTTON, lp)
	postMessage(hwnd, WM_LBUTTONUP, 0, lp)
	if hold > 0 {
		time.Sleep(hold)
	}

	setCursorPos(origX, origY)
}
