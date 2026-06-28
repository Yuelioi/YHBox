package input

import (
	"testing"
	"unsafe"
)

func TestNewBackend_SendInput(t *testing.T) {
	b, err := NewBackend("sendinput")
	if err != nil {
		t.Fatalf("NewBackend(sendinput): %v", err)
	}
	if b == nil {
		t.Fatal("nil backend")
	}
	if b.Name() != "sendinput" {
		t.Errorf("Name = %q, want sendinput", b.Name())
	}
	caps := b.Capabilities()
	if caps.BackgroundInput || !caps.RelativeMouse || !caps.GlobalInput {
		t.Errorf("caps = %+v, want {BackgroundInput:false RelativeMouse:true GlobalInput:true}", caps)
	}
}

func TestSendInputBackend_ReleaseAll_ClearsState(t *testing.T) {
	b := newSendInputBackend()
	// 未知 vk (0xFE/0xFF): MapVirtualKey 返 0, SendInput 静默丢 — 不真敲键盘.
	b.heldKeys[0xFF] = struct{}{}
	b.heldKeys[0xFE] = struct{}{}
	if err := b.ReleaseAll(); err != nil {
		t.Fatalf("ReleaseAll: %v", err)
	}
	if len(b.heldKeys) != 0 {
		t.Errorf("heldKeys not cleared: %v", b.heldKeys)
	}
	if len(b.heldBtns) != 0 {
		t.Errorf("heldBtns not cleared: %v", b.heldBtns)
	}
}

func TestSendInputKeyboardBlockSizeMatchesWin32Input(t *testing.T) {
	got := unsafe.Sizeof(sendInputKeyBlock{})
	want := unsafe.Sizeof(sendInputBlock{})
	if got != want {
		t.Fatalf("sendInputKeyBlock size = %d, want %d", got, want)
	}
}
