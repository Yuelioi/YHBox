package controller

import (
	"context"
	"testing"

	"yotta/internal/automation/target"
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
	clickHWND uintptr
	clickX    float64
	clickY    float64
	keyDown   []string
	keyUp     []string
	text      string
}

func (f *fakeWin32Input) Click(hwnd uintptr, xRatio, yRatio float64, button string, durMs int) error {
	f.clickHWND = hwnd
	f.clickX = xRatio
	f.clickY = yRatio
	return nil
}

func (f *fakeWin32Input) KeyDown(hwnd uintptr, key string) error {
	f.keyDown = append(f.keyDown, key)
	return nil
}

func (f *fakeWin32Input) KeyUp(hwnd uintptr, key string) error {
	f.keyUp = append(f.keyUp, key)
	return nil
}

func (f *fakeWin32Input) TypeText(hwnd uintptr, text string) error {
	f.text = text
	return nil
}

func (f *fakeWin32Input) MoveTo(hwnd uintptr, xRatio, yRatio float64) error { return nil }

func (f *fakeWin32Input) Scroll(hwnd uintptr, xRatio, yRatio float64, notches int, horizontal bool) error {
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
