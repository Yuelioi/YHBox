package controller

import (
	"context"
	"errors"
	"testing"

	"github.com/yottaapp/yotta/internal/automation/target"
	automationtrace "github.com/yottaapp/yotta/internal/automation/trace"
)

func TestWin32ControllerTargetAndCapabilities(t *testing.T) {
	tg := target.Target{
		ID:          "win32:42",
		Kind:        target.KindWin32Window,
		DisplayName: "Test Window",
		Ref:         target.TargetRef{HWND: 42},
		Resolution:  target.Size{W: 1280, H: 720},
	}
	ctrl, err := NewWin32Controller(tg, Win32Deps{Input: &fakeWin32Input{}, Capture: fakeWin32Capture{}})
	if err != nil {
		t.Fatalf("NewWin32Controller() error = %v", err)
	}
	if got := ctrl.Target(); got.ID != tg.ID {
		t.Fatalf("target id = %q, want %q", got.ID, tg.ID)
	}
	caps := ctrl.Capabilities(context.Background())
	if !caps.Screenshot || !caps.Click || !caps.PointerPosition || !caps.KeyState || !caps.Text {
		t.Fatalf("unexpected caps: %#v", caps)
	}
	if caps.StartApp || caps.StopApp {
		t.Fatalf("win32 phase1 should not expose app lifecycle: %#v", caps)
	}
}

func TestWin32ControllerCapabilitiesReflectInjectedDependencies(t *testing.T) {
	ctrl, err := NewWin32Controller(target.NewWin32WindowTarget(target.WindowHandle{HWND: 42}), Win32Deps{})
	if err != nil {
		t.Fatalf("NewWin32Controller() error = %v", err)
	}
	if caps := ctrl.Capabilities(context.Background()); caps.Screenshot || caps.Click || caps.PointerPosition {
		t.Fatalf("capabilities should not advertise missing dependencies: %#v", caps)
	}
}

func TestWin32ControllerFullCapabilitiesMatchProfile(t *testing.T) {
	ctrl, err := NewWin32Controller(
		target.NewWin32WindowTarget(target.WindowHandle{HWND: 42}),
		Win32Deps{Input: &fakeWin32Input{}, Capture: fakeWin32Capture{}},
	)
	if err != nil {
		t.Fatalf("NewWin32Controller() error = %v", err)
	}
	profile, ok := Profile(BackendWin32)
	if !ok {
		t.Fatal("Win32 backend profile not found")
	}
	if got := ctrl.Capabilities(context.Background()); got != profile.Capabilities {
		t.Fatalf("controller capabilities = %#v, profile = %#v", got, profile.Capabilities)
	}
}

func TestWin32ControllerRejectsNonWin32Target(t *testing.T) {
	_, err := NewWin32Controller(target.Target{
		ID:   "adb:device",
		Kind: target.KindAndroidADB,
		Ref:  target.TargetRef{ADBSerial: "device"},
	}, Win32Deps{})
	if err == nil {
		t.Fatalf("expected error for non-win32 target")
	}
}

type fakeWin32Input struct {
	clickHWND        uintptr
	clickX           float64
	clickY           float64
	keyDown          []string
	keyUp            []string
	moveHWND         uintptr
	moveX            float64
	moveY            float64
	scrollHWND       uintptr
	scrollX          float64
	scrollY          float64
	scrollNotches    int
	scrollHorizontal bool
	mouseDownHWND    uintptr
	mouseDownX       float64
	mouseDownY       float64
	mouseDownButton  string
	mouseUpHWND      uintptr
	mouseUpButton    string
	dragHWND         uintptr
	dragX1           float64
	dragY1           float64
	dragX2           float64
	dragY2           float64
	dragButton       string
	dragDurationMs   int
	moveRelHWND      uintptr
	moveRelDx        int
	moveRelDy        int
	moveRelDuration  int
	text             string
	err              error
}

func (f *fakeWin32Input) CursorRatio(uintptr) (float64, float64, error) {
	return 0.25, 0.75, f.err
}

type fakeWin32Capture struct{}

func (fakeWin32Capture) Frame(uintptr) (Frame, error) { return Frame{}, nil }

