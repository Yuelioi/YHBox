package runtime

import (
	"context"
	"sync"
	"testing"
	"time"

	"yhbox/internal/services/inputclip"
	"yhbox/internal/services/inputclip/backends"
)

// captureBackend mock 后端: 记录 Send 顺序 + 调用时点 (QPC us).
type captureBackend struct {
	mu        sync.Mutex
	sent      []inputclip.Event
	sentTimes []uint64
	released  int
}

func (c *captureBackend) Send(ev inputclip.Event) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sent = append(c.sent, ev)
	c.sentTimes = append(c.sentTimes, QPCMicros())
	return nil
}

func (c *captureBackend) SendBatch(evs []inputclip.Event) error {
	for _, e := range evs {
		if err := c.Send(e); err != nil {
			return err
		}
	}
	return nil
}

func (c *captureBackend) ReleaseHeld() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.released++
	return nil
}

func (c *captureBackend) Name() string { return "capture" }

func (c *captureBackend) snapshot() ([]inputclip.Event, []uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	evs := make([]inputclip.Event, len(c.sent))
	copy(evs, c.sent)
	ts := make([]uint64, len(c.sentTimes))
	copy(ts, c.sentTimes)
	return evs, ts
}

var _ backends.IInputBackend = (*captureBackend)(nil)

// 时间 tolerance: QPC stub 路径 (跨平台 CI) 用 time.Now() 精度差, 给 +/-15ms 余量.
const timeToleranceUs uint64 = 15_000

func TestClipPlayer_PlayWithoutRanges_PlaysAllEvents(t *testing.T) {
	clip := &inputclip.InputClip{
		ID:         "c1",
		DurationUs: 100_000,
		Events: []inputclip.Event{
			{TUs: 0, Type: inputclip.EventTypeKeyDown, A: 0x41},
			{TUs: 50_000, Type: inputclip.EventTypeKeyUp, A: 0x41},
			{TUs: 100_000, Type: inputclip.EventTypeKeyDown, A: 0x42},
		},
	}
	cap := &captureBackend{}
	p := NewClipPlayer(clip, nil, cap, DefaultPlaybackPolicy(), 0, nil)
	p.Start(context.Background())
	if err := p.Wait(); err != nil {
		t.Fatalf("Wait: %v", err)
	}
	evs, _ := cap.snapshot()
	if len(evs) != 3 {
		t.Fatalf("expected 3 sent, got %d", len(evs))
	}
	if evs[0].A != 0x41 || evs[1].A != 0x41 || evs[2].A != 0x42 {
		t.Fatalf("event order wrong: %+v", evs)
	}
}

func TestClipPlayer_PlayWithRanges_SkipsCutRegion(t *testing.T) {
	// 5 events at 0/30/60/90/120 ms.
	// ranges: keep [0,30] + [90,120] → 期望发送 events @ TUs 0, 30, 90, 120 (4 个).
	// 关键时间检验: ev[90] wallclock = startQPC + (90 - 60_cut) ms = startQPC + 30ms.
	clip := &inputclip.InputClip{
		ID:         "c2",
		DurationUs: 120_000,
		Events: []inputclip.Event{
			{TUs: 0, Type: inputclip.EventTypeKeyDown, A: 1},
			{TUs: 30_000, Type: inputclip.EventTypeKeyDown, A: 2},
			{TUs: 60_000, Type: inputclip.EventTypeKeyDown, A: 3},
			{TUs: 90_000, Type: inputclip.EventTypeKeyDown, A: 4},
			{TUs: 120_000, Type: inputclip.EventTypeKeyDown, A: 5},
		},
	}
	ranges := []inputclip.Range{
		{FromUs: 0, ToUs: 30_000},
		{FromUs: 90_000, ToUs: 120_000},
	}
	cap := &captureBackend{}
	p := NewClipPlayer(clip, ranges, cap, DefaultPlaybackPolicy(), 0, nil)
	startQPC := QPCMicros()
	p.Start(context.Background())
	if err := p.Wait(); err != nil {
		t.Fatalf("Wait: %v", err)
	}
	evs, ts := cap.snapshot()
	if len(evs) != 4 {
		t.Fatalf("expected 4 sent, got %d: %+v", len(evs), evs)
	}
	if evs[0].A != 1 || evs[1].A != 2 || evs[2].A != 4 || evs[3].A != 5 {
		t.Fatalf("event order wrong (should drop A=3): %+v", evs)
	}

	// ev[4] @ TUs=90_000 → effective = 90_000 - 60_000_cut = 30_000us, wallclock ≈ startQPC + 30ms.
	// ev[5] @ TUs=120_000 → effective = 60_000us.
	expects := []uint64{0, 30_000, 30_000, 60_000}
	for i, exp := range expects {
		actual := ts[i] - startQPC
		diff := int64(actual) - int64(exp)
		if diff < 0 {
			diff = -diff
		}
		if uint64(diff) > timeToleranceUs {
			t.Errorf("ev[%d] wallclock offset = %dus, expected %dus +/- %dus (diff=%d)",
				i, actual, exp, timeToleranceUs, diff)
		}
	}
}

