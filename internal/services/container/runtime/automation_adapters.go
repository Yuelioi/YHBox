package runtime

import (
	"fmt"

	"yotta/internal/automation/target"
	"yotta/pkg/winutil"
)

func windowHandleToTarget(wh winutil.WindowHandle) target.Target {
	return target.Target{
		ID:          fmt.Sprintf("win32:%d", wh.HWND),
		Kind:        target.KindWin32Window,
		DisplayName: wh.Title,
		Ref:         target.TargetRef{HWND: wh.HWND},
		Resolution:  target.Size{W: wh.ClientW, H: wh.ClientH},
		Metadata: map[string]any{
			"class":   wh.Class,
			"process": wh.ProcessName,
			"pid":     wh.PID,
		},
	}
}