func TestWin32ControllerPointerPosition(t *testing.T) {
	ctrl, err := NewWin32Controller(
		target.NewWin32WindowTarget(target.WindowHandle{HWND: 42}),
		Win32Deps{Input: &fakeWin32Input{}},
	)
	if err != nil {
		t.Fatalf("NewWin32Controller() error = %v", err)
	}
	point, err := ctrl.PointerPosition(context.Background())
	if err != nil {
		t.Fatalf("PointerPosition() error = %v", err)
	}
	if point != target.NewNormalizedPoint(0.25, 0.75) {
		t.Fatalf("PointerPosition() = %#v, want normalized 0.25,0.75", point)
	}
}

func (f *fakeWin32Input) Click(hwnd uintptr, xRatio, yRatio float64, button string, durMs int) error {
	f.clickHWND = hwnd
	f.clickX = xRatio
	f.clickY = yRatio
	return nil
}

func (f *fakeWin32Input) KeyDown(hwnd uintptr, key string) error {
	f.keyDown = append(f.keyDown, key)
	return f.err
}

func (f *fakeWin32Input) KeyUp(hwnd uintptr, key string) error {
	f.keyUp = append(f.keyUp, key)
	return nil
}

func (f *fakeWin32Input) TypeText(hwnd uintptr, text string) error {
	f.text = text
	return nil
}

func (f *fakeWin32Input) MoveTo(hwnd uintptr, xRatio, yRatio float64) error {
	f.moveHWND = hwnd
	f.moveX = xRatio
	f.moveY = yRatio
	return nil
}

func (f *fakeWin32Input) Scroll(hwnd uintptr, xRatio, yRatio float64, notches int, horizontal bool) error {
	f.scrollHWND = hwnd
	f.scrollX = xRatio
	f.scrollY = yRatio
	f.scrollNotches = notches
	f.scrollHorizontal = horizontal
	return nil
}

func (f *fakeWin32Input) MouseDown(hwnd uintptr, xRatio, yRatio float64, button string) error {
	f.mouseDownHWND = hwnd
	f.mouseDownX = xRatio
	f.mouseDownY = yRatio
	f.mouseDownButton = button
	return nil
}

func (f *fakeWin32Input) MouseUp(hwnd uintptr, button string) error {
	f.mouseUpHWND = hwnd
	f.mouseUpButton = button
	return nil
}

func (f *fakeWin32Input) Drag(hwnd uintptr, x1Ratio, y1Ratio, x2Ratio, y2Ratio float64, button string, durationMs int) error {
	f.dragHWND = hwnd
	f.dragX1 = x1Ratio
	f.dragY1 = y1Ratio
	f.dragX2 = x2Ratio
	f.dragY2 = y2Ratio
	f.dragButton = button
	f.dragDurationMs = durationMs
	return nil
}

func (f *fakeWin32Input) MouseMoveRel(hwnd uintptr, dx, dy, durationMs int) error {
	f.moveRelHWND = hwnd
	f.moveRelDx = dx
	f.moveRelDy = dy
	f.moveRelDuration = durationMs
	return nil
}

func TestWin32ControllerClickDelegatesNormalizedPoint(t *testing.T) {
	in := &fakeWin32Input{}
	ctrl, err := NewWin32Controller(target.Target{
		ID:   "win32:42",
		Kind: target.KindWin32Window,
		Ref:  target.TargetRef{HWND: 42},
	}, Win32Deps{Input: in})
	if err != nil {
		t.Fatalf("NewWin32Controller() error = %v", err)
	}
	err = ctrl.Click(context.Background(), ClickRequest{
		Point:  target.NewNormalizedPoint(0.25, 0.75),
		Button: "left",
	})
	if err != nil {
		t.Fatalf("Click() error = %v", err)
	}
	if in.clickHWND != 42 || in.clickX != 0.25 || in.clickY != 0.75 {
		t.Fatalf("delegated click = hwnd %d (%f,%f)", in.clickHWND, in.clickX, in.clickY)
	}
}

