package actiontransform

import (
	"testing"

	"yhbox/internal/services/actions"
)

func TestPairEvents_KeyDownUp(t *testing.T) {
	evs := []Event{
		{TimestampMs: 1000, Type: EventKeyboard, Vk: "F", IsDown: true},
		{TimestampMs: 1080, Type: EventKeyboard, Vk: "F", IsDown: false},
	}
	steps := PairEvents(evs, 1920, 1080)
	if len(steps) != 1 {
		t.Fatalf("len=%d want 1: %+v", len(steps), steps)
	}
	if steps[0].Kind != actions.StepKey {
		t.Errorf("kind=%q want %q", steps[0].Kind, actions.StepKey)
	}
	if steps[0].Vk != "F" {
		t.Errorf("vk=%q want F", steps[0].Vk)
	}
	if steps[0].DurationMs != 80 {
		t.Errorf("dur=%d want 80", steps[0].DurationMs)
	}
}

func TestPairEvents_KeyOnlyDown(t *testing.T) {
	evs := []Event{
		{TimestampMs: 1000, Type: EventKeyboard, Vk: "Shift", IsDown: true},
	}
	steps := PairEvents(evs, 1920, 1080)
	if len(steps) != 1 {
		t.Fatalf("len=%d want 1: %+v", len(steps), steps)
	}
	if steps[0].Kind != actions.StepKeyDown {
		t.Errorf("kind=%q want %q", steps[0].Kind, actions.StepKeyDown)
	}
	if steps[0].Vk != "Shift" {
		t.Errorf("vk=%q want Shift", steps[0].Vk)
	}
}

func TestPairEvents_KeyOnlyUp(t *testing.T) {
	evs := []Event{
		{TimestampMs: 1000, Type: EventKeyboard, Vk: "Shift", IsDown: false},
	}
	steps := PairEvents(evs, 1920, 1080)
	if len(steps) != 0 {
		t.Errorf("len=%d want 0 (orphan up should be dropped): %+v", len(steps), steps)
	}
}

func TestPairEvents_KeyAutoRepeatMerged(t *testing.T) {
	// 用户按住 W 不松手，Windows auto-repeat 发出多个 KEYDOWN（间隔 ~30ms），
	// 最终一个 KEYUP。recorder 全部捕获。
	// 旧实现：每个 down push 栈 → up pop 一个 → 剩余孤立 down 产 N 个 key_down step（错）。
	// 新实现：auto-repeat 被忽略，合并成单一 key 从第一个 down 到 up（200ms hold）。
	evs := []Event{
		{TimestampMs: 1000, Type: EventKeyboard, Vk: "W", IsDown: true},
		{TimestampMs: 1030, Type: EventKeyboard, Vk: "W", IsDown: true},
		{TimestampMs: 1060, Type: EventKeyboard, Vk: "W", IsDown: true},
		{TimestampMs: 1090, Type: EventKeyboard, Vk: "W", IsDown: true},
		{TimestampMs: 1120, Type: EventKeyboard, Vk: "W", IsDown: true},
		{TimestampMs: 1200, Type: EventKeyboard, Vk: "W", IsDown: false},
	}
	steps := PairEvents(evs, 1920, 1080)
	if len(steps) != 1 {
		t.Fatalf("len=%d want 1 (auto-repeat 应合并): %+v", len(steps), steps)
	}
	if steps[0].Kind != actions.StepKey || steps[0].Vk != "W" || steps[0].DurationMs != 200 {
		t.Errorf("step0=%+v want key W dur=200", steps[0])
	}
}

func TestPairEvents_KeySeparatePresses(t *testing.T) {
	// 真正的两次独立按键（中间有 up 然后再 down）— 应当产 2 个 key step。
	evs := []Event{
		{TimestampMs: 1000, Type: EventKeyboard, Vk: "F", IsDown: true},
		{TimestampMs: 1080, Type: EventKeyboard, Vk: "F", IsDown: false},
		{TimestampMs: 1500, Type: EventKeyboard, Vk: "F", IsDown: true},
		{TimestampMs: 1580, Type: EventKeyboard, Vk: "F", IsDown: false},
	}
	steps := PairEvents(evs, 1920, 1080)
	// 期望：key(F, 80ms) + sleep(420ms) + key(F, 80ms)
	if len(steps) != 3 {
		t.Fatalf("len=%d want 3: %+v", len(steps), steps)
	}
	if steps[0].Kind != actions.StepKey || steps[0].Vk != "F" || steps[0].DurationMs != 80 {
		t.Errorf("step0=%+v want key F dur=80", steps[0])
	}
	if steps[1].Kind != actions.StepSleep || steps[1].DurationMs != 420 {
		t.Errorf("step1=%+v want sleep 420", steps[1])
	}
	if steps[2].Kind != actions.StepKey || steps[2].Vk != "F" || steps[2].DurationMs != 80 {
		t.Errorf("step2=%+v want key F dur=80", steps[2])
	}
}

