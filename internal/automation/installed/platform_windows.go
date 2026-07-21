//go:build windows

package installed

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"slices"
	"sync"
	"time"

	"github.com/yottaapp/yotta/internal/automation/controller"
	"github.com/yottaapp/yotta/internal/automation/target"
	pkgcapture "github.com/yottaapp/yotta/pkg/capture"
	pkginput "github.com/yottaapp/yotta/pkg/input"
	"github.com/yottaapp/yotta/pkg/winutil"
)

type windowsDriver struct {
	profile Profile
	backend pkginput.Backend
	capture pkgcapture.IBackend
	gate    chan struct{}
	closed  bool
}

type windowsHeldInput struct {
	parent  *windowsDriver
	backend pkginput.Backend
	mu      sync.Mutex
	closed  bool
}

type windowsPlayback struct {
	parent *windowsDriver
	window winutil.WindowHandle
}

type controllerInputAdapter struct{ backend pkginput.Backend }

func (a controllerInputAdapter) Click(hwnd uintptr, x, y float64, button string, duration int) error {
	return a.backend.Click(pkginput.Handle(hwnd), x, y, button, duration)
}
func (a controllerInputAdapter) MouseDown(hwnd uintptr, x, y float64, button string) error {
	return a.backend.MouseDown(pkginput.Handle(hwnd), x, y, button)
}
func (a controllerInputAdapter) MouseUp(hwnd uintptr, button string) error {
	return a.backend.MouseUp(pkginput.Handle(hwnd), button)
}
func (a controllerInputAdapter) Drag(hwnd uintptr, x1, y1, x2, y2 float64, button string, duration int) error {
	return a.backend.Drag(pkginput.Handle(hwnd), x1, y1, x2, y2, button, duration)
}
func (a controllerInputAdapter) MouseMoveRel(hwnd uintptr, dx, dy, duration int) error {
	return a.backend.MouseMoveRel(pkginput.Handle(hwnd), dx, dy, duration)
}
func (a controllerInputAdapter) KeyDown(hwnd uintptr, key string) error {
	return a.backend.KeyDown(pkginput.Handle(hwnd), key)
}
func (a controllerInputAdapter) KeyUp(hwnd uintptr, key string) error {
	return a.backend.KeyUp(pkginput.Handle(hwnd), key)
}
func (a controllerInputAdapter) TypeText(hwnd uintptr, value string) error {
	return a.backend.TypeText(pkginput.Handle(hwnd), value)
}
func (a controllerInputAdapter) MoveTo(hwnd uintptr, x, y float64) error {
	return a.backend.MoveTo(pkginput.Handle(hwnd), x, y)
}
func (a controllerInputAdapter) Scroll(hwnd uintptr, x, y float64, notches int, horizontal bool) error {
	return a.backend.Scroll(pkginput.Handle(hwnd), x, y, notches, horizontal)
}
func (a controllerInputAdapter) CursorRatio(hwnd uintptr) (float64, float64, error) {
	return a.backend.CursorRatio(pkginput.Handle(hwnd))
}

type controllerCaptureAdapter struct{ backend pkgcapture.IBackend }

func (a controllerCaptureAdapter) Frame(hwnd uintptr) (controller.Frame, error) {
	frame, err := a.backend.Frame(pkgcapture.Handle(hwnd))
	if err != nil {
		return controller.Frame{}, err
	}
	bounds := frame.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, frame, bounds.Min, draw.Src)
	return controller.Frame{Image: rgba, Space: target.SpaceWindowClient, Size: target.Size{W: bounds.Dx(), H: bounds.Dy()}}, nil
}

func PlatformSupported() bool { return true }

func newPlatformDriver(profile Profile) (driver, error) {
	machine, ok := DesktopProfile(profile)
	if !ok {
		return nil, failure(CodeContractViolation, errors.New("Win32 driver received another adapter profile"))
	}
	backend, err := pkginput.NewBackend(machine.InputBackend)
	if err != nil {
		return nil, failure(CodeUnsupportedHost, err)
	}
	captureBackend, warning, err := pkgcapture.NewIBackend(machine.CaptureBackend)
	if err != nil {
		_ = backend.Close()
		return nil, failure(CodeUnsupportedHost, err)
	}
	if warning != "" {
		_ = captureBackend.Close()
		_ = backend.Close()
		return nil, failure(CodeUnsupportedHost, errors.New(warning))
	}
	gate := make(chan struct{}, 1)
	gate <- struct{}{}
	return &windowsDriver{profile: profile, backend: backend, capture: captureBackend, gate: gate}, nil
}

