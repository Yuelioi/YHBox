// Package input 后台键盘 / 鼠标控制。
//
// 关键 trick：
//   - 异环 IMC 在窗口内部 IsActive=false 时丢弃 PostMessage 键盘消息。
//     必须先 SendMessage(WM_ACTIVATE, WA_ACTIVE) 翻 IsActive=true，再 PostMessage。
//   - Slate UI 按钮必须先 SetCursorPos 移真实光标，再 PostMessage DOWN/UP。
package input

import (
	"fmt"
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
	WM_RBUTTONDOWN = 0x0204
	WM_RBUTTONUP   = 0x0205
	WM_MBUTTONDOWN = 0x0207
	WM_MBUTTONUP   = 0x0208
	WM_MOUSEWHEEL  = 0x020A
	MK_LBUTTON     = 0x0001
	MK_RBUTTON     = 0x0002
	MK_MBUTTON     = 0x0010
	// 一格滚轮 = 120 单位（Win32 约定）
	WheelDelta = 120
)

// MouseButton 区分鼠标三键。Click 走默认 MouseLeft；ClickButton 接受任意。
type MouseButton int

const (
	MouseLeft MouseButton = iota
	MouseMiddle
	MouseRight
)

var (
	user32                = syscall.NewLazyDLL("user32.dll")
	procPostMessageW      = user32.NewProc("PostMessageW")
	procSendMessageW      = user32.NewProc("SendMessageW")
	procGetCursorPos      = user32.NewProc("GetCursorPos")
	procSetCursorPos      = user32.NewProc("SetCursorPos")
	procClientToScreen    = user32.NewProc("ClientToScreen")
	procGetClientRect     = user32.NewProc("GetClientRect")
	procMapVirtualKeyW    = user32.NewProc("MapVirtualKeyW")
	procSendInput         = user32.NewProc("SendInput")
	procSetForegroundWnd  = user32.NewProc("SetForegroundWindow")
	procShowWindow        = user32.NewProc("ShowWindow")
	procIsIconic          = user32.NewProc("IsIconic")
)

// rect 对应 Win32 RECT。
type rect struct {
	Left, Top, Right, Bottom int32
}

const swRestore = 9

// BringToForeground 把窗口拽到前台 + 最小化时还原。返 true 表示 SetForegroundWindow OS 调用成功（r != 0）。
func BringToForeground(hwnd win.HWND) bool {
	r, _, _ := procIsIconic.Call(uintptr(hwnd))
	if r != 0 {
		procShowWindow.Call(uintptr(hwnd), swRestore)
	}
	r2, _, _ := procSetForegroundWnd.Call(uintptr(hwnd))
	return r2 != 0
}

// SendInput INPUT 结构（amd64：type 4 + pad 4 + MOUSEINPUT 24 + tail pad = 40 bytes）。
// 用于注入 raw mouse input，绕开 PostMessage/SetCursorPos 走 cursor 位置的路径，
// 直接进 OS raw input pipeline —— 游戏读 Raw Input API（FPS / 摄像机控制）能收到。
type mouseInput struct {
	Dx        int32
	Dy        int32
	MouseData uint32
	Flags     uint32
	Time      uint32
	ExtraInfo uintptr
}

type sendInputBlock struct {
	Type uint32
	_    uint32 // pad on amd64
	Mi   mouseInput
}

const (
	mouseEventFMove uint32 = 0x0001 // 相对位移（不带 ABSOLUTE）
)

// sendInputMouseRel 单次注入相对 mouse 位移到 OS raw input。
// 走 SendInput 而不是 mouse_event：后者已 deprecated，内部也走 SendInput。
// 注：dx/dy 是**屏幕像素**单位，跟 SetCursorPos 一致；游戏内部按它自己的灵敏度换算转角。
func sendInputMouseRel(dx, dy int32) {
	in := sendInputBlock{
		Type: 0, // INPUT_MOUSE
		Mi: mouseInput{
			Dx:    dx,
			Dy:    dy,
			Flags: mouseEventFMove,
		},
	}
	procSendInput.Call(1, uintptr(unsafe.Pointer(&in)), unsafe.Sizeof(in))
}

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

