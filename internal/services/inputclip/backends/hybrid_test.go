//go:build windows

package backends

import (
	"testing"

	"yotta/internal/services/inputclip"
)

func TestHybridBackend_Name(t *testing.T) {
	b := NewHybridBackend(nil)
	name := b.Name()
	if name != "postmessage+sendinput" {
		t.Fatalf("unexpected name: %q", name)
	}
}

func TestHybridBackend_Smoke(t *testing.T) {
	b := NewHybridBackend(nil)
	// 不真敲键盘 — 用 ReleaseHeld 验状态机.
	if err := b.ReleaseHeld(); err != nil {
		t.Fatalf("empty ReleaseHeld should be no-op, got: %v", err)
	}
	// SendBatch 空切片不应报错
	if err := b.SendBatch(nil); err != nil {
		t.Fatalf("empty SendBatch err: %v", err)
	}
	// 未知 EventType 应静默
	if err := b.Send(inputclip.Event{Type: 0}); err != nil {
		t.Fatalf("unknown EventType should noop, got: %v", err)
	}
}

func TestHybridBackend_ReleaseHeld_ClearsState(t *testing.T) {
	b := NewHybridBackend(nil).(*HybridBackend)
	// 直接塞 held 状态. 用不存在的 vk (0xFE/0xFF) 避免测试真敲键盘 —
	// MapVirtualKey 对未知 vk 返 0, sendInput 静默丢. 鼠标按钮不模拟,
	// 否则 ReleaseHeld 会真发 LeftUp 影响测试机.
	b.heldKeys[0xFF] = struct{}{}
	b.heldKeys[0xFE] = struct{}{}
	if err := b.ReleaseHeld(); err != nil {
		t.Fatalf("ReleaseHeld err: %v", err)
	}
	if len(b.heldKeys) != 0 {
		t.Fatalf("heldKeys should be empty, got %v", b.heldKeys)
	}
	if len(b.heldButtons) != 0 {
		t.Fatalf("heldButtons should be empty, got %v", b.heldButtons)
	}
}
