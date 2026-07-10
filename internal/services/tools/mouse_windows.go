package tools

import (
	"syscall"
	"unsafe"
)

var (
	user32             = syscall.NewLazyDLL("user32.dll")
	procGetCursorPos   = user32.NewProc("GetCursorPos")
	procScreenToClient = user32.NewProc("ScreenToClient")
)

type point struct {
	X int32
	Y int32
}

// readCursor 调 Win32 GetCursorPos 拿屏幕坐标。
func readCursor() (int, int, bool) {
	var p point
	r, _, _ := procGetCursorPos.Call(uintptr(unsafe.Pointer(&p)))
	if r == 0 {
		return 0, 0, false
	}
	return int(p.X), int(p.Y), true
}

// screenToClient 屏幕坐标 → hwnd 客户区坐标。
func screenToClient(hwnd uintptr, sx, sy int) (int, int, bool) {
	p := point{X: int32(sx), Y: int32(sy)}
	r, _, _ := procScreenToClient.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&p)))
	if r == 0 {
		return 0, 0, false
	}
	return int(p.X), int(p.Y), true
}