func (d *windowsDriver) ResolveTarget(ctx context.Context) (target.Target, error) {
	select {
	case <-ctx.Done():
		return target.Target{}, ctx.Err()
	case <-d.gate:
	}
	defer func() { d.gate <- struct{}{} }()
	if d.closed {
		return target.Target{}, failure(CodeContractViolation, errors.New("automation target driver is closed"))
	}
	window, err := d.resolve(ctx)
	if err != nil {
		return target.Target{}, err
	}
	return target.NewWin32WindowTarget(target.WindowHandle(window)), nil
}

func (d *windowsDriver) Capture(ctx context.Context) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-d.gate:
	}
	defer func() { d.gate <- struct{}{} }()
	if d.closed || d.capture == nil {
		return nil, failure(CodeContractViolation, errors.New("automation capture driver is closed"))
	}
	window, err := d.resolve(ctx)
	if err != nil {
		return nil, err
	}
	resolved, err := d.controller(window)
	if err != nil {
		return nil, failure(CodeCaptureFailed, err)
	}
	frame, err := resolved.Screenshot(ctx, controller.ScreenshotRequest{Space: target.SpaceWindowClient})
	if err != nil {
		return nil, failure(CodeCaptureFailed, err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, frame.Image); err != nil {
		return nil, failure(CodeCaptureFailed, err)
	}
	if int64(encoded.Len()) > MaxCaptureBytes {
		return nil, failure(CodeCaptureFailed, errors.New("captured PNG exceeds byte budget"))
	}
	return encoded.Bytes(), nil
}

func (d *windowsDriver) controller(window winutil.WindowHandle) (*controller.Win32Controller, error) {
	return controller.NewWin32Controller(
		target.NewWin32WindowTarget(target.WindowHandle(window)),
		controller.Win32Deps{
			Input: controllerInputAdapter{backend: d.backend}, Capture: controllerCaptureAdapter{backend: d.capture}, Backend: d.profile.AdapterKind(),
		},
	)
}

func (d *windowsDriver) PlayEvent(ctx context.Context, event PlaybackEvent) error {
	opened, err := d.OpenPlayback(ctx)
	if err != nil {
		return err
	}
	return opened.PlayEvent(ctx, event)
}

func (d *windowsDriver) OpenPlayback(ctx context.Context) (playbackSessionDriver, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-d.gate:
	}
	defer func() { d.gate <- struct{}{} }()
	if d.closed || d.backend == nil {
		return nil, failure(CodeContractViolation, errors.New("automation playback driver is closed"))
	}
	window, err := d.resolve(ctx)
	if err != nil {
		return nil, err
	}
	if d.backend.Name() == "sendinput" {
		if err := winutil.BringToFront(window.HWND); err != nil {
			return nil, failure(CodePlaybackFailed, err)
		}
	}
	return &windowsPlayback{parent: d, window: window}, nil
}

func (playback *windowsPlayback) PlayEvent(ctx context.Context, event PlaybackEvent) error {
	d := playback.parent
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-d.gate:
	}
	defer func() { d.gate <- struct{}{} }()
	if d.closed || d.backend == nil {
		return failure(CodeContractViolation, errors.New("automation playback driver is closed"))
	}
	window := playback.window
	handle := pkginput.Handle(window.HWND)
	switch event.Kind {
	case PlaybackKeyDown:
		return d.backend.KeyDownCode(handle, event.KeyCode)
	case PlaybackKeyUp:
		return d.backend.KeyUpCode(handle, event.KeyCode)
	case PlaybackButtonDown:
		return d.backend.MouseDown(handle, event.Point.X, event.Point.Y, event.Button)
	case PlaybackButtonUp:
		return d.backend.MouseUp(handle, event.Button)
	case PlaybackMove:
		return d.backend.MoveTo(handle, event.Point.X, event.Point.Y)
	case PlaybackMoveRelative:
		return d.backend.MouseMoveRel(handle, int(event.DeltaX), int(event.DeltaY), 0)
	case PlaybackScroll:
		return d.backend.Scroll(handle, event.Point.X, event.Point.Y, int(event.Notches), false)
	default:
		return failure(CodeContractViolation, errors.New("automation playback event is unsupported"))
	}
}

