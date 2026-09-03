package installed

import (
	"context"
	"image"
	"image/color"
	"os"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/automation/target"
	"github.com/yottaapp/yotta/internal/resource"
)

type fakeDriver struct {
	operation            string
	request              any
	err                  error
	closed               int
	capture              []byte
	frame                *image.RGBA
	window               target.WindowHandle
	held                 *fakeHeldInput
	waited               bool
	playbackOpens        int
	recordingActivations int
}

func (driver *fakeDriver) ActivateAndResolveTarget(context.Context) (target.Target, error) {
	driver.recordingActivations++
	driver.operation = OperationActivate
	return target.NewWin32WindowTarget(driver.window), driver.err
}

type fakeHeldInput struct {
	operation string
	request   any
	closed    int
	err       error
}

func (driver *fakeDriver) ResolveTarget(context.Context) (target.Target, error) {
	return target.NewWin32WindowTarget(driver.window), driver.err
}

func (driver *fakeDriver) Execute(_ context.Context, operation string, request any) error {
	driver.operation, driver.request = operation, request
	return driver.err
}
func (driver *fakeDriver) Capture(_ context.Context) ([]byte, error) {
	return append([]byte(nil), driver.capture...), driver.err
}
func (driver *fakeDriver) CaptureFrame(_ context.Context) (*image.RGBA, error) {
	return driver.frame, driver.err
}
func (driver *fakeDriver) PlayEvent(_ context.Context, event PlaybackEvent) error {
	driver.operation, driver.request = OperationPlayEvent, event
	return driver.err
}
func (driver *fakeDriver) OpenPlayback(context.Context) (playbackSessionDriver, error) {
	driver.playbackOpens++
	return driver, driver.err
}
func (driver *fakeDriver) ReleaseInput() error {
	driver.operation = OperationReleaseHeld
	return driver.err
}
func (driver *fakeDriver) Close() error { driver.closed++; return driver.err }
func (driver *fakeDriver) OpenHeldInput() (heldInputDriver, error) {
	if driver.held == nil {
		driver.held = &fakeHeldInput{}
	}
	return driver.held, driver.err
}
func (driver *fakeDriver) WaitWindow(_ context.Context, present bool, _ time.Duration) (bool, error) {
	driver.waited = present
	return present, driver.err
}
func (driver *fakeDriver) WindowState(context.Context) (WindowStateResponse, error) {
	return WindowStateResponse{State: "normal", Foreground: true, X: 10, Y: 20, Width: 800, Height: 600}, driver.err
}
func (driver *fakeDriver) PointerPosition(context.Context) (Point, error) {
	return Point{X: 0.25, Y: 0.75, Unit: "ratio"}, driver.err
}
func (driver *fakeHeldInput) Execute(_ context.Context, operation string, request any) error {
	driver.operation, driver.request = operation, request
	return driver.err
}
func (driver *fakeHeldInput) Close() error {
	driver.closed++
	return driver.err
}

