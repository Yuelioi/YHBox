package installed

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/automation/controller"
	"github.com/yottaapp/yotta/internal/automation/target"
	"github.com/yottaapp/yotta/internal/resource"
)

const (
	ProviderABI             = "https://schemas.yotta.dev/provider-abi/resource/v1"
	TargetKindDesktopWindow = target.KindDesktopWindow
	TargetKindAndroidDevice = target.KindAndroidDevice
	TargetKindBrowserCDP    = target.KindBrowserCDP
	KindInput               = "automation/input-session"
	KindHeldInput           = "automation/held-input-session"
	KindWindow              = "automation/window-session"
	KindCapture             = "automation/capture-session"
	KindPlayback            = "automation/playback-session"

	OperationClick            = "click"
	OperationMove             = "move"
	OperationScroll           = "scroll"
	OperationDrag             = "drag"
	OperationMoveRelative     = "move-relative"
	OperationPressKeys        = "press-keys"
	OperationTypeText         = "type-text"
	OperationHoldKeys         = "hold-keys"
	OperationHoldButton       = "hold-button"
	OperationActivate         = "activate"
	OperationCloseWindow      = "close-window"
	OperationMoveResizeWindow = "move-resize-window"
	OperationSetWindowState   = "set-window-state"
	OperationGetWindowState   = "get-window-state"
	OperationWaitWindow       = "wait-window"
	OperationWaitWindowGone   = "wait-window-gone"
	OperationStopApp          = "stop-app"
	OperationCapture          = "capture"
	OperationReadCapture      = "read-capture"
	OperationPlayEvent        = "play-event"
	OperationReleaseHeld      = "release-held"

	CodeInvalidRequest    = "automation.invalid_request"
	CodeIdentityChanged   = "automation.identity_changed"
	CodeTargetNotFound    = "automation.target_not_found"
	CodeTargetAmbiguous   = "automation.target_ambiguous"
	CodeInputFailed       = "automation.input_failed"
	CodeWindowFailed      = "automation.window_failed"
	CodeCaptureFailed     = "automation.capture_failed"
	CodePlaybackFailed    = "automation.playback_failed"
	CodePlaybackBusy      = "automation.playback_busy"
	CodeUnsupportedHost   = "automation.unsupported_host"
	CodeContractViolation = "automation.contract_violation"

	providerImplementation = "installed-automation-target/v2"
	MaxInputDurationMs     = int64(60_000)
	MaxCaptureBytes        = int64(64 << 20)
	MaxCaptureChunkBytes   = int64(64 << 10)
)

var inputOperations = []string{
	OperationClick, OperationDrag, OperationMove, OperationMoveRelative,
	OperationPressKeys, OperationScroll, OperationTypeText,
}
var heldInputOperations = []string{OperationHoldButton, OperationHoldKeys, OperationReleaseHeld}
var windowOperations = []string{
	OperationActivate, OperationCloseWindow, OperationMoveResizeWindow, OperationSetWindowState,
	OperationGetWindowState, OperationStopApp, OperationWaitWindow, OperationWaitWindowGone,
}
var captureOperations = []string{OperationCapture, OperationReadCapture}
var playbackOperations = []string{OperationPlayEvent, OperationReleaseHeld}

func InputOperations() []string     { return append([]string(nil), inputOperations...) }
func HeldInputOperations() []string { return append([]string(nil), heldInputOperations...) }
func WindowOperations() []string    { return append([]string(nil), windowOperations...) }
func CaptureOperations() []string   { return append([]string(nil), captureOperations...) }
func PlaybackOperations() []string  { return append([]string(nil), playbackOperations...) }
func Operations() []string {
	operations := append(InputOperations(), heldInputOperations...)
	operations = append(operations, windowOperations...)
	operations = append(operations, captureOperations...)
	return append(operations, playbackOperations...)
}

func validSessionOperation(kind, operation string) bool {
	switch kind {
	case KindInput:
		return slices.Contains(inputOperations, operation)
	case KindWindow:
		return slices.Contains(windowOperations, operation)
	default:
		return false
	}
}

type Failure struct {
	Code  string
	Cause error
}

func (e *Failure) Error() string {
	if e == nil {
		return "automation failure"
	}
	if e.Cause == nil {
		return e.Code
	}
	return e.Code + ": " + e.Cause.Error()
}
func (e *Failure) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

