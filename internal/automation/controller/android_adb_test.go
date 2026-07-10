package controller

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"reflect"
	"testing"

	"github.com/yottaapp/yotta/internal/automation/target"
	automationtrace "github.com/yottaapp/yotta/internal/automation/trace"
)

type adbCall struct {
	serial string
	args   []string
}

type fakeADBRunner struct {
	calls []adbCall
	outs  map[string][]byte
	out   []byte
	err   error
}

func (f *fakeADBRunner) Run(_ context.Context, serial string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, adbCall{serial: serial, args: append([]string(nil), args...)})
	if f.outs != nil {
		if out, ok := f.outs[adbCallKey(serial, args...)]; ok {
			return out, f.err
		}
	}
	return f.out, f.err
}

func adbCallKey(serial string, args ...string) string {
	out := serial + "|"
	for i, arg := range args {
		if i > 0 {
			out += " "
		}
		out += arg
	}
	return out
}

func TestAndroidADBControllerTargetAndCapabilities(t *testing.T) {
	ctrl, err := NewAndroidADBController(androidTarget(), AndroidADBDeps{Runner: &fakeADBRunner{}})
	if err != nil {
		t.Fatalf("NewAndroidADBController() error = %v", err)
	}
	if ctrl.Target().ID != "android:emulator-5554" {
		t.Fatalf("target id = %q", ctrl.Target().ID)
	}
	caps := ctrl.Capabilities(context.Background())
	if !caps.Screenshot || !caps.Click || !caps.StartApp || caps.KeyState {
		t.Fatalf("unexpected caps: %#v", caps)
	}
}

func TestAndroidADBControllerRejectsNonAndroidTarget(t *testing.T) {
	_, err := NewAndroidADBController(target.Target{
		ID:   "win32:42",
		Kind: target.KindWin32Window,
		Ref:  target.TargetRef{HWND: 42},
	}, AndroidADBDeps{})
	if err == nil {
		t.Fatalf("expected error for non-android target")
	}
}

func TestAndroidADBControllerClickUsesTapAndTrace(t *testing.T) {
	runner := &fakeADBRunner{}
	rec := automationtrace.NewMemoryRecorder()
	ctrl, err := NewAndroidADBController(androidTarget(), AndroidADBDeps{Runner: runner, Trace: rec})
	if err != nil {
		t.Fatalf("NewAndroidADBController() error = %v", err)
	}
	if err := ctrl.Click(context.Background(), ClickRequest{Point: target.NewNormalizedPoint(0.5, 0.25)}); err != nil {
		t.Fatalf("Click() error = %v", err)
	}
	wantArgs := []string{"shell", "input", "tap", "540", "480"}
	if len(runner.calls) != 2 || runner.calls[1].serial != "emulator-5554" || !reflect.DeepEqual(runner.calls[1].args, wantArgs) {
		t.Fatalf("adb calls = %#v, want args %#v", runner.calls, wantArgs)
	}
	records := rec.Records()
	if len(records) != 1 || records[0].Action != "click" || records[0].Backend != string(BackendAndroidADB) {
		t.Fatalf("trace records = %#v", records)
	}
	if len(records[0].CoordinateSteps) != 1 {
		t.Fatalf("coordinate steps len = %d, want 1", len(records[0].CoordinateSteps))
	}
	step := records[0].CoordinateSteps[0]
	if step.From != target.SpaceNormalized || step.To != target.SpaceAndroidDevice {
		t.Fatalf("step spaces = %s -> %s", step.From, step.To)
	}
}

func TestAndroidADBControllerClickUsesCurrentLandscapeOrientation(t *testing.T) {
	tg := androidTarget()
	tg.Resolution = target.Size{W: 720, H: 1280}
	runner := &fakeADBRunner{outs: map[string][]byte{
		adbCallKey("emulator-5554", "shell", "dumpsys", "input"): []byte("SurfaceOrientation: 1\n"),
	}}
	ctrl, err := NewAndroidADBController(tg, AndroidADBDeps{Runner: runner})
	if err != nil {
		t.Fatalf("NewAndroidADBController() error = %v", err)
	}
	if err := ctrl.Click(context.Background(), ClickRequest{Point: target.NewNormalizedPoint(0.7651, 0.2106)}); err != nil {
		t.Fatalf("Click() error = %v", err)
	}
	wantArgs := []string{"shell", "input", "tap", "979", "152"}
	if len(runner.calls) != 2 || !reflect.DeepEqual(runner.calls[1].args, wantArgs) {
		t.Fatalf("adb calls = %#v, want tap args %#v", runner.calls, wantArgs)
	}
}

