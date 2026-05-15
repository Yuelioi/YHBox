// Package tools 杂项调试/辅助工具。MousePos 给 HUD 用；OpenMouseHUD /
// OpenScreenPicker 用 wails3 独立窗口承载交互式工具。
package tools

import (
	"syscall"
	"unsafe"

	"github.com/lxn/win"
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

// MousePosInfo 给前端 HUD 用。包含屏幕坐标 + 客户区坐标 + ratio。
// HWND 可为 0（未检测到游戏窗口），此时 Client/Ratio 字段为 0。
type MousePosInfo struct {
	ScreenX int     `json:"screenX"`
	ScreenY int     `json:"screenY"`
	HasGame bool    `json:"hasGame"`
	ClientX int     `json:"clientX"`
	ClientY int     `json:"clientY"`
	XRatio  float64 `json:"xRatio"`
	YRatio  float64 `json:"yRatio"`
	ClientW int     `json:"clientW"`
	ClientH int     `json:"clientH"`
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
func screenToClient(hwnd win.HWND, sx, sy int) (int, int, bool) {
	p := point{X: int32(sx), Y: int32(sy)}
	r, _, _ := procScreenToClient.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&p)))
	if r == 0 {
		return 0, 0, false
	}
	return int(p.X), int(p.Y), true
}