type CapabilityScope struct {
	Operation string `json:"operation"`
}
type Point struct {
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	Unit string  `json:"unit"`
}
type ClickRequest struct {
	Point                Point  `json:"point"`
	Button               string `json:"button"`
	DurationMilliseconds int64  `json:"durationMilliseconds"`
}
type MoveRequest struct {
	Point Point `json:"point"`
}
type ScrollRequest struct {
	Point      Point `json:"point"`
	Notches    int64 `json:"notches"`
	Horizontal bool  `json:"horizontal"`
}
type DragRequest struct {
	From                 Point  `json:"from"`
	To                   Point  `json:"to"`
	Button               string `json:"button"`
	DurationMilliseconds int64  `json:"durationMilliseconds"`
}
type RelativeMoveRequest struct {
	DeltaX               int64 `json:"deltaX"`
	DeltaY               int64 `json:"deltaY"`
	DurationMilliseconds int64 `json:"durationMilliseconds"`
}
type PressKeysRequest struct {
	Keys                 []string `json:"keys"`
	DurationMilliseconds int64    `json:"durationMilliseconds"`
}
type HoldKeysRequest struct {
	Keys []string `json:"keys"`
}
type HoldButtonRequest struct {
	Point  Point  `json:"point"`
	Button string `json:"button"`
}
type MoveResizeWindowRequest struct {
	X      int64 `json:"x"`
	Y      int64 `json:"y"`
	Width  int64 `json:"width"`
	Height int64 `json:"height"`
}
type SetWindowStateRequest struct {
	State string `json:"state"`
}
type WaitWindowRequest struct {
	TimeoutMilliseconds int64 `json:"timeoutMilliseconds"`
}
type WaitWindowResponse struct {
	Matched bool `json:"matched"`
}
type WindowStateResponse struct {
	State      string `json:"state"`
	Foreground bool   `json:"foreground"`
	X          int64  `json:"x"`
	Y          int64  `json:"y"`
	Width      int64  `json:"width"`
	Height     int64  `json:"height"`
}
type TypeTextRequest struct {
	Text string `json:"text"`
}
type CaptureResponse struct {
	MediaType string `json:"mediaType"`
	Size      int64  `json:"size"`
}
type CaptureRangeRequest struct {
	Offset int64 `json:"offset"`
	Length int64 `json:"length"`
}

const (
	PlaybackKeyDown      = "key-down"
	PlaybackKeyUp        = "key-up"
	PlaybackButtonDown   = "button-down"
	PlaybackButtonUp     = "button-up"
	PlaybackMove         = "move"
	PlaybackMoveRelative = "move-relative"
	PlaybackScroll       = "scroll"
)

type PlaybackEvent struct {
	Kind            string `json:"kind"`
	KeyCode         uint32 `json:"keyCode,omitempty"`
	Point           *Point `json:"point,omitempty"`
	Button          string `json:"button,omitempty"`
	DeltaX          int64  `json:"deltaX,omitempty"`
	DeltaY          int64  `json:"deltaY,omitempty"`
	SourceCounts360 int64  `json:"sourceCounts360,omitempty"`
	Notches         int64  `json:"notches,omitempty"`
}

type driver interface {
	ResolveTarget(context.Context) (target.Target, error)
	Execute(context.Context, string, any) error
	Capture(context.Context) ([]byte, error)
	PlayEvent(context.Context, PlaybackEvent) error
	ReleaseInput() error
	Close() error
}

type playbackSessionDriver interface {
	PlayEvent(context.Context, PlaybackEvent) error
	ReleaseInput() error
}

type playbackOpener interface {
	OpenPlayback(context.Context) (playbackSessionDriver, error)
}

type directPlaybackSession struct{ driver driver }

func (session directPlaybackSession) PlayEvent(ctx context.Context, event PlaybackEvent) error {
	return session.driver.PlayEvent(ctx, event)
}

func (session directPlaybackSession) ReleaseInput() error { return session.driver.ReleaseInput() }

type provider struct {
	profile               Profile
	driver                driver
	verify                func(Profile) error
	operations            []string
	runtimeMouseCounts360 int64
	closeOnce             sync.Once
	closeErr              error
	stateMu               sync.Mutex
	inputSessions         int
	playbackOpen          bool
}
type session struct {
	mu             sync.Mutex
	operation      string
	inputAuthority bool
	closed         bool
}
type heldInputDriver interface {
	Execute(context.Context, string, any) error
	Close() error
}
type heldInputOpener interface {
	OpenHeldInput() (heldInputDriver, error)
}
type windowWaiter interface {
	WaitWindow(context.Context, bool, time.Duration) (bool, error)
}
type windowStateReader interface {
	WindowState(context.Context) (WindowStateResponse, error)
}
type heldInputSession struct {
	mu         sync.Mutex
	driver     heldInputDriver
	operations []string
	active     bool
	engaged    bool
	closed     bool
}
type playbackSession struct {
	mu        sync.Mutex
	driver    playbackSessionDriver
	residualX float64
	residualY float64
	closed    bool
}
type captureSession struct {
	mu     sync.Mutex
	data   []byte
	closed bool
}

