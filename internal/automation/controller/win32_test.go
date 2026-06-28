package controller

import (
	"context"
	"errors"
	"testing"

	"yotta/internal/automation/target"
	automationtrace "yotta/internal/automation/trace"
)

func TestWin32ControllerTargetAndCapabilities(t *testing.T) {
	tg := target.Target{
		ID:          "win32:42",
		Kind:        target.KindWin32Window,
		DisplayName: "Test Window",
		Ref:         target.TargetRef{HWND: 42},
		Resolution:  target.Size{W: 1280, H: 720},
	}
	ctrl, err := NewWin32Controller(tg, Win32Deps{})
	if err != nil {
		t.Fatalf("NewWin32Controller() error = %v", err)
	}
	if got := ctrl.Target(); got.ID != tg.ID {
		t.Fatalf("target id = %q, want %q", got.ID, tg.ID)
	}
	caps := ctrl.Capabilities(context.Background())
	if !caps.Screenshot || !caps.Click || !caps.KeyState || !caps.Text {
		t.Fatalf("unexpected caps: %#v", caps)
	}
	if caps.StartApp || caps.StopApp {
		t.Fatalf("win32 phase1 should not expose app lifecycle: %#v", caps)
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
	text             string
	err              error
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