func openInputSession(t *testing.T, provider *provider, operation string) any {
	t.Helper()
	object, err := provider.Open(context.Background(), resource.ProviderOpenRequest{
		Kind: KindInput, Operations: []string{operation}, Config: []byte(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	return object
}

func openWindowSession(t *testing.T, provider *provider, operation string) any {
	t.Helper()
	object, err := provider.Open(context.Background(), resource.ProviderOpenRequest{
		Kind: KindWindow, Operations: []string{operation}, Config: []byte(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	return object
}

func openCaptureSession(t *testing.T, provider *provider) any {
	t.Helper()
	object, err := provider.Open(context.Background(), resource.ProviderOpenRequest{
		Kind: KindCapture, Operations: CaptureOperations(), Config: []byte(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	return object
}

func openPlaybackSession(t *testing.T, provider *provider) any {
	t.Helper()
	object, err := provider.Open(context.Background(), resource.ProviderOpenRequest{
		Kind: KindPlayback, Operations: PlaybackOperations(), Config: []byte(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	return object
}

func openHeldInputSession(t *testing.T, provider *provider) any {
	t.Helper()
	object, err := provider.Open(context.Background(), resource.ProviderOpenRequest{
		Kind: KindHeldInput, Operations: HeldInputOperations(), Config: []byte(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	return object
}

func TestProviderUsesConfiguredOperationAndCanonicalPayload(t *testing.T) {
	profile, _ := testProfile(t)
	driver := &fakeDriver{}
	provider := &provider{profile: profile, driver: driver}
	object := openInputSession(t, provider, OperationClick)
	payload, err := artifact.Marshal(ClickRequest{Point: Point{X: 0.5, Y: 0.25, Unit: "ratio"}, Button: "left", DurationMilliseconds: 25})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := provider.Invoke(context.Background(), object, OperationClick, payload)
	if err != nil || OpenEffectResponse(raw) != nil || driver.operation != OperationClick {
		t.Fatalf("Invoke() operation=%q response=%s error=%v", driver.operation, raw, err)
	}
	request, ok := driver.request.(ClickRequest)
	if !ok || request.Point.X != 0.5 || request.DurationMilliseconds != 25 {
		t.Fatalf("driver request = %#v", driver.request)
	}
	if _, err := provider.Invoke(context.Background(), object, OperationMove, []byte(`{"point":{"x":0.5,"y":0.5,"unit":"ratio"}}`)); err == nil {
		t.Fatal("click session accepted move operation")
	}
}

func TestProviderReturnsCanonicalPointerPosition(t *testing.T) {
	profile, _ := testProfile(t)
	provider := &provider{profile: profile, driver: &fakeDriver{}}
	object := openInputSession(t, provider, OperationPointerPosition)
	raw, err := provider.Invoke(context.Background(), object, OperationPointerPosition, []byte(`{}`))
	response, decodeErr := OpenPointerPositionResponse(raw)
	if err != nil || decodeErr != nil || response.Point != (Point{X: 0.25, Y: 0.75, Unit: "ratio"}) {
		t.Fatalf("pointer response=%#v raw=%s invoke=%v decode=%v", response, raw, err, decodeErr)
	}
}

func TestProviderRejectsInvalidPayload(t *testing.T) {
	profile, _ := testProfile(t)
	provider := &provider{profile: profile, driver: &fakeDriver{}}
	object := openInputSession(t, provider, OperationPressKeys)
	for _, payload := range [][]byte{
		[]byte(`{"keys":[],"durationMilliseconds":10}`),
		[]byte(`{"keys":["CTRL","ctrl"],"durationMilliseconds":10}`),
		[]byte(`{"keys":["NOT-A-KEY"],"durationMilliseconds":10}`),
		[]byte(`{"keys":["CTRL"],"durationMilliseconds":10,"command":"forged"}`),
	} {
		if _, err := provider.Invoke(context.Background(), object, OperationPressKeys, payload); err == nil {
			t.Fatalf("provider accepted payload %s", payload)
		}
	}
}

func TestProviderRoutesOperationsBySessionKind(t *testing.T) {
	profile, _ := testProfile(t)
	driver := &fakeDriver{}
	provider := &provider{profile: profile, driver: driver}
	object := openWindowSession(t, provider, OperationActivate)
	raw, err := provider.Invoke(context.Background(), object, OperationActivate, []byte(`{}`))
	if err != nil || OpenEffectResponse(raw) != nil || driver.operation != OperationActivate {
		t.Fatalf("activate operation=%q response=%s error=%v", driver.operation, raw, err)
	}
	if _, ok := driver.request.(struct{}); !ok {
		t.Fatalf("activate request = %#v", driver.request)
	}
	for _, request := range []resource.ProviderOpenRequest{
		{Kind: KindInput, Operations: []string{OperationActivate}, Config: []byte(`{}`)},
		{Kind: KindWindow, Operations: []string{OperationClick}, Config: []byte(`{}`)},
		{Kind: KindCapture, Operations: []string{OperationCapture}, Config: []byte(`{}`)},
		{Kind: KindPlayback, Operations: []string{OperationPlayEvent}, Config: []byte(`{}`)},
	} {
		if _, err := provider.Open(context.Background(), request); err == nil {
			t.Fatalf("provider accepted an operation for the wrong session kind %#v", request)
		}
	}
}

func TestProviderPlaybackIsExclusiveScaledAndReleasesHeldState(t *testing.T) {
	profile, _ := testProfile(t)
	desktopProfile := desktopPayload(t, profile)
	desktopProfile.MouseCounts360 = 800
	profile, err := SealProfile(NewDesktopProfileDraft(desktopProfile))
	if err != nil {
		t.Fatal(err)
	}
	driver := &fakeDriver{}
	provider := &provider{profile: profile, driver: driver, runtimeMouseCounts360: 4134}
	input := openInputSession(t, provider, OperationMove)
	if _, err := provider.Open(context.Background(), resource.ProviderOpenRequest{
		Kind: KindPlayback, Operations: PlaybackOperations(), Config: []byte(`{}`),
	}); err == nil {
		t.Fatal("playback opened while an input session was active")
	}
	if err := provider.Close(context.Background(), input); err != nil {
		t.Fatal(err)
	}
	playback := openPlaybackSession(t, provider)
	if _, err := provider.Open(context.Background(), resource.ProviderOpenRequest{
		Kind: KindInput, Operations: []string{OperationClick}, Config: []byte(`{}`),
	}); err == nil {
		t.Fatal("input session opened while playback was active")
	}
	payload, err := artifact.Marshal(PlaybackEvent{Kind: PlaybackMoveRelative, DeltaX: 3, DeltaY: -2, SourceCounts360: 400})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := provider.Invoke(context.Background(), playback, OperationPlayEvent, payload)
	event, ok := driver.request.(PlaybackEvent)
	if err != nil || OpenEffectResponse(raw) != nil || !ok || event.DeltaX != 6 || event.DeltaY != -4 {
		t.Fatalf("playback event=%#v response=%s error=%v", driver.request, raw, err)
	}
	if err := provider.Close(context.Background(), playback); err != nil || driver.operation != OperationReleaseHeld || driver.playbackOpens != 1 {
		t.Fatalf("close operation=%q error=%v", driver.operation, err)
	}
}

func TestProviderPlaybackAcceptsAtomicClick(t *testing.T) {
	profile, _ := testProfile(t)
	driver := &fakeDriver{}
	provider := &provider{profile: profile, driver: driver}
	playback := openPlaybackSession(t, provider)
	payload := []byte(`{"kind":"click","point":{"x":0.25,"y":0.75,"unit":"ratio"},"button":"left","durationMilliseconds":94}`)
	raw, err := provider.Invoke(context.Background(), playback, OperationPlayEvent, payload)
	event, ok := driver.request.(PlaybackEvent)
	if err != nil || OpenEffectResponse(raw) != nil || !ok || event.Kind != PlaybackClick || event.DurationMilliseconds != 94 {
		t.Fatalf("playback click=%#v response=%s error=%v", driver.request, raw, err)
	}
}

func TestProviderRelativePlaybackUsesSourceScaleWhenTargetFollowsActiveCalibration(t *testing.T) {
	base, _ := testProfile(t)
	desktopProfile := desktopPayload(t, base)
	desktopProfile.MouseCounts360 = 0
	profile, err := SealProfile(NewDesktopProfileDraft(desktopProfile))
	if err != nil {
		t.Fatal(err)
	}
	driver := &fakeDriver{}
	provider := &provider{profile: profile, driver: driver, runtimeMouseCounts360: 4134}
	playback := openPlaybackSession(t, provider)
	payload, err := artifact.Marshal(PlaybackEvent{Kind: PlaybackMoveRelative, DeltaX: 3, DeltaY: -2, SourceCounts360: 4134})
	if err != nil {
		t.Fatal(err)
	}

	raw, err := provider.Invoke(context.Background(), playback, OperationPlayEvent, payload)
	event, ok := driver.request.(PlaybackEvent)
	if err != nil || OpenEffectResponse(raw) != nil || !ok || event.DeltaX != 3 || event.DeltaY != -2 {
		t.Fatalf("playback event=%#v response=%s error=%v", driver.request, raw, err)
	}
}

func TestProviderRelativePlaybackUsesExactCalibrationRatio(t *testing.T) {
	profile, _ := testProfile(t)
	desktopProfile := desktopPayload(t, profile)
	desktopProfile.MouseCounts360 = 1000
	profile, err := SealProfile(NewDesktopProfileDraft(desktopProfile))
	if err != nil {
		t.Fatal(err)
	}
	driver := &fakeDriver{}
	provider := &provider{profile: profile, driver: driver}
	playback := openPlaybackSession(t, provider)
	payload, err := artifact.Marshal(PlaybackEvent{Kind: PlaybackMoveRelative, DeltaX: 3, DeltaY: -2, SourceCounts360: 400})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := provider.Invoke(context.Background(), playback, OperationPlayEvent, payload)
	event, ok := driver.request.(PlaybackEvent)
	if err != nil || OpenEffectResponse(raw) != nil || !ok || event.DeltaX != 8 || event.DeltaY != -5 {
		t.Fatalf("playback event=%#v response=%s error=%v", driver.request, raw, err)
	}
}

func TestProviderHeldInputOwnsASeparateSessionAndReleasesState(t *testing.T) {
	profile, _ := testProfile(t)
	driver := &fakeDriver{}
	provider := &provider{profile: profile, driver: driver}
	held := openHeldInputSession(t, provider)
	payload, err := artifact.Marshal(HoldKeysRequest{Keys: []string{"CTRL", "A"}})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := provider.Invoke(context.Background(), held, OperationHoldKeys, payload)
	request, ok := driver.held.request.(HoldKeysRequest)
	if err != nil || OpenEffectResponse(raw) != nil || !ok || len(request.Keys) != 2 {
		t.Fatalf("held request=%#v response=%s error=%v", driver.held.request, raw, err)
	}
	if _, err := provider.Open(context.Background(), resource.ProviderOpenRequest{
		Kind: KindPlayback, Operations: PlaybackOperations(), Config: []byte(`{}`),
	}); err == nil {
		t.Fatal("playback opened while held input was active")
	}
	raw, err = provider.Invoke(context.Background(), held, OperationReleaseHeld, []byte(`{}`))
	if err != nil || OpenEffectResponse(raw) != nil || driver.held.closed != 1 {
		t.Fatalf("release response=%s closed=%d error=%v", raw, driver.held.closed, err)
	}
	playback := openPlaybackSession(t, provider)
	if err := provider.Close(context.Background(), playback); err != nil {
		t.Fatal(err)
	}
	if err := provider.Close(context.Background(), held); err != nil || driver.held.closed != 1 {
		t.Fatalf("idempotent close count=%d error=%v", driver.held.closed, err)
	}

	driver.held = nil
	leaked := openHeldInputSession(t, provider)
	if _, err := provider.Invoke(context.Background(), leaked, OperationHoldButton, []byte(`{"point":{"x":0.5,"y":0.5,"unit":"ratio"},"button":"left"}`)); err != nil {
		t.Fatal(err)
	}
	if err := provider.Close(context.Background(), leaked); err != nil || driver.held.closed != 1 {
		t.Fatalf("run cleanup closed=%d error=%v", driver.held.closed, err)
	}
}

func TestProviderWindowWaitReturnsAControlResult(t *testing.T) {
	profile, _ := testProfile(t)
	driver := &fakeDriver{}
	provider := &provider{profile: profile, driver: driver}
	object := openWindowSession(t, provider, OperationWaitWindow)
	raw, err := provider.Invoke(context.Background(), object, OperationWaitWindow, []byte(`{"timeoutMilliseconds":25}`))
	response, decodeErr := OpenWaitWindowResponse(raw)
	if err != nil || decodeErr != nil || !response.Matched || !driver.waited {
		t.Fatalf("wait response=%s decoded=%#v error=%v decode=%v", raw, response, err, decodeErr)
	}
}

func TestProviderWindowStateReturnsAValidatedObservation(t *testing.T) {
	profile, _ := testProfile(t)
	provider := &provider{profile: profile, driver: &fakeDriver{}}
	object := openWindowSession(t, provider, OperationGetWindowState)
	raw, err := provider.Invoke(context.Background(), object, OperationGetWindowState, []byte(`{}`))
	response, decodeErr := OpenWindowStateResponse(raw)
	if err != nil || decodeErr != nil || response.State != "normal" || !response.Foreground || response.Width != 800 {
		t.Fatalf("state response=%s decoded=%#v error=%v decode=%v", raw, response, err, decodeErr)
	}
}

func TestProviderCaptureIsBoundedAndRequiresCaptureBeforeRead(t *testing.T) {
	profile, _ := testProfile(t)
	driver := &fakeDriver{capture: []byte("png-bytes")}
	provider := &provider{profile: profile, driver: driver}
	object := openCaptureSession(t, provider)
	if _, err := provider.Invoke(context.Background(), object, OperationReadCapture, []byte(`{"offset":0,"length":1}`)); err == nil {
		t.Fatal("capture session allowed read before capture")
	}
	raw, err := provider.Invoke(context.Background(), object, OperationCapture, []byte(`{}`))
	response, decodeErr := OpenCaptureResponse(raw)
	if err != nil || decodeErr != nil || response.Size != 9 || response.MediaType != "image/png" {
		t.Fatalf("capture response=%s decoded=%#v error=%v decode=%v", raw, response, err, decodeErr)
	}
	chunk, err := provider.Invoke(context.Background(), object, OperationReadCapture, []byte(`{"offset":4,"length":5}`))
	if err != nil || string(chunk) != "bytes" {
		t.Fatalf("read capture=%q error=%v", chunk, err)
	}
	oversizedRange, marshalErr := artifact.Marshal(CaptureRangeRequest{Offset: 0, Length: MaxCaptureChunkBytes + 1})
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	if _, err := provider.Invoke(context.Background(), object, OperationReadCapture, oversizedRange); err == nil {
		t.Fatal("capture session accepted oversized range")
	}
	if err := provider.Close(context.Background(), object); err != nil {
		t.Fatal(err)
	}
	if _, err := provider.Invoke(context.Background(), object, OperationCapture, []byte(`{}`)); err == nil {
		t.Fatal("closed capture session accepted capture")
	}
}

func TestProviderCaptureProjectsRawRGBAWithoutPNGEncoding(t *testing.T) {
	profile, _ := testProfile(t)
	frame := image.NewRGBA(image.Rect(0, 0, 2, 1))
	frame.SetRGBA(0, 0, color.RGBA{R: 1, G: 2, B: 3, A: 4})
	frame.SetRGBA(1, 0, color.RGBA{R: 5, G: 6, B: 7, A: 8})
	provider := &provider{profile: profile, driver: &fakeDriver{frame: frame}}
	object := openCaptureSession(t, provider)

	raw, err := provider.Invoke(context.Background(), object, OperationCapture, []byte(`{"format":"rgba"}`))
	response, decodeErr := OpenCaptureResponse(raw)
	if err != nil || decodeErr != nil || response.MediaType != CaptureMediaTypeRGBA ||
		response.Width != 2 || response.Height != 1 || response.Size != 8 {
		t.Fatalf("capture response=%s decoded=%#v error=%v decode=%v", raw, response, err, decodeErr)
	}
	content, err := provider.Invoke(context.Background(), object, OperationReadCapture, []byte(`{"offset":0,"length":8}`))
	if err != nil || string(content) != string([]byte{1, 2, 3, 4, 5, 6, 7, 8}) {
		t.Fatalf("raw capture=%v error=%v", content, err)
	}
}

func TestProviderCaptureProjectsOnlyRequestedRGBARegion(t *testing.T) {
	profile, _ := testProfile(t)
	frame := image.NewRGBA(image.Rect(0, 0, 4, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 4; x++ {
			frame.SetRGBA(x, y, color.RGBA{R: uint8(y*4 + x + 1), A: 255})
		}
	}
	provider := &provider{profile: profile, driver: &fakeDriver{frame: frame}}
	object := openCaptureSession(t, provider)

	raw, err := provider.Invoke(context.Background(), object, OperationCapture, []byte(`{"format":"rgba","region":{"x":0.25,"y":0,"width":0.5,"height":1,"unit":"ratio"}}`))
	response, decodeErr := OpenCaptureResponse(raw)
	if err != nil || decodeErr != nil || response.Width != 2 || response.Height != 2 || response.Size != 16 {
		t.Fatalf("capture response=%s decoded=%#v error=%v decode=%v", raw, response, err, decodeErr)
	}
	content, err := provider.Invoke(context.Background(), object, OperationReadCapture, []byte(`{"offset":0,"length":16}`))
	if err != nil || len(content) != 16 || content[0] != 2 || content[4] != 3 || content[8] != 6 || content[12] != 7 {
		t.Fatalf("cropped raw capture=%v error=%v", content, err)
	}
}

func TestProviderCaptureAcceptsCanonicalWorkflowRegionRequest(t *testing.T) {
	profile, _ := testProfile(t)
	frame := image.NewRGBA(image.Rect(0, 0, 100, 100))
	provider := &provider{profile: profile, driver: &fakeDriver{frame: frame}}
	object := openCaptureSession(t, provider)
	payload, err := artifact.Marshal(CaptureRequest{Format: CaptureFormatRGBA, Region: &CaptureRegion{
		X: 0.3416666666666667, Y: 0.1527777777777778,
		Width: 0.33125, Height: 0.1824074074074074, Unit: "ratio",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(payload) <= 128 {
		t.Fatalf("workflow capture request unexpectedly fits legacy decoder limit: %d bytes", len(payload))
	}
	if _, err := provider.Invoke(context.Background(), object, OperationCapture, payload); err != nil {
		t.Fatalf("canonical workflow capture request (%d bytes) was rejected: %v", len(payload), err)
	}
}

func TestProviderContinuesAfterConfiguredExecutableUpdate(t *testing.T) {
	profile, path := testProfile(t)
	driver := &fakeDriver{}
	provider := &provider{profile: profile, driver: driver}
	object := openInputSession(t, provider, OperationTypeText)
	if err := os.WriteFile(path, []byte("installed-automation-target-v2"), 0o700); err != nil {
		t.Fatal(err)
	}
	_, err := provider.Invoke(context.Background(), object, OperationTypeText, []byte(`{"text":"hello"}`))
	if err != nil || driver.operation != OperationTypeText {
		t.Fatalf("Invoke() error=%v operation=%q", err, driver.operation)
	}
}

func TestConfiguredInstallationAndUnsupportedHost(t *testing.T) {
	profile, _ := testProfile(t)
	draft := InstallationDraft{Slot: "editor", Profile: profile.Machine()}
	installations, err := Install([]InstallationDraft{draft})
	if !PlatformSupported() {
		if err == nil {
			t.Fatal("Install succeeded on unsupported host")
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	if len(installations.Entries()) != 1 {
		t.Fatalf("entries = %d", len(installations.Entries()))
	}
	if err := installations.Close(); err != nil {
		t.Fatal(err)
	}
}