func (p *provider) supports(operation string) bool {
	// Direct provider fixtures predate the adapter registry. Production
	// providers always carry an explicit operation set.
	return len(p.operations) == 0 || slices.Contains(p.operations, operation)
}

func (p *provider) verifyProfile() error {
	if p.verify != nil {
		return p.verify(p.profile)
	}
	return VerifyProfile(p.profile)
}

func newProvider(profile Profile, manifest InstallationManifest, registry adapterRegistry) (*provider, error) {
	document := manifest.Machine()
	if !profile.Valid() || !manifest.Valid() || document.ProfileDigest != profile.Digest() ||
		document.TargetKind != profile.TargetKind() || document.AdapterKind != profile.AdapterKind() ||
		document.ProfileVersion != profile.Machine().ProfileVersion || document.ProviderABI != ProviderABI {
		return nil, errors.New("automation provider requires a matching installation manifest")
	}
	registered, err := registry.registration(profile)
	if err != nil {
		return nil, err
	}
	driver, err := registered.open(profile)
	if err != nil {
		return nil, err
	}
	return &provider{profile: profile, driver: driver, verify: registered.verify, operations: operations(document.Capabilities)}, nil
}

func (p *provider) Open(ctx context.Context, request resource.ProviderOpenRequest) (any, error) {
	if request.CredentialBindingID != "" {
		return nil, failure(CodeContractViolation, errors.New("invalid automation session request"))
	}
	var config map[string]any
	if err := decodeExact(request.Config, &config, 1024); err != nil || len(config) != 0 {
		return nil, failure(CodeContractViolation, errors.New("automation session config must be empty"))
	}
	if request.Kind == KindCapture {
		if !p.supports(OperationCapture) || !p.supports(OperationReadCapture) {
			return nil, failure(CodeContractViolation, errors.New("capture is unsupported by this automation adapter"))
		}
		if !slices.Equal(request.Operations, captureOperations) {
			return nil, failure(CodeContractViolation, errors.New("capture session requires exact operations"))
		}
		var scope CapabilityScope
		if err := decodeExact(request.CapabilityScope, &scope, 1024); err != nil || scope.Operation != OperationCapture {
			return nil, failure(CodeContractViolation, errors.New("automation capability scope is invalid"))
		}
		return &captureSession{}, nil
	}
	if request.Kind == KindPlayback {
		if !p.supports(OperationPlayEvent) || !p.supports(OperationReleaseHeld) {
			return nil, failure(CodeContractViolation, errors.New("playback is unsupported by this automation adapter"))
		}
		if !slices.Equal(request.Operations, playbackOperations) {
			return nil, failure(CodeContractViolation, errors.New("playback session requires exact operations"))
		}
		var scope CapabilityScope
		if err := decodeExact(request.CapabilityScope, &scope, 1024); err != nil || scope.Operation != "play" {
			return nil, failure(CodeContractViolation, errors.New("automation capability scope is invalid"))
		}
		p.stateMu.Lock()
		if p.playbackOpen || p.inputSessions != 0 {
			p.stateMu.Unlock()
			return nil, failure(CodePlaybackBusy, errors.New("installed target input authority is already in use"))
		}
		p.playbackOpen = true
		p.stateMu.Unlock()
		failOpen := func(err error) (any, error) {
			p.stateMu.Lock()
			p.playbackOpen = false
			p.stateMu.Unlock()
			return nil, err
		}
		if err := p.verifyProfile(); err != nil {
			return failOpen(failure(CodeIdentityChanged, err))
		}
		openedDriver := playbackSessionDriver(directPlaybackSession{driver: p.driver})
		if opener, ok := p.driver.(playbackOpener); ok {
			var err error
			openedDriver, err = opener.OpenPlayback(ctx)
			if err != nil {
				return failOpen(classifyPlaybackFailure(err))
			}
		}
		return &playbackSession{driver: openedDriver}, nil
	}
	if request.Kind == KindHeldInput {
		if !slices.Equal(request.Operations, heldInputOperations) ||
			!p.supports(OperationHoldKeys) || !p.supports(OperationHoldButton) || !p.supports(OperationReleaseHeld) {
			return nil, failure(CodeContractViolation, errors.New("held input session requires exact operations"))
		}
		var scope CapabilityScope
		if err := decodeExact(request.CapabilityScope, &scope, 1024); err != nil || scope.Operation != "held-input" {
			return nil, failure(CodeContractViolation, errors.New("automation capability scope is invalid"))
		}
		opener, ok := p.driver.(heldInputOpener)
		if !ok {
			return nil, failure(CodeUnsupportedHost, errors.New("held input is unsupported by this automation adapter"))
		}
		p.stateMu.Lock()
		if p.playbackOpen {
			p.stateMu.Unlock()
			return nil, failure(CodePlaybackBusy, errors.New("installed target playback is active"))
		}
		p.inputSessions++
		p.stateMu.Unlock()
		driver, err := opener.OpenHeldInput()
		if err != nil {
			p.stateMu.Lock()
			p.inputSessions--
			p.stateMu.Unlock()
			return nil, err
		}
		return &heldInputSession{driver: driver, operations: HeldInputOperations(), active: true}, nil
	}
	if len(request.Operations) != 1 || !validSessionOperation(request.Kind, request.Operations[0]) {
		return nil, failure(CodeContractViolation, errors.New("invalid automation session request"))
	}
	if !p.supports(request.Operations[0]) {
		return nil, failure(CodeContractViolation, fmt.Errorf("operation %q is unsupported by this automation adapter", request.Operations[0]))
	}
	var scope CapabilityScope
	if err := decodeExact(request.CapabilityScope, &scope, 1024); err != nil || scope.Operation != request.Operations[0] {
		return nil, failure(CodeContractViolation, errors.New("automation capability scope is invalid"))
	}
	inputAuthority := request.Kind == KindInput
	if inputAuthority {
		p.stateMu.Lock()
		defer p.stateMu.Unlock()
		if p.playbackOpen {
			return nil, failure(CodePlaybackBusy, errors.New("installed target playback is active"))
		}
		p.inputSessions++
	}
	return &session{operation: scope.Operation, inputAuthority: inputAuthority}, nil
}

