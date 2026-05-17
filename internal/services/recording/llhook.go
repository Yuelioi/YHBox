// Package recording 是 actions 录制层：Win32 LL hook 包装 + 上层 recorder。
//
// 本文件 llhook.go 只做最薄一层：把 SetWindowsHookExW(WH_KEYBOARD_LL/WH_MOUSE_LL)
// 的 raw 事件解码成 Go struct HookEvent，并通过 channel 透传出去。
//
// 它不知道 gameHwnd，也不做 ScreenToClient —— 这些是 recorder.go 的事。
//
// 整个项目都是 Windows-only（cook/fish/piano/rhythm 全部直接调 Win32），
// 所以这里也不加 //go:build windows tag。
package recording

import (
	"errors"
	"fmt"
	"sync/atomic"
	"syscall"
	"unsafe"

	"github.com/lxn/win"
)

// --- Win32 常量 ---

const (
	whKeyboardLL = 13
	whMouseLL    = 14
	hcAction     = 0
	gaRoot       = 2
	wmQuit       = 0x0012

	wmKeyDown    = 0x0100
	wmKeyUp      = 0x0101
	wmSysKeyDown = 0x0104
	wmSysKeyUp   = 0x0105

	wmLButtonDown = 0x0201
	wmLButtonUp   = 0x0202
	wmRButtonDown = 0x0204
	wmRButtonUp   = 0x0205
	wmMButtonDown = 0x0207
	wmMButtonUp   = 0x0208
	wmMouseMove   = 0x0200
	wmMouseWheel  = 0x020A

	// 鼠标移动事件节流：每像素一个事件会把 256-cap channel 1 秒内打爆。
	// 30ms ≈ 33Hz，对 drag 中间帧轨迹够细，对录制器又不会爆。
	mouseMoveThrottleMs = 30
)

// HookMouseBtn 鼠标按键枚举（LL hook 层用）。
type HookMouseBtn int

const (
	HookBtnLeft   HookMouseBtn = 0
	HookBtnMiddle HookMouseBtn = 1
	HookBtnRight  HookMouseBtn = 2
)

// kbdllhookstruct 对应 Win32 KBDLLHOOKSTRUCT。
type kbdllhookstruct struct {
	VkCode      uint32
	ScanCode    uint32
	Flags       uint32
	Time        uint32
	DwExtraInfo uintptr
}

// msllhookstruct 对应 Win32 MSLLHOOKSTRUCT。Pt 是 POINT { int32 x, int32 y }。
type msllhookstruct struct {
	Pt          struct{ X, Y int32 }
	MouseData   uint32
	Flags       uint32
	Time        uint32
	DwExtraInfo uintptr
}

// --- Win32 proc 句柄 ---

var (
	user32                  = syscall.NewLazyDLL("user32.dll")
	procSetWindowsHookExW   = user32.NewProc("SetWindowsHookExW")
	procUnhookWindowsHookEx = user32.NewProc("UnhookWindowsHookEx")
	procCallNextHookEx      = user32.NewProc("CallNextHookEx")
	procGetMessageW         = user32.NewProc("GetMessageW")
	procWindowFromPoint     = user32.NewProc("WindowFromPoint")
	procGetParent           = user32.NewProc("GetParent")
	procGetAncestor         = user32.NewProc("GetAncestor")
	procPostThreadMessageW  = user32.NewProc("PostThreadMessageW")
	procScreenToClient      = user32.NewProc("ScreenToClient")

	kernel32               = syscall.NewLazyDLL("kernel32.dll")
	procGetCurrentThreadId = kernel32.NewProc("GetCurrentThreadId")
)

