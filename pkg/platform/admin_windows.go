//go:build windows

// Package platform Win32 控制台和 UAC 基础设施。
package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

var (
	kernel32          = syscall.NewLazyDLL("kernel32.dll")
	shell32           = syscall.NewLazyDLL("shell32.dll")
	procIsUserAnAdmin = shell32.NewProc("IsUserAnAdmin")
	procShellExecuteW = shell32.NewProc("ShellExecuteW")
)

func IsAdmin() bool {
	ret, _, _ := procIsUserAnAdmin.Call()
	return ret != 0
}

func RelaunchAsAdmin() bool {
	exe, _ := os.Executable()
	exe, _ = filepath.Abs(exe)
	cwd, _ := os.Getwd()

	verb, _ := syscall.UTF16PtrFromString("runas")
	file, _ := syscall.UTF16PtrFromString(exe)
	args, _ := syscall.UTF16PtrFromString("")
	dir, _ := syscall.UTF16PtrFromString(cwd)

	ret, _, _ := procShellExecuteW.Call(
		0,
		uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(file)),
		uintptr(unsafe.Pointer(args)),
		uintptr(unsafe.Pointer(dir)),
		1,
	)
	return ret > 32
}

// EnsureAdmin 检查管理员权限，不足则提权重启。
func EnsureAdmin() {
	if IsAdmin() {
		return
	}
	fmt.Println("正在请求管理员权限...")
	if RelaunchAsAdmin() {
		os.Exit(0)
	}
	fmt.Fprintln(os.Stderr, "无法获得管理员权限（用户拒绝 UAC 或系统不支持）")
	os.Exit(1)
}
