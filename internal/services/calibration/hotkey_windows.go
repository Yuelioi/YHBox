package calibration

// RegisterHotKey cannot reliably observe keys reserved by a foreground game,
// so calibration owns an isolated single-key low-level hook. A session starts
// and stops the hook once; callbacks run asynchronously, auto-repeat fires only
// on the up-to-down transition, and matching key-down/key-up events are blocked.

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"

	"github.com/yottaapp/yotta/pkg/winutil"
)

const (
	whKeyboardLL = 13
	hcAction     = 0
	wmKeyDown    = 0x0100
	wmSysKeyDown = 0x0104
)

var (
	procSetWindowsHookExW   = user32.NewProc("SetWindowsHookExW")
	procUnhookWindowsHookEx = user32.NewProc("UnhookWindowsHookEx")
	procCallNextHookEx      = user32.NewProc("CallNextHookEx")
	procPostThreadMessageW  = user32.NewProc("PostThreadMessageW")
	procGetCurrentThreadId  = kernel32.NewProc("GetCurrentThreadId")
)

// kbdllhookstruct 对应 Win32 KBDLLHOOKSTRUCT (只用 VkCode).
type kbdllhookstruct struct {
	VkCode      uint32
	ScanCode    uint32
	Flags       uint32
	Time        uint32
	DwExtraInfo uintptr
}

// 全局 hook 状态 — C callback 走 stdcall ABI 不能捕获 Go 闭包, 只能走包级.
// 单 VK / 实例: 同时刻只装一份 (HotkeyHook.Start 串行, installHotkeyHook 再 guard).
var (
	hkHookHandle   uintptr // SetWindowsHookExW 句柄; 0 = 未装
	hkVK           atomic.Uint32
	hkCallback     atomic.Pointer[func()]
	hkKeyHeld      atomic.Bool // autorepeat 去抖: down 已按住时不重复 fire
	hkProcCallback uintptr     // syscall.NewCallback(hotkeyKeyboardProc), 建一次
	hkProcOnce     sync.Once
)

// HotkeyHook 是一个单键 LL 键盘热键. Start 装钩 (返 error), Stop 卸钩 (幂等).
type HotkeyHook struct {
	vk uint32
	cb func()

	mu      sync.Mutex
	tid     uint32
	done    chan struct{}
	started bool
}

// NewHotkeyHook 建一个监听 vk 的热键; 命中时调 cb. 还没装钩 — 要 Start().
func NewHotkeyHook(vk uint32, cb func()) *HotkeyHook {
	return &HotkeyHook{vk: vk, cb: cb}
}

// Start 在自己的 OS 线程上装钩 + 跑 message loop. 装钩失败返 error (杀软拦截等, 不静默).
func (h *HotkeyHook) Start() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.started {
		return errors.New("hotkey hook 已启动")
	}

	ready := make(chan error, 1)
	done := make(chan struct{})
	go func() {
		// SetWindowsHookExW 与 GetMessage 必须同一 OS 线程.
		runtimeLockOSThread()
		defer runtimeUnlockOSThread()
		defer close(done)

		h.tid = getCurrentThreadID()
		if err := installHotkeyHook(h.vk, h.cb); err != nil {
			ready <- err
			return
		}
		ready <- nil
		runHotkeyMessageLoop()
		// message loop 退出 (收到 WM_QUIT) 后由本 worker 自己卸钩 —— 绝不跨 goroutine
		// 调 UnhookWindowsHookEx (跨线程返 FALSE 且静默失败).
		uninstallHotkeyHook()
	}()

	if err := <-ready; err != nil {
		return err
	}
	h.started = true
	h.done = done
	return nil
}

// Stop 让 worker 退出并等它卸完钩. 幂等 — 多次调安全.
func (h *HotkeyHook) Stop() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.started {
		return
	}
	postThreadQuit(h.tid)
	<-h.done
	h.started = false
}

func installHotkeyHook(vk uint32, cb func()) error {
	if hkHookHandle != 0 {
		return errors.New("已有 hotkey hook 在跑")
	}
	hkProcOnce.Do(func() {
		hkProcCallback = syscall.NewCallback(hotkeyKeyboardProc)
	})
	hkVK.Store(vk)
	hkKeyHeld.Store(false)
	if cb == nil {
		hkCallback.Store(nil)
	} else {
		hkCallback.Store(&cb)
	}
	// SetWindowsHookExW(WH_KEYBOARD_LL, lpfn, hMod=0, dwThreadId=0): LL hook 标准用法.
	hk, _, callErr := procSetWindowsHookExW.Call(uintptr(whKeyboardLL), hkProcCallback, 0, 0)
	if hk == 0 {
		hkVK.Store(0)
		hkCallback.Store(nil)
		return fmt.Errorf("SetWindowsHookExW(WH_KEYBOARD_LL): %v", callErr)
	}
	hkHookHandle = hk
	return nil
}

func uninstallHotkeyHook() {
	if hkHookHandle != 0 {
		procUnhookWindowsHookEx.Call(hkHookHandle)
		hkHookHandle = 0
	}
	hkVK.Store(0)
	hkCallback.Store(nil)
	hkKeyHeld.Store(false)
}

// hotkeyKeyboardProc 必须 fast return — 耗时操作会让 OS ~300ms 后自动摘钩.
// 命中键: down 跳变 fire 一次 (autorepeat 不重复), up 复位; down/up 都 return 1 拦截.
// 非命中键: CallNextHookEx 透传, 不影响其它应用收键.
func hotkeyKeyboardProc(nCode, wParam, lParam uintptr) uintptr {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "[calibration hotkey recover] %v\n", r)
		}
	}()
	if nCode == hcAction {
		vk := hkVK.Load()
		if vk != 0 {
			kbd := winutil.ReadStructFromPointer[kbdllhookstruct](lParam)
			if kbd.VkCode == vk {
				isDown := wParam == wmKeyDown || wParam == wmSysKeyDown
				if isDown {
					if !hkKeyHeld.Swap(true) {
						if cbp := hkCallback.Load(); cbp != nil {
							go (*cbp)()
						}
					}
				} else {
					hkKeyHeld.Store(false)
				}
				return 1
			}
		}
	}
	next, _, _ := procCallNextHookEx.Call(0, nCode, wParam, lParam)
	return next
}

// runHotkeyMessageLoop 拉本线程消息直到 WM_QUIT.
func runHotkeyMessageLoop() {
	var m msg
	for {
		ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(ret) <= 0 { // 0 = WM_QUIT, -1 = error
			return
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
}

func postThreadQuit(tid uint32) {
	procPostThreadMessageW.Call(uintptr(tid), wmQuit, 0, 0)
}

func getCurrentThreadID() uint32 {
	r, _, _ := procGetCurrentThreadId.Call()
	return uint32(r)
}
