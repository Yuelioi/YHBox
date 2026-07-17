//go:build windows

package tools

import "golang.org/x/sys/windows"

func processIsElevated() bool {
	return windows.GetCurrentProcessToken().IsElevated()
}
