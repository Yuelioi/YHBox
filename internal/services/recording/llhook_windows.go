package recording

import (
	"errors"
	"fmt"
	"os"
	"sync/atomic"
	"syscall"
	"unsafe"

	"github.com/lxn/win"

	"github.com/yottaapp/yotta/pkg/winutil"
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

// HookHandle 是注册后的 hook 句柄。Windows recorder 持有它，停录时 Uninstall。
type HookHandle struct {
	kb syscall.Handle
	ms syscall.Handle
}

// 全局 channel 指针 —— callback 是 C ABI 不能闭包捕获 Go 闭包变量，所以走全局。
// 同时只能装一份 hook（Windows recorder 串行启停），全局 ok。
//
// 不需要 mutex 保护：activeEvents 只在 InstallHooks/Uninstall（同一 caller
// 线程）里读写；callback 只读。Install 完才会有 callback 被调用，Uninstall
// 之前 callback 已经走完。
// lastMouseMoveMs 给 mouseProc 做 30ms 节流。atomic 保证 callback 并发安全。
var lastMouseMoveMs atomic.Uint32

var activeEvents chan<- HookEvent

// activeStopHotkeyVK F12 (或用户配置) 的 vk. recorder.Start 时 atomic 写入,
// keyboardProc 检测; 0 = 没启用停录热键.
var activeStopHotkeyVK uint32

// activeStopCallback 检测到停录热键时调. 用 atomic.Pointer 保证 hook 线程读到
// 跟 Start/Stop 线程写的同步 (普通指针赋值 Go 不保证跨线程可见).
var activeStopCallback atomic.Pointer[func()]

// activePauseHotkeyVK / activePauseCallback 暂停/继续切换热键. 跟停录不同: callback
// 可重复触发 (录制全程多次暂停/继续), 不能 Swap(nil) 一次性消费 — 用 Load.
// pauseKeyHeld 去抖 autorepeat: 按住 OS 高频重发 keydown, 只在 down 跳变 (held false→true)
// 时 fire 一次, keyup 复位 (对照 ll-hook-keydown-coalesce incident).
var activePauseHotkeyVK uint32
var activePauseCallback atomic.Pointer[func()]
var pauseKeyHeld atomic.Bool

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
	activeEvents = nil
}

// keyboardProc / mouseProc 必须 fast return。任何耗时操作（包括 channel send 阻塞）
// 都会让 OS 在 ~300ms 后自动 unhook。所以 channel 是非阻塞 push（select default drop）。
//
// 始终调 CallNextHookEx 把事件透传给下一个 hook —— 不影响其它应用收到键鼠。
func keyboardProc(nCode, wParam, lParam uintptr) uintptr {
	defer func() {
		if r := recover(); r != nil {
			// LL hook 不能死: 单条事件 panic 静默 swallow, hook 继续监听
			fmt.Fprintf(os.Stderr, "[llhook recover] keyboardProc: %v\n", r)
		}
	}()
	if nCode == hcAction && activeEvents != nil {
		kbd := winutil.ReadStructFromPointer[kbdllhookstruct](lParam)
		isKeyDown := wParam == wmKeyDown || wParam == wmSysKeyDown

		// 停录热键拦截: 触发 stop callback + 不透传游戏.
		// Swap 而不是 Load — 同一 session 内只允许一次 callback (F12 keydown auto-repeat /
		// 用户按住, OS 会高频重发 keydown; 不 swap 会 spawn 多个 StopAsync, 后面的全部
		// 撞 "recorder not active").
		stopVK := atomic.LoadUint32(&activeStopHotkeyVK)
		if isKeyDown && stopVK != 0 && kbd.VkCode == stopVK {
			if cbp := activeStopCallback.Swap(nil); cbp != nil {
				go (*cbp)()
			}
			return 1 // 不调 CallNextHookEx — 游戏收不到这个 keydown
		}

		// 暂停/继续切换热键拦截: down 跳变 fire (autorepeat 不重复), up 复位.
		// down+up 都 return 1 拦截 → 既不透传游戏也不进 clip (录制器收不到这个键).
		pauseVK := atomic.LoadUint32(&activePauseHotkeyVK)
		if pauseVK != 0 && kbd.VkCode == pauseVK {
			if isKeyDown {
				if !pauseKeyHeld.Swap(true) {
					if cbp := activePauseCallback.Load(); cbp != nil {
						go (*cbp)()
					}
				}
			} else {
				pauseKeyHeld.Store(false)
			}
			return 1
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

// setActiveStopHotkey recorder.Start 时调一次, Stop 时调一次清零.
func setActiveStopHotkey(vk uint32, callback func()) {
	atomic.StoreUint32(&activeStopHotkeyVK, vk)
	if callback == nil {
		activeStopCallback.Store(nil)
	} else {
		activeStopCallback.Store(&callback)
	}
}

// setActivePauseHotkey 暂停/继续切换热键. Start 时设, Stop 时清 (vk=0,nil).
// 复位 pauseKeyHeld 防上个 session 残留的"按住"态串到新 session.
func setActivePauseHotkey(vk uint32, callback func()) {
	atomic.StoreUint32(&activePauseHotkeyVK, vk)
	pauseKeyHeld.Store(false)
	if callback == nil {
		activePauseCallback.Store(nil)
	} else {
		activePauseCallback.Store(&callback)
	}
}

func mouseProc(nCode, wParam, lParam uintptr) uintptr {
	defer func() {
		if r := recover(); r != nil {
			// LL hook 不能死: 单条事件 panic 静默 swallow, hook 继续监听
			fmt.Fprintf(os.Stderr, "[llhook recover] mouseProc: %v\n", r)
		}
	}()
	if nCode == hcAction && activeEvents != nil {
		ms := winutil.ReadStructFromPointer[msllhookstruct](lParam)
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
