//go:build !windows

package runtime

import "github.com/yottaapp/yotta/pkg/platform"

func newWin32ControllerProvider(*RuntimeContext) (win32ControllerProvider, error) {
	return nil, platform.NewUnsupportedError("Win32 automation controller")
}