func TestWin32ControllerKeyChordDelegatesDownReverseUp(t *testing.T) {
	in := &fakeWin32Input{}
	ctrl, err := NewWin32Controller(target.Target{
		ID:   "win32:42",
		Kind: target.KindWin32Window,
		Ref:  target.TargetRef{HWND: 42},
	}, Win32Deps{Input: in})
	if err != nil {
		t.Fatalf("NewWin32Controller() error = %v", err)
	}
	err = ctrl.KeyChord(context.Background(), KeyChordRequest{Keys: []string{"ctrl", "n"}})
	if err != nil {
		t.Fatalf("KeyChord() error = %v", err)
	}
	if got := in.keyDown; len(got) != 2 || got[0] != "ctrl" || got[1] != "n" {
		t.Fatalf("keyDown = %#v", got)
	}
	if got := in.keyUp; len(got) != 2 || got[0] != "n" || got[1] != "ctrl" {
		t.Fatalf("keyUp = %#v", got)
	}
}

func TestWin32ControllerMoveRecordsCoordinateStep(t *testing.T) {
	in := &fakeWin32Input{}
	rec := automationtrace.NewMemoryRecorder()
	ctrl, err := NewWin32Controller(target.Target{
		ID:   "win32:42",
		Kind: target.KindWin32Window,
		Ref:  target.TargetRef{HWND: 42},
	}, Win32Deps{Input: in, Trace: rec})
	if err != nil {
		t.Fatalf("NewWin32Controller() error = %v", err)
	}
	if err := ctrl.Move(context.Background(), MoveRequest{Point: target.NewNormalizedPoint(0.25, 0.75)}); err != nil {
		t.Fatalf("Move() error = %v", err)
	}
	if in.moveHWND != 42 || in.moveX != 0.25 || in.moveY != 0.75 {
		t.Fatalf("delegate move = hwnd %d (%f,%f)", in.moveHWND, in.moveX, in.moveY)
	}
	records := rec.Records()
	if len(records) != 1 {
		t.Fatalf("trace records len = %d, want 1", len(records))
	}
	got := records[0]
	if got.Action != "move" {
		t.Fatalf("trace action = %q, want move", got.Action)
	}
	if len(got.CoordinateSteps) != 1 {
		t.Fatalf("coordinate steps len = %d, want 1", len(got.CoordinateSteps))
	}
	step := got.CoordinateSteps[0]
	if step.From != target.SpaceNormalized || step.To != target.SpaceWindowClient {
		t.Fatalf("coordinate step spaces = %s -> %s", step.From, step.To)
	}
}

func TestWin32ControllerScrollRecordsCoordinateStep(t *testing.T) {
	in := &fakeWin32Input{}
	rec := automationtrace.NewMemoryRecorder()
	ctrl, err := NewWin32Controller(target.Target{
		ID:   "win32:42",
		Kind: target.KindWin32Window,
		Ref:  target.TargetRef{HWND: 42},
	}, Win32Deps{Input: in, Trace: rec})
	if err != nil {
		t.Fatalf("NewWin32Controller() error = %v", err)
	}
	if err := ctrl.Scroll(context.Background(), ScrollRequest{
		Point:      target.NewNormalizedPoint(0.2, 0.8),
		Notches:    -3,
		Horizontal: true,
	}); err != nil {
		t.Fatalf("Scroll() error = %v", err)
	}
	if in.scrollHWND != 42 || in.scrollX != 0.2 || in.scrollY != 0.8 || in.scrollNotches != -3 || !in.scrollHorizontal {
		t.Fatalf("delegate scroll = hwnd %d (%f,%f) notches %d horizontal %v", in.scrollHWND, in.scrollX, in.scrollY, in.scrollNotches, in.scrollHorizontal)
	}
	records := rec.Records()
	if len(records) != 1 {
		t.Fatalf("trace records len = %d, want 1", len(records))
	}
	got := records[0]
	if got.Action != "scroll" {
		t.Fatalf("trace action = %q, want scroll", got.Action)
	}
	if len(got.CoordinateSteps) != 1 {
		t.Fatalf("coordinate steps len = %d, want 1", len(got.CoordinateSteps))
	}
	step := got.CoordinateSteps[0]
	if step.From != target.SpaceNormalized || step.To != target.SpaceWindowClient {
		t.Fatalf("coordinate step spaces = %s -> %s", step.From, step.To)
	}
}

