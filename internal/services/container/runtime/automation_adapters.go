package runtime

import (
	"github.com/yottaapp/yotta/internal/automation/controller"
	"github.com/yottaapp/yotta/internal/automation/target"
	pkgcapture "github.com/yottaapp/yotta/pkg/capture"
	pkginput "github.com/yottaapp/yotta/pkg/input"
)

type runtimeWin32Input struct {
	backend pkginput.Backend
}

type runtimeWin32Capture struct {
	backend pkgcapture.IBackend
}

func (a runtimeWin32Capture) Frame(hwnd uintptr) (controller.Frame, error) {
	img, err := a.backend.Frame(pkgcapture.Handle(hwnd))
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
	return a.backend.Click(pkginput.Handle(hwnd), xRatio, yRatio, button, durMs)
}

func (a runtimeWin32Input) MouseDown(hwnd uintptr, xRatio, yRatio float64, button string) error {
	return a.backend.MouseDown(pkginput.Handle(hwnd), xRatio, yRatio, button)
}

func (a runtimeWin32Input) MouseUp(hwnd uintptr, button string) error {
	return a.backend.MouseUp(pkginput.Handle(hwnd), button)
}

func (a runtimeWin32Input) Drag(hwnd uintptr, x1Ratio, y1Ratio, x2Ratio, y2Ratio float64, button string, durationMs int) error {
	return a.backend.Drag(pkginput.Handle(hwnd), x1Ratio, y1Ratio, x2Ratio, y2Ratio, button, durationMs)
}

func (a runtimeWin32Input) MouseMoveRel(hwnd uintptr, dx, dy, durationMs int) error {
	return a.backend.MouseMoveRel(pkginput.Handle(hwnd), dx, dy, durationMs)
}

func (a runtimeWin32Input) KeyDown(hwnd uintptr, key string) error {
	return a.backend.KeyDown(pkginput.Handle(hwnd), key)
}

func (a runtimeWin32Input) KeyUp(hwnd uintptr, key string) error {
	return a.backend.KeyUp(pkginput.Handle(hwnd), key)
}

func (a runtimeWin32Input) TypeText(hwnd uintptr, text string) error {
	return a.backend.TypeText(pkginput.Handle(hwnd), text)
}

func (a runtimeWin32Input) MoveTo(hwnd uintptr, xRatio, yRatio float64) error {
	return a.backend.MoveTo(pkginput.Handle(hwnd), xRatio, yRatio)
}

func (a runtimeWin32Input) Scroll(hwnd uintptr, xRatio, yRatio float64, notches int, horizontal bool) error {
	return a.backend.Scroll(pkginput.Handle(hwnd), xRatio, yRatio, notches, horizontal)
}
