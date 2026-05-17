// recorder_test.go：单测只覆盖不需要真 hwnd + 真键鼠输入的路径。
// 真录制集成阶段手测。
package recording

import (
	"testing"
	"time"

	"yhbox/internal/services/inputclip"
)

func TestVKName(t *testing.T) {
	cases := []struct {
		in   uint32
		want string
	}{
		{'A', "A"},
		{'Z', "Z"},
		{'0', "0"},
		{'9', "9"},
		{0x70, "F1"},
		{0x7B, "F12"},
		{VK_SPACE, "Space"},
		{VK_ESCAPE, "Esc"},
		{VK_RETURN, "Enter"},
		{VK_LCONTROL, "Ctrl"},
		{VK_RCONTROL, "Ctrl"},
		{VK_CONTROL, "Ctrl"},
		{VK_LSHIFT, "Shift"},
		{VK_LMENU, "Alt"},
		{VK_UP, "Up"},
		{VK_DOWN, "Down"},
		{VK_LEFT, "Left"},
		{VK_RIGHT, "Right"},
		{0xFF, ""},
		{0x00, ""},
		{0x6F, ""},
		{0x7C, ""},
	}
	for _, c := range cases {
		if got := vkName(c.in); got != c.want {
			t.Errorf("vkName(0x%X) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestRecorder_NotActiveInitially(t *testing.T) {
	r := NewRecorder()
	if r.Active() {
		t.Error("新建 Recorder 不应是 active")
	}
}

func TestRecorder_StopWhenNotActive(t *testing.T) {
	r := NewRecorder()
	if _, err := r.Stop(); err == nil {
		t.Error("非 active 时 Stop 应返 error")
	}
}

func TestRecorder_CancelWhenNotActive(t *testing.T) {
	r := NewRecorder()
	defer func() {
		if rec := recover(); rec != nil {
			t.Errorf("Cancel 不应 panic, 但 panic 了: %v", rec)
		}
	}()
	r.Cancel()
	if r.Active() {
		t.Error("Cancel 后仍 Active")
	}
}

// TestRecorderAppendOrdering: dumb drain loop 不做 dedupe — 100 个 keydown 全部
// 进 clipEvents (不合并 auto-repeat), seq 单调递增.
//
// 用 mouseMode='relative' 避免 IsPointInsideGameWindow 真 Win32 调用.
func TestRecorderAppendOrdering(t *testing.T) {
	r := &Recorder{
		tStartUs:   uint64(time.Now().UnixMicro()),
		seqCounter: 0,
		meta:       inputclip.ClipMeta{MouseMode: "relative"},
		rawEvents:  make(chan HookEvent, 200),
		drainDone:  make(chan struct{}),
	}
	go r.drainLoop()
	for i := 0; i < 100; i++ {
		r.rawEvents <- HookEvent{IsKeyboard: true, IsKeyDown: true, Vk: 0x57}
	}
	r.rawEvents <- HookEvent{IsKeyboard: true, IsKeyDown: false, Vk: 0x57}
	close(r.rawEvents)
	<-r.drainDone

	r.eventMu.Lock()
	defer r.eventMu.Unlock()
	if len(r.clipEvents) != 101 {
		t.Errorf("got %d events, want 101 (recorder 不能 dedupe auto-repeat)", len(r.clipEvents))
	}
	for i := 1; i < len(r.clipEvents); i++ {
		if r.clipEvents[i].Seq <= r.clipEvents[i-1].Seq {
			t.Errorf("seq not monotonic at %d", i)
		}
	}
}