func TestPairEvents_MouseLeftDownUp(t *testing.T) {
	evs := []Event{
		{TimestampMs: 1000, Type: EventMouse, MouseBtn: BtnLeft, X: 960, Y: 540, IsDown: true},
		{TimestampMs: 1050, Type: EventMouse, MouseBtn: BtnLeft, X: 960, Y: 540, IsDown: false},
	}
	steps := PairEvents(evs, 1920, 1080)
	if len(steps) != 1 {
		t.Fatalf("len=%d want 1: %+v", len(steps), steps)
	}
	s := steps[0]
	if s.Kind != actions.StepClickLeft {
		t.Errorf("kind=%q want %q", s.Kind, actions.StepClickLeft)
	}
	if s.XRatio != 0.5 {
		t.Errorf("xRatio=%v want 0.5", s.XRatio)
	}
	if s.YRatio != 0.5 {
		t.Errorf("yRatio=%v want 0.5", s.YRatio)
	}
	if s.DurationMs != 50 {
		t.Errorf("dur=%d want 50", s.DurationMs)
	}
}

func TestPairEvents_MouseOnlyDown(t *testing.T) {
	evs := []Event{
		{TimestampMs: 1000, Type: EventMouse, MouseBtn: BtnLeft, X: 100, Y: 100, IsDown: true},
	}
	steps := PairEvents(evs, 1920, 1080)
	if len(steps) != 0 {
		t.Errorf("len=%d want 0 (orphan mouse down should be dropped): %+v", len(steps), steps)
	}
}

func TestPairEvents_GapInsertsSleep(t *testing.T) {
	evs := []Event{
		{TimestampMs: 1000, Type: EventKeyboard, Vk: "A", IsDown: true},
		{TimestampMs: 1030, Type: EventKeyboard, Vk: "A", IsDown: false}, // dur=30→normalize 20，但实际 dur 是 30
		{TimestampMs: 1200, Type: EventKeyboard, Vk: "B", IsDown: true},  // gap 1200-1030=170
		{TimestampMs: 1250, Type: EventKeyboard, Vk: "B", IsDown: false},
	}
	steps := PairEvents(evs, 1920, 1080)
	// 期望: key(A dur=30 但 <20? 30>=20 保留), sleep 170, key(B dur=50)
	// 注：第一个 key 的 duration=1030-1000=30, 不 <20, 保留 30
	if len(steps) != 3 {
		t.Fatalf("len=%d want 3: %+v", len(steps), steps)
	}
	if steps[0].Kind != actions.StepKey || steps[0].Vk != "A" || steps[0].DurationMs != 30 {
		t.Errorf("step0=%+v want key A dur=30", steps[0])
	}
	if steps[1].Kind != actions.StepSleep || steps[1].DurationMs != 170 {
		t.Errorf("step1=%+v want sleep 170", steps[1])
	}
	if steps[2].Kind != actions.StepKey || steps[2].Vk != "B" || steps[2].DurationMs != 50 {
		t.Errorf("step2=%+v want key B dur=50", steps[2])
	}
}

func TestPairEvents_ShortClickNormalizedTo20(t *testing.T) {
	evs := []Event{
		{TimestampMs: 1000, Type: EventMouse, MouseBtn: BtnLeft, X: 0, Y: 0, IsDown: true},
		{TimestampMs: 1005, Type: EventMouse, MouseBtn: BtnLeft, X: 0, Y: 0, IsDown: false},
	}
	steps := PairEvents(evs, 1920, 1080)
	if len(steps) != 1 {
		t.Fatalf("len=%d want 1: %+v", len(steps), steps)
	}
	if steps[0].DurationMs != 20 {
		t.Errorf("dur=%d want 20 (short click normalized)", steps[0].DurationMs)
	}
}

func TestPairEvents_ShortKeyNormalizedTo20(t *testing.T) {
	evs := []Event{
		{TimestampMs: 1000, Type: EventKeyboard, Vk: "X", IsDown: true},
		{TimestampMs: 1005, Type: EventKeyboard, Vk: "X", IsDown: false},
	}
	steps := PairEvents(evs, 1920, 1080)
	if len(steps) != 1 {
		t.Fatalf("len=%d want 1", len(steps))
	}
	if steps[0].DurationMs != 20 {
		t.Errorf("dur=%d want 20", steps[0].DurationMs)
	}
}

