//go:build windows

package desktopapp

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

var (
	shell32                   = syscall.NewLazyDLL("shell32.dll")
	procShellExecuteWElevated = shell32.NewProc("ShellExecuteW")
)

func launchElevated() error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve Yotta executable: %w", err)
	}
	verb, err := syscall.UTF16PtrFromString("runas")
	if err != nil {
		return err
	}
	path, err := syscall.UTF16PtrFromString(executable)
	if err != nil {
		return err
	}
	directory, err := syscall.UTF16PtrFromString(filepath.Dir(executable))
	if err != nil {
		return err
	}
	result, _, callErr := procShellExecuteWElevated.Call(
		0,
		uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(path)),
		0,
		uintptr(unsafe.Pointer(directory)),
		1,
	)
	if result <= 32 {
		return fmt.Errorf("request administrator restart: %v (ShellExecuteW=%d)", callErr, result)
	}
	return nil
}
