//go:build windows

package installed

import (
	"bytes"
	"context"
	"errors"
	"image/png"
	"math"
	"time"

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
	cached  uintptr
	closed  bool
}

func PlatformSupported() bool { return true }

func newPlatformDriver(profile Profile) (driver, error) {
	backend, err := pkginput.NewBackend(profile.Machine().InputBackend)
	if err != nil {
		return nil, failure(CodeUnsupportedHost, err)
	}
	captureBackend, warning, err := pkgcapture.NewIBackend(profile.Machine().CaptureBackend)
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

func (d *windowsDriver) ResolveWindow(ctx context.Context) (target.WindowHandle, error) {
	select {
	case <-ctx.Done():
		return target.WindowHandle{}, ctx.Err()
	case <-d.gate:
	}
	defer func() { d.gate <- struct{}{} }()
	if d.closed {
		return target.WindowHandle{}, failure(CodeContractViolation, errors.New("automation target driver is closed"))
	}
	window, err := d.resolve(ctx)
	if err != nil {
		return target.WindowHandle{}, err
	}
	return target.WindowHandle{
		HWND: window.HWND, Title: window.Title, Class: window.Class,
		ProcessName: window.ProcessName, PID: window.PID,
		ClientW: window.ClientW, ClientH: window.ClientH,
	}, nil
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
	frame, err := d.capture.Frame(pkgcapture.Handle(window.HWND))
	if err != nil {
		return nil, failure(CodeCaptureFailed, err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, frame); err != nil {
		return nil, failure(CodeCaptureFailed, err)
	}
	if int64(encoded.Len()) > MaxCaptureBytes {
		return nil, failure(CodeCaptureFailed, errors.New("captured PNG exceeds byte budget"))
	}
	return encoded.Bytes(), nil
}

func (d *windowsDriver) PlayEvent(ctx context.Context, event PlaybackEvent) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-d.gate:
	}
	defer func() { d.gate <- struct{}{} }()
	if d.closed || d.backend == nil {
		return failure(CodeContractViolation, errors.New("automation playback driver is closed"))
	}
	window, err := d.resolve(ctx)
	if err != nil {
		return err
	}
	if d.backend.Name() == "sendinput" {
		if err := winutil.BringToFront(window.HWND); err != nil {
			return failure(CodePlaybackFailed, err)
		}
	}
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

func (d *windowsDriver) ReleaseInput() error {
	<-d.gate
	defer func() { d.gate <- struct{}{} }()
	if d.closed || d.backend == nil {
		return nil
	}
	return d.backend.ReleaseAll()
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
	if d.backend.Name() == "sendinput" && operation != OperationActivate {
		if err := winutil.BringToFront(window.HWND); err != nil {
			return failure(CodeInputFailed, err)
		}
	}
	defer func() { runErr = errors.Join(runErr, d.backend.ReleaseAll()) }()
	handle := pkginput.Handle(window.HWND)
	switch request := raw.(type) {
	case struct{}:
		if operation != OperationActivate {
			return failure(CodeContractViolation, errors.New("empty automation request is unsupported"))
		}
		if err := winutil.BringToFront(window.HWND); err != nil {
			return failure(CodeWindowFailed, err)
		}
		return nil
	case ClickRequest:
		point, err := windowPoint(request.Point, window.ClientW, window.ClientH)
		if err != nil {
			return err
		}
		if err := d.backend.MouseDown(handle, point.X, point.Y, request.Button); err != nil {
			return err
		}
		if err := waitContext(ctx, request.DurationMilliseconds); err != nil {
			return err
		}
		return d.backend.MouseUp(handle, request.Button)
	case MoveRequest:
		point, err := windowPoint(request.Point, window.ClientW, window.ClientH)
		if err != nil {
			return err
		}
		return d.backend.MoveTo(handle, point.X, point.Y)
	case ScrollRequest:
		point, err := windowPoint(request.Point, window.ClientW, window.ClientH)
		if err != nil {
			return err
		}
		return d.backend.Scroll(handle, point.X, point.Y, int(request.Notches), request.Horizontal)
	case DragRequest:
		from, err := windowPoint(request.From, window.ClientW, window.ClientH)
		if err != nil {
			return err
		}
		to, err := windowPoint(request.To, window.ClientW, window.ClientH)
		if err != nil {
			return err
		}
		return d.drag(ctx, handle, from, to, request.Button, request.DurationMilliseconds)
	case RelativeMoveRequest:
		return d.moveRelative(ctx, handle, request)
	case PressKeysRequest:
		for _, key := range request.Keys {
			if err := d.backend.KeyDown(handle, key); err != nil {
				return err
			}
		}
		if err := waitContext(ctx, request.DurationMilliseconds); err != nil {
			return err
		}
		for index := len(request.Keys) - 1; index >= 0; index-- {
			if err := d.backend.KeyUp(handle, request.Keys[index]); err != nil {
				return err
			}
		}
		return nil
	case TypeTextRequest:
		for _, character := range request.Text {
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := d.backend.TypeText(handle, string(character)); err != nil {
				return err
			}
		}
		return nil
	default:
		return failure(CodeContractViolation, errors.New("automation input request type is unsupported"))
	}
}

func (d *windowsDriver) resolve(ctx context.Context) (winutil.WindowHandle, error) {
	machine := d.profile.Machine()
	executable := machine.Application.Executable
	if d.cached != 0 {
		window, err := winutil.VerifyExecutableWindow(d.cached, executable, machine.WindowTitle, machine.WindowClass)
		if err == nil {
			return window, nil
		}
		d.cached = 0
	}
	timeout := time.Duration(machine.ResolveTimeoutMilliseconds) * time.Millisecond
	window, err := winutil.ResolveUniqueExecutableWindow(ctx, executable, machine.WindowTitle, machine.WindowClass, timeout, min(100*time.Millisecond, timeout))
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
	d.cached = window.HWND
	return window, nil
}

func (d *windowsDriver) drag(ctx context.Context, handle pkginput.Handle, from, to normalizedPoint, button string, duration int64) error {
	if err := d.backend.MoveTo(handle, from.X, from.Y); err != nil {
		return err
	}
	if err := d.backend.MouseDown(handle, from.X, from.Y, button); err != nil {
		return err
	}
	steps := durationSteps(duration)
	for step := 1; step <= steps; step++ {
		if err := waitContext(ctx, duration/int64(steps)); err != nil {
			return err
		}
		progress := float64(step) / float64(steps)
		if err := d.backend.MoveTo(handle, from.X+(to.X-from.X)*progress, from.Y+(to.Y-from.Y)*progress); err != nil {
			return err
		}
	}
	return d.backend.MouseUp(handle, button)
}

func (d *windowsDriver) moveRelative(ctx context.Context, handle pkginput.Handle, request RelativeMoveRequest) error {
	steps := durationSteps(request.DurationMilliseconds)
	previousX, previousY := int64(0), int64(0)
	for step := 1; step <= steps; step++ {
		if err := waitContext(ctx, request.DurationMilliseconds/int64(steps)); err != nil {
			return err
		}
		currentX := int64(math.Round(float64(request.DeltaX) * float64(step) / float64(steps)))
		currentY := int64(math.Round(float64(request.DeltaY) * float64(step) / float64(steps)))
		if err := d.backend.MouseMoveRel(handle, int(currentX-previousX), int(currentY-previousY), 0); err != nil {
			return err
		}
		previousX, previousY = currentX, currentY
	}
	return nil
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

func durationSteps(duration int64) int {
	if duration <= 0 {
		return 1
	}
	return max(1, int(math.Ceil(float64(duration)/16)))
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
