//go:build windows

package runtime

import (
	"errors"
	"fmt"

	"github.com/yottaapp/yotta/internal/automation/controller"
	"github.com/yottaapp/yotta/internal/automation/target"
	automationtrace "github.com/yottaapp/yotta/internal/automation/trace"
	"github.com/yottaapp/yotta/internal/services/container"
	pkgcapture "github.com/yottaapp/yotta/pkg/capture"
	pkginput "github.com/yottaapp/yotta/pkg/input"
)

type nativeWin32ControllerProvider struct {
	input   pkginput.Backend
	capture pkgcapture.IBackend
}

func newWin32ControllerProvider(rt *RuntimeContext) (win32ControllerProvider, error) {
	inputName := container.ReadWin32WindowTargetInputBackend(rt.Container)
	rawInput, err := pkginput.NewBackend(inputName)
	if err != nil {
		return nil, fmt.Errorf("input backend %q: %w", inputName, err)
	}
	input := NewSafeInputBackend(rawInput, rt)

	captureName := rt.Container.CaptureBackend
	if captureName == "" {
		captureName = "auto"
	}
	rawCapture, warning, err := pkgcapture.NewIBackend(captureName)
	if err != nil {
		_ = input.ReleaseAll()
		_ = input.Close()
		return nil, fmt.Errorf("capture backend %q: %w", captureName, err)
	}
	if warning != "" && rt.Emit != nil {
		rt.Emit("container:warning", map[string]any{"message": warning})
	}
	return &nativeWin32ControllerProvider{input: input, capture: NewSafeCaptureBackend(rawCapture, rt)}, nil
}

func (p *nativeWin32ControllerProvider) NewController(tg target.Target, rec automationtrace.Recorder, need controllerNeed) (controller.Controller, error) {
	deps := controller.Win32Deps{Trace: rec}
	if need.Input {
		if p.input == nil {
			return nil, fmt.Errorf("input backend not initialised")
		}
		deps.Input = runtimeWin32Input{backend: p.input}
		deps.Backend = p.input.Name()
	}
	if need.Capture {
		if p.capture == nil {
			return nil, fmt.Errorf("capture backend not initialised")
		}
		deps.Capture = runtimeWin32Capture{backend: p.capture}
		deps.Backend = p.capture.Name()
	}
	return controller.NewWin32Controller(tg, deps)
}

func (p *nativeWin32ControllerProvider) Close() error {
	var errs []error
	if p.input != nil {
		errs = append(errs, p.input.ReleaseAll(), p.input.Close())
		p.input = nil
	}
	if p.capture != nil {
		errs = append(errs, p.capture.Close())
		p.capture = nil
	}
	return errors.Join(errs...)
}

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

func (a runtimeWin32Input) CursorRatio(hwnd uintptr) (float64, float64, error) {
	return a.backend.CursorRatio(pkginput.Handle(hwnd))
}
