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
