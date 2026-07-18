package winutil

import (
	"fmt"
	"unsafe"

	"github.com/lxn/win"
)

var (
	procIsWindow           = user32.NewProc("IsWindow")
	procSetWindowPos       = user32.NewProc("SetWindowPos")
	procGetWindowLongPtr   = user32.NewProc("GetWindowLongPtrW")
	procSetWindowLongPtr   = user32.NewProc("SetWindowLongPtrW")
	procMonitorFromWindow  = user32.NewProc("MonitorFromWindow")
	procGetMonitorInfo     = user32.NewProc("GetMonitorInfoW")
	procGetWindowPlacement = user32.NewProc("GetWindowPlacement")
	procSetWindowPlacement = user32.NewProc("SetWindowPlacement")
	procPostMessage        = user32.NewProc("PostMessageW")
	procGetWindowRect      = user32.NewProc("GetWindowRect")
	procIsZoomed           = user32.NewProc("IsZoomed")
)

// gwlStyleIdx is stored as a var (not const) so uintptr(gwlStyleIdx) compiles:
// Go refuses uintptr(-16) as a constant expression.
var gwlStyleIdx = int32(-16)

const (
	swMaximize      = 3
	swMinimize      = 6
	wsCaption       = 0x00C00000
	wsThickFrame    = 0x00040000
	wsOverlapped    = 0x00CF0000 // WS_OVERLAPPEDWINDOW
	swpFrameChanged = 0x0020
	swpNoZOrder     = 0x0004
	swpShowWindow   = 0x0040
	swpNoMove       = 0x0002
	swpNoSize       = 0x0001
	wmClose         = 0x0010
	monitorNearest  = 0x00000002
)

func IsWindow(hwnd uintptr) bool {
	if hwnd == 0 {
		return false
	}
	r, _, _ := procIsWindow.Call(hwnd)
	return r != 0
}

func Maximize(hwnd uintptr) error { return showErr(hwnd, swMaximize) }
func Minimize(hwnd uintptr) error { return showErr(hwnd, swMinimize) }
func Restore(hwnd uintptr) error  { return showErr(hwnd, swRestore) }

func showErr(hwnd uintptr, cmd int) error {
	if hwnd == 0 {
		return fmt.Errorf("hwnd 0")
	}
	procShowWindow.Call(hwnd, uintptr(cmd)) // ShowWindow 返回值是「之前是否可见」, 非成功标志
	return nil
}

func MoveResize(hwnd uintptr, x, y, w, h int) error {
	if hwnd == 0 {
		return fmt.Errorf("hwnd 0")
	}
	r, _, err := procSetWindowPos.Call(hwnd, 0, uintptr(x), uintptr(y), uintptr(w), uintptr(h), swpNoZOrder)
	if r == 0 {
		return fmt.Errorf("SetWindowPos: %v", err)
	}
	return nil
}

func CloseWindow(hwnd uintptr) error {
	if hwnd == 0 {
		return fmt.Errorf("hwnd 0")
	}
	result, _, callErr := procPostMessage.Call(hwnd, wmClose, 0, 0) // 发送即返, 不等关闭
	if result == 0 {
		return fmt.Errorf("PostMessageW(WM_CLOSE): %v", callErr)
	}
	return nil
}

type WindowState struct {
	State      string
	Foreground bool
	X          int
	Y          int
	Width      int
	Height     int
}

func InspectWindowState(hwnd uintptr) (WindowState, error) {
	if hwnd == 0 || !IsWindow(hwnd) {
		return WindowState{}, fmt.Errorf("invalid hwnd")
	}
	var rect win.RECT
	result, _, callErr := procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&rect)))
	if result == 0 {
		return WindowState{}, fmt.Errorf("GetWindowRect: %v", callErr)
	}
	state := "normal"
	if iconic, _, _ := procIsIconic.Call(hwnd); iconic != 0 {
		state = "minimized"
	} else if zoomed, _, _ := procIsZoomed.Call(hwnd); zoomed != 0 {
		state = "maximized"
	}
	return WindowState{
		State: state, Foreground: ForegroundWindow() == hwnd,
		X: int(rect.Left), Y: int(rect.Top), Width: int(rect.Right - rect.Left), Height: int(rect.Bottom - rect.Top),
	}, nil
}

// SavedWindow — borderless 进入前快照, 供 ExitBorderless 还原。
type SavedWindow struct {
	Style     uintptr
	Placement win.WINDOWPLACEMENT
	PID       uint32
}

func EnterBorderless(hwnd uintptr) (SavedWindow, error) {
	if hwnd == 0 {
		return SavedWindow{}, fmt.Errorf("hwnd 0")
	}
	style, _, _ := procGetWindowLongPtr.Call(hwnd, uintptr(gwlStyleIdx))
	var wp win.WINDOWPLACEMENT
	wp.Length = uint32(unsafe.Sizeof(wp))
	procGetWindowPlacement.Call(hwnd, uintptr(unsafe.Pointer(&wp)))
	saved := SavedWindow{Style: style, Placement: wp, PID: getWindowPID(win.HWND(hwnd))}

	// 去标题/边框
	procSetWindowLongPtr.Call(hwnd, uintptr(gwlStyleIdx), style&^(wsCaption|wsThickFrame))
	// 铺满所在显示器
	mon, _, _ := procMonitorFromWindow.Call(hwnd, monitorNearest)
	var mi win.MONITORINFO
	mi.CbSize = uint32(unsafe.Sizeof(mi))
	procGetMonitorInfo.Call(mon, uintptr(unsafe.Pointer(&mi)))
	r := mi.RcMonitor
	procSetWindowPos.Call(hwnd, 0, uintptr(r.Left), uintptr(r.Top),
		uintptr(r.Right-r.Left), uintptr(r.Bottom-r.Top), swpFrameChanged|swpNoZOrder|swpShowWindow)
	return saved, nil
}

func ExitBorderless(hwnd uintptr, saved SavedWindow) error {
	if hwnd == 0 {
		return fmt.Errorf("hwnd 0")
	}
	style := saved.Style
	if style == 0 {
		style = wsOverlapped // 无记录退化
	}
	procSetWindowLongPtr.Call(hwnd, uintptr(gwlStyleIdx), style)
	if saved.Placement.Length != 0 {
		procSetWindowPlacement.Call(hwnd, uintptr(unsafe.Pointer(&saved.Placement)))
	} else {
		procShowWindow.Call(hwnd, swRestore)
	}
	procSetWindowPos.Call(hwnd, 0, 0, 0, 0, 0, swpFrameChanged|swpNoZOrder|swpNoMove|swpNoSize)
	return nil
}

// WindowPID 暴露 hwnd 的进程 PID, 给 RestoreBorders 防 HWND 复用校验用。
func WindowPID(hwnd uintptr) uint32 { return getWindowPID(win.HWND(hwnd)) }
