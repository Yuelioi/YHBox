//go:build windows

package runtime

import "syscall"

var (
	safeUser32       = syscall.NewLazyDLL("user32.dll")
	procIsWindowSafe = safeUser32.NewProc("IsWindow")
)

func isValidHwnd(hwnd uintptr) bool {
	if hwnd == 0 {
		return false
	}
	r, _, _ := procIsWindowSafe.Call(hwnd)
	return r != 0
}