func TestClipPlayer_PlayCancelMidway(t *testing.T) {
	// 100 events 跨 1s (每 10ms 一个). ctx 200ms 取消. 应收到约 20 events.
	var evs []inputclip.Event
	for i := 0; i < 100; i++ {
		evs = append(evs, inputclip.Event{
			TUs:  uint64(i) * 10_000,
			Type: inputclip.EventTypeKeyDown,
			A:    int32(i),
		})
	}
	clip := &inputclip.InputClip{ID: "c3", DurationUs: 990_000, Events: evs}

	cap := &captureBackend{}
	p := NewClipPlayer(clip, nil, cap, DefaultPlaybackPolicy(), 0, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	p.Start(ctx)

	err := p.Wait()
	if err == nil {
		t.Fatal("expected cancel error, got nil")
	}
	sent, _ := cap.snapshot()
	if len(sent) >= 100 {
		t.Fatalf("cancel was ignored, all %d events sent", len(sent))
	}
	// 容差: 10-40 区间内, 不严格 (CI 波动).
	if len(sent) < 5 || len(sent) > 50 {
		t.Errorf("expected ~20 events sent, got %d", len(sent))
	}
	if cap.released != 1 {
		t.Errorf("expected ReleaseHeld called exactly once, got %d", cap.released)
	}
}

func TestClipPlayer_PlayRawDeltaScales(t *testing.T) {
	clip := &inputclip.InputClip{
		ID:         "c4",
		DurationUs: 0,
		Meta:       inputclip.ClipMeta{MouseCounts360: 800},
		Events: []inputclip.Event{
			{TUs: 0, Type: inputclip.EventTypeRawDelta, B: 100, C: 50},
		},
	}
	cap := &captureBackend{}
	p := NewClipPlayer(clip, nil, cap, DefaultPlaybackPolicy(), 1600, nil) // factor = 2.0
	p.Start(context.Background())
	if err := p.Wait(); err != nil {
		t.Fatalf("Wait: %v", err)
	}
	evs, _ := cap.snapshot()
	if len(evs) != 1 {
		t.Fatalf("expected 1 sent, got %d", len(evs))
	}
	if evs[0].B != 200 || evs[0].C != 100 {
		t.Fatalf("scale failed: expected B=200 C=100, got B=%d C=%d", evs[0].B, evs[0].C)
	}
}

func TestClipPlayer_PlayRawDelta_NoScaleWhenSourceZero(t *testing.T) {
	clip := &inputclip.InputClip{
		ID:         "c5",
		Meta:       inputclip.ClipMeta{MouseCounts360: 0},
		DurationUs: 0,
		Events: []inputclip.Event{
			{TUs: 0, Type: inputclip.EventTypeRawDelta, B: 100, C: 50},
		},
	}
	cap := &captureBackend{}
	p := NewClipPlayer(clip, nil, cap, DefaultPlaybackPolicy(), 1600, nil)
	p.Start(context.Background())
	if err := p.Wait(); err != nil {
		t.Fatalf("Wait: %v", err)
	}
	evs, _ := cap.snapshot()
	if evs[0].B != 100 || evs[0].C != 50 {
		t.Fatalf("source=0 should skip scaling, got B=%d C=%d", evs[0].B, evs[0].C)
	}
}

func TestClipPlayer_PlayRawDelta_NoScaleWhenEqual(t *testing.T) {
	clip := &inputclip.InputClip{
		Meta: inputclip.ClipMeta{MouseCounts360: 1600},
		Events: []inputclip.Event{
			{TUs: 0, Type: inputclip.EventTypeRawDelta, B: 100, C: 50},
		},
	}
	cap := &captureBackend{}
	p := NewClipPlayer(clip, nil, cap, DefaultPlaybackPolicy(), 1600, nil)
	p.Start(context.Background())
	if err := p.Wait(); err != nil {
		t.Fatalf("Wait: %v", err)
	}
	evs, _ := cap.snapshot()
	if evs[0].B != 100 || evs[0].C != 50 {
		t.Fatalf("equal source/target should skip scaling, got B=%d C=%d", evs[0].B, evs[0].C)
	}
}

func TestClipPlayer_EmptyClip(t *testing.T) {
	clip := &inputclip.InputClip{ID: "empty", Events: nil}
	cap := &captureBackend{}
	p := NewClipPlayer(clip, nil, cap, DefaultPlaybackPolicy(), 0, nil)
	p.Start(context.Background())
	if err := p.Wait(); err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if cap.released != 1 {
		t.Errorf("ReleaseHeld 应仍被调一次 (defer 兜底), 实际 %d", cap.released)
	}
}

func TestClipPlayer_ExplicitCancelReleasesHeld(t *testing.T) {
	var evs []inputclip.Event
	for i := 0; i < 50; i++ {
		evs = append(evs, inputclip.Event{TUs: uint64(i) * 20_000, Type: inputclip.EventTypeKeyDown})
	}
	clip := &inputclip.InputClip{Events: evs, DurationUs: 980_000}
	cap := &captureBackend{}
	p := NewClipPlayer(clip, nil, cap, DefaultPlaybackPolicy(), 0, nil)
	p.Start(context.Background())
	time.Sleep(80 * time.Millisecond)
	p.Cancel()
	err := p.Wait()
	if err != ErrCancelled {
		t.Fatalf("expected ErrCancelled, got %v", err)
	}
	if cap.released != 1 {
		t.Errorf("expected ReleaseHeld == 1, got %d", cap.released)
	}
	// 再调 Cancel 不 panic (幂等).
	p.Cancel()
}

// 录制时 1920x1080, 中心点击 (960, 540), 回放 960x540 应缩放到 (480, 270).
func TestPlayback_AbsoluteCoordScaling(t *testing.T) {
	clip := &inputclip.InputClip{
		ID:         "scale1",
		DurationUs: 0,
		Meta:       inputclip.ClipMeta{BaseResolution: [2]int{1920, 1080}},
		Events: []inputclip.Event{
			{TUs: 0, Type: inputclip.EventTypeMouseBtnDown, A: 0, B: 960, C: 540},
		},
	}
	cap := &captureBackend{}
	p := NewClipPlayer(clip, nil, cap, DefaultPlaybackPolicy(), 0,
		func() (int, int) { return 960, 540 },
	)
	p.Start(context.Background())
	if err := p.Wait(); err != nil {
		t.Fatalf("Wait: %v", err)
	}
	evs, _ := cap.snapshot()
	if len(evs) != 1 {
		t.Fatalf("expected 1 sent, got %d", len(evs))
	}
	if evs[0].B != 480 || evs[0].C != 270 {
		t.Fatalf("coord scaling failed: expected B=480 C=270, got B=%d C=%d", evs[0].B, evs[0].C)
	}
}

// BaseResolution = (0,0) → 不缩放, 原样透传.
func TestPlayback_NoScaleWhenBaseZero(t *testing.T) {
	clip := &inputclip.InputClip{
		ID:         "scale2",
		DurationUs: 0,
		Meta:       inputclip.ClipMeta{BaseResolution: [2]int{0, 0}},
		Events: []inputclip.Event{
			{TUs: 0, Type: inputclip.EventTypeMouseBtnDown, A: 0, B: 100, C: 100},
		},
	}
	cap := &captureBackend{}
	p := NewClipPlayer(clip, nil, cap, DefaultPlaybackPolicy(), 0,
		func() (int, int) { return 960, 540 },
	)
	p.Start(context.Background())
	if err := p.Wait(); err != nil {
		t.Fatalf("Wait: %v", err)
	}
	evs, _ := cap.snapshot()
	if evs[0].B != 100 || evs[0].C != 100 {
		t.Fatalf("base=0 should skip scaling, got B=%d C=%d", evs[0].B, evs[0].C)
	}
}
