//go:build !windows

package runtime

import (
	"context"
	"time"

	"github.com/yottaapp/yotta/internal/automation/target"
	"github.com/yottaapp/yotta/pkg/platform"
)

type nativeBorderlessState struct{}

var (
	isWindowFn      = func(uintptr) bool { return false }
	resolveWindowFn = func(context.Context, target.WindowMatchSpec, time.Duration, time.Duration) (target.WindowHandle, error) {
		return target.WindowHandle{}, platform.NewUnsupportedError("native window resolution")
	}
	clientSizeFn = func(uintptr) (int, int, error) {
		return 0, 0, platform.NewUnsupportedError("native window client size")
	}
)

func maximizeNativeWindow(uintptr) error {
	return platform.NewUnsupportedError("maximize native window")
}
func minimizeNativeWindow(uintptr) error {
	return platform.NewUnsupportedError("minimize native window")
}
func restoreNativeWindow(uintptr) error { return platform.NewUnsupportedError("restore native window") }
func moveResizeNativeWindow(uintptr, int, int, int, int) error {
	return platform.NewUnsupportedError("move or resize native window")
}
func closeNativeWindow(uintptr) error { return platform.NewUnsupportedError("close native window") }
func enterNativeBorderless(uintptr) (nativeBorderlessState, error) {
	return nativeBorderlessState{}, platform.NewUnsupportedError("borderless native window")
}
func exitNativeBorderless(uintptr, nativeBorderlessState) error {
	return platform.NewUnsupportedError("restore native window borders")
}
