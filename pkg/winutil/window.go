// Package winutil 找异环游戏窗口。lxn/win 这版缺 EnumWindows / GetWindowTextW，
// 用 LazyDLL 自包。匹配规则：标题以"异环"开头 且 类名 = UnrealWindow（两条都要满足，
// 防止浏览器标签等"异环 ... - Chrome"被误匹配）。
package winutil

import (
	"syscall"
	"unsafe"

	"github.com/lxn/win"
)

var (
	user32             = syscall.NewLazyDLL("user32.dll")
	procEnumWindows    = user32.NewProc("EnumWindows")
	procGetWindowTextW = user32.NewProc("GetWindowTextW")
)

type Target struct {
	HWND  win.HWND
	Title string
	Class string
}

func utf16ToString(buf []uint16) string {
	n := 0
	for n < len(buf) && buf[n] != 0 {
		n++
	}
	return syscall.UTF16ToString(buf[:n])
}

// FindGame 枚举所有可见顶层窗口，返回所有匹配的（通常只有一个）。
func FindGame() []Target {
	var found []Target
	cb := syscall.NewCallback(func(hwnd win.HWND, _ uintptr) uintptr {
		if !win.IsWindowVisible(hwnd) {
			return 1
		}
		var tBuf [256]uint16
		var cBuf [256]uint16
		procGetWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&tBuf[0])), uintptr(len(tBuf)))
		win.GetClassName(hwnd, &cBuf[0], len(cBuf))
		t := utf16ToString(tBuf[:])
		c := utf16ToString(cBuf[:])
		titleMatch := len(t) >= len("异环") && t[:len("异环")] == "异环"
		classMatch := c == "UnrealWindow"
		if titleMatch && classMatch {
			found = append(found, Target{hwnd, t, c})
		}
		return 1
	})
	procEnumWindows.Call(cb, 0)
	return found
}