func (p *provider) Invoke(ctx context.Context, object any, operation string, payload []byte) ([]byte, error) {
	opened, ok := object.(*session)
	if !ok {
		held, heldOK := object.(*heldInputSession)
		if heldOK {
			return p.invokeHeldInput(ctx, held, operation, payload)
		}
		capture, captureOK := object.(*captureSession)
		if captureOK {
			return p.invokeCapture(ctx, capture, operation, payload)
		}
		playback, playbackOK := object.(*playbackSession)
		if playbackOK {
			return p.invokePlayback(ctx, playback, operation, payload)
		}
		return nil, failure(CodeContractViolation, errors.New("automation object has the wrong type"))
	}
	opened.mu.Lock()
	defer opened.mu.Unlock()
	if opened.closed || operation != opened.operation {
		return nil, failure(CodeContractViolation, errors.New("automation session is closed or operation is not granted"))
	}
	request, err := decodeOperationRequest(operation, payload)
	if err != nil {
		return nil, failure(CodeInvalidRequest, err)
	}
	if err := p.verifyProfile(); err != nil {
		return nil, failure(CodeIdentityChanged, err)
	}
	if operation == OperationWaitWindow || operation == OperationWaitWindowGone {
		waiter, ok := p.driver.(windowWaiter)
		if !ok {
			return nil, failure(CodeUnsupportedHost, errors.New("window waiting is unsupported by this automation adapter"))
		}
		waitRequest := request.(WaitWindowRequest)
		matched, err := waiter.WaitWindow(ctx, operation == OperationWaitWindow, time.Duration(waitRequest.TimeoutMilliseconds)*time.Millisecond)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil, err
			}
			return nil, failure(CodeWindowFailed, err)
		}
		return artifact.Marshal(WaitWindowResponse{Matched: matched})
	}
	if operation == OperationGetWindowState {
		reader, ok := p.driver.(windowStateReader)
		if !ok {
			return nil, failure(CodeUnsupportedHost, errors.New("window state is unsupported by this automation adapter"))
		}
		response, err := reader.WindowState(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil, err
			}
			return nil, failure(CodeWindowFailed, err)
		}
		return artifact.Marshal(response)
	}
	if err := p.driver.Execute(ctx, operation, request); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		var classified *Failure
		if errors.As(err, &classified) {
			return nil, err
		}
		code := CodeInputFailed
		if slices.Contains(windowOperations, operation) {
			code = CodeWindowFailed
		}
		return nil, failure(code, err)
	}
	return artifact.Marshal(struct{}{})
}

