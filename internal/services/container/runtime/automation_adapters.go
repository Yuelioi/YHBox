package runtime

import (
	"fmt"

	"github.com/lxn/win"

	"yotta/internal/automation/target"
	pkginput "yotta/pkg/input"
	"yotta/pkg/winutil"
)

type runtimeWin32Input struct {
	backend pkginput.Backend
}

func (a runtimeWin32Input) Click(hwnd uintptr, xRatio, yRatio float64, button string, durMs int) error {
	return a.backend.Click(win.HWND(hwnd), xRatio, yRatio, button, durMs)
}

func (a runtimeWin32Input) KeyDown(hwnd uintptr, key string) error {
	return a.backend.KeyDown(win.HWND(hwnd), key)
}

func (a runtimeWin32Input) KeyUp(hwnd uintptr, key string) error {
	return a.backend.KeyUp(win.HWND(hwnd), key)
}

func (a runtimeWin32Input) TypeText(hwnd uintptr, text string) error {
	return a.backend.TypeText(win.HWND(hwnd), text)
}

func (a runtimeWin32Input) MoveTo(hwnd uintptr, xRatio, yRatio float64) error {
	return a.backend.MoveTo(win.HWND(hwnd), xRatio, yRatio)
}

func (a runtimeWin32Input) Scroll(hwnd uintptr, xRatio, yRatio float64, notches int, horizontal bool) error {
	return a.backend.Scroll(win.HWND(hwnd), xRatio, yRatio, notches, horizontal)
}

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