// HookEvent 给上层 recorder 的事件结构。
// 跟 actiontransform.Event 类似但含原始屏幕绝对坐标（ScreenX/ScreenY），
// recorder 拿到后再做 IsPointInsideGameWindow 过滤 + ScreenToClient 转换。
//
// llhook 这一层不知道 gameHwnd —— 它只负责把 Win32 hook 事件解码成 Go struct
// 透传出去。
type HookEvent struct {
	// TimestampMs: KBDLLHOOKSTRUCT.time / MSLLHOOKSTRUCT.time（uint32 ms tick）。
	// 用它而不是 time.Now() 是为了避开 channel + goroutine scheduling jitter。
	TimestampMs uint32

	// IsKeyboard: true=键盘事件，false=鼠标事件（按键/移动/滚轮 三种之一）。
	IsKeyboard bool

	// keyboard 字段（IsKeyboard=true 时有效）
	Vk        uint32 // virtual key code（Win32 VK_*）
	IsKeyDown bool

	// mouse 公共：屏幕绝对坐标（任何 mouse 事件都填）
	ScreenX int32
	ScreenY int32

	// mouse 按键事件（IsKeyboard=false 且不是 move/scroll 时有效）
	MouseBtn    HookMouseBtn // 0=left, 1=middle, 2=right
	IsMouseDown bool

	// mouse 移动事件（IsMouseMove=true 时；recorder 用来追踪 drag 中间轨迹）
	IsMouseMove bool

	// mouse 滚轮事件（IsScroll=true 时；WheelNotches 正=上推/下滑页面，负=下推）
	IsScroll     bool
	WheelNotches int

	// Raw Input 相对鼠标位移（IsRawDelta=true 时；camera 转向走这个）。
	// RawDx/RawDy 单位是设备 mickeys，跟 SendInput MOUSEEVENTF_MOVE 一一对应，
	// 录制端记下来回放时 driver.MouseMoveRel 用同一组数字即可还原相机角度。
	IsRawDelta bool
	RawDx      int
	RawDy      int
}

// HookHandle 是注册后的 hook 句柄。recorder.go 持有它，停录时 Uninstall。
type HookHandle struct {
	kb syscall.Handle
	ms syscall.Handle
}

// 全局 channel 指针 —— callback 是 C ABI 不能闭包捕获 Go 闭包变量，所以走全局。
// 同时只能装一份 hook（recorder.go 串行启停），全局 ok。
//
// 不需要 mutex 保护：activeEvents 只在 InstallHooks/Uninstall（同一 caller
// 线程）里读写；callback 只读。Install 完才会有 callback 被调用，Uninstall
// 之前 callback 已经走完。
// lastMouseMoveMs 给 mouseProc 做 30ms 节流。atomic 保证 callback 并发安全。
var lastMouseMoveMs atomic.Uint32

var (
	activeEvents chan<- HookEvent
	// 保留 hook id 以便万一同进程多次启停 debug
	kbHook uintptr
	msHook uintptr
)

// activeStopHotkeyVK F12 (或用户配置) 的 vk. recorder.Start 时 atomic 写入,
// keyboardProc 检测; 0 = 没启用停录热键.
var activeStopHotkeyVK uint32

// activeStopCallback 检测到停录热键时调. 用 atomic.Pointer 保证 hook 线程读到
// 跟 Start/Stop 线程写的同步 (普通指针赋值 Go 不保证跨线程可见).
var activeStopCallback atomic.Pointer[func()]

// InstallHooks 注册键盘+鼠标 LL hook。callback 在 OS hook 线程被调，
// 非阻塞 push 到 events channel；channel 满则 drop（caller 自己定 cap）。
// 返回的 HookHandle 必须由 caller 在停录时 Uninstall。
//
// IMPORTANT: caller 必须先 runtime.LockOSThread() —— Win32 hook 跟线程绑定，
// 注册线程和 GetMessage 必须是同一个 OS thread。
func InstallHooks(events chan<- HookEvent) (*HookHandle, error) {
	if activeEvents != nil {
		return nil, errors.New("LL hook 已注册，stop 旧的再装新的")
	}
	if events == nil {
		return nil, errors.New("events channel 不能为 nil")
	}
	activeEvents = events

	// syscall.NewCallback 把 Go func 包成 stdcall 兼容的 C 函数指针。
	// windows/amd64 上 calling convention 自动对齐。
	kbProc := syscall.NewCallback(keyboardProc)
	msProc := syscall.NewCallback(mouseProc)

	// SetWindowsHookExW(idHook, lpfn, hMod=0, dwThreadId=0)：
	// hMod=0 + dwThreadId=0 是 LL hook 标准用法（系统级，不需要 DLL）。
	kb, _, callErr := procSetWindowsHookExW.Call(uintptr(whKeyboardLL), kbProc, 0, 0)
	if kb == 0 {
		activeEvents = nil
		return nil, fmt.Errorf("SetWindowsHookExW(WH_KEYBOARD_LL): %v", callErr)
	}
	ms, _, callErr := procSetWindowsHookExW.Call(uintptr(whMouseLL), msProc, 0, 0)
	if ms == 0 {
		procUnhookWindowsHookEx.Call(kb)
		activeEvents = nil
		return nil, fmt.Errorf("SetWindowsHookExW(WH_MOUSE_LL): %v", callErr)
	}
	kbHook = kb
	msHook = ms
	return &HookHandle{kb: syscall.Handle(kb), ms: syscall.Handle(ms)}, nil
}