func (p *provider) invokeHeldInput(ctx context.Context, opened *heldInputSession, operation string, payload []byte) ([]byte, error) {
	opened.mu.Lock()
	defer opened.mu.Unlock()
	if opened.closed || !opened.active || !slices.Contains(opened.operations, operation) {
		return nil, failure(CodeContractViolation, errors.New("held input session is closed or operation is not granted"))
	}
	request, err := decodeOperationRequest(operation, payload)
	if err != nil {
		return nil, failure(CodeInvalidRequest, err)
	}
	if err := p.verifyProfile(); err != nil {
		return nil, failure(CodeIdentityChanged, err)
	}
	if operation == OperationReleaseHeld {
		closeErr := opened.driver.Close()
		opened.closed = true
		opened.active = false
		p.releaseInputAuthority()
		if closeErr != nil {
			return nil, failure(CodeInputFailed, closeErr)
		}
		return artifact.Marshal(struct{}{})
	}
	if opened.engaged {
		return nil, failure(CodeContractViolation, errors.New("held input session already owns input state"))
	}
	if err := opened.driver.Execute(ctx, operation, request); err != nil {
		closeErr := opened.driver.Close()
		opened.closed = true
		opened.active = false
		p.releaseInputAuthority()
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, errors.Join(err, closeErr)
		}
		return nil, failure(CodeInputFailed, errors.Join(err, closeErr))
	}
	opened.engaged = true
	return artifact.Marshal(struct{}{})
}

func (p *provider) releaseInputAuthority() {
	p.stateMu.Lock()
	p.inputSessions--
	p.stateMu.Unlock()
}

func (p *provider) invokePlayback(ctx context.Context, opened *playbackSession, operation string, payload []byte) ([]byte, error) {
	opened.mu.Lock()
	defer opened.mu.Unlock()
	if opened.closed {
		return nil, failure(CodeContractViolation, errors.New("automation playback session is closed"))
	}
	switch operation {
	case OperationPlayEvent:
		var event PlaybackEvent
		if err := decodeExact(payload, &event, 4096); err != nil || validatePlaybackEvent(event) != nil {
			return nil, failure(CodeInvalidRequest, errors.New("playback event is invalid"))
		}
		if event.Kind == PlaybackMoveRelative {
			desktop, ok := DesktopProfile(p.profile)
			if !ok {
				return nil, failure(CodeContractViolation, errors.New("relative playback requires a desktop profile"))
			}
			targetCounts := desktop.MouseCounts360
			if targetCounts <= 0 {
				targetCounts = p.runtimeMouseCounts360
			}
			if event.SourceCounts360 <= 0 || targetCounts <= 0 {
				return nil, failure(CodePlaybackFailed, errors.New("relative playback requires source and target calibration"))
			}
			factor := float64(targetCounts) / float64(event.SourceCounts360)
			scaledX := float64(event.DeltaX)*factor + opened.residualX
			scaledY := float64(event.DeltaY)*factor + opened.residualY
			roundedX, roundedY := math.Round(scaledX), math.Round(scaledY)
			opened.residualX, opened.residualY = scaledX-roundedX, scaledY-roundedY
			event.DeltaX, event.DeltaY = int64(roundedX), int64(roundedY)
		}
		if err := opened.driver.PlayEvent(ctx, event); err != nil {
			return nil, classifyPlaybackFailure(err)
		}
		return artifact.Marshal(struct{}{})
	case OperationReleaseHeld:
		var request struct{}
		if err := decodeExact(payload, &request, 32); err != nil {
			return nil, failure(CodeInvalidRequest, err)
		}
		if err := opened.driver.ReleaseInput(); err != nil {
			return nil, classifyPlaybackFailure(err)
		}
		return artifact.Marshal(struct{}{})
	default:
		return nil, failure(CodeContractViolation, errors.New("playback operation is not granted"))
	}
}

func classifyPlaybackFailure(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	var classified *Failure
	if errors.As(err, &classified) {
		return err
	}
	return failure(CodePlaybackFailed, err)
}

