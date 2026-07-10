package runtime

import (
	"fmt"

	"github.com/lxn/win"

	"github.com/yottaapp/yotta/internal/automation/controller"
	"github.com/yottaapp/yotta/internal/automation/target"
	pkgcapture "github.com/yottaapp/yotta/pkg/capture"
	pkginput "github.com/yottaapp/yotta/pkg/input"
	"github.com/yottaapp/yotta/pkg/winutil"
)

type runtimeWin32Input struct {
	backend pkginput.Backend
}

type runtimeWin32Capture struct {
	backend pkgcapture.IBackend
}

func (a runtimeWin32Capture) Frame(hwnd uintptr) (controller.Frame, error) {
	img, err := a.backend.Frame(win.HWND(hwnd))
	if err != nil {
		return controller.Frame{}, err
	}
	size := target.Size{}
	if img != nil {
		bounds := img.Bounds()
		size = target.Size{W: bounds.Dx(), H: bounds.Dy()}
	}
	return controller.Frame{
		Image: img,
		Space: target.SpaceWindowClient,
		Size:  size,
	}, nil
}

func (a runtimeWin32Input) Click(hwnd uintptr, xRatio, yRatio float64, button string, durMs int) error {
	return a.backend.Click(win.HWND(hwnd), xRatio, yRatio, button, durMs)
}

func (a runtimeWin32Input) MouseDown(hwnd uintptr, xRatio, yRatio float64, button string) error {
	return a.backend.MouseDown(win.HWND(hwnd), xRatio, yRatio, button)
}

func (a runtimeWin32Input) MouseUp(hwnd uintptr, button string) error {
	return a.backend.MouseUp(win.HWND(hwnd), button)
}

func (a runtimeWin32Input) Drag(hwnd uintptr, x1Ratio, y1Ratio, x2Ratio, y2Ratio float64, button string, durationMs int) error {
	return a.backend.Drag(win.HWND(hwnd), x1Ratio, y1Ratio, x2Ratio, y2Ratio, button, durationMs)
}

func (a runtimeWin32Input) MouseMoveRel(hwnd uintptr, dx, dy, durationMs int) error {
	return a.backend.MouseMoveRel(win.HWND(hwnd), dx, dy, durationMs)
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
