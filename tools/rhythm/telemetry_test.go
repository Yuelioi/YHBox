package rhythm

import (
	"sort"
	"testing"
	"time"
)

func TestTelemetry_PercentilesOnFixedSamples(t *testing.T) {
	tm := NewTelemetry(120)
	for i := 1; i <= 100; i++ {
		tm.RecordTick(
			time.Duration(i)*time.Millisecond, // capture
			time.Duration(i)*time.Microsecond, // detect
			time.Duration(i)*time.Millisecond, // tick_total
		)
	}
	stats := tm.Snapshot()
	// 检查 tick_total 的 p50/p95/p99 落在合理 buckets
	if stats.TickP50 < 40*time.Millisecond || stats.TickP50 > 60*time.Millisecond {
		t.Errorf("p50 want ~50ms, got %v", stats.TickP50)
	}
	if stats.TickP95 < 90*time.Millisecond || stats.TickP95 > 100*time.Millisecond {
		t.Errorf("p95 want ~95ms, got %v", stats.TickP95)
	}
	if stats.TickP99 < 95*time.Millisecond || stats.TickP99 > 100*time.Millisecond {
		t.Errorf("p99 want ~99ms, got %v", stats.TickP99)
	}
}

func TestTelemetry_LateTicksCounted(t *testing.T) {
	tm := NewTelemetry(120)
	// 8ms TickInterval → 任何 tick > 8ms 算 late
	tm.RecordTick(5*time.Millisecond, 0, 7*time.Millisecond)  // not late
	tm.RecordTick(5*time.Millisecond, 0, 9*time.Millisecond)  // late
	tm.RecordTick(5*time.Millisecond, 0, 20*time.Millisecond) // late
	stats := tm.Snapshot()
	if stats.LateTicks != 2 {
		t.Errorf("LateTicks want 2, got %d", stats.LateTicks)
	}
}

func TestTelemetry_RollingWindow(t *testing.T) {
	tm := NewTelemetry(3) // 只保留最近 3 条
	tm.RecordTick(1*time.Millisecond, 0, 1*time.Millisecond)
	tm.RecordTick(2*time.Millisecond, 0, 2*time.Millisecond)
	tm.RecordTick(3*time.Millisecond, 0, 3*time.Millisecond)
	tm.RecordTick(4*time.Millisecond, 0, 4*time.Millisecond) // 顶掉 1ms
	stats := tm.Snapshot()
	// 应该只看到 2/3/4
	wantSet := []time.Duration{2 * time.Millisecond, 3 * time.Millisecond, 4 * time.Millisecond}
	got := append([]time.Duration{}, stats.Ticks...)
	sort.Slice(got, func(i, j int) bool { return got[i] < got[j] })
	for i, w := range wantSet {
		if got[i] != w {
			t.Errorf("rolling window expected %v, got %v", wantSet, got)
			break
		}
	}
}