func TestAndroidADBControllerTracesBackendErrors(t *testing.T) {
	runner := &fakeADBRunner{err: errors.New("adb failed")}
	rec := automationtrace.NewMemoryRecorder()
	ctrl, err := NewAndroidADBController(androidTarget(), AndroidADBDeps{Runner: runner, Trace: rec})
	if err != nil {
		t.Fatalf("NewAndroidADBController() error = %v", err)
	}
	err = ctrl.Click(context.Background(), ClickRequest{Point: target.NewNormalizedPoint(0.5, 0.25)})
	if err == nil {
		t.Fatalf("Click() error = nil, want backend error")
	}
	records := rec.Records()
	if len(records) != 1 {
		t.Fatalf("trace records len = %d, want 1: %#v", len(records), records)
	}
	if records[0].Status != automationtrace.StatusError || records[0].Error != "adb failed" {
		t.Fatalf("trace error record = %#v", records[0])
	}
	if len(records[0].CoordinateSteps) != 1 {
		t.Fatalf("coordinate steps len = %d, want 1", len(records[0].CoordinateSteps))
	}
}

func TestAndroidADBControllerDragUsesSwipe(t *testing.T) {
	runner := &fakeADBRunner{}
	ctrl, err := NewAndroidADBController(androidTarget(), AndroidADBDeps{Runner: runner})
	if err != nil {
		t.Fatalf("NewAndroidADBController() error = %v", err)
	}
	err = ctrl.Drag(context.Background(), DragRequest{
		From:       target.NewNormalizedPoint(0.1, 0.2),
		To:         target.NewNormalizedPoint(0.8, 0.9),
		DurationMs: 450,
	})
	if err != nil {
		t.Fatalf("Drag() error = %v", err)
	}
	wantArgs := []string{"shell", "input", "swipe", "108", "384", "864", "1728", "450"}
	if len(runner.calls) != 2 || !reflect.DeepEqual(runner.calls[1].args, wantArgs) {
		t.Fatalf("adb calls = %#v, want args %#v", runner.calls, wantArgs)
	}
}

func TestAndroidADBControllerPointConversionBoundaries(t *testing.T) {
	ctrl, err := NewAndroidADBController(androidTarget(), AndroidADBDeps{Runner: &fakeADBRunner{}})
	if err != nil {
		t.Fatalf("NewAndroidADBController() error = %v", err)
	}
	x, y, err := ctrl.pointToDevice(target.NewNormalizedPoint(1, 1))
	if err != nil {
		t.Fatalf("pointToDevice(normalized edge) error = %v", err)
	}
	if x != 1079 || y != 1919 {
		t.Fatalf("pointToDevice(normalized edge) = (%d,%d), want (1079,1919)", x, y)
	}
	x, y, err = ctrl.pointToDevice(target.Point{X: 12.4, Y: 56.6, Space: target.SpaceAndroidDevice})
	if err != nil {
		t.Fatalf("pointToDevice(device) error = %v", err)
	}
	if x != 12 || y != 57 {
		t.Fatalf("pointToDevice(device) = (%d,%d), want (12,57)", x, y)
	}
}

func TestAndroidADBControllerRejectsUnsupportedPointSpaceBeforeADBCall(t *testing.T) {
	runner := &fakeADBRunner{}
	ctrl, err := NewAndroidADBController(androidTarget(), AndroidADBDeps{Runner: runner})
	if err != nil {
		t.Fatalf("NewAndroidADBController() error = %v", err)
	}
	err = ctrl.Click(context.Background(), ClickRequest{Point: target.Point{X: 10, Y: 10, Space: target.SpaceBrowserView}})
	if err == nil {
		t.Fatalf("Click() error = nil, want unsupported space error")
	}
	if len(runner.calls) != 0 {
		t.Fatalf("adb calls = %#v, want none", runner.calls)
	}
}

