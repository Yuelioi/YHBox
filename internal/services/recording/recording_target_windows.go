//go:build windows

package recording

import (
	"context"
	"time"

	"github.com/yottaapp/yotta/internal/automation/target"
	"github.com/yottaapp/yotta/pkg/winutil"
)

func resolveRecordingWindow(spec target.WindowMatchSpec) (target.WindowHandle, error) {
	window, err := winutil.ResolveWindow(context.Background(), spec, 3*time.Second, 500*time.Millisecond)
	if err != nil {
		return target.WindowHandle{}, err
	}
	_ = winutil.BringToFront(window.HWND)
	return window, nil
}
