package target

import "testing"

func TestTargetIsValid(t *testing.T) {
	t.Run("valid win32 target", func(t *testing.T) {
		tg := Target{
			ID:          "win32:100",
			Kind:        KindWin32Window,
			DisplayName: "After Effects",
			Ref:         TargetRef{HWND: 100},
			Resolution:  Size{W: 1920, H: 1080},
		}
		if err := tg.Validate(); err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
	})

	t.Run("missing id", func(t *testing.T) {
		tg := Target{Kind: KindWin32Window, Ref: TargetRef{HWND: 100}}
		if err := tg.Validate(); err == nil {
			t.Fatalf("Validate() expected error")
		}
	})

	t.Run("win32 target requires hwnd", func(t *testing.T) {
		tg := Target{ID: "win32:0", Kind: KindWin32Window}
		if err := tg.Validate(); err == nil {
			t.Fatalf("Validate() expected error")
		}
	})
}

func TestPointSpaceDefaults(t *testing.T) {
	p := NewNormalizedPoint(0.25, 0.75)
	if p.Space != SpaceNormalized {
		t.Fatalf("space = %q, want %q", p.Space, SpaceNormalized)
	}
	if p.X != 0.25 || p.Y != 0.75 {
		t.Fatalf("point = %#v", p)
	}
}
