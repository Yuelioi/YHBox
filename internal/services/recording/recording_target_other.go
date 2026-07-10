//go:build !windows

package recording

import (
	"github.com/yottaapp/yotta/internal/automation/target"
	"github.com/yottaapp/yotta/pkg/platform"
)

func resolveRecordingWindow(target.WindowMatchSpec) (target.WindowHandle, error) {
	return target.WindowHandle{}, platform.NewUnsupportedError("Win32 input recording target")
}