func (playback *windowsPlayback) ReleaseInput() error { return playback.parent.ReleaseInput() }

func (d *windowsDriver) ReleaseInput() error {
	<-d.gate
	defer func() { d.gate <- struct{}{} }()
	if d.closed || d.backend == nil {
		return nil
	}
	return d.backend.ReleaseAll()
}

func (d *windowsDriver) OpenHeldInput() (heldInputDriver, error) {
	machine, ok := DesktopProfile(d.profile)
	if !ok {
		return nil, failure(CodeContractViolation, errors.New("Win32 driver received another adapter profile"))
	}
	backend, err := pkginput.NewBackend(machine.InputBackend)
	if err != nil {
		return nil, failure(CodeUnsupportedHost, err)
	}
	return &windowsHeldInput{parent: d, backend: backend}, nil
}

func (h *windowsHeldInput) Execute(ctx context.Context, operation string, raw any) (runErr error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed || h.backend == nil {
		return failure(CodeContractViolation, errors.New("held input driver is closed"))
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-h.parent.gate:
	}
	defer func() { h.parent.gate <- struct{}{} }()
	if h.parent.closed {
		return failure(CodeContractViolation, errors.New("automation target driver is closed"))
	}
	window, err := h.parent.resolve(ctx)
	if err != nil {
		return err
	}
	if h.backend.Name() == "sendinput" {
		if err := winutil.BringToFront(window.HWND); err != nil {
			return failure(CodeInputFailed, err)
		}
	}
	handle := pkginput.Handle(window.HWND)
	switch request := raw.(type) {
	case HoldKeysRequest:
		for _, key := range request.Keys {
			if err := h.backend.KeyDown(handle, key); err != nil {
				return errors.Join(err, h.backend.ReleaseAll())
			}
		}
		return nil
	case HoldButtonRequest:
		point, err := windowPoint(request.Point, window.ClientW, window.ClientH)
		if err != nil {
			return err
		}
		if err := h.backend.MouseDown(handle, point.X, point.Y, request.Button); err != nil {
			return errors.Join(err, h.backend.ReleaseAll())
		}
		return nil
	default:
		return failure(CodeContractViolation, fmt.Errorf("held input operation %q is unsupported", operation))
	}
}

func (h *windowsHeldInput) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return nil
	}
	h.closed = true
	err := errors.Join(h.backend.ReleaseAll(), h.backend.Close())
	h.backend = nil
	return err
}