func TestPairEvents_MixedButtonsNoCrossPair(t *testing.T) {
	evs := []Event{
		{TimestampMs: 1000, Type: EventMouse, MouseBtn: BtnLeft, X: 100, Y: 100, IsDown: true},
		{TimestampMs: 1010, Type: EventMouse, MouseBtn: BtnRight, X: 200, Y: 200, IsDown: true},
		{TimestampMs: 1080, Type: EventMouse, MouseBtn: BtnLeft, X: 100, Y: 100, IsDown: false},
		{TimestampMs: 1090, Type: EventMouse, MouseBtn: BtnRight, X: 200, Y: 200, IsDown: false},
	}
	steps := PairEvents(evs, 1000, 1000)
	// 左 down @1000 + 左 up @1080 → click_left dur=80, xR=0.1, yR=0.1
	// emitGap(left_down @1000 vs lastEmit @1000) → no gap
	// 右 down @1010 + 右 up @1090 → click_right dur=80, xR=0.2, yR=0.2
	// emitGap(right_down @1010 vs lastEmit @1080) → no gap (10 < 50 actually negative)
	if len(steps) != 2 {
		t.Fatalf("len=%d want 2: %+v", len(steps), steps)
	}
	if steps[0].Kind != actions.StepClickLeft || steps[0].XRatio != 0.1 {
		t.Errorf("step0=%+v want click_left xR=0.1", steps[0])
	}
	if steps[1].Kind != actions.StepClickRight || steps[1].XRatio != 0.2 {
		t.Errorf("step1=%+v want click_right xR=0.2", steps[1])
	}
}

func TestPairEvents_MiddleButton(t *testing.T) {
	evs := []Event{
		{TimestampMs: 1000, Type: EventMouse, MouseBtn: BtnMiddle, X: 0, Y: 0, IsDown: true},
		{TimestampMs: 1050, Type: EventMouse, MouseBtn: BtnMiddle, X: 0, Y: 0, IsDown: false},
	}
	steps := PairEvents(evs, 1920, 1080)
	if len(steps) != 1 || steps[0].Kind != actions.StepClickMiddle {
		t.Fatalf("want click_middle, got %+v", steps)
	}
}

func TestPairEvents_Empty(t *testing.T) {
	if got := PairEvents(nil, 1920, 1080); len(got) != 0 {
		t.Errorf("want 0 steps for empty input, got %d", len(got))
	}
}

func TestPairEvents_ZeroClientDimsSafe(t *testing.T) {
	evs := []Event{
		{TimestampMs: 1000, Type: EventMouse, MouseBtn: BtnLeft, X: 100, Y: 100, IsDown: true},
		{TimestampMs: 1050, Type: EventMouse, MouseBtn: BtnLeft, X: 100, Y: 100, IsDown: false},
	}
	steps := PairEvents(evs, 0, 0)
	if len(steps) != 1 {
		t.Fatalf("len=%d want 1", len(steps))
	}
	if steps[0].XRatio != 0 || steps[0].YRatio != 0 {
		t.Errorf("ratios=%v,%v want 0,0 (zero dims)", steps[0].XRatio, steps[0].YRatio)
	}
}

func TestPairEvents_MouseDragDetected(t *testing.T) {
	// down 在 (100, 100)，中间 move 到 (200, 200)，up 在 (200, 200)
	// dx*dx + dy*dy = 100*100 * 2 = 20000 > dragThresholdPx → drag
	evs := []Event{
		{TimestampMs: 1000, Type: EventMouse, MouseBtn: BtnLeft, X: 100, Y: 100, IsDown: true},
		{TimestampMs: 1050, Type: EventMouseMove, X: 200, Y: 200},
		{TimestampMs: 1100, Type: EventMouse, MouseBtn: BtnLeft, X: 200, Y: 200, IsDown: false},
	}
	steps := PairEvents(evs, 1000, 1000)
	if len(steps) != 1 {
		t.Fatalf("len=%d want 1 (drag step), got %+v", len(steps), steps)
	}
	if steps[0].Kind != "mouse_drag" {
		t.Errorf("kind=%s want mouse_drag", steps[0].Kind)
	}
	if steps[0].XRatio != 0.1 || steps[0].YRatio != 0.1 {
		t.Errorf("起点 ratio %v,%v want 0.1,0.1", steps[0].XRatio, steps[0].YRatio)
	}
	if steps[0].X2Ratio != 0.2 || steps[0].Y2Ratio != 0.2 {
		t.Errorf("终点 ratio %v,%v want 0.2,0.2", steps[0].X2Ratio, steps[0].Y2Ratio)
	}
	if steps[0].MouseButton != "left" {
		t.Errorf("button=%q want left", steps[0].MouseButton)
	}
}

