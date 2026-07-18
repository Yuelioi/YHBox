package installed

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image/png"
	"math"
	"sync"
	"time"

	"github.com/yottaapp/yotta/internal/automation/browsercdp"
	"github.com/yottaapp/yotta/internal/automation/controller"
	"github.com/yottaapp/yotta/internal/automation/target"
)

type browserDriver struct {
	profile Profile
	mu      sync.Mutex
	closed  bool
}

type BrowserHealthProbe struct{ driver *browserDriver }

func NewBrowserHealthProbe(profile Profile) (BrowserHealthProbe, error) {
	opened, err := newBrowserDriver(profile)
	if err != nil {
		return BrowserHealthProbe{}, err
	}
	return BrowserHealthProbe{driver: opened.(*browserDriver)}, nil
}

func (probe BrowserHealthProbe) Resolve(ctx context.Context) (target.Target, error) {
	if probe.driver == nil {
		return target.Target{}, errors.New("browser CDP health probe is unavailable")
	}
	return probe.driver.ResolveTarget(ctx)
}

func newBrowserDriver(profile Profile) (driver, error) {
	if !profile.Valid() || profile.AdapterKind() != AdapterKindBrowserCDP || profile.TargetKind() != TargetKindBrowserCDP {
		return nil, failure(CodeContractViolation, errors.New("browser CDP adapter requires a browser page profile"))
	}
	return &browserDriver{profile: profile}, nil
}

func (driver *browserDriver) ResolveTarget(ctx context.Context) (target.Target, error) {
	driver.mu.Lock()
	defer driver.mu.Unlock()
	resolved, client, err := driver.resolveLocked(ctx)
	if client != nil {
		_ = client.Close()
	}
	return resolved, err
}

func (driver *browserDriver) resolveLocked(ctx context.Context) (target.Target, *browsercdp.WebSocketClient, error) {
	if driver.closed {
		return target.Target{}, nil, failure(CodeContractViolation, errors.New("browser CDP automation driver is closed"))
	}
	machine, ok := BrowserProfile(driver.profile)
	if !ok {
		return target.Target{}, nil, failure(CodeContractViolation, errors.New("browser driver received another adapter profile"))
	}
	resolveCtx, cancel := context.WithTimeout(ctx, time.Duration(machine.ResolveTimeoutMilliseconds)*time.Millisecond)
	defer cancel()
	info, ok, err := browsercdp.NewService(machine.BrowserEndpoint).TargetByID(resolveCtx, machine.BrowserEndpoint, machine.BrowserTargetID)
	if err != nil {
		return target.Target{}, nil, failure(CodeTargetNotFound, err)
	}
	if !ok {
		return target.Target{}, nil, failure(CodeTargetNotFound, fmt.Errorf("browser CDP page %q is offline; reopen the page and install it again", machine.BrowserTitle))
	}
	if info.WebSocketDebuggerURL != machine.BrowserWebSocketURL {
		return target.Target{}, nil, failure(CodeIdentityChanged, errors.New("browser CDP page websocket identity changed; install the page again"))
	}
	client, err := browsercdp.DialWebSocketClient(resolveCtx, info.WebSocketDebuggerURL)
	if err != nil {
		return target.Target{}, nil, failure(CodeTargetNotFound, fmt.Errorf("connect browser CDP page: %w", err))
	}
	metrics, err := client.Call(resolveCtx, "Page.getLayoutMetrics", nil)
	if err != nil {
		_ = client.Close()
		return target.Target{}, nil, failure(CodeTargetNotFound, fmt.Errorf("read browser viewport: %w", err))
	}
	size, err := browserViewportSize(metrics)
	if err != nil {
		_ = client.Close()
		return target.Target{}, nil, failure(CodeContractViolation, err)
	}
	return browsercdp.TargetFromInfo(machine.BrowserEndpoint, info, size.W, size.H, machine.BrowserTitle), client, nil
}

func browserViewportSize(metrics map[string]any) (target.Size, error) {
	viewport, ok := metrics["cssLayoutViewport"].(map[string]any)
	if !ok {
		return target.Size{}, errors.New("browser CDP layout metrics omitted cssLayoutViewport")
	}
	width, widthOK := viewport["clientWidth"].(float64)
	height, heightOK := viewport["clientHeight"].(float64)
	if !widthOK || !heightOK || width <= 0 || height <= 0 || width > 100_000 || height > 100_000 {
		return target.Size{}, errors.New("browser CDP layout metrics reported an invalid viewport")
	}
	return target.Size{W: int(math.Round(width)), H: int(math.Round(height))}, nil
}

func (driver *browserDriver) Execute(ctx context.Context, operation string, raw any) error {
	driver.mu.Lock()
	defer driver.mu.Unlock()
	resolvedTarget, client, err := driver.resolveLocked(ctx)
	if err != nil {
		return err
	}
	defer client.Close()
	resolved, err := controller.NewBrowserCDPController(resolvedTarget, controller.BrowserCDPDeps{Client: client, Backend: AdapterKindBrowserCDP})
	if err != nil {
		return failure(CodeContractViolation, err)
	}
	switch request := raw.(type) {
	case ClickRequest:
		return resolved.Click(ctx, controller.ClickRequest{Point: browserPoint(request.Point), Button: request.Button, DurationMs: int(request.DurationMilliseconds)})
	case MoveRequest:
		return resolved.Move(ctx, controller.MoveRequest{Point: browserPoint(request.Point)})
	case ScrollRequest:
		return resolved.Scroll(ctx, controller.ScrollRequest{Point: browserPoint(request.Point), Notches: int(request.Notches), Horizontal: request.Horizontal})
	case DragRequest:
		return resolved.Drag(ctx, controller.DragRequest{From: browserPoint(request.From), To: browserPoint(request.To), Button: request.Button, DurationMs: int(request.DurationMilliseconds)})
	case PressKeysRequest:
		return resolved.KeyChord(ctx, controller.KeyChordRequest{Keys: request.Keys})
	case TypeTextRequest:
		return resolved.Text(ctx, controller.TextRequest{Text: request.Text})
	}
	return failure(CodeContractViolation, fmt.Errorf("browser CDP operation %q is unsupported", operation))
}

func browserPoint(point Point) target.Point {
	if point.Unit == "px" {
		return target.Point{X: point.X, Y: point.Y, Space: target.SpaceBrowserView}
	}
	return target.NewNormalizedPoint(point.X, point.Y)
}

func (driver *browserDriver) Capture(ctx context.Context) ([]byte, error) {
	driver.mu.Lock()
	defer driver.mu.Unlock()
	resolvedTarget, client, err := driver.resolveLocked(ctx)
	if err != nil {
		return nil, err
	}
	defer client.Close()
	resolved, err := controller.NewBrowserCDPController(resolvedTarget, controller.BrowserCDPDeps{Client: client, Backend: AdapterKindBrowserCDP})
	if err != nil {
		return nil, failure(CodeContractViolation, err)
	}
	frame, err := resolved.Screenshot(ctx, controller.ScreenshotRequest{Space: target.SpaceBrowserView})
	if err != nil {
		return nil, failure(CodeCaptureFailed, err)
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

func (driver *browserDriver) PlayEvent(context.Context, PlaybackEvent) error {
	return failure(CodePlaybackFailed, errors.New("browser CDP does not support low-level recording playback"))
}

func (driver *browserDriver) ReleaseInput() error { return nil }

func (driver *browserDriver) Close() error {
	driver.mu.Lock()
	defer driver.mu.Unlock()
	driver.closed = true
	return nil
}