func (p *provider) invokeCapture(ctx context.Context, opened *captureSession, operation string, payload []byte) ([]byte, error) {
	opened.mu.Lock()
	defer opened.mu.Unlock()
	if opened.closed {
		return nil, failure(CodeContractViolation, errors.New("automation capture session is closed"))
	}
	switch operation {
	case OperationCapture:
		if err := p.verifyProfile(); err != nil {
			return nil, failure(CodeIdentityChanged, err)
		}
		var request struct{}
		if err := decodeExact(payload, &request, 32); err != nil {
			return nil, failure(CodeInvalidRequest, err)
		}
		data, err := p.driver.Capture(ctx)
		if err != nil {
			return nil, classifyCaptureFailure(err)
		}
		if len(data) == 0 || int64(len(data)) > MaxCaptureBytes {
			return nil, failure(CodeCaptureFailed, errors.New("capture result exceeds byte budget"))
		}
		opened.data = append(opened.data[:0], data...)
		return artifact.Marshal(CaptureResponse{MediaType: "image/png", Size: int64(len(opened.data))})
	case OperationReadCapture:
		if opened.data == nil {
			return nil, failure(CodeContractViolation, errors.New("capture must complete before reading"))
		}
		var request CaptureRangeRequest
		if err := decodeExact(payload, &request, 1024); err != nil || request.Offset < 0 || request.Length <= 0 || request.Length > MaxCaptureChunkBytes || request.Offset > int64(len(opened.data)) || request.Length > int64(len(opened.data))-request.Offset {
			return nil, failure(CodeInvalidRequest, errors.New("capture range is invalid"))
		}
		return append([]byte(nil), opened.data[request.Offset:request.Offset+request.Length]...), nil
	default:
		return nil, failure(CodeContractViolation, errors.New("capture operation is not granted"))
	}
}

func classifyCaptureFailure(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	var classified *Failure
	if errors.As(err, &classified) {
		return err
	}
	return failure(CodeCaptureFailed, err)
}

func (p *provider) Close(_ context.Context, object any) error {
	opened, ok := object.(*session)
	if !ok {
		held, heldOK := object.(*heldInputSession)
		if heldOK {
			held.mu.Lock()
			if held.closed {
				held.mu.Unlock()
				return nil
			}
			held.closed = true
			wasActive := held.active
			held.active = false
			closeErr := held.driver.Close()
			held.mu.Unlock()
			if wasActive {
				p.releaseInputAuthority()
			}
			if closeErr != nil {
				return failure(CodeInputFailed, closeErr)
			}
			return nil
		}
		capture, captureOK := object.(*captureSession)
		if captureOK {
			capture.mu.Lock()
			clear(capture.data)
			capture.data = nil
			capture.closed = true
			capture.mu.Unlock()
			return nil
		}
		playback, playbackOK := object.(*playbackSession)
		if !playbackOK {
			return failure(CodeContractViolation, errors.New("automation object has the wrong type"))
		}
		playback.mu.Lock()
		if playback.closed {
			playback.mu.Unlock()
			return nil
		}
		playback.closed = true
		releaseErr := playback.driver.ReleaseInput()
		playback.mu.Unlock()
		p.stateMu.Lock()
		p.playbackOpen = false
		p.stateMu.Unlock()
		return classifyPlaybackCloseFailure(releaseErr)
	}
	opened.mu.Lock()
	if opened.closed {
		opened.mu.Unlock()
		return nil
	}
	opened.closed = true
	inputAuthority := opened.inputAuthority
	opened.mu.Unlock()
	if inputAuthority {
		p.stateMu.Lock()
		p.inputSessions--
		p.stateMu.Unlock()
	}
	return nil
}

func classifyPlaybackCloseFailure(err error) error {
	if err == nil {
		return nil
	}
	return classifyPlaybackFailure(err)
}

func (p *provider) CloseHost() error {
	p.closeOnce.Do(func() { p.closeErr = p.driver.Close() })
	return p.closeErr
}

func ProviderArtifactDigest(profile Profile) (artifact.Digest, error) {
	if !profile.Valid() {
		return "", errors.New("automation provider artifact requires a profile")
	}
	raw, err := artifact.Marshal(map[string]any{"implementation": providerImplementation, "profileDigest": profile.Digest(), "providerAbi": ProviderABI})
	if err != nil {
		return "", err
	}
	return artifact.Sum("yotta/automation-provider-artifact/v1", raw)
}

func OpenEffectResponse(raw []byte) error {
	var response struct{}
	if err := decodeExact(raw, &response, 32); err != nil {
		return failure(CodeContractViolation, errors.New("invalid automation response"))
	}
	canonical, err := artifact.Marshal(response)
	if err != nil || !bytes.Equal(canonical, raw) {
		return failure(CodeContractViolation, errors.New("automation response is not canonical"))
	}
	return nil
}

func OpenCaptureResponse(raw []byte) (CaptureResponse, error) {
	var response CaptureResponse
	if err := decodeExact(raw, &response, 1024); err != nil || response.MediaType != "image/png" || response.Size <= 0 || response.Size > MaxCaptureBytes {
		return CaptureResponse{}, failure(CodeContractViolation, errors.New("invalid capture response"))
	}
	canonical, err := artifact.Marshal(response)
	if err != nil || !bytes.Equal(canonical, raw) {
		return CaptureResponse{}, failure(CodeContractViolation, errors.New("capture response is not canonical"))
	}
	return response, nil
}

