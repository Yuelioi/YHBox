package tools

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"

	"github.com/lxn/win"

	"github.com/yottaapp/yotta/internal/apperr"
	"github.com/yottaapp/yotta/pkg/winutil"
)

func win32WindowCaptureSupported() error { return nil }

// captureSession 一次 capture 调用的状态. 同时只能有一个.
type captureSession struct {
	id         string
	hotkeyMods uint32
	hotkeyVK   uint32
	threadID   uint32        // 给 PostThreadMessage 用; worker 启动后写一次, 之后只读
	done       chan struct{} // worker 退出信号
	cancel     chan struct{} // 主动 cancel 信号 (worker 收到后退出, 不 emit)
	window     chan uintptr
	emit       func(name string, data any)
	fired      atomic.Bool // 防重 emit (双触发 / cancel+触发竞态)
	cancelOnce sync.Once
}

// activeCapture 全局当前活跃 capture session. nil = 无.
var (
	activeCapture *captureSession
	captureMu     sync.Mutex
	hookSession   atomic.Pointer[captureSession]
	hookProc      uintptr
	hookProcOnce  sync.Once
)

// startWin32WindowTargetCapture 安装临时全局键盘钩子并等待按键。
// 返 captureID 给前端 cancel 用. hotkeyMods/hotkeyVK 是 Win32 MOD_*/VK_* 码 (例 vk 0x78 = F9).
// emit 在 hotkey 触发时调 (name="win32windowtarget:captured", data=map).
// 同时只能一个 session — 之前的没清就报错.
func startWin32WindowTargetCapture(hotkeyMods, hotkeyVK uint32, emit func(name string, data any)) (string, error) {
	captureMu.Lock()
	if activeCapture != nil {
		captureMu.Unlock()
		return "", errors.New("已有 capture session 在等待, 先 cancel")
	}
	sess := &captureSession{
		id:         "wt-capture-" + randID(),
		hotkeyMods: hotkeyMods,
		hotkeyVK:   hotkeyVK,
		done:       make(chan struct{}),
		cancel:     make(chan struct{}),
		window:     make(chan uintptr, 1),
		emit:       emit,
	}
	activeCapture = sess
	captureMu.Unlock()

	started := make(chan error, 1)
	go sess.workerThread(started)
	if err := <-started; err != nil {
		// worker 启动失败, 清掉 activeCapture
		captureMu.Lock()
		if activeCapture == sess {
			activeCapture = nil
		}
		captureMu.Unlock()
		return "", err
	}
	return sess.id, nil
}

// cancelWin32WindowTargetCapture 主动 cancel. id 必须匹配当前 active session.
// idempotent: id 不匹配或没 active 都返 nil.
func cancelWin32WindowTargetCapture(id string) error {
	captureMu.Lock()
	sess := activeCapture
	captureMu.Unlock()
	if sess == nil || sess.id != id {
		return nil
	}
	return cancelCaptureSession(context.Background(), sess)
}

func shutdownWin32WindowTargetCapture(ctx context.Context) error {
	captureMu.Lock()
	sess := activeCapture
	captureMu.Unlock()
	if sess == nil {
		return nil
	}
	return cancelCaptureSession(ctx, sess)
}

func cancelCaptureSession(ctx context.Context, sess *captureSession) error {
	sess.cancelOnce.Do(func() { close(sess.cancel) })
	postQuitToThread(sess.threadID) // unblock GetMessage
	select {
	case <-sess.done:
	case <-ctx.Done():
		return ctx.Err()
	}
	captureMu.Lock()
	if activeCapture == sess {
		activeCapture = nil
	}
	captureMu.Unlock()
	return nil
}

// clearActiveSession worker 退出前自调, 保证 activeCapture 不留指针挡下次 Start.
// cancel 路径外面也会清, 但 hotkey-fired 路径只能 worker 自己清.
func clearActiveSession(sess *captureSession) {
	captureMu.Lock()
	if activeCapture == sess {
		activeCapture = nil
	}
	captureMu.Unlock()
}

// workerThread 锁 OS 线程并持有临时低级键盘钩子，避免目标程序占用相同快捷键时注册失败。
// PostThreadMessage(WM_QUIT) → GetMessage 返 0 → 退出 (cancel 路径).
func (s *captureSession) workerThread(started chan<- error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer close(s.done)
	defer clearActiveSession(s)

	s.threadID = win.GetCurrentThreadId()

	hookProcOnce.Do(func() { hookProc = syscall.NewCallback(captureKeyboardProc) })
	hookSession.Store(s)
	hook, _, err := procSetWindowsHookExW.Call(whKeyboardLL, hookProc, 0, 0)
	if hook == 0 {
		hookSession.CompareAndSwap(s, nil)
		started <- fmt.Errorf("SetWindowsHookExW(WH_KEYBOARD_LL): %v", err)
		return
	}
	releaseHook := func() {
		if hook == 0 {
			return
		}
		hookSession.CompareAndSwap(s, nil)
		procUnhookWindowsHookEx.Call(hook)
		hook = 0
	}
	defer releaseHook()

	var m msg
	procPeekMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0, 0)
	started <- nil

	for {
		ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(ret) <= 0 {
			break
		}
	}
	releaseHook()
	select {
	case <-s.cancel:
		return
	default:
	}
	select {
	case hwnd := <-s.window:
		s.handleWindow(hwnd)
	default:
	}
}

