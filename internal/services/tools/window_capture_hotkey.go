package tools

import (
	"errors"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"

	"github.com/lxn/win"

	"yotta/pkg/winutil"
)

// captureSession 一次 capture 调用的状态. 同时只能有一个.
type captureSession struct {
	id         string
	hotkeyMods uint32
	hotkeyVK   uint32
	threadID   uint32        // 给 PostThreadMessage 用; worker 启动后写一次, 之后只读
	done     chan struct{} // worker 退出信号
	cancel   chan struct{} // 主动 cancel 信号 (worker 收到后退出, 不 emit)
	emit     func(name string, data any)
	fired    atomic.Bool // 防重 emit (双触发 / cancel+触发竞态)
}

// activeCapture 全局当前活跃 capture session. nil = 无.
var (
	activeCapture *captureSession
	captureMu     sync.Mutex
)

// startWindowTargetCapture 注册 hotkey + 起 goroutine 等待按键.
// 返 captureID 给前端 cancel 用. hotkeyMods/hotkeyVK 是 Win32 MOD_*/VK_* 码 (例 vk 0x78 = F9).
// emit 在 hotkey 触发时调 (name="windowtarget:captured", data=map).
// 同时只能一个 session — 之前的没清就报错.
func startWindowTargetCapture(hotkeyMods, hotkeyVK uint32, emit func(name string, data any)) (string, error) {
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

// cancelWindowTargetCapture 主动 cancel. id 必须匹配当前 active session.
// idempotent: id 不匹配或没 active 都返 nil.
func cancelWindowTargetCapture(id string) error {
	captureMu.Lock()
	sess := activeCapture
	captureMu.Unlock()
	if sess == nil || sess.id != id {
		return nil
	}
	// 防止重复 close panic (双 cancel 路径)
	select {
	case <-sess.cancel:
		// already canceled
	default:
		close(sess.cancel)
	}
	postQuitToThread(sess.threadID) // unblock GetMessage
	<-sess.done                     // wait for worker exit
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

// workerThread 锁 OS 线程, RegisterHotKey, 起 GetMessage loop.
// hotkey 触发 → 查 foreground window → emit → 退出.
// PostThreadMessage(WM_QUIT) → GetMessage 返 0 → 退出 (cancel 路径).
func (s *captureSession) workerThread(started chan<- error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer close(s.done)
	defer clearActiveSession(s)

	s.threadID = win.GetCurrentThreadId()

	// RegisterHotKey id 用一个固定值 (一个 session 一个 hotkey, 不冲突)
	const hotkeyID = 0x9001
	r, _, err := procRegisterHotKey.Call(0, uintptr(hotkeyID), uintptr(s.hotkeyMods), uintptr(s.hotkeyVK))
	if r == 0 {
		started <- fmt.Errorf("RegisterHotKey mods=0x%x vk=0x%x: %v", s.hotkeyMods, s.hotkeyVK, err)
		return
	}
	defer procUnregisterHotKey.Call(0, uintptr(hotkeyID))

	started <- nil

	// Message loop. WM_HOTKEY = 0x0312. WM_QUIT = 0x0012.
	const wmHotkey = 0x0312
	var m msg
	for {
		ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(ret) <= 0 {
			// 0 = WM_QUIT, -1 = error. 都退.
			return
		}
		if m.message == wmHotkey {
			// 收到 hotkey, 但先检查是否已 cancel (竞态: cancel 跟 hotkey 同时触发)
			select {
			case <-s.cancel:
				return
			default:
			}
			if !s.fired.CompareAndSwap(false, true) {
				return // 二次触发, 忽略
			}
			s.handleHotkey()
			return
		}
	}
}

// handleHotkey 查 foreground window + emit.
func (s *captureSession) handleHotkey() {
	hwnd := winutil.ForegroundWindow()
	if hwnd == 0 {
		if s.emit != nil {
			s.emit("windowtarget:captured", map[string]any{"error": "无前台窗口"})
		}
		return
	}
	wh, err := winutil.WindowMetadata(hwnd)
	if err != nil {
		if s.emit != nil {
			s.emit("windowtarget:captured", map[string]any{"error": err.Error()})
		}
		return
	}
	if s.emit != nil {
		s.emit("windowtarget:captured", map[string]any{
			"title":       wh.Title,
			"class":       wh.Class,
			"processName": wh.ProcessName,
			"clientW":     wh.ClientW,
			"clientH":     wh.ClientH,
		})
	}
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

// --- Win32 procs ---
// 注意: 不能复用 tools 包里 mouse.go 的 user32 var (那个只挂了 cursor 相关 proc).
// 用独立 LazyDLL — NewLazyDLL 同名只 load 一次, 不会重复.

var (
	user32CaptureHotkey    = syscall.NewLazyDLL("user32.dll")
	procRegisterHotKey     = user32CaptureHotkey.NewProc("RegisterHotKey")
	procUnregisterHotKey   = user32CaptureHotkey.NewProc("UnregisterHotKey")
	procGetMessageW        = user32CaptureHotkey.NewProc("GetMessageW")
	procPostThreadMessageW = user32CaptureHotkey.NewProc("PostThreadMessageW")
)

func postQuitToThread(tid uint32) {
	const wmQuit = 0x0012
	procPostThreadMessageW.Call(uintptr(tid), uintptr(wmQuit), 0, 0)
}
