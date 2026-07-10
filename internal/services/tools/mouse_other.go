//go:build !windows

package tools

import "github.com/yottaapp/yotta/pkg/platform"

func readCursor() (int, int, error) {
	return 0, 0, platform.NewUnsupportedError("pointer position")
}

func screenToClient(uintptr, int, int) (int, int, error) {
	return 0, 0, platform.NewUnsupportedError("Win32 client coordinates")
}