func (s *captureSession) handleWindow(hwnd uintptr) {
	if hwnd == 0 {
		if s.emit != nil {
			s.emit("win32windowtarget:captured", map[string]any{"problem": apperr.Project(apperr.New("automation.window_capture.no_foreground", nil))})
		}
		return
	}
	wh, err := winutil.WindowMetadata(hwnd)
	if err != nil {
		if s.emit != nil {
			s.emit("win32windowtarget:captured", map[string]any{"problem": apperr.Project(fmt.Errorf("%w: %v", apperr.New("automation.window_capture.metadata_failed", nil), err))})
		}
		return
	}
	executable, err := winutil.WindowExecutable(hwnd)
	if err != nil {
		if s.emit != nil {
			s.emit("win32windowtarget:captured", map[string]any{"problem": apperr.Project(fmt.Errorf("%w: %v", apperr.New("automation.window_capture.executable_failed", nil), err))})
		}
		return
	}
	if s.emit != nil {
		s.emit("win32windowtarget:captured", map[string]any{
			"title":      wh.Title,
			"class":      wh.Class,
			"executable": executable,
		})
	}
}

func captureKeyboardProc(nCode, wParam, lParam uintptr) (result uintptr) {
	defer func() {
		if recover() != nil {
			result, _, _ = procCallNextHookEx.Call(0, nCode, wParam, lParam)
		}
	}()
	if nCode == hcAction {
		sess := hookSession.Load()
		if sess != nil {
			key := winutil.ReadStructFromPointer[captureKeyboardEvent](lParam)
			if key.VKCode == sess.hotkeyVK {
				if wParam == wmKeyDown || wParam == wmSysKeyDown {
					if sess.fired.Load() || captureModifiersPressed(sess.hotkeyMods) {
						if sess.fired.CompareAndSwap(false, true) {
							select {
							case sess.window <- winutil.ForegroundWindow():
							default:
							}
						}
						return 1
					}
				}
				if (wParam == wmKeyUp || wParam == wmSysKeyUp) && sess.fired.Load() {
					postQuitToThread(sess.threadID)
					return 1
				}
			}
		}
	}
	result, _, _ = procCallNextHookEx.Call(0, nCode, wParam, lParam)
	return result
}

func captureModifiersPressed(mods uint32) bool {
	return keyPressed(vkControl) == (mods&modControl != 0) &&
		keyPressed(vkShift) == (mods&modShift != 0) &&
		keyPressed(vkMenu) == (mods&modAlt != 0)
}

func keyPressed(vk uintptr) bool {
	state, _, _ := procGetAsyncKeyState.Call(vk)
	return state&0x8000 != 0
}

type captureKeyboardEvent struct {
	VKCode    uint32
	ScanCode  uint32
	Flags     uint32
	Time      uint32
	ExtraInfo uintptr
}

// msg Win32 MSG struct.
type msg struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      struct{ X, Y int32 }
}

// randID 短 ID. 用 atomic counter + time, 不需 crypto-grade — session 同时只有一个,
// 但前端 stale cancel 用旧 ID 不该误中新 session, 所以序号要单调递增.
func randID() string {
	n := atomic.AddUint64(&idCounter, 1)
	const charset = "0123456789abcdef"
	b := make([]byte, 12)
	for i := 11; i >= 0; i-- {
		b[i] = charset[n&0xF]
		n >>= 4
	}
	return string(b)
}

var idCounter uint64

const (
	whKeyboardLL = 13
	hcAction     = 0
	wmKeyDown    = 0x0100
	wmKeyUp      = 0x0101
	wmSysKeyDown = 0x0104
	wmSysKeyUp   = 0x0105
	modAlt       = 0x0001
	modControl   = 0x0002
	modShift     = 0x0004
	vkShift      = 0x10
	vkControl    = 0x11
	vkMenu       = 0x12
)

// --- Win32 procs ---
// 捕获线程使用独立的 LazyDLL proc 集，避免和鼠标位置 adapter 共享可变初始化状态。
// 用独立 LazyDLL — NewLazyDLL 同名只 load 一次, 不会重复.

var (
	user32CaptureHotkey     = syscall.NewLazyDLL("user32.dll")
	procSetWindowsHookExW   = user32CaptureHotkey.NewProc("SetWindowsHookExW")
	procUnhookWindowsHookEx = user32CaptureHotkey.NewProc("UnhookWindowsHookEx")
	procCallNextHookEx      = user32CaptureHotkey.NewProc("CallNextHookEx")
	procGetAsyncKeyState    = user32CaptureHotkey.NewProc("GetAsyncKeyState")
	procGetMessageW         = user32CaptureHotkey.NewProc("GetMessageW")
	procPeekMessageW        = user32CaptureHotkey.NewProc("PeekMessageW")
	procPostThreadMessageW  = user32CaptureHotkey.NewProc("PostThreadMessageW")
)

func postQuitToThread(tid uint32) {
	const wmQuit = 0x0012
	procPostThreadMessageW.Call(uintptr(tid), uintptr(wmQuit), 0, 0)
}
