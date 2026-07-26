package tools

import (
	"fmt"
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

func readCursor() (int, int, error) {
	var p point
	r, _, _ := procGetCursorPos.Call(uintptr(unsafe.Pointer(&p)))
	if r == 0 {
		return 0, 0, fmt.Errorf("GetCursorPos failed")
	}
	return int(p.X), int(p.Y), nil
}

func screenToClient(hwnd uintptr, sx, sy int) (int, int, error) {
	p := point{X: int32(sx), Y: int32(sy)}
	r, _, _ := procScreenToClient.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&p)))
	if r == 0 {
		return 0, 0, fmt.Errorf("ScreenToClient failed")
	}
	return int(p.X), int(p.Y), nil
}
