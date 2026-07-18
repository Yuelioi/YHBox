package winutil

import (
	"os"
	"testing"
)

func TestInspectForegroundWindowState(t *testing.T) {
	if os.Getenv("YOTTA_WINDOWS_NATIVE_SMOKE") != "1" {
		t.Skip("set YOTTA_WINDOWS_NATIVE_SMOKE=1 to run desktop window smoke")
	}
	hwnd := ForegroundWindow()
	if hwnd == 0 {
		t.Fatal("native window-state feedback loop requires a foreground window")
	}
	state, err := InspectWindowState(hwnd)
	if err != nil {
		t.Fatal(err)
	}
	if state.Width <= 0 || state.Height <= 0 || !state.Foreground ||
		(state.State != "normal" && state.State != "minimized" && state.State != "maximized") {
		t.Fatalf("foreground window state = %#v", state)
	}
}
