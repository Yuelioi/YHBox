package runtime

import (
	"testing"
)

func TestSafeInputBackend_WarnOnceOnInvalidHwnd(t *testing.T) {
	rt := &RuntimeContext{} // Emit nil 也 OK
	inner := &fakeInputBackend{}
	safe := NewSafeInputBackend(inner, rt)
	// hwnd=0 → invalid → 5 次 Click 都 skip inner, warned flag 翻 true
	for i := 0; i < 5; i++ {
		_ = safe.Click(0, 0.5, 0.5, "left", 50)
	}
	if inner.clicks != 0 {
		t.Errorf("invalid hwnd should skip inner, got %d clicks", inner.clicks)
	}
	if !safe.warned {
		t.Error("warned flag should be true after first invalid hwnd")
	}
}

func TestSafeInputBackend_PassThroughForValidHwnd(t *testing.T) {
	// 真 hwnd 测不到 (需要真窗口). 这里 skip — 集成测覆盖.
	t.Skip("passthrough requires real window, covered by integration test")
}

func TestUintToHex(t *testing.T) {
	cases := []struct {
		in   uintptr
		want string
	}{
		{0, "0x0"},
		{1, "0x1"},
		{15, "0xf"},
		{16, "0x10"},
		{255, "0xff"},
		{0xDEADBEEF, "0xdeadbeef"},
	}
	for _, c := range cases {
		if got := uintToHex(c.in); got != c.want {
			t.Errorf("uintToHex(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}
