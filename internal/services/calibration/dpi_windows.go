package calibration

import (
	"errors"
	"fmt"
	"sync"
	"syscall"
	"unsafe"
)

const (
	wmInput   = 0x00FF
	wmDestroy = 0x0002

	hwndMessage = ^uintptr(2) // (HWND)-3

	ridevInputSink    = 0x00000100
	ridInput          = 0x10000003
	rimTypeMouse      = 0
	mouseMoveRelative = uint16(0)
	wmQuit            = 0x0012
)

type rawinputdevice struct {
	UsagePage  uint16
	Usage      uint16
	Flags      uint32
	HwndTarget syscall.Handle
}

type rawinputheader struct {
	Type   uint32
	Size   uint32
	Device syscall.Handle
	WParam uintptr
}

type rawmouse struct {
	UsFlags            uint16
	_                  uint16
	UlButtons          uint32
	UlRawButtons       uint32
	LLastX             int32
	LLastY             int32
	UlExtraInformation uint32
}

type wndclassex struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   syscall.Handle
	Icon       syscall.Handle
	Cursor     syscall.Handle
	Background syscall.Handle
	MenuName   *uint16
	ClassName  *uint16
	IconSm     syscall.Handle
}

type msg struct {
	HWND    syscall.Handle
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	PtX     int32
	PtY     int32
}

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procRegisterClassExW        = user32.NewProc("RegisterClassExW")
	procCreateWindowExW         = user32.NewProc("CreateWindowExW")
	procDestroyWindow           = user32.NewProc("DestroyWindow")
	procDefWindowProcW          = user32.NewProc("DefWindowProcW")
	procDispatchMessageW        = user32.NewProc("DispatchMessageW")
	procTranslateMessage        = user32.NewProc("TranslateMessage")
	procGetMessageW             = user32.NewProc("GetMessageW")
	procPostMessageW            = user32.NewProc("PostMessageW")
	procRegisterRawInputDevices = user32.NewProc("RegisterRawInputDevices")
	procGetRawInputData         = user32.NewProc("GetRawInputData")
	procGetModuleHandleW        = kernel32.NewProc("GetModuleHandleW")
)

// 全局：WndProc 必须 C ABI；run() 同时刻只一个，全局够。
var (
	wndProcCallback uintptr
	classRegistered bool
	classRegMu      sync.Mutex

	// done channel + window handle 给 Stop 用
	stopMu sync.Mutex
	stopFn func() // 关闭当前会话；nil = 没在跑
)

// Start 启动 raw input 监听 goroutine。同时刻只能一个。
// 返回错误 = 启动失败（class 注册失败 / window 创建失败 / RegisterRawInputDevices 失败）。
func Start() error {
	stopMu.Lock()
	defer stopMu.Unlock()
	if stopFn != nil {
		return errors.New("calibration already active")
	}

	ready := make(chan error, 1)
	done := make(chan struct{})
	var hwnd syscall.Handle

	go func() {
		// WndProc + message loop 必须同一 OS 线程
		// （RegisterClassExW + CreateWindowExW + GetMessageW 都依赖 thread 状态）
		runtimeLockOSThread()
		defer runtimeUnlockOSThread()

		h, err := makeWindow()
		if err != nil {
			ready <- err
			close(done)
			return
		}
		hwnd = h
		if err := registerRawMouse(hwnd); err != nil {
			procDestroyWindow.Call(uintptr(hwnd))
			ready <- err
			close(done)
			return
		}
		currentState.setActive(true)
		ready <- nil
		pumpMessages(hwnd)
		currentState.setActive(false)
		procDestroyWindow.Call(uintptr(hwnd))
		close(done)
	}()

	if err := <-ready; err != nil {
		return err
	}
	stopFn = func() {
		// PostMessage WM_QUIT → GetMessage 返 0 → pumpMessages 退出
		procPostMessageW.Call(uintptr(hwnd), wmQuit, 0, 0)
		<-done
	}
	return nil
}