func TestAndroidADBControllerRequiresResolutionForNormalizedPoint(t *testing.T) {
	tg := androidTarget()
	tg.Resolution = target.Size{}
	ctrl, err := NewAndroidADBController(tg, AndroidADBDeps{Runner: &fakeADBRunner{}})
	if err != nil {
		t.Fatalf("NewAndroidADBController() error = %v", err)
	}
	if _, _, err := ctrl.pointToDevice(target.NewNormalizedPoint(0.5, 0.5)); err == nil {
		t.Fatalf("pointToDevice() error = nil, want missing resolution error")
	}
}

func TestAndroidADBControllerTextEscapesADBInput(t *testing.T) {
	runner := &fakeADBRunner{}
	ctrl, err := NewAndroidADBController(androidTarget(), AndroidADBDeps{Runner: runner})
	if err != nil {
		t.Fatalf("NewAndroidADBController() error = %v", err)
	}
	if err := ctrl.Text(context.Background(), TextRequest{Text: "hello 100%"}); err != nil {
		t.Fatalf("Text() error = %v", err)
	}
	wantArgs := []string{"shell", "input", "text", "hello%s100\\%"}
	if len(runner.calls) != 1 || !reflect.DeepEqual(runner.calls[0].args, wantArgs) {
		t.Fatalf("adb calls = %#v, want args %#v", runner.calls, wantArgs)
	}
}

func TestAndroidADBControllerScreenshotDecodesPNGAndTracesResult(t *testing.T) {
	runner := &fakeADBRunner{out: tinyPNG(t)}
	rec := automationtrace.NewMemoryRecorder()
	ctrl, err := NewAndroidADBController(androidTarget(), AndroidADBDeps{Runner: runner, Trace: rec})
	if err != nil {
		t.Fatalf("NewAndroidADBController() error = %v", err)
	}
	frame, err := ctrl.Screenshot(context.Background(), ScreenshotRequest{})
	if err != nil {
		t.Fatalf("Screenshot() error = %v", err)
	}
	wantArgs := []string{"exec-out", "screencap", "-p"}
	if len(runner.calls) != 1 || !reflect.DeepEqual(runner.calls[0].args, wantArgs) {
		t.Fatalf("adb calls = %#v, want args %#v", runner.calls, wantArgs)
	}
	if frame.Image == nil || frame.Size.W != 2 || frame.Size.H != 1 || frame.Space != target.SpaceAndroidDevice {
		t.Fatalf("unexpected frame: %#v", frame)
	}
	records := rec.Records()
	if len(records) != 1 || records[0].Action != "screenshot" {
		t.Fatalf("trace records = %#v", records)
	}
	result, _ := records[0].Result.(map[string]any)
	if result["width"] != 2 || result["height"] != 1 {
		t.Fatalf("trace result = %#v", records[0].Result)
	}
}

func TestAndroidADBControllerStartStopApp(t *testing.T) {
	runner := &fakeADBRunner{}
	ctrl, err := NewAndroidADBController(androidTarget(), AndroidADBDeps{Runner: runner})
	if err != nil {
		t.Fatalf("NewAndroidADBController() error = %v", err)
	}
	if err := ctrl.StartApp(context.Background(), StartAppRequest{Intent: "com.example.app"}); err != nil {
		t.Fatalf("StartApp() error = %v", err)
	}
	if err := ctrl.StopApp(context.Background(), StopAppRequest{Intent: "com.example.app"}); err != nil {
		t.Fatalf("StopApp() error = %v", err)
	}
	wantStart := []string{"shell", "monkey", "-p", "com.example.app", "-c", "android.intent.category.LAUNCHER", "1"}
	wantStop := []string{"shell", "am", "force-stop", "com.example.app"}
	if len(runner.calls) != 2 || !reflect.DeepEqual(runner.calls[0].args, wantStart) || !reflect.DeepEqual(runner.calls[1].args, wantStop) {
		t.Fatalf("adb calls = %#v", runner.calls)
	}
}

func androidTarget() target.Target {
	return target.Target{
		ID:         "android:emulator-5554",
		Kind:       target.KindAndroidADB,
		Ref:        target.TargetRef{ADBSerial: "emulator-5554"},
		Resolution: target.Size{W: 1080, H: 1920},
	}
}

func tinyPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 1))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	img.Set(1, 0, color.RGBA{G: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}
