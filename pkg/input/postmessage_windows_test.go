//go:build windows

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

// TypeText 走 PostMessage WM_CHAR (PostText) —— targeted hwnd, 不依赖前台焦点.
// 死 hwnd 下 postMessage 静默失败 (同既有 KeyDown 测试模式); 这里验逐 rune 拆 UTF-16
// (含 BMP 外 surrogate pair) 不 panic + 返回 nil. 实际「后台窗口能收到字符」靠真机 smoke.
func TestPostMessageBackend_TypeText_NoPanic(t *testing.T) {
	b := newPostMessageBackend()
	fakeHwnd := win.HWND(0xDEADBEEF) // 假 hwnd, postMessage 静默失败
	// ASCII + CJK (BMP 内) + emoji (BMP 外, surrogate pair)
	if err := b.TypeText(fakeHwnd, "abc你好😀"); err != nil {
		t.Errorf("TypeText returned err: %v", err)
	}
	// 空串也不能炸
	if err := b.TypeText(fakeHwnd, ""); err != nil {
		t.Errorf("TypeText(empty) returned err: %v", err)
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

func TestNewBackend_Unknown(t *testing.T) {
	if _, err := NewBackend("xxx"); err == nil {
		t.Error("unknown backend should err")
	}
}
