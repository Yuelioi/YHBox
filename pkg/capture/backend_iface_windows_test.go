//go:build windows

package capture

import (
	"testing"

	"github.com/lxn/win"
)

func TestNewIBackend_GDI(t *testing.T) {
	b, warning, err := NewIBackend("gdi")
	if err != nil {
		t.Fatalf("gdi NewIBackend err: %v", err)
	}
	if warning != "" {
		t.Errorf("gdi explicit should have no warning, got %q", warning)
	}
	if b.Name() != "gdi" {
		t.Errorf("name = %q, want gdi", b.Name())
	}
}

func TestNewIBackend_Auto(t *testing.T) {
	b, _, err := NewIBackend("auto")
	if err != nil {
		t.Fatalf("auto NewIBackend err: %v", err)
	}
	// auto 在 Win10 1903+ 应返 wgc, 否则 gdi. 不强测哪个 — 只测不报错 + 名字合理.
	if b.Name() != "wgc" && b.Name() != "gdi" {
		t.Errorf("auto name = %q, want wgc or gdi", b.Name())
	}
}

func TestNewIBackend_Mock(t *testing.T) {
	b, _, err := NewIBackend("mock")
	if err != nil {
		// mock 可能因没 mock dir 失败 — 测试环境下接受
		t.Skipf("mock init failed (expected in test env without mock dir): %v", err)
	}
	if b.Name() != "mock" {
		t.Errorf("name = %q, want mock", b.Name())
	}
}

func TestNewIBackend_Unknown(t *testing.T) {
	if _, _, err := NewIBackend("xxx"); err == nil {
		t.Error("unknown backend should err")
	}
}

func TestGDIBackend_InvalidHwnd(t *testing.T) {
	b, _, _ := NewIBackend("gdi")
	_, err := b.Frame(win.HWND(0xDEADBEEF))
	if err == nil {
		t.Error("invalid hwnd should err, not panic or return nil img with nil err")
	}
}

func TestGDIBackend_ClientSize_InvalidHwnd(t *testing.T) {
	b, _, _ := NewIBackend("gdi")
	_, _, err := b.ClientSize(win.HWND(0xDEADBEEF))
	if err == nil {
		t.Error("invalid hwnd ClientSize should err")
	}
}

func TestMockBackend_ClientSize_InvalidHwnd(t *testing.T) {
	backend := &mockBackend{}
	_, _, err := backend.ClientSize(win.HWND(0xDEADBEEF))
	if err == nil {
		t.Fatal("invalid hwnd ClientSize should return an error on Windows")
	}
}