func TestWin32ControllerMouseDownRecordsCoordinateStep(t *testing.T) {
	in := &fakeWin32Input{}
	rec := automationtrace.NewMemoryRecorder()
	ctrl, err := NewWin32Controller(target.Target{
		ID:   "win32:42",
		Kind: target.KindWin32Window,
		Ref:  target.TargetRef{HWND: 42},
	}, Win32Deps{Input: in, Trace: rec})
	if err != nil {
		t.Fatalf("NewWin32Controller() error = %v", err)
	}
	if err := ctrl.MouseDown(context.Background(), MouseButtonRequest{
		Point:  target.NewNormalizedPoint(0.3, 0.7),
		Button: "right",
	}); err != nil {
		t.Fatalf("MouseDown() error = %v", err)
	}
	if in.mouseDownHWND != 42 || in.mouseDownX != 0.3 || in.mouseDownY != 0.7 || in.mouseDownButton != "right" {
		t.Fatalf("delegate mouseDown = hwnd %d (%f,%f) %s", in.mouseDownHWND, in.mouseDownX, in.mouseDownY, in.mouseDownButton)
	}
	records := rec.Records()
	if len(records) != 1 || records[0].Action != "mouse-down" {
		t.Fatalf("trace records = %#v", records)
	}
	if len(records[0].CoordinateSteps) != 1 {
		t.Fatalf("coordinate steps len = %d, want 1", len(records[0].CoordinateSteps))
	}
}

func TestWin32ControllerMouseUpRecordsTrace(t *testing.T) {
	in := &fakeWin32Input{}
	rec := automationtrace.NewMemoryRecorder()
	ctrl, err := NewWin32Controller(target.Target{
		ID:   "win32:42",
		Kind: target.KindWin32Window,
		Ref:  target.TargetRef{HWND: 42},
	}, Win32Deps{Input: in, Trace: rec})
	if err != nil {
		t.Fatalf("NewWin32Controller() error = %v", err)
	}
	if err := ctrl.MouseUp(context.Background(), MouseButtonRequest{Button: "middle"}); err != nil {
		t.Fatalf("MouseUp() error = %v", err)
	}
	if in.mouseUpHWND != 42 || in.mouseUpButton != "middle" {
		t.Fatalf("delegate mouseUp = hwnd %d %s", in.mouseUpHWND, in.mouseUpButton)
	}
	records := rec.Records()
	if len(records) != 1 || records[0].Action != "mouse-up" {
		t.Fatalf("trace records = %#v", records)
	}
	if len(records[0].CoordinateSteps) != 0 {
		t.Fatalf("coordinate steps len = %d, want 0", len(records[0].CoordinateSteps))
	}
}

func TestWin32ControllerDragRecordsBeginEndCoordinateSteps(t *testing.T) {
	in := &fakeWin32Input{}
	rec := automationtrace.NewMemoryRecorder()
	ctrl, err := NewWin32Controller(target.Target{
		ID:   "win32:42",
		Kind: target.KindWin32Window,
		Ref:  target.TargetRef{HWND: 42},
	}, Win32Deps{Input: in, Trace: rec})
	if err != nil {
		t.Fatalf("NewWin32Controller() error = %v", err)
	}
	if err := ctrl.Drag(context.Background(), DragRequest{
		From:       target.NewNormalizedPoint(0.1, 0.2),
		To:         target.NewNormalizedPoint(0.8, 0.9),
		Button:     "left",
		DurationMs: 300,
	}); err != nil {
		t.Fatalf("Drag() error = %v", err)
	}
	if in.dragHWND != 42 || in.dragX1 != 0.1 || in.dragY1 != 0.2 || in.dragX2 != 0.8 || in.dragY2 != 0.9 || in.dragButton != "left" || in.dragDurationMs != 300 {
		t.Fatalf("delegate drag = hwnd %d (%f,%f)->(%f,%f) %s %d", in.dragHWND, in.dragX1, in.dragY1, in.dragX2, in.dragY2, in.dragButton, in.dragDurationMs)
	}
	records := rec.Records()
	if len(records) != 1 || records[0].Action != "drag" {
		t.Fatalf("trace records = %#v", records)
	}
	if len(records[0].CoordinateSteps) != 2 {
		t.Fatalf("coordinate steps len = %d, want 2", len(records[0].CoordinateSteps))
	}
}