func OpenWaitWindowResponse(raw []byte) (WaitWindowResponse, error) {
	var response WaitWindowResponse
	if err := decodeExact(raw, &response, 128); err != nil {
		return WaitWindowResponse{}, failure(CodeContractViolation, errors.New("invalid window wait response"))
	}
	canonical, err := artifact.Marshal(response)
	if err != nil || !bytes.Equal(canonical, raw) {
		return WaitWindowResponse{}, failure(CodeContractViolation, errors.New("window wait response is not canonical"))
	}
	return response, nil
}

func OpenWindowStateResponse(raw []byte) (WindowStateResponse, error) {
	var response WindowStateResponse
	if err := decodeExact(raw, &response, 1024); err != nil ||
		(response.State != "normal" && response.State != "minimized" && response.State != "maximized") ||
		response.Width <= 0 || response.Height <= 0 {
		return WindowStateResponse{}, failure(CodeContractViolation, errors.New("invalid window state response"))
	}
	canonical, err := artifact.Marshal(response)
	if err != nil || !bytes.Equal(canonical, raw) {
		return WindowStateResponse{}, failure(CodeContractViolation, errors.New("window state response is not canonical"))
	}
	return response, nil
}

func decodeOperationRequest(operation string, raw []byte) (any, error) {
	if len(raw) == 0 || len(raw) > 64<<10 {
		return nil, errors.New("automation request exceeds byte budget")
	}
	decode := func(target any) error { return decodeExact(raw, target, 64<<10) }
	switch operation {
	case OperationActivate, OperationCloseWindow, OperationGetWindowState, OperationStopApp, OperationReleaseHeld:
		var request struct{}
		if err := decode(&request); err != nil {
			return nil, err
		}
		return request, nil
	case OperationHoldKeys:
		var request HoldKeysRequest
		if err := decode(&request); err != nil {
			return nil, err
		}
		if err := validateKeys(request.Keys); err != nil {
			return nil, err
		}
		return request, nil
	case OperationHoldButton:
		var request HoldButtonRequest
		if err := decode(&request); err != nil {
			return nil, err
		}
		if err := validatePoint(request.Point); err != nil || !validButton(request.Button) {
			return nil, errors.New("automation held button request is invalid")
		}
		return request, nil
	case OperationMoveResizeWindow:
		var request MoveResizeWindowRequest
		if err := decode(&request); err != nil {
			return nil, err
		}
		if request.X < -1_000_000 || request.X > 1_000_000 || request.Y < -1_000_000 || request.Y > 1_000_000 ||
			request.Width < 1 || request.Width > 1_000_000 || request.Height < 1 || request.Height > 1_000_000 {
			return nil, errors.New("automation move/resize window request is invalid")
		}
		return request, nil
	case OperationSetWindowState:
		var request SetWindowStateRequest
		if err := decode(&request); err != nil {
			return nil, err
		}
		if request.State != "maximize" && request.State != "minimize" && request.State != "restore" {
			return nil, errors.New("automation window state request is invalid")
		}
		return request, nil
	case OperationWaitWindow, OperationWaitWindowGone:
		var request WaitWindowRequest
		if err := decode(&request); err != nil {
			return nil, err
		}
		if request.TimeoutMilliseconds < 0 || request.TimeoutMilliseconds > MaxInputDurationMs {
			return nil, errors.New("automation window wait request is invalid")
		}
		return request, nil
	case OperationClick:
		var request ClickRequest
		if err := decode(&request); err != nil {
			return nil, err
		}
		return request, validateClick(request)
	case OperationMove:
		var request MoveRequest
		if err := decode(&request); err != nil {
			return nil, err
		}
		return request, validatePoint(request.Point)
	case OperationScroll:
		var request ScrollRequest
		if err := decode(&request); err != nil {
			return nil, err
		}
		if err := validatePoint(request.Point); err != nil || request.Notches == 0 || request.Notches < -100 || request.Notches > 100 {
			return nil, errors.New("automation scroll request is invalid")
		}
		return request, nil
	case OperationDrag:
		var request DragRequest
		if err := decode(&request); err != nil {
			return nil, err
		}
		if err := validatePoint(request.From); err != nil {
			return nil, err
		}
		if err := validatePoint(request.To); err != nil || !validButton(request.Button) || !validDuration(request.DurationMilliseconds) {
			return nil, errors.New("automation drag request is invalid")
		}
		return request, nil
	case OperationMoveRelative:
		var request RelativeMoveRequest
		if err := decode(&request); err != nil {
			return nil, err
		}
		if request.DeltaX < -1_000_000 || request.DeltaX > 1_000_000 || request.DeltaY < -1_000_000 || request.DeltaY > 1_000_000 || !validDuration(request.DurationMilliseconds) {
			return nil, errors.New("automation relative move request is invalid")
		}
		return request, nil
	case OperationPressKeys:
		var request PressKeysRequest
		if err := decode(&request); err != nil {
			return nil, err
		}
		if !validDuration(request.DurationMilliseconds) {
			return nil, errors.New("automation key request is invalid")
		}
		if err := validateKeys(request.Keys); err != nil {
			return nil, err
		}
		return request, nil
	case OperationTypeText:
		var request TypeTextRequest
		if err := decode(&request); err != nil {
			return nil, err
		}
		if request.Text == "" || len(request.Text) > 4096 || !utf8.ValidString(request.Text) || bytes.IndexByte([]byte(request.Text), 0) >= 0 {
			return nil, errors.New("automation text request is invalid")
		}
		return request, nil
	default:
		return nil, errors.New("unsupported automation operation")
	}
}