// 虚拟键码表。键值必须覆盖 recorder vkName / hotkey parseHotkeyString 的所有输出，
// 否则录制 → 回放 / 热键触发 这条路径会断（VK=0 报"unknown vk"中断 action）。
var vkMap = map[string]uint32{
	"esc": 0x1B, "escape": 0x1B,
	"space": 0x20,
	"enter": 0x0D, "return": 0x0D,
	"shift": 0x10, "ctrl": 0x11, "control": 0x11, "alt": 0x12,
	"tab":       0x09,
	"backspace": 0x08, "back": 0x08,
	"delete": 0x2E, "del": 0x2E,
	"insert": 0x2D, "ins": 0x2D,
	"home": 0x24, "end": 0x23,
	"pgup": 0x21, "pageup": 0x21,
	"pgdn": 0x22, "pagedown": 0x22,
	"up": 0x26, "down": 0x28, "left": 0x25, "right": 0x27,
	",": 0xBC, ".": 0xBE,
	"caps": 0x14, "capslock": 0x14,
	// F1-F12 走 VK() 里的循环匹配，不入表
}

// VK 解析键名（不区分大小写），返回 0 表示未知。
func VK(name string) uint32 {
	n := strings.ToLower(strings.TrimSpace(name))
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
	// F1-F12: VK_F1=0x70, VK_F12=0x7B
	if len(n) >= 2 && n[0] == 'f' {
		var num int
		if _, err := fmt.Sscanf(n[1:], "%d", &num); err == nil && num >= 1 && num <= 12 {
			return 0x70 + uint32(num) - 1
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
//   - hold:          DOWN 到 UP 之间按住的时长（>0 给录制的长按蓄力技能用；
//                    cook bot UI 按钮传 0 退化为紧贴 DOWN/UP）
func Click(hwnd win.HWND, clientX, clientY int, hold, activateDelay, cursorSettle time.Duration) {
	ClickButton(hwnd, clientX, clientY, MouseLeft, hold, activateDelay, cursorSettle)
}

// ClickButton 同 Click 但可选左/中/右键。
// down/up message + MK_* wParam 按 button 派发；其余流程（FakeActivate / 光标移动 / hover 入队）不变。
func ClickButton(hwnd win.HWND, clientX, clientY int, button MouseButton, hold, activateDelay, cursorSettle time.Duration) {
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

	var downMsg, upMsg uint32
	var downMK uintptr
	switch button {
	case MouseMiddle:
		downMsg, upMsg, downMK = WM_MBUTTONDOWN, WM_MBUTTONUP, MK_MBUTTON
	case MouseRight:
		downMsg, upMsg, downMK = WM_RBUTTONDOWN, WM_RBUTTONUP, MK_RBUTTON
	default:
		downMsg, upMsg, downMK = WM_LBUTTONDOWN, WM_LBUTTONUP, MK_LBUTTON
	}

	// hold = 0 时退化为旧的"紧贴 DOWN/UP" 行为（cook bot UI 按钮走这条）。
	// hold > 0 时把 sleep 放在 DOWN→UP 之间 —— 回放录制的长按蓄力技能。
	postMessage(hwnd, downMsg, downMK, lp)
	if hold > 0 {
		time.Sleep(hold)
	}
	postMessage(hwnd, upMsg, 0, lp)

	setCursorPos(origX, origY)
}

// ClickButtonNoRestore 同 ClickButton 但**不**把光标移回起点 —— action 回放专用。
//
// 为什么分两个函数：cook bot 在桌面跑用户也在用鼠标，cursor restore 避免抢用户光标
// （cook 用 ClickButton）。action 回放时游戏已经在前台，用户希望"按录制时的光标轨迹走"，
// restore 反而造成"飘"—— 鼠标快速从点击位置弹回原位置很显眼（runner 用这个）。
func ClickButtonNoRestore(hwnd win.HWND, clientX, clientY int, button MouseButton, hold, activateDelay, cursorSettle time.Duration) {
	FakeActivate(hwnd)
	if activateDelay > 0 {
		time.Sleep(activateDelay)
	}

	sx, sy := clientToScreen(hwnd, int32(clientX), int32(clientY))
	setCursorPos(sx, sy)

	lp := makeLParam(int32(clientX), int32(clientY))
	postMessage(hwnd, WM_MOUSEMOVE, 0, lp)
	if cursorSettle > 0 {
		time.Sleep(cursorSettle)
	}

	var downMsg, upMsg uint32
	var mk uintptr
	switch button {
	case MouseMiddle:
		downMsg, upMsg, mk = WM_MBUTTONDOWN, WM_MBUTTONUP, MK_MBUTTON
	case MouseRight:
		downMsg, upMsg, mk = WM_RBUTTONDOWN, WM_RBUTTONUP, MK_RBUTTON
	default:
		downMsg, upMsg, mk = WM_LBUTTONDOWN, WM_LBUTTONUP, MK_LBUTTON
	}
	// hold > 0 → sleep 放 DOWN→UP 之间，回放录制的长按蓄力技能。
	// 0 → 退化为紧贴 DOWN/UP。
	postMessage(hwnd, downMsg, mk, lp)
	if hold > 0 {
		time.Sleep(hold)
	}
	postMessage(hwnd, upMsg, 0, lp)
	// 不 setCursorPos(origX, origY) —— 让光标留在 click 处
}

// MouseMove 把光标移到指定客户区坐标 + 给游戏发 WM_MOUSEMOVE。
// 不还原光标位置（跟 Click 不同 —— 用户明确要光标停在那）。
//   - 游戏读 cursor 位置（旧 UI/Slate 等）：生效
//   - 游戏读 Raw Input（FPS camera 等）：**不生效**，需要 v2 SendInput driver
func MouseMove(hwnd win.HWND, clientX, clientY int, activateDelay, cursorSettle time.Duration) {
	FakeActivate(hwnd)
	if activateDelay > 0 {
		time.Sleep(activateDelay)
	}
	sx, sy := clientToScreen(hwnd, int32(clientX), int32(clientY))
	setCursorPos(sx, sy)
	postMessage(hwnd, WM_MOUSEMOVE, 0, makeLParam(int32(clientX), int32(clientY)))
	if cursorSettle > 0 {
		time.Sleep(cursorSettle)
	}
}

// MouseDrag 按住 button 从 (x1,y1) 拖到 (x2,y2)，整体过程 durationMs。
// 中间均匀生成 N 个 MOUSEMOVE 帧让游戏感知到"拖动中"。durationMs=0 → 瞬移 down+up。
//
// 适用：拖物品 / 拖滑块 / 拖编辑器手柄。
// 不适用：camera 转向（Raw Input）。
func MouseDrag(hwnd win.HWND, x1, y1, x2, y2 int, button MouseButton, duration, activateDelay, cursorSettle time.Duration) {
	FakeActivate(hwnd)
	if activateDelay > 0 {
		time.Sleep(activateDelay)
	}

	sx1, sy1 := clientToScreen(hwnd, int32(x1), int32(y1))
	setCursorPos(sx1, sy1)
	postMessage(hwnd, WM_MOUSEMOVE, 0, makeLParam(int32(x1), int32(y1)))
	if cursorSettle > 0 {
		time.Sleep(cursorSettle)
	}

	var downMsg, upMsg uint32
	var mk uintptr
	switch button {
	case MouseMiddle:
		downMsg, upMsg, mk = WM_MBUTTONDOWN, WM_MBUTTONUP, MK_MBUTTON
	case MouseRight:
		downMsg, upMsg, mk = WM_RBUTTONDOWN, WM_RBUTTONUP, MK_RBUTTON
	default:
		downMsg, upMsg, mk = WM_LBUTTONDOWN, WM_LBUTTONUP, MK_LBUTTON
	}

	postMessage(hwnd, downMsg, mk, makeLParam(int32(x1), int32(y1)))

	// 中间过渡：每 ~16ms 一帧，按 duration 计算帧数。最少 1 帧（终点）。
	frames := int(duration / (16 * time.Millisecond))
	if frames < 1 {
		frames = 1
	}
	for i := 1; i <= frames; i++ {
		t := float64(i) / float64(frames)
		cx := int32(float64(x1) + t*float64(x2-x1))
		cy := int32(float64(y1) + t*float64(y2-y1))
		sx, sy := clientToScreen(hwnd, cx, cy)
		setCursorPos(sx, sy)
		postMessage(hwnd, WM_MOUSEMOVE, mk, makeLParam(cx, cy))
		if frames > 1 {
			time.Sleep(duration / time.Duration(frames))
		}
	}

	postMessage(hwnd, upMsg, 0, makeLParam(int32(x2), int32(y2)))
}

// MouseMoveRel 注入相对 mouse 位移走 OS raw input —— 游戏相机/视角转向走这条。
//
// 跟 MouseMove / Click 的区别：后者改 cursor 位置 + 发 WM_MOUSEMOVE，FPS/TPS 游戏读
// Raw Input API 收不到。SendInput 直接进 raw input pipeline，凡是读 GetRawInputData /
// DirectInput 的游戏都能收到（异环实测可用）。
//
// 实现：duration 内拆 ~16ms 一帧，把总 (dx, dy) 均分到帧。一帧 dx 太大游戏可能丢失中间值，
// 拆帧让相机平滑转。duration=0 或 ≤16ms → 单帧一次性投递。
//
// 注意：SendInput 走 OS 全局，**不需要游戏在前台也能投** —— 但游戏内部 RegisterRawInputDevices
// 时若指定了 RIDEV_CAPTUREMOUSE / RIDEV_EXINPUTSINK 才能在 background 收。多数游戏不开后台 sink，
// 所以仍然建议前台跑。FakeActivate 仍调一下保险。
func MouseMoveRel(hwnd win.HWND, totalDx, totalDy int, duration, activateDelay time.Duration) {
	FakeActivate(hwnd)
	if activateDelay > 0 {
		time.Sleep(activateDelay)
	}

	// 锚点：客户区中心 screen 坐标。每帧 SendInput 之后把 OS cursor 拽回这里，
	// 抵消"OS cursor 位置随 SendInput 累加位移"的视觉飘动。
	//
	// 原理：SendInput MOUSEEVENTF_MOVE 的 dx/dy 单位是 mickeys（设备相对单位），
	// 进 OS raw input pipeline → 游戏读到 → 视角转向。同时 OS 把 dx/dy 按 mouse
	// speed + DPI scaling 转 pixel 累加到 cursor。150% DPI 下一个 mickey 拖 cursor
	// 更多 pixel，几百 mickey 的视角转向就让 cursor 飞出屏幕。屏外被 clamp 在边缘，
	// 下一个 click step 再 SetCursorPos 拽回，视觉上是"瞬移到边再瞬移回点击点"。
	//
	// 修复：每帧 SendInput 后立刻 SetCursorPos(中心)。SetCursorPos 只改 OS cursor
	// 位置，不发 raw input、不发 WM_MOUSEMOVE，所以游戏视角累计不受影响。
	// 视觉上 cursor 始终钉在窗口中心，完全消除飘动。
	var anchorSx, anchorSy int32
	var hasAnchor bool
	var rc rect
	procGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rc)))
	if rc.Right > 0 && rc.Bottom > 0 {
		cx := rc.Right / 2
		cy := rc.Bottom / 2
		anchorSx, anchorSy = clientToScreen(hwnd, cx, cy)
		setCursorPos(anchorSx, anchorSy)
		hasAnchor = true
	}

	frames := int(duration / (16 * time.Millisecond))
	if frames < 1 {
		frames = 1
	}

	// 累积取整避免分母不整除时丢/多 px
	var emittedDx, emittedDy int
	for i := 1; i <= frames; i++ {
		targetDx := totalDx * i / frames
		targetDy := totalDy * i / frames
		stepDx := targetDx - emittedDx
		stepDy := targetDy - emittedDy
		if stepDx != 0 || stepDy != 0 {
			sendInputMouseRel(int32(stepDx), int32(stepDy))
			// 每帧 SendInput 后立刻拽回锚点，cursor 视觉上不动。
			if hasAnchor {
				setCursorPos(anchorSx, anchorSy)
			}
		}
		emittedDx, emittedDy = targetDx, targetDy
		if frames > 1 {
			time.Sleep(duration / time.Duration(frames))
		}
	}
}

