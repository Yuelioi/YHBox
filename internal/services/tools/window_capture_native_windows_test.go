package tools

import (
	"os"
	"testing"
	"time"
	"unsafe"

	inputdriver "github.com/yottaapp/yotta/pkg/input"
	"github.com/yottaapp/yotta/pkg/winutil"
)

func TestWindowCaptureReturnsExactForegroundMetadata(t *testing.T) {
	if os.Getenv("YOTTA_WINDOWS_NATIVE_SMOKE") != "1" {
		t.Skip("set YOTTA_WINDOWS_NATIVE_SMOKE=1 to run desktop input smoke")
	}
	const vkF10 = 0x79
	hwnd := winutil.ForegroundWindow()
	if hwnd == 0 {
		t.Fatal("native capture feedback loop requires a foreground window")
	}
	wantWindow, err := winutil.WindowMetadata(hwnd)
	if err != nil {
		t.Fatalf("inspect foreground window: %v", err)
	}
	wantExecutable, err := winutil.WindowExecutable(hwnd)
	if err != nil {
		t.Fatalf("inspect foreground executable: %v", err)
	}

	emitted := make(chan map[string]any, 1)
	captureID, err := startWin32WindowTargetCapture(0, vkF10, func(name string, data any) {
		if name != "win32windowtarget:captured" {
			return
		}
		payload, ok := data.(map[string]any)
		if ok {
			emitted <- payload
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	defer cancelWin32WindowTargetCapture(captureID)
	captureMu.Lock()
	session := activeCapture
	captureMu.Unlock()
	if session == nil {
		t.Fatal("capture session disappeared before input")
	}
	select {
	case <-session.done:
		t.Fatal("capture worker exited before input")
	default:
	}

	inputBackend, err := inputdriver.NewBackend("sendinput")
	if err != nil {
		t.Fatal(err)
	}
	defer inputBackend.Close()
	defer inputBackend.ReleaseAll()
	if err := inputBackend.KeyDownCode(0, vkF10); err != nil {
		t.Fatalf("inject capture key down: %v", err)
	}
	if err := inputBackend.KeyUpCode(0, vkF10); err != nil {
		t.Fatalf("inject capture key up: %v", err)
	}

	select {
	case payload := <-emitted:
		if payload["error"] != nil {
			t.Fatalf("capture error = %v", payload["error"])
		}
		if title, _ := payload["title"].(string); title != wantWindow.Title {
			t.Fatalf("captured title = %q, want exact %q", title, wantWindow.Title)
		}
		if class, _ := payload["class"].(string); class != wantWindow.Class {
			t.Fatalf("captured class = %q, want exact %q", class, wantWindow.Class)
		}
		if executable, _ := payload["executable"].(string); executable != wantExecutable {
			t.Fatalf("captured executable = %q, want exact %q", executable, wantExecutable)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("capture key did not produce foreground window metadata")
	}
}

func TestCaptureKeyboardProcQueuesExactForegroundWindow(t *testing.T) {
	if os.Getenv("YOTTA_WINDOWS_NATIVE_SMOKE") != "1" {
		t.Skip("set YOTTA_WINDOWS_NATIVE_SMOKE=1 to run desktop input smoke")
	}
	mods := uint32(0)
	if keyPressed(vkControl) {
		mods |= modControl
	}
	if keyPressed(vkShift) {
		mods |= modShift
	}
	if keyPressed(vkMenu) {
		mods |= modAlt
	}
	session := &captureSession{
		hotkeyMods: mods,
		hotkeyVK:   0x79,
		window:     make(chan uintptr, 1),
	}
	previous := hookSession.Swap(session)
	defer hookSession.Store(previous)
	event := captureKeyboardEvent{VKCode: session.hotkeyVK}
	if result := captureKeyboardProc(hcAction, wmKeyDown, uintptr(unsafe.Pointer(&event))); result != 1 {
		t.Fatalf("hook callback result = %d, want suppressed input", result)
	}
	if !session.fired.Load() {
		t.Fatal("hook callback did not claim matching key")
	}
	select {
	case hwnd := <-session.window:
		if hwnd == 0 || hwnd != winutil.ForegroundWindow() {
			t.Fatalf("queued foreground HWND = %d", hwnd)
		}
	default:
		t.Fatal("hook callback did not queue foreground HWND")
	}
}
