// rawinput.go 给 recorder 加 Raw Input 通路抓**相对鼠标位移**（camera 转向）。
//
// 为什么：游戏走 Raw Input API（GetRawInputData / DirectInput）读 mouse 设备
// delta，不走 WM_MOUSEMOVE。LL hook (WH_MOUSE_LL) 只看消息层的鼠标，camera 转
// 向期间根本看不到事件（游戏多半还把光标锁中心，看到的也是 noise）。
//
// 解法：注册 RegisterRawInputDevices 接 raw HID Mouse → 走 WM_INPUT → 在我们自己
// 的消息-only window 的 WndProc 里读 RAWINPUT.mouse.LastX/LastY。
//
// 加 RIDEV_INPUTSINK flag 让游戏在前台时本进程也能后台收 raw mouse。

package recording

import (
	"fmt"
	"syscall"
	"unsafe"
)

const (
	wmInput   = 0x00FF
	wmDestroy = 0x0002

	// HWND_MESSAGE = (HWND)-3 在 amd64 是 0xFFFFFFFFFFFFFFFD
	hwndMessage = ^uintptr(2)

	ridevInputSink = 0x00000100
	ridevRemove    = 0x00000001

	ridInput = 0x10000003

	rimTypeMouse = 0

	mouseMoveRelative uint16 = 0
)

// rawinputdevice 对应 Win32 RAWINPUTDEVICE。
type rawinputdevice struct {
	UsagePage  uint16
	Usage      uint16
	Flags      uint32
	HwndTarget syscall.Handle
}

// rawinputheader 对应 Win32 RAWINPUTHEADER。
type rawinputheader struct {
	Type   uint32
	Size   uint32
	Device syscall.Handle
	WParam uintptr
}

// rawmouse 对应 Win32 RAWMOUSE。注意 UlButtons 是 union 头一个字段，
// 我们只用 LastX/LastY，其它不动也行。
type rawmouse struct {
	UsFlags            uint16
	_                  uint16
	UlButtons          uint32
	UlRawButtons       uint32
	LLastX             int32
	LLastY             int32
	UlExtraInformation uint32
}

// wndclassex 对应 Win32 WNDCLASSEXW。
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

// 多 proc 句柄（同 user32 / kernel32）
var (
	procRegisterClassExW        = user32.NewProc("RegisterClassExW")
	procCreateWindowExW         = user32.NewProc("CreateWindowExW")
	procDestroyWindow           = user32.NewProc("DestroyWindow")
	procDefWindowProcW          = user32.NewProc("DefWindowProcW")
	procDispatchMessageW        = user32.NewProc("DispatchMessageW")
	procTranslateMessage        = user32.NewProc("TranslateMessage")
	procRegisterRawInputDevices = user32.NewProc("RegisterRawInputDevices")
	procGetRawInputData         = user32.NewProc("GetRawInputData")
	procGetModuleHandleW        = kernel32.NewProc("GetModuleHandleW")
	procGetTickCount            = kernel32.NewProc("GetTickCount")
)

// 全局 window class atom + 缓存的 callback。WndProc 是 C ABI 不能闭包，走全局。
// 跟 keyboardProc/mouseProc 一样 —— 同一进程同一时刻只有一份 recorder。
var (
	rawClassRegistered bool
	rawWndProcCallback uintptr // 启动时 syscall.NewCallback(rawInputWndProc) 缓存
)

// rawInputWndProc WndProc 处理 WM_INPUT。从 lParam(HRAWINPUT) 读 RAWINPUT，
// 提取 mouse delta，push 到 activeEvents（沿用 LL hook 同一 channel）。
//
// 必须 fast return —— DefWindowProcW 必走完不然 OS 抱怨。
func rawInputWndProc(hwnd, msg, wParam, lParam uintptr) uintptr {
	if msg == wmInput && activeEvents != nil {
		var size uint32
		headerSize := uint32(unsafe.Sizeof(rawinputheader{}))
		// 先查需要多少 byte
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
				m := (*rawmouse)(unsafe.Pointer(&buf[headerSize]))
				// 只关心相对位移；absolute mode（数位板/触控）忽略
				if (m.UsFlags&1) == mouseMoveRelative && (m.LLastX != 0 || m.LLastY != 0) {
					tick, _, _ := procGetTickCount.Call()
					select {
					case activeEvents <- HookEvent{
						TimestampMs: uint32(tick),
						IsRawDelta:  true,
						RawDx:       int(m.LLastX),
						RawDy:       int(m.LLastY),
					}:
					default:
					}
				}
			}
		}
	}
	next, _, _ := procDefWindowProcW.Call(hwnd, msg, wParam, lParam)
	return next
}

// registerRawInputClass 注册 window class（进程内一次）。
// 必须在创建 window 之前调一次。
func registerRawInputClass() error {
	if rawClassRegistered {
		return nil
	}
	if rawWndProcCallback == 0 {
		rawWndProcCallback = syscall.NewCallback(rawInputWndProc)
	}
	className, _ := syscall.UTF16PtrFromString("YottaRawInputWnd")
	hInst, _, _ := procGetModuleHandleW.Call(0)

	wc := wndclassex{
		Size:      uint32(unsafe.Sizeof(wndclassex{})),
		WndProc:   rawWndProcCallback,
		Instance:  syscall.Handle(hInst),
		ClassName: className,
	}
	atom, _, callErr := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	if atom == 0 {
		return fmt.Errorf("RegisterClassExW: %v", callErr)
	}
	rawClassRegistered = true
	return nil
}

// createRawInputWindow 在当前线程创建消息-only 窗口（HWND_MESSAGE 父）。
// 配合 caller 已经 LockOSThread，让 WndProc 跟 message loop 同线程。
func createRawInputWindow() (syscall.Handle, error) {
	if err := registerRawInputClass(); err != nil {
		return 0, err
	}
	className, _ := syscall.UTF16PtrFromString("YottaRawInputWnd")
	hInst, _, _ := procGetModuleHandleW.Call(0)

	hwnd, _, callErr := procCreateWindowExW.Call(
		0, // dwExStyle
		uintptr(unsafe.Pointer(className)),
		0, // lpWindowName
		0, // dwStyle
		0, 0, 0, 0, // x y w h
		hwndMessage, // parent = HWND_MESSAGE
		0,           // hMenu
		hInst,       // hInstance
		0,           // lpParam
	)
	if hwnd == 0 {
		return 0, fmt.Errorf("CreateWindowExW: %v", callErr)
	}
	return syscall.Handle(hwnd), nil
}

// destroyRawInputWindow 删窗口。caller 应该在结束 message loop 后调。
func destroyRawInputWindow(hwnd syscall.Handle) {
	if hwnd != 0 {
		procDestroyWindow.Call(uintptr(hwnd))
	}
}

// registerRawMouse 注册 raw mouse input 到 hwnd。RIDEV_INPUTSINK 让窗口在
// 非前台也收事件（录制时用户切到游戏窗口我们的窗口失去前台是常态）。
func registerRawMouse(hwnd syscall.Handle) error {
	rid := rawinputdevice{
		UsagePage:  0x01, // Generic Desktop
		Usage:      0x02, // Mouse
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

// unregisterRawMouse 反注册。Stop 录时调。
func unregisterRawMouse() {
	rid := rawinputdevice{
		UsagePage: 0x01,
		Usage:     0x02,
		Flags:     ridevRemove,
	}
	procRegisterRawInputDevices.Call(
		uintptr(unsafe.Pointer(&rid)),
		1,
		unsafe.Sizeof(rid),
	)
}
