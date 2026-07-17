//go:build !windows

package winutil

import (
	"context"
	"time"

	"github.com/yottaapp/yotta/pkg/platform"
)

func BringToFront(uintptr) error { return platform.NewUnsupportedError("bring native window to front") }

func ResolveWindow(context.Context, MatchSpec, time.Duration, time.Duration) (WindowHandle, error) {
	return WindowHandle{}, platform.NewUnsupportedError("native window resolution")
}

func ResolveUniqueExecutableWindow(context.Context, string, string, string, time.Duration, time.Duration) (WindowHandle, error) {
	return WindowHandle{}, platform.NewUnsupportedError("exact executable window resolution")
}

func VerifyExecutableWindow(uintptr, string, string, string) (WindowHandle, error) {
	return WindowHandle{}, platform.NewUnsupportedError("exact executable window verification")
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

func WindowExecutable(uintptr) (string, error) {
	return "", platform.NewUnsupportedError("native window executable")
}
