package tools

import (
	"runtime"
	"syscall"
	"testing"
)

var (
	testUser32           = syscall.NewLazyDLL("user32.dll")
	testRegisterHotKey   = testUser32.NewProc("RegisterHotKey")
	testUnregisterHotKey = testUser32.NewProc("UnregisterHotKey")
)

func TestCancelWin32WindowTargetCapture_NoActive(t *testing.T) {
	captureMu.Lock()
	activeCapture = nil
	captureMu.Unlock()

	if err := cancelWin32WindowTargetCapture("any-id"); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestCancelWin32WindowTargetCapture_WrongID(t *testing.T) {
	captureMu.Lock()
	activeCapture = &captureSession{
		id:     "real-id",
		done:   make(chan struct{}),
		cancel: make(chan struct{}),
	}
	captureMu.Unlock()
	defer func() {
		captureMu.Lock()
		activeCapture = nil
		captureMu.Unlock()
	}()

	if err := cancelWin32WindowTargetCapture("wrong-id"); err != nil {
		t.Fatalf("expected nil error for wrong id, got %v", err)
	}
	captureMu.Lock()
	stillActive := activeCapture != nil
	captureMu.Unlock()
	if !stillActive {
		t.Fatal("active session was cleared by wrong-id cancel")
	}
}

func TestRandID_Unique(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		id := randID()
		if seen[id] {
			t.Fatalf("randID collision after %d iterations: %q", i, id)
		}
		seen[id] = true
	}
}

func TestWindowCaptureStartsWhenAnotherApplicationOwnsTheConfiguredHotkey(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	const (
		ownerID = 0x9011
		mods    = 0x0001 | 0x0002 | 0x0004
		vkF10   = 0x79
	)
	registered, _, err := testRegisterHotKey.Call(0, ownerID, mods, vkF10)
	if registered == 0 {
		t.Fatalf("reserve configured hotkey: %v", err)
	}
	defer testUnregisterHotKey.Call(0, ownerID)

	for attempt := 1; attempt <= 2; attempt++ {
		captureID, err := startWin32WindowTargetCapture(mods, vkF10, nil)
		if err != nil {
			t.Fatalf("start capture attempt %d while configured key is reserved: %v", attempt, err)
		}
		if captureID == "" {
			t.Fatalf("capture ID is empty on attempt %d", attempt)
		}
		if err := cancelWin32WindowTargetCapture(captureID); err != nil {
			t.Fatalf("cancel capture attempt %d: %v", attempt, err)
		}
	}
}
