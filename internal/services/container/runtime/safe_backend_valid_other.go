//go:build !windows

package runtime

func isValidHwnd(hwnd uintptr) bool { return hwnd != 0 }