func TestWin32ControllerMoveRelativeRecordsTrace(t *testing.T) {
	in := &fakeWin32Input{}
	rec := automationtrace.NewMemoryRecorder()
	ctrl, err := NewWin32Controller(target.Target{
		ID:   "win32:42",
		Kind: target.KindWin32Window,
		Ref:  target.TargetRef{HWND: 42},
	}, Win32Deps{Input: in, Trace: rec})
	if err != nil {
		t.Fatalf("NewWin32Controller() error = %v", err)
	}
	if err := ctrl.MoveRelative(context.Background(), RelativeMoveRequest{Dx: 10, Dy: -20, DurationMs: 150}); err != nil {
		t.Fatalf("MoveRelative() error = %v", err)
	}
	if in.moveRelHWND != 42 || in.moveRelDx != 10 || in.moveRelDy != -20 || in.moveRelDuration != 150 {
		t.Fatalf("delegate move relative = hwnd %d dx %d dy %d duration %d", in.moveRelHWND, in.moveRelDx, in.moveRelDy, in.moveRelDuration)
	}
	records := rec.Records()
	if len(records) != 1 || records[0].Action != "move-relative" {
		t.Fatalf("trace records = %#v", records)
	}
	if len(records[0].CoordinateSteps) != 0 {
		t.Fatalf("coordinate steps len = %d, want 0", len(records[0].CoordinateSteps))
	}
}

func TestWin32ControllerClickRecordsTrace(t *testing.T) {
	in := &fakeWin32Input{}
	rec := automationtrace.NewMemoryRecorder()
	ctrl, err := NewWin32Controller(target.Target{
		ID:   "win32:42",
		Kind: target.KindWin32Window,
		Ref:  target.TargetRef{HWND: 42},
	}, Win32Deps{Input: in, Trace: rec})
	if err != nil {
		t.Fatalf("NewWin32Controller() error = %v", err)
	}
	if err := ctrl.Click(context.Background(), ClickRequest{Point: target.NewNormalizedPoint(0.1, 0.2)}); err != nil {
		t.Fatalf("Click() error = %v", err)
	}

	records := rec.Records()
	if len(records) != 1 {
		t.Fatalf("trace records len = %d, want 1", len(records))
	}
	got := records[0]
	if got.Action != "click" || got.Target.ID != "win32:42" || got.Backend != "win32" {
		t.Fatalf("unexpected trace record: %#v", got)
	}
	if got.Status != automationtrace.StatusSuccess || got.Error != "" {
		t.Fatalf("trace status/error = %q/%q", got.Status, got.Error)
	}
}

func TestWin32ControllerKeyChordRecordsErrorTrace(t *testing.T) {
	wantErr := errors.New("keyboard denied")
	in := &fakeWin32Input{err: wantErr}
	rec := automationtrace.NewMemoryRecorder()
	ctrl, err := NewWin32Controller(target.Target{
		ID:   "win32:42",
		Kind: target.KindWin32Window,
		Ref:  target.TargetRef{HWND: 42},
	}, Win32Deps{Input: in, Trace: rec, Backend: "sendinput"})
	if err != nil {
		t.Fatalf("NewWin32Controller() error = %v", err)
	}
	if err := ctrl.KeyChord(context.Background(), KeyChordRequest{Keys: []string{"ctrl", "n"}}); !errors.Is(err, wantErr) {
		t.Fatalf("KeyChord() error = %v, want %v", err, wantErr)
	}

	records := rec.Records()
	if len(records) != 1 {
		t.Fatalf("trace records len = %d, want 1", len(records))
	}
	got := records[0]
	if got.Action != "key-chord" || got.Backend != "sendinput" {
		t.Fatalf("unexpected trace record: %#v", got)
	}
	if got.Status != automationtrace.StatusError || got.Error != wantErr.Error() {
		t.Fatalf("trace status/error = %q/%q", got.Status, got.Error)
	}
}