func TestPairEvents_TinyMoveStaysClick(t *testing.T) {
	// down + 微小 move (2px) + up：dx*dx+dy*dy = 4*2 = 8 < 25，仍判 click 不 drag
	evs := []Event{
		{TimestampMs: 1000, Type: EventMouse, MouseBtn: BtnLeft, X: 100, Y: 100, IsDown: true},
		{TimestampMs: 1050, Type: EventMouseMove, X: 102, Y: 102},
		{TimestampMs: 1100, Type: EventMouse, MouseBtn: BtnLeft, X: 102, Y: 102, IsDown: false},
	}
	steps := PairEvents(evs, 1000, 1000)
	if len(steps) != 1 || steps[0].Kind != "click_left" {
		t.Errorf("微移应当 click 不 drag, got %+v", steps)
	}
}

func TestPairEvents_ScrollEmitsStep(t *testing.T) {
	evs := []Event{
		{TimestampMs: 1000, Type: EventMouseScroll, X: 500, Y: 500, WheelNotches: 3},
		{TimestampMs: 1100, Type: EventMouseScroll, X: 500, Y: 500, WheelNotches: -1},
	}
	steps := PairEvents(evs, 1000, 1000)
	scrollCount := 0
	for _, s := range steps {
		if s.Kind == "scroll" {
			scrollCount++
		}
	}
	if scrollCount != 2 {
		t.Errorf("want 2 scroll steps, got %d in %+v", scrollCount, steps)
	}
}

func TestPairEvents_BareMouseMoveIgnored(t *testing.T) {
	// 没 down 关联的 move 应被丢
	evs := []Event{
		{TimestampMs: 1000, Type: EventMouseMove, X: 100, Y: 100},
		{TimestampMs: 1100, Type: EventMouseMove, X: 200, Y: 200},
	}
	steps := PairEvents(evs, 1000, 1000)
	if len(steps) != 0 {
		t.Errorf("裸 move 不应产 step, got %+v", steps)
	}
}

func TestPairEvents_RawDeltaAggregatesWithin50ms(t *testing.T) {
	// 3 个 raw delta，前 2 个在 50ms 窗口内（10ms,30ms），第 3 个在 100ms（超窗）
	evs := []Event{
		{TimestampMs: 1000, Type: EventRawMouseDelta, Dx: 10, Dy: 0},
		{TimestampMs: 1010, Type: EventRawMouseDelta, Dx: 20, Dy: 5},
		{TimestampMs: 1030, Type: EventRawMouseDelta, Dx: 30, Dy: 10},
		{TimestampMs: 1100, Type: EventRawMouseDelta, Dx: 50, Dy: 0},
	}
	steps := PairEvents(evs, 1920, 1080)
	// 应产 2 个 mouse_move_rel：前 3 个合 (60, 15)；最后 1 个 (50, 0)
	rels := []actions.Step{}
	for _, s := range steps {
		if s.Kind == "mouse_move_rel" {
			rels = append(rels, s)
		}
	}
	if len(rels) != 2 {
		t.Fatalf("应产 2 个 mouse_move_rel, got %d: %+v", len(rels), rels)
	}
	if rels[0].Dx != 60 || rels[0].Dy != 15 {
		t.Errorf("第一桶累加错 got dx=%d dy=%d want 60,15", rels[0].Dx, rels[0].Dy)
	}
	if rels[1].Dx != 50 || rels[1].Dy != 0 {
		t.Errorf("第二桶 got dx=%d dy=%d want 50,0", rels[1].Dx, rels[1].Dy)
	}
}

func TestPairEvents_RawDeltaFlushedByNonRawEvent(t *testing.T) {
	// raw delta 之后立刻一个 key event → raw 必须先 flush 才能 emit key
	evs := []Event{
		{TimestampMs: 1000, Type: EventRawMouseDelta, Dx: 100, Dy: 0},
		{TimestampMs: 1010, Type: EventKeyboard, Vk: "W", IsDown: true},
		{TimestampMs: 1100, Type: EventKeyboard, Vk: "W", IsDown: false},
	}
	steps := PairEvents(evs, 1920, 1080)
	if len(steps) < 2 {
		t.Fatalf("应产 ≥2 step, got %+v", steps)
	}
	if steps[0].Kind != "mouse_move_rel" {
		t.Errorf("第一个 step 应是 mouse_move_rel, got %s", steps[0].Kind)
	}
}

