//go:build !windows

package tools

import "github.com/yottaapp/yotta/pkg/platform"

func (*Service) win32PixelAt(string) (PixelInfo, error) {
	return PixelInfo{}, platform.NewUnsupportedError("Win32 target pixel sampling")
}
