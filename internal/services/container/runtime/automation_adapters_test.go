package runtime

import (
	"testing"

	"yotta/internal/automation/target"
	"yotta/pkg/winutil"
)

func TestWindowHandleToTarget(t *testing.T) {
	wh := winutil.WindowHandle{
		HWND:        123,
		Title:       "After Effects",
		Class:       "AE_CApplication",
		ProcessName: "AfterFX.exe",
		ClientW:     1920,
		ClientH:     1080,
	}
	tg := windowHandleToTarget(wh)
	if tg.ID != "win32:123" {
		t.Fatalf("target id = %q", tg.ID)
	}
	if tg.Kind != target.KindWin32Window {
		t.Fatalf("target kind = %q", tg.Kind)
	}
	if tg.Ref.HWND != 123 {
		t.Fatalf("hwnd = %d", tg.Ref.HWND)
	}
	if tg.Resolution.W != 1920 || tg.Resolution.H != 1080 {
		t.Fatalf("resolution = %#v", tg.Resolution)
	}
}