func TestPairEvents_ShiftWrapperW(t *testing.T) {
	// Shift down → W down → W up → Shift up = 经典组合键 Shift+W
	// 期望：key_down(Shift) / key(W, 100ms) / key_up(Shift)
	evs := []Event{
		{TimestampMs: 1000, Type: EventKeyboard, Vk: "Shift", IsDown: true},
		{TimestampMs: 1100, Type: EventKeyboard, Vk: "W", IsDown: true},
		{TimestampMs: 1200, Type: EventKeyboard, Vk: "W", IsDown: false},
		{TimestampMs: 1300, Type: EventKeyboard, Vk: "Shift", IsDown: false},
	}
	steps := PairEvents(evs, 1920, 1080)
	// 期望非 sleep 步骤序列: key_down(Shift), key(W, 100), key_up(Shift)
	nonSleep := []actions.Step{}
	for _, s := range steps {
		if s.Kind != "sleep" {
			nonSleep = append(nonSleep, s)
		}
	}
	if len(nonSleep) != 3 {
		t.Fatalf("want 3 non-sleep steps, got %d: %+v", len(nonSleep), nonSleep)
	}
	if nonSleep[0].Kind != "key_down" || nonSleep[0].Vk != "Shift" {
		t.Errorf("step0=%+v, want key_down(Shift)", nonSleep[0])
	}
	if nonSleep[1].Kind != "key" || nonSleep[1].Vk != "W" || nonSleep[1].DurationMs != 100 {
		t.Errorf("step1=%+v, want key(W, 100)", nonSleep[1])
	}
	if nonSleep[2].Kind != "key_up" || nonSleep[2].Vk != "Shift" {
		t.Errorf("step2=%+v, want key_up(Shift)", nonSleep[2])
	}
}

func TestPairEvents_NonOverlappingKeysStayAsKey(t *testing.T) {
	// W down/up + Shift down/up（不重叠）→ 两个独立 key step，不该错认成 wrapper
	evs := []Event{
		{TimestampMs: 1000, Type: EventKeyboard, Vk: "W", IsDown: true},
		{TimestampMs: 1100, Type: EventKeyboard, Vk: "W", IsDown: false},
		{TimestampMs: 1200, Type: EventKeyboard, Vk: "Shift", IsDown: true},
		{TimestampMs: 1300, Type: EventKeyboard, Vk: "Shift", IsDown: false},
	}
	steps := PairEvents(evs, 0, 0)
	for _, s := range steps {
		if s.Kind == "key_down" || s.Kind == "key_up" {
			t.Errorf("不应有 key_down/key_up（无嵌套）, got %+v", s)
		}
	}
}

func TestMergeMouseMoveRel_AdjacentMerged(t *testing.T) {
	in := []actions.Step{
		{Kind: "mouse_move_rel", Dx: 3, DurationMs: 16},
		{Kind: "mouse_move_rel", Dx: 1, Dy: 1, DurationMs: 16},
	}
	out := MergeMouseMoveRel(in)
	if len(out) != 1 {
		t.Fatalf("应合成 1 step, got %d: %+v", len(out), out)
	}
	if out[0].Dx != 4 || out[0].Dy != 1 || out[0].DurationMs != 32 {
		t.Errorf("got %+v want dx=4 dy=1 32ms", out[0])
	}
}

func TestMergeMouseMoveRel_ShortSleepMerged(t *testing.T) {
	in := []actions.Step{
		{Kind: "mouse_move_rel", Dx: 3, DurationMs: 16},
		{Kind: "sleep", DurationMs: 80},
		{Kind: "mouse_move_rel", Dx: 1, Dy: 1, DurationMs: 16},
	}
	out := MergeMouseMoveRel(in)
	if len(out) != 1 {
		t.Fatalf("短 sleep 应吞，合成 1 step, got %d: %+v", len(out), out)
	}
	if out[0].DurationMs != 112 { // 16 + 80 + 16
		t.Errorf("durationMs=%d want 112", out[0].DurationMs)
	}
}

func TestMergeMouseMoveRel_LongSleepNotMerged(t *testing.T) {
	in := []actions.Step{
		{Kind: "mouse_move_rel", Dx: 3, DurationMs: 16},
		{Kind: "sleep", DurationMs: 500}, // 超 300ms 阈值
		{Kind: "mouse_move_rel", Dx: 1, DurationMs: 16},
	}
	out := MergeMouseMoveRel(in)
	if len(out) != 3 {
		t.Errorf("长 sleep 不应合并, got %d steps", len(out))
	}
}
