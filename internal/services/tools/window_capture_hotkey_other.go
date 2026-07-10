//go:build !windows

package tools

import "github.com/yottaapp/yotta/pkg/platform"

func startWin32WindowTargetCapture(uint32, uint32, func(string, any)) (string, error) {
	return "", platform.NewUnsupportedError("Win32 window target capture")
}

func cancelWin32WindowTargetCapture(string) error { return nil }