// SendInputMouseRel 暴露 sendInputMouseRel 给 inputclip backends 用 (相机转向唯一路径).
// 不调 FakeActivate / setCursorPos — caller (ClipPlayer) 已按帧调度.
func SendInputMouseRel(dx, dy int32) {
	sendInputMouseRel(dx, dy)
}

// PostKeyDownVK 直接按 vk uint32 PostMessage 键盘按下到 hwnd (clip 回放用).
// keyLParam 自带 scancode (UE InputComponent 必需). 不调 FakeActivate (ClipPlayer 控时序).
func PostKeyDownVK(hwnd win.HWND, vk uint32) {
	postMessage(hwnd, WM_KEYDOWN, uintptr(vk), keyLParam(vk, false))
}

// PostKeyUpVK 同上, 松开.
func PostKeyUpVK(hwnd win.HWND, vk uint32) {
	postMessage(hwnd, WM_KEYUP, uintptr(vk), keyLParam(vk, true))
}

// MouseBtnDown 鼠标按键按下 (PostMessage 路径, 不松开). clip 回放 down/up 分离用.
// clientX/Y 客户区像素; button 见 MouseButton enum. 不 FakeActivate (caller 节奏控制).
func MouseBtnDown(hwnd win.HWND, clientX, clientY int, button MouseButton) {
	var downMsg uint32
	var mk uintptr
	switch button {
	case MouseMiddle:
		downMsg, mk = WM_MBUTTONDOWN, MK_MBUTTON
	case MouseRight:
		downMsg, mk = WM_RBUTTONDOWN, MK_RBUTTON
	default:
		downMsg, mk = WM_LBUTTONDOWN, MK_LBUTTON
	}
	postMessage(hwnd, downMsg, mk, makeLParam(int32(clientX), int32(clientY)))
}