// Stop 关闭监听并返回累积值。idempotent — 多次调安全。
func Stop() (State, error) {
	stopMu.Lock()
	fn := stopFn
	stopFn = nil
	stopMu.Unlock()
	if fn == nil {
		return Get(), nil
	}
	fn()
	return Get(), nil
}

// ---- Win32 内部 ----

func wndProc(hwnd, m, wParam, lParam uintptr) uintptr {
	if m == wmInput {
		var size uint32
		headerSize := uint32(unsafe.Sizeof(rawinputheader{}))
		procGetRawInputData.Call(lParam, ridInput, 0,
			uintptr(unsafe.Pointer(&size)), uintptr(headerSize))
		if size > 0 && size < 1024 {
			buf := make([]byte, size)
			procGetRawInputData.Call(lParam, ridInput,
				uintptr(unsafe.Pointer(&buf[0])),
				uintptr(unsafe.Pointer(&size)),
				uintptr(headerSize))
			hdr := (*rawinputheader)(unsafe.Pointer(&buf[0]))
			if hdr.Type == rimTypeMouse && len(buf) >= int(headerSize)+int(unsafe.Sizeof(rawmouse{})) {
				rm := (*rawmouse)(unsafe.Pointer(&buf[headerSize]))
				if (rm.UsFlags & 1) == mouseMoveRelative {
					if rm.LLastX != 0 {
						currentState.addRelative(int64(absInt32(rm.LLastX)), 0)
					}
					if rm.LLastY != 0 {
						currentState.addRelative(0, int64(absInt32(rm.LLastY)))
					}
				}
			}
		}
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, m, wParam, lParam)
	return r
}

func absInt32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}

func ensureClass() error {
	classRegMu.Lock()
	defer classRegMu.Unlock()
	if classRegistered {
		return nil
	}
	if wndProcCallback == 0 {
		wndProcCallback = syscall.NewCallback(wndProc)
	}
	className, _ := syscall.UTF16PtrFromString("YottaCalibrationWnd")
	hInst, _, _ := procGetModuleHandleW.Call(0)
	wc := wndclassex{
		Size:      uint32(unsafe.Sizeof(wndclassex{})),
		WndProc:   wndProcCallback,
		Instance:  syscall.Handle(hInst),
		ClassName: className,
	}
	atom, _, callErr := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	if atom == 0 {
		return fmt.Errorf("RegisterClassExW: %v", callErr)
	}
	classRegistered = true
	return nil
}

func makeWindow() (syscall.Handle, error) {
	if err := ensureClass(); err != nil {
		return 0, err
	}
	className, _ := syscall.UTF16PtrFromString("YottaCalibrationWnd")
	hInst, _, _ := procGetModuleHandleW.Call(0)
	h, _, callErr := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		0,
		0,
		0, 0, 0, 0,
		hwndMessage,
		0,
		hInst,
		0,
	)
	if h == 0 {
		return 0, fmt.Errorf("CreateWindowExW: %v", callErr)
	}
	return syscall.Handle(h), nil
}

func registerRawMouse(hwnd syscall.Handle) error {
	rid := rawinputdevice{
		UsagePage:  0x01,
		Usage:      0x02,
		Flags:      ridevInputSink,
		HwndTarget: hwnd,
	}
	r, _, callErr := procRegisterRawInputDevices.Call(
		uintptr(unsafe.Pointer(&rid)),
		1,
		unsafe.Sizeof(rid),
	)
	if r == 0 {
		return fmt.Errorf("RegisterRawInputDevices: %v", callErr)
	}
	return nil
}

func pumpMessages(_ syscall.Handle) {
	var m msg
	for {
		ret, _, _ := procGetMessageW.Call(
			uintptr(unsafe.Pointer(&m)),
			0, // 任意 hwnd
			0, 0,
		)
		// 0 = WM_QUIT，-1 = 错误
		if int32(ret) <= 0 {
			return
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
}