// Uninstall 反注册 hook。必须在跟 InstallHooks 同一个 OS 线程调。
// 多次调安全（idempotent）。
func (h *HookHandle) Uninstall() {
	if h == nil {
		return
	}
	if h.ms != 0 {
		procUnhookWindowsHookEx.Call(uintptr(h.ms))
		h.ms = 0
	}
	if h.kb != 0 {
		procUnhookWindowsHookEx.Call(uintptr(h.kb))
		h.kb = 0
	}
	kbHook = 0
	msHook = 0
	activeEvents = nil
}

// keyboardProc / mouseProc 必须 fast return。任何耗时操作（包括 channel send 阻塞）
// 都会让 OS 在 ~300ms 后自动 unhook。所以 channel 是非阻塞 push（select default drop）。
//
// 始终调 CallNextHookEx 把事件透传给下一个 hook —— 不影响其它应用收到键鼠。
func keyboardProc(nCode, wParam, lParam uintptr) uintptr {
	if nCode == hcAction && activeEvents != nil {
		kbd := (*kbdllhookstruct)(unsafe.Pointer(lParam))
		isKeyDown := wParam == wmKeyDown || wParam == wmSysKeyDown

		// 停录热键拦截: 触发 stop callback + 不透传游戏
		stopVK := atomic.LoadUint32(&activeStopHotkeyVK)
		if isKeyDown && stopVK != 0 && kbd.VkCode == stopVK {
			if cbp := activeStopCallback.Load(); cbp != nil {
				go (*cbp)()
			}
			return 1 // 不调 CallNextHookEx — 游戏收不到这个 keydown
		}

		ev := HookEvent{
			TimestampMs: kbd.Time,
			IsKeyboard:  true,
			Vk:          kbd.VkCode,
			IsKeyDown:   isKeyDown,
		}
		// 非阻塞 push；channel 满则 drop（buffer 256 一般够）
		select {
		case activeEvents <- ev:
		default:
		}
	}
	next, _, _ := procCallNextHookEx.Call(0, nCode, wParam, lParam)
	return next
}

// SetActiveStopHotkey recorder.Start 时调一次, Stop 时调一次清零.
func SetActiveStopHotkey(vk uint32, callback func()) {
	atomic.StoreUint32(&activeStopHotkeyVK, vk)
	if callback == nil {
		activeStopCallback.Store(nil)
	} else {
		activeStopCallback.Store(&callback)
	}
}

func mouseProc(nCode, wParam, lParam uintptr) uintptr {
	if nCode == hcAction && activeEvents != nil {
		ms := (*msllhookstruct)(unsafe.Pointer(lParam))
		ev := HookEvent{
			TimestampMs: ms.Time,
			IsKeyboard:  false,
			ScreenX:     ms.Pt.X,
			ScreenY:     ms.Pt.Y,
		}
		emit := false

		switch wParam {
		case wmLButtonDown:
			ev.MouseBtn, ev.IsMouseDown, emit = HookBtnLeft, true, true
		case wmLButtonUp:
			ev.MouseBtn, ev.IsMouseDown, emit = HookBtnLeft, false, true
		case wmRButtonDown:
			ev.MouseBtn, ev.IsMouseDown, emit = HookBtnRight, true, true
		case wmRButtonUp:
			ev.MouseBtn, ev.IsMouseDown, emit = HookBtnRight, false, true
		case wmMButtonDown:
			ev.MouseBtn, ev.IsMouseDown, emit = HookBtnMiddle, true, true
		case wmMButtonUp:
			ev.MouseBtn, ev.IsMouseDown, emit = HookBtnMiddle, false, true

		case wmMouseMove:
			// 30ms 节流，否则 channel cap=256 撑不过 1 秒。
			last := lastMouseMoveMs.Load()
			if ms.Time-last >= mouseMoveThrottleMs {
				lastMouseMoveMs.Store(ms.Time)
				ev.IsMouseMove = true
				emit = true
			}

		case wmMouseWheel:
			// HIWORD(mouseData) = wheel delta（有符号 int16）
			delta := int16(ms.MouseData >> 16)
			if delta != 0 {
				ev.IsScroll = true
				ev.WheelNotches = int(delta) / 120 // WHEEL_DELTA = 120
				if ev.WheelNotches == 0 {
					// 高 DPI 鼠标可能不到 120，至少保留方向
					if delta > 0 {
						ev.WheelNotches = 1
					} else {
						ev.WheelNotches = -1
					}
				}
				emit = true
			}
		}

		if emit {
			select {
			case activeEvents <- ev:
			default:
			}
		}
	}
	next, _, _ := procCallNextHookEx.Call(0, nCode, wParam, lParam)
	return next
}