func (d *windowsDriver) Execute(ctx context.Context, operation string, raw any) (runErr error) {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-d.gate:
	}
	defer func() { d.gate <- struct{}{} }()
	if d.closed || d.backend == nil {
		return failure(CodeContractViolation, errors.New("automation input driver is closed"))
	}
	window, err := d.resolve(ctx)
	if err != nil {
		return err
	}
	inputOperation := slices.Contains(inputOperations, operation)
	if d.backend.Name() == "sendinput" && inputOperation {
		if err := winutil.BringToFront(window.HWND); err != nil {
			return failure(CodeInputFailed, err)
		}
	}
	if inputOperation {
		defer func() { runErr = errors.Join(runErr, d.backend.ReleaseAll()) }()
	}
	switch request := raw.(type) {
	case struct{}:
		switch operation {
		case OperationActivate:
			if err := winutil.BringToFront(window.HWND); err != nil {
				return failure(CodeWindowFailed, err)
			}
			return nil
		case OperationCloseWindow:
			if err := winutil.CloseWindow(window.HWND); err != nil {
				return failure(CodeWindowFailed, err)
			}
			return nil
		default:
			return failure(CodeContractViolation, errors.New("empty automation request is unsupported"))
		}
	case MoveResizeWindowRequest:
		if err := winutil.MoveResize(window.HWND, int(request.X), int(request.Y), int(request.Width), int(request.Height)); err != nil {
			return failure(CodeWindowFailed, err)
		}
		return nil
	case SetWindowStateRequest:
		var err error
		switch request.State {
		case "maximize":
			err = winutil.Maximize(window.HWND)
		case "minimize":
			err = winutil.Minimize(window.HWND)
		case "restore":
			err = winutil.Restore(window.HWND)
		}
		if err != nil {
			return failure(CodeWindowFailed, err)
		}
		return nil
	case ClickRequest:
		resolved, err := d.controller(window)
		if err != nil {
			return failure(CodeContractViolation, err)
		}
		point, err := windowPoint(request.Point, window.ClientW, window.ClientH)
		if err != nil {
			return err
		}
		return resolved.Click(ctx, controller.ClickRequest{
			Point: target.NewNormalizedPoint(point.X, point.Y), Button: request.Button, DurationMs: int(request.DurationMilliseconds),
		})
	case MoveRequest:
		resolved, err := d.controller(window)
		if err != nil {
			return failure(CodeContractViolation, err)
		}
		point, err := windowPoint(request.Point, window.ClientW, window.ClientH)
		if err != nil {
			return err
		}
		return resolved.Move(ctx, controller.MoveRequest{Point: target.NewNormalizedPoint(point.X, point.Y)})
	case ScrollRequest:
		resolved, err := d.controller(window)
		if err != nil {
			return failure(CodeContractViolation, err)
		}
		point, err := windowPoint(request.Point, window.ClientW, window.ClientH)
		if err != nil {
			return err
		}
		return resolved.Scroll(ctx, controller.ScrollRequest{
			Point: target.NewNormalizedPoint(point.X, point.Y), Notches: int(request.Notches), Horizontal: request.Horizontal,
		})
	case DragRequest:
		resolved, err := d.controller(window)
		if err != nil {
			return failure(CodeContractViolation, err)
		}
		from, err := windowPoint(request.From, window.ClientW, window.ClientH)
		if err != nil {
			return err
		}
		to, err := windowPoint(request.To, window.ClientW, window.ClientH)
		if err != nil {
			return err
		}
		return resolved.Drag(ctx, controller.DragRequest{
			From: target.NewNormalizedPoint(from.X, from.Y), To: target.NewNormalizedPoint(to.X, to.Y), Button: request.Button, DurationMs: int(request.DurationMilliseconds),
		})
	case RelativeMoveRequest:
		resolved, err := d.controller(window)
		if err != nil {
			return failure(CodeContractViolation, err)
		}
		return resolved.MoveRelative(ctx, controller.RelativeMoveRequest{Dx: int(request.DeltaX), Dy: int(request.DeltaY), DurationMs: int(request.DurationMilliseconds)})
	case PressKeysRequest:
		resolved, err := d.controller(window)
		if err != nil {
			return failure(CodeContractViolation, err)
		}
		for _, key := range request.Keys {
			if err := resolved.KeyDown(ctx, controller.KeyRequest{Key: key}); err != nil {
				return err
			}
		}
		if err := waitContext(ctx, request.DurationMilliseconds); err != nil {
			return err
		}
		for index := len(request.Keys) - 1; index >= 0; index-- {
			if err := resolved.KeyUp(ctx, controller.KeyRequest{Key: request.Keys[index]}); err != nil {
				return err
			}
		}
		return nil
	case TypeTextRequest:
		resolved, err := d.controller(window)
		if err != nil {
			return failure(CodeContractViolation, err)
		}
		return resolved.Text(ctx, controller.TextRequest{Text: request.Text})
	default:
		return failure(CodeContractViolation, errors.New("automation input request type is unsupported"))
	}
}

