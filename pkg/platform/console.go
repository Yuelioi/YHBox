package platform

import "unsafe"

var (
	procGetConsoleMode     = kernel32.NewProc("GetConsoleMode")
	procSetConsoleMode     = kernel32.NewProc("SetConsoleMode")
	procGetStdHandle       = kernel32.NewProc("GetStdHandle")
	procSetConsoleOutputCP = kernel32.NewProc("SetConsoleOutputCP")
)

// InitConsole 禁用 QuickEdit、启用 ANSI 颜色、设置 UTF-8 输出。
func InitConsole() {
	disableQuickEdit()
	enableVirtualTerminal()
	procSetConsoleOutputCP.Call(65001)
}

func disableQuickEdit() {
	const (
		stdInputHandle      = ^uintptr(0) - 10 + 1
		enableQuickEditMode = 0x0040
		enableExtendedFlags = 0x0080
	)
	h, _, _ := procGetStdHandle.Call(stdInputHandle)
	if h == 0 {
		return
	}
	var mode uint32
	ret, _, _ := procGetConsoleMode.Call(h, uintptr(unsafe.Pointer(&mode)))
	if ret == 0 {
		return
	}
	newMode := (mode | enableExtendedFlags) &^ enableQuickEditMode
	procSetConsoleMode.Call(h, uintptr(newMode))
}

func enableVirtualTerminal() {
	const (
		stdOutputHandle              = ^uintptr(0) - 11 + 1
		enableVirtualTerminalProcess = 0x0004
	)
	h, _, _ := procGetStdHandle.Call(stdOutputHandle)
	if h == 0 {
		return
	}
	var mode uint32
	ret, _, _ := procGetConsoleMode.Call(h, uintptr(unsafe.Pointer(&mode)))
	if ret == 0 {
		return
	}
	procSetConsoleMode.Call(h, uintptr(mode|enableVirtualTerminalProcess))
}