// RunMessageLoop 阻塞拉本线程消息直到收到 WM_QUIT。caller 已 LockOSThread。
//
// LL hook callback 的 dispatch 是 SetWindowsHookEx 内部走的 —— OS 在
// hook 线程的消息泵 idle 时投递 callback。但我们还注册了一个消息-only window
// 接 WM_INPUT (Raw Input)，那个**需要** DispatchMessage 才能路由到我们的 WndProc。
// 所以这里走 Translate + Dispatch，对 LL hook 无害。
//
// GetMessage 返回：
//   - 0      → WM_QUIT，正常退出
//   - -1     → error
//   - 其它   → 普通消息
func RunMessageLoop() {
	var msg win.MSG
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		ret := int32(r)
		if ret == 0 || ret == -1 {
			return
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

// PostQuitToThread 给指定线程发 WM_QUIT，让 RunMessageLoop 返回。
// recorder.Stop 调它来让 worker goroutine 退出。
func PostQuitToThread(threadID uint32) {
	procPostThreadMessageW.Call(uintptr(threadID), wmQuit, 0, 0)
}

// GetCurrentThreadID 返回当前 OS 线程 ID。
// recorder worker 启动后用它把 thread id 暴露给 main goroutine，
// 让 Stop 时能 PostThreadMessage 过去。
func GetCurrentThreadID() uint32 {
	tid, _, _ := procGetCurrentThreadId.Call()
	return uint32(tid)
}

// IsPointInsideGameWindow 判断 screen 坐标 (screenX, screenY) 是否在 gameHwnd
// 的窗口（含其 child / overlay / IME / DX surface）内。
//
// 检查链：
//  1. WindowFromPoint(p) 拿到点下面的 hwnd h
//  2. h == gameHwnd 命中
//  3. 沿 GetParent chain 向上走，任一祖先 == gameHwnd 命中
//  4. GetAncestor(h, GA_ROOT) == gameHwnd 命中（top-level 覆盖）
//
// 单纯 WindowFromPoint == gameHwnd 不够 —— 游戏经常有 child overlay / IME
// 候选窗 / DirectX swap chain surface 盖在主窗口之上。
//
// gameHwnd == 0 时永远返 false（recorder 没设 hwnd 之前别记录）。
func IsPointInsideGameWindow(gameHwnd win.HWND, screenX, screenY int32) bool {
	if gameHwnd == 0 {
		return false
	}

	// WindowFromPoint 接受 POINT 结构（8 byte: int32 x + int32 y）。
	// windows/amd64 上 stdcall by-value 8-byte 结构通过单个 64-bit 寄存器/栈
	// 槽传递，可以 pack 进单个 uintptr：低 32 位 x，高 32 位 y。
	// 这是 syscall.Syscall 调 WindowFromPoint 的常见做法（lxn/walk 也这么干）。
	pt := uintptr(uint32(screenX)) | (uintptr(uint32(screenY)) << 32)
	r, _, _ := procWindowFromPoint.Call(pt)
	h := win.HWND(r)
	if h == 0 {
		return false
	}
	if h == gameHwnd {
		return true
	}

	// 沿 GetParent chain 向上找。GetParent 对 top-level 窗口返 0，
	// 对 owned 窗口返 owner，对 child 返 parent。
	// 加 16 步上限防意外死循环（正常 chain 不会超过几层）。
	cur := h
	for i := 0; i < 16; i++ {
		parent, _, _ := procGetParent.Call(uintptr(cur))
		p := win.HWND(parent)
		if p == 0 || p == cur {
			break
		}
		if p == gameHwnd {
			return true
		}
		cur = p
	}

	// GA_ROOT 拿 owner chain 顶端 top-level window —— 比 GetParent chain
	// 多覆盖 owned popup 情况。
	root, _, _ := procGetAncestor.Call(uintptr(h), gaRoot)
	return win.HWND(root) == gameHwnd
}

// screenToClient 屏幕坐标转 client 坐标。Win32 ScreenToClient in-place 修改 POINT。
// 失败（hwnd 无效等）返 (0,0,false)。
func screenToClient(hwnd win.HWND, screenX, screenY int32) (cx, cy int32, ok bool) {
	var pt struct{ X, Y int32 }
	pt.X = screenX
	pt.Y = screenY
	r, _, _ := procScreenToClient.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&pt)))
	if r == 0 {
		return 0, 0, false
	}
	return pt.X, pt.Y, true
}