// MouseBtnUp 鼠标按键松开.
func MouseBtnUp(hwnd win.HWND, clientX, clientY int, button MouseButton) {
	var upMsg uint32
	switch button {
	case MouseMiddle:
		upMsg = WM_MBUTTONUP
	case MouseRight:
		upMsg = WM_RBUTTONUP
	default:
		upMsg = WM_LBUTTONUP
	}
	postMessage(hwnd, upMsg, 0, makeLParam(int32(clientX), int32(clientY)))
}

// MouseScroll 在当前光标位置滚 notches 格滚轮（正=上推 = 通常向上滚页面）。
// 一格 = 120 wheel-delta 单位。
// 走 WM_MOUSEWHEEL —— wheel 事件需要光标坐标走 screen 坐标，按 Win32 约定。
func MouseScroll(hwnd win.HWND, notches int, activateDelay time.Duration) {
	FakeActivate(hwnd)
	if activateDelay > 0 {
		time.Sleep(activateDelay)
	}
	cx, cy := getCursorPos()
	// WM_MOUSEWHEEL wParam: HIWORD=delta, LOWORD=MK_* flags（这里 0）
	// lParam: HIWORD=screen Y, LOWORD=screen X
	delta := int16(notches * WheelDelta)
	wp := uintptr(uint32(uint16(delta))) << 16
	lp := uintptr(uint32(uint16(cx))) | uintptr(uint32(uint16(cy)))<<16
	postMessage(hwnd, WM_MOUSEWHEEL, wp, lp)
}