func validateKeys(keys []string) error {
	if len(keys) == 0 || len(keys) > 16 {
		return errors.New("automation key request is invalid")
	}
	seen := map[string]struct{}{}
	for _, key := range keys {
		if !controller.ValidKeyName(key) {
			return errors.New("automation key request is invalid")
		}
		folded := strings.ToUpper(key)
		if _, exists := seen[folded]; exists {
			return errors.New("automation key request contains duplicates")
		}
		seen[folded] = struct{}{}
	}
	return nil
}

func validateClick(request ClickRequest) error {
	if err := validatePoint(request.Point); err != nil || !validButton(request.Button) || !validDuration(request.DurationMilliseconds) {
		return errors.New("automation click request is invalid")
	}
	return nil
}
func validatePoint(point Point) error {
	if point.Unit == "ratio" && point.X >= 0 && point.X <= 1 && point.Y >= 0 && point.Y <= 1 {
		return nil
	}
	if point.Unit == "px" && point.X >= 0 && point.X <= 1_000_000 && point.Y >= 0 && point.Y <= 1_000_000 {
		return nil
	}
	return errors.New("automation point is invalid")
}
func validButton(button string) bool {
	return button == "left" || button == "right" || button == "middle"
}
func validDuration(value int64) bool { return value >= 0 && value <= MaxInputDurationMs }

func validatePlaybackEvent(event PlaybackEvent) error {
	emptyPoint := event.Point == nil
	unusedKey := event.KeyCode == 0
	unusedButton := event.Button == ""
	unusedDelta := event.DeltaX == 0 && event.DeltaY == 0 && event.SourceCounts360 == 0
	unusedNotches := event.Notches == 0
	switch event.Kind {
	case PlaybackKeyDown, PlaybackKeyUp:
		if !emptyPoint || !unusedButton || !unusedDelta || !unusedNotches || event.KeyCode == 0 || event.KeyCode > 255 {
			return errors.New("invalid playback key event")
		}
	case PlaybackButtonDown, PlaybackButtonUp:
		if !unusedKey || !unusedDelta || !unusedNotches || emptyPoint || validatePoint(*event.Point) != nil || !validButton(event.Button) {
			return errors.New("invalid playback button event")
		}
	case PlaybackMove:
		if !unusedKey || !unusedButton || !unusedDelta || !unusedNotches || emptyPoint || validatePoint(*event.Point) != nil {
			return errors.New("invalid playback move event")
		}
	case PlaybackMoveRelative:
		if !unusedKey || !unusedButton || !emptyPoint || !unusedNotches || event.SourceCounts360 <= 0 || event.SourceCounts360 > 10_000_000 || event.DeltaX < -1<<31 || event.DeltaX > 1<<31-1 || event.DeltaY < -1<<31 || event.DeltaY > 1<<31-1 {
			return errors.New("invalid playback relative event")
		}
	case PlaybackScroll:
		if !unusedKey || !unusedButton || !unusedDelta || emptyPoint || validatePoint(*event.Point) != nil || event.Notches == 0 || event.Notches < -100 || event.Notches > 100 {
			return errors.New("invalid playback scroll event")
		}
	default:
		return errors.New("invalid playback event kind")
	}
	return nil
}
func decodeExact(raw []byte, target any, maximum int) error {
	if len(raw) == 0 || len(raw) > maximum {
		return errors.New("JSON document exceeds byte budget")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return errors.New("JSON document contains trailing values")
	}
	return nil
}
func failure(code string, cause error) error { return &Failure{Code: code, Cause: cause} }

var _ resource.Provider = (*provider)(nil)