func (d *windowsDriver) WaitWindow(ctx context.Context, present bool, timeout time.Duration) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	case <-d.gate:
	}
	defer func() { d.gate <- struct{}{} }()
	if d.closed {
		return false, failure(CodeContractViolation, errors.New("automation target driver is closed"))
	}
	machine, ok := DesktopProfile(d.profile)
	if !ok {
		return false, failure(CodeContractViolation, errors.New("Win32 driver received another adapter profile"))
	}
	selector := winutil.MatchSpec{Title: machine.WindowTitle, TitleMatch: machine.WindowTitleMatch, Class: machine.WindowClass}
	deadline := time.Now().Add(timeout)
	for {
		probeTimeout := min(25*time.Millisecond, max(time.Millisecond, time.Until(deadline)))
		_, err := winutil.ResolveExecutableWindow(ctx, machine.Application.Executable, selector, machine.WindowSelection, probeTimeout, probeTimeout)
		found := err == nil || errors.Is(err, winutil.ErrWindowAmbiguous)
		if (present && found) || (!present && !found && errors.Is(err, winutil.ErrWindowNotFound)) {
			return true, nil
		}
		if err != nil && !errors.Is(err, winutil.ErrWindowNotFound) && !errors.Is(err, winutil.ErrWindowAmbiguous) {
			return false, err
		}
		if time.Now().After(deadline) {
			return false, nil
		}
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-time.After(min(100*time.Millisecond, max(time.Millisecond, time.Until(deadline)))):
		}
	}
}

func (d *windowsDriver) WindowState(ctx context.Context) (WindowStateResponse, error) {
	select {
	case <-ctx.Done():
		return WindowStateResponse{}, ctx.Err()
	case <-d.gate:
	}
	defer func() { d.gate <- struct{}{} }()
	if d.closed {
		return WindowStateResponse{}, failure(CodeContractViolation, errors.New("automation target driver is closed"))
	}
	window, err := d.resolve(ctx)
	if err != nil {
		return WindowStateResponse{}, err
	}
	state, err := winutil.InspectWindowState(window.HWND)
	if err != nil {
		return WindowStateResponse{}, failure(CodeWindowFailed, err)
	}
	return WindowStateResponse{
		State: state.State, Foreground: state.Foreground,
		X: int64(state.X), Y: int64(state.Y), Width: int64(state.Width), Height: int64(state.Height),
	}, nil
}

func (d *windowsDriver) resolve(ctx context.Context) (winutil.WindowHandle, error) {
	machine, ok := DesktopProfile(d.profile)
	if !ok {
		return winutil.WindowHandle{}, failure(CodeContractViolation, errors.New("Win32 driver received another adapter profile"))
	}
	executable := machine.Application.Executable
	selector := winutil.MatchSpec{Title: machine.WindowTitle, TitleMatch: machine.WindowTitleMatch, Class: machine.WindowClass}
	timeout := time.Duration(machine.ResolveTimeoutMilliseconds) * time.Millisecond
	window, err := winutil.ResolveExecutableWindow(ctx, executable, selector, machine.WindowSelection, timeout, min(100*time.Millisecond, timeout))
	if err != nil {
		switch {
		case errors.Is(err, winutil.ErrWindowAmbiguous):
			return winutil.WindowHandle{}, failure(CodeTargetAmbiguous, err)
		case errors.Is(err, winutil.ErrWindowNotFound):
			return winutil.WindowHandle{}, failure(CodeTargetNotFound, err)
		default:
			return winutil.WindowHandle{}, err
		}
	}
	return window, nil
}

func (d *windowsDriver) Close() error {
	<-d.gate
	defer func() { d.gate <- struct{}{} }()
	if d.closed {
		return nil
	}
	d.closed = true
	err := errors.Join(d.backend.ReleaseAll(), d.backend.Close(), d.capture.Close())
	d.backend = nil
	d.capture = nil
	return err
}

type normalizedPoint struct{ X, Y float64 }

func windowPoint(point Point, width, height int) (normalizedPoint, error) {
	if point.Unit == "ratio" {
		return normalizedPoint{X: point.X, Y: point.Y}, nil
	}
	if width <= 0 || height <= 0 || point.X >= float64(width) || point.Y >= float64(height) {
		return normalizedPoint{}, failure(CodeInvalidRequest, errors.New("pixel point is outside the installed target client area"))
	}
	return normalizedPoint{X: point.X / float64(width), Y: point.Y / float64(height)}, nil
}

func waitContext(ctx context.Context, milliseconds int64) error {
	if milliseconds <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(time.Duration(milliseconds) * time.Millisecond)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
