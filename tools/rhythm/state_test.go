package rhythm

import (
	"testing"
	"time"
)

// 上升沿 (prev=false → ratio≥阈值) → fire
func TestTrackState_RisingEdgeFires(t *testing.T) {
	var s TrackState
	now := time.Now()
	if !s.Tick(DarkRatioThreshold+0.1, now) {
		t.Fatal("上升沿应触发按键")
	}
	if !s.prev {
		t.Error("触发后 prev 应为 true")
	}
}

// 持续 ratio≥阈值 + 时间不到 retrigger → 不再 fire
func TestTrackState_HoldNoRefireBeforeRetrigger(t *testing.T) {
	var s TrackState
	now := time.Now()
	s.Tick(DarkRatioThreshold+0.1, now) // 第一次触发
	for i := 1; i < 10; i++ {
		if s.Tick(DarkRatioThreshold+0.1, now.Add(time.Duration(i)*time.Millisecond)) {
			t.Errorf("retrigger 间隔内不该再触发 (i=%d)", i)
		}
	}
}

// 持续 ratio≥阈值 + 超过 retrigger 间隔 → 重新 fire（应付长按音符）
func TestTrackState_HoldRefiresAfterRetrigger(t *testing.T) {
	var s TrackState
	now := time.Now()
	s.Tick(DarkRatioThreshold+0.1, now)
	if !s.Tick(DarkRatioThreshold+0.1, now.Add(RetriggerInterval+time.Millisecond)) {
		t.Fatal("超过 retrigger 间隔后持续 has 应该重新触发")
	}
}

// 下降到阈值以下 → prev=false，下次上升沿才能再 fire
func TestTrackState_FallingEdgeResetsPrev(t *testing.T) {
	var s TrackState
	now := time.Now()
	s.Tick(DarkRatioThreshold+0.1, now) // press
	if s.Tick(DarkRatioThreshold-0.01, now.Add(10*time.Millisecond)) {
		t.Fatal("下降沿不该触发按键")
	}
	if s.prev {
		t.Error("下降后 prev 应为 false")
	}
	// 再次上升应该触发
	if !s.Tick(DarkRatioThreshold+0.1, now.Add(20*time.Millisecond)) {
		t.Fatal("下降后再次上升应该触发")
	}
}

// 边界：ratio 恰好等于阈值（>= 比较）→ fire
func TestTrackState_BoundaryEqualThresholdFires(t *testing.T) {
	var s TrackState
	now := time.Now()
	if !s.Tick(DarkRatioThreshold, now) {
		t.Fatal("ratio == threshold 应该触发 (用的是 >= 比较)")
	}
}

// 长序列：模拟音符飞过 + 间隔再来一个
func TestTrackState_SequenceTable(t *testing.T) {
	cases := []struct {
		name      string
		ratios    []float32
		wantPress int
	}{
		{"single hit", []float32{0.0, 0.0, 0.5, 0.5, 0.0, 0.0}, 1},
		{"two separate hits", []float32{0.0, 0.5, 0.0, 0.0, 0.5, 0.0}, 2},
		{"continuous hold no refire in window", []float32{0.0, 0.5, 0.5, 0.5, 0.5}, 1},
		{"all background", []float32{0.01, 0.02, 0.0, 0.03}, 0},
		// 阈值边界 0.06：0.05 不触发，0.07 触发
		{"sub threshold no fire", []float32{0.0, 0.05, 0.05, 0.0}, 0},
		{"barely over threshold fires once", []float32{0.0, 0.07, 0.07, 0.0}, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var s TrackState
			now := time.Now()
			presses := 0
			// 每 tick 8ms，比 RetriggerInterval(85ms) 短得多，连续 hold 不会跨 retrigger
			for i, r := range c.ratios {
				if s.Tick(r, now.Add(time.Duration(i)*8*time.Millisecond)) {
					presses++
				}
			}
			if presses != c.wantPress {
				t.Errorf("ratios=%v: presses=%d want=%d", c.ratios, presses, c.wantPress)
			}
		})
	}
}
