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
	// TargetKind remains as a source-compatible alias while callers migrate to
	// descriptors. It is semantic and no longer identifies Win32.
	TargetKind   = TargetKindDesktopWindow
	KindInput    = "automation/input-session"
	KindWindow   = "automation/window-session"
	KindCapture  = "automation/capture-session"
	KindPlayback = "automation/playback-session"

	OperationClick        = "click"
	OperationMove         = "move"
	OperationScroll       = "scroll"
	OperationDrag         = "drag"
	OperationMoveRelative = "move-relative"
	OperationPressKeys    = "press-keys"
	OperationTypeText     = "type-text"
	OperationActivate     = "activate"
	OperationStopApp      = "stop-app"
	OperationCapture      = "capture"
	OperationReadCapture  = "read-capture"
	OperationPlayEvent    = "play-event"
	OperationReleaseHeld  = "release-held"

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
var windowOperations = []string{OperationActivate, OperationStopApp}
var captureOperations = []string{OperationCapture, OperationReadCapture}
var playbackOperations = []string{OperationPlayEvent, OperationReleaseHeld}

func InputOperations() []string    { return append([]string(nil), inputOperations...) }
func WindowOperations() []string   { return append([]string(nil), windowOperations...) }
func CaptureOperations() []string  { return append([]string(nil), captureOperations...) }
func PlaybackOperations() []string { return append([]string(nil), playbackOperations...) }
func Operations() []string {
	operations := append(InputOperations(), windowOperations...)
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

type provider struct {
	profile       Profile
	driver        driver
	operations    []string
	closeOnce     sync.Once
	closeErr      error
	stateMu       sync.Mutex
	inputSessions int
	playbackOpen  bool
}
type session struct {
	mu             sync.Mutex
	operation      string
	inputAuthority bool
	closed         bool
}
type playbackSession struct {
	mu        sync.Mutex
	verified  bool
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

func newProvider(profile Profile, registry adapterRegistry) (*provider, error) {
	if !profile.Valid() {
		return nil, errors.New("automation provider requires a profile")
	}
	registered, err := registry.registration(profile)
	if err != nil {
		return nil, err
	}
	driver, err := registered.open(profile)
	if err != nil {
		return nil, err
	}
	return &provider{profile: profile, driver: driver, operations: append([]string(nil), registered.descriptor.Operations...)}, nil
}

func (p *provider) Open(_ context.Context, request resource.ProviderOpenRequest) (any, error) {
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
		defer p.stateMu.Unlock()
		if p.playbackOpen || p.inputSessions != 0 {
			return nil, failure(CodePlaybackBusy, errors.New("installed target input authority is already in use"))
		}
		p.playbackOpen = true
		return &playbackSession{}, nil
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
	if err := VerifyProfile(p.profile); err != nil {
		return nil, failure(CodeIdentityChanged, err)
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
		if operation == OperationActivate || operation == OperationStopApp {
			code = CodeWindowFailed
		}
		return nil, failure(code, err)
	}
	return artifact.Marshal(struct{}{})
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
		if !opened.verified {
			if err := VerifyProfile(p.profile); err != nil {
				return nil, failure(CodeIdentityChanged, err)
			}
			opened.verified = true
		}
		if event.Kind == PlaybackMoveRelative {
			targetCounts := p.profile.Machine().MouseCounts360
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
		if err := p.driver.PlayEvent(ctx, event); err != nil {
			return nil, classifyPlaybackFailure(err)
		}
		return artifact.Marshal(struct{}{})
	case OperationReleaseHeld:
		var request struct{}
		if err := decodeExact(payload, &request, 32); err != nil {
			return nil, failure(CodeInvalidRequest, err)
		}
		if err := p.driver.ReleaseInput(); err != nil {
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
		if err := VerifyProfile(p.profile); err != nil {
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
		releaseErr := p.driver.ReleaseInput()
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

func decodeOperationRequest(operation string, raw []byte) (any, error) {
	if len(raw) == 0 || len(raw) > 64<<10 {
		return nil, errors.New("automation request exceeds byte budget")
	}
	decode := func(target any) error { return decodeExact(raw, target, 64<<10) }
	switch operation {
	case OperationActivate, OperationStopApp:
		var request struct{}
		if err := decode(&request); err != nil {
			return nil, err
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
		if len(request.Keys) == 0 || len(request.Keys) > 16 || !validDuration(request.DurationMilliseconds) {
			return nil, errors.New("automation key request is invalid")
		}
		seen := map[string]struct{}{}
		for _, key := range request.Keys {
			if !controller.ValidKeyName(key) {
				return nil, errors.New("automation key request is invalid")
			}
			folded := strings.ToUpper(key)
			if _, exists := seen[folded]; exists {
				return nil, errors.New("automation key request contains duplicates")
			}
			seen[folded] = struct{}{}
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
