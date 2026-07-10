//go:build !windows

package winutil

import (
	"context"
	"time"

	"github.com/yottaapp/yotta/pkg/platform"
)

func BringToFront(uintptr) bool { return false }

func ResolveWindow(context.Context, MatchSpec, time.Duration, time.Duration) (WindowHandle, error) {
	return WindowHandle{}, platform.NewUnsupportedError("native window resolution")
}

func WaitWindowGone(context.Context, MatchSpec, time.Duration, time.Duration) error {
	return platform.NewUnsupportedError("native window wait")
}

func EnumTopWindows() []WindowHandle { return nil }

func ClientSize(uintptr) (int, int, error) {
	return 0, 0, platform.NewUnsupportedError("native window client size")
}

func ForegroundWindow() uintptr { return 0 }

func WindowMetadata(uintptr) (WindowHandle, error) {
	return WindowHandle{}, platform.NewUnsupportedError("native window metadata")
}
