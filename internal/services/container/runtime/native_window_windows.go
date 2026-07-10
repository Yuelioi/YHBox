//go:build windows

package runtime

import (
	"context"
	"time"

	"github.com/yottaapp/yotta/internal/automation/target"
	"github.com/yottaapp/yotta/pkg/winutil"
)

type nativeBorderlessState = winutil.SavedWindow

var (
	isWindowFn      = winutil.IsWindow
	resolveWindowFn = func(ctx context.Context, spec target.WindowMatchSpec, timeout, interval time.Duration) (target.WindowHandle, error) {
		return winutil.ResolveWindow(ctx, spec, timeout, interval)
	}
	clientSizeFn = winutil.ClientSize
)

func maximizeNativeWindow(hwnd uintptr) error { return winutil.Maximize(hwnd) }
func minimizeNativeWindow(hwnd uintptr) error { return winutil.Minimize(hwnd) }
func restoreNativeWindow(hwnd uintptr) error  { return winutil.Restore(hwnd) }
func moveResizeNativeWindow(hwnd uintptr, x, y, w, h int) error {
	return winutil.MoveResize(hwnd, x, y, w, h)
}
func closeNativeWindow(hwnd uintptr) error { return winutil.CloseWindow(hwnd) }

func enterNativeBorderless(hwnd uintptr) (nativeBorderlessState, error) {
	return winutil.EnterBorderless(hwnd)
}

func exitNativeBorderless(hwnd uintptr, saved nativeBorderlessState) error {
	if winutil.WindowPID(hwnd) != saved.PID {
		saved = winutil.SavedWindow{}
	}
	return winutil.ExitBorderless(hwnd, saved)
}
