package input

import (
	"testing"

	"github.com/lxn/win"
)

func TestPostMessageBackend_NameAndCapabilities(t *testing.T) {
	b := newPostMessageBackend()
	if b.Name() != "postmessage" {
		t.Errorf("Name() = %q, want postmessage", b.Name())
	}
	caps := b.Capabilities()
	if !caps.BackgroundInput {
		t.Error("PostMessage should support BackgroundInput")
	}
	if caps.GlobalInput {
		t.Error("PostMessage should NOT be global input")
	}
}

func TestPostMessageBackend_ReleaseAllEmpty(t *testing.T) {
	b := newPostMessageBackend()
	// 没 down 过任何东西, ReleaseAll 不能 panic
	if err := b.ReleaseAll(); err != nil {
		t.Errorf("empty ReleaseAll should not error: %v", err)
	}
}

func TestPostMessageBackend_StateTracking(t *testing.T) {
	b := newPostMessageBackend()
	fakeHwnd := win.HWND(0xDEADBEEF) // 假 hwnd
	_ = b.KeyDown(fakeHwnd, "W")
	b.mu.Lock()
	_, hasW := b.heldKeys["W"]
	b.mu.Unlock()
	if !hasW {
		t.Error("KeyDown should track W in heldKeys")
	}
	_ = b.KeyUp(fakeHwnd, "W")
	b.mu.Lock()
	_, stillHasW := b.heldKeys["W"]
	b.mu.Unlock()
	if stillHasW {
		t.Error("KeyUp should remove W from heldKeys")
	}
}

func TestNewBackend_PostMessage(t *testing.T) {
	b, err := NewBackend("postmessage")
	if err != nil {
		t.Fatalf("NewBackend postmessage err: %v", err)
	}
	if b.Name() != "postmessage" {
		t.Errorf("got %q", b.Name())
	}
}

func TestNewBackend_EmptyName(t *testing.T) {
	b, err := NewBackend("")
	if err != nil {
		t.Fatalf("NewBackend with empty name should default to postmessage, got err: %v", err)
	}
	if b.Name() != "postmessage" {
		t.Errorf("empty name should default to postmessage, got %q", b.Name())
	}
}

func TestNewBackend_Unknown(t *testing.T) {
	if _, err := NewBackend("xxx"); err == nil {
		t.Error("unknown backend should err")
	}
}
