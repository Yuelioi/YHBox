//go:build windows

package runtime

import (
	"context"
	"fmt"
	"time"

	"github.com/yottaapp/yotta/internal/automation/target"
	"github.com/yottaapp/yotta/pkg/winutil"
)

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

func enterNativeBorderless(hwnd uintptr) (any, error) {
	return winutil.EnterBorderless(hwnd)
}

func exitNativeBorderless(hwnd uintptr, state any) error {
	saved := winutil.SavedWindow{}
	if state != nil {
		var ok bool
		saved, ok = state.(winutil.SavedWindow)
		if !ok {
			return fmt.Errorf("invalid borderless window state %T", state)
		}
		if winutil.WindowPID(hwnd) != saved.PID {
			saved = winutil.SavedWindow{}
		}
	}
	return winutil.ExitBorderless(hwnd, saved)
}
