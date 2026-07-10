// recorder_test.go：单测只覆盖不需要真 hwnd + 真键鼠输入的路径。
// 真录制集成阶段手测。
package recording

import (
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/services/inputclip"
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

// TestRecorder_PausedDropsEvents: paused 期到达的事件在 drain 点被丢弃 (全 0 落盘).
// 预填 + 关闭 channel 后同步 drain, paused 全程 true → 不 append 任何 event.
func TestRecorder_PausedDropsEvents(t *testing.T) {
	r := &Recorder{
		tStartUs:  uint64(time.Now().UnixMicro()),
		meta:      inputclip.ClipMeta{MouseMode: "relative"},
		rawEvents: make(chan HookEvent, 10),
		drainDone: make(chan struct{}),
	}
	r.paused.Store(true)
	for i := 0; i < 5; i++ {
		r.rawEvents <- HookEvent{IsKeyboard: true, IsKeyDown: true, Vk: 0x57}
	}
	close(r.rawEvents)
	r.drainLoop() // 同步: channel 已关, 处理完所有 (全丢) 后返回

	r.eventMu.Lock()
	defer r.eventMu.Unlock()
	if len(r.clipEvents) != 0 {
		t.Errorf("paused 期事件应全丢, got %d", len(r.clipEvents))
	}
}

// TestRecorder_TimestampSubtractsPausedAccum: 事件时间戳扣除累计暂停时长 → 切除间隔.
// 造 tStartUs = 100ms 前 + accum = 50ms → 事件 TUs ≈ 50ms (真实录制时长, 不含暂停段).
func TestRecorder_TimestampSubtractsPausedAccum(t *testing.T) {
	r := &Recorder{
		tStartUs:  uint64(time.Now().UnixMicro()) - 100_000, // 录制开始于 100ms 前
		meta:      inputclip.ClipMeta{MouseMode: "relative"},
		rawEvents: make(chan HookEvent, 2),
		drainDone: make(chan struct{}),
	}
	r.pausedAccumUs.Store(50_000) // 已累计暂停 50ms
	r.rawEvents <- HookEvent{IsKeyboard: true, IsKeyDown: true, Vk: 0x57}
	close(r.rawEvents)
	r.drainLoop()

	r.eventMu.Lock()
	defer r.eventMu.Unlock()
	if len(r.clipEvents) != 1 {
		t.Fatalf("want 1 event, got %d", len(r.clipEvents))
	}
	tus := r.clipEvents[0].TUs
	if tus < 45_000 || tus > 70_000 {
		t.Errorf("TUs = %d us, want ≈50000 (wall 100ms - paused 50ms); drain jitter 容差内", tus)
	}
}

// TestRecorder_ResumeAccumulatesPauseDuration: Pause→sleep→Resume 把暂停时长累加进 accum.
func TestRecorder_ResumeAccumulatesPauseDuration(t *testing.T) {
	r := &Recorder{tStartUs: uint64(time.Now().UnixMicro())}
	r.active = true
	r.Pause()
	if !r.paused.Load() {
		t.Fatal("Pause 后应 paused")
	}
	time.Sleep(30 * time.Millisecond)
	r.Resume()
	if r.paused.Load() {
		t.Fatal("Resume 后不应 paused")
	}
	accum := r.pausedAccumUs.Load()
	if accum < 20_000 || accum > 100_000 {
		t.Errorf("accum = %d us, want ≈30000 (30ms 暂停); 容差内", accum)
	}
}

// TestRecorder_PauseResumeGuards: 非 active / 未暂停 / 重复暂停 全幂等无副作用.
func TestRecorder_PauseResumeGuards(t *testing.T) {
	r := &Recorder{}
	r.Pause() // 非 active
	if r.paused.Load() {
		t.Error("非 active Pause 不应置 paused")
	}
	r.active = true
	r.Resume() // 未暂停
	if r.pausedAccumUs.Load() != 0 {
		t.Error("未暂停 Resume 不应累加 accum")
	}
	r.Pause()
	first := r.pauseStartUs
	time.Sleep(2 * time.Millisecond)
	r.Pause() // 重复 Pause
	if r.pauseStartUs != first {
		t.Error("重复 Pause 不应覆盖 pauseStartUs")
	}
}
