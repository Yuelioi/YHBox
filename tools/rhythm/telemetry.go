package rhythm

import (
	"sort"
	"sync"
	"time"
)

// Telemetry 收集 rhythm 主循环的帧时间，提供 rolling p50/p95/p99 + late_ticks。
// 用 ring buffer 保留最近 N 条。不是 hot-path performance sensitive
// (RecordTick 加 mutex 没问题，120Hz 跟锁开销可忽略)。
type Telemetry struct {
	mu      sync.Mutex
	cap     int
	capture []time.Duration
	detect  []time.Duration
	tick    []time.Duration
	idx     int   // ring write pointer
	filled  int   // 已写条数 (≤cap)
	late    int64 // 累计 late tick 数（tick_total > TickInterval）
}

// TelemetrySnapshot 是某一时刻的 stats 快照。
type TelemetrySnapshot struct {
	CaptureP50, CaptureP95, CaptureP99 time.Duration
	DetectP50, DetectP95, DetectP99    time.Duration
	TickP50, TickP95, TickP99          time.Duration
	LateTicks                          int64
	Ticks                              []time.Duration // 调试用
}

// NewTelemetry capacity = ring buffer 长度。120 ≈ 1 秒数据 @ 120Hz。
func NewTelemetry(capacity int) *Telemetry {
	return &Telemetry{
		cap:     capacity,
		capture: make([]time.Duration, capacity),
		detect:  make([]time.Duration, capacity),
		tick:    make([]time.Duration, capacity),
	}
}

// RecordTick 写一条样本。捕获/识别/总时长。
func (t *Telemetry) RecordTick(capDur, detDur, tickDur time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.capture[t.idx] = capDur
	t.detect[t.idx] = detDur
	t.tick[t.idx] = tickDur
	t.idx = (t.idx + 1) % t.cap
	if t.filled < t.cap {
		t.filled++
	}
	if tickDur > TickInterval {
		t.late++
	}
}

// Snapshot 计算当前 rolling stats。
func (t *Telemetry) Snapshot() TelemetrySnapshot {
	t.mu.Lock()
	cs := append([]time.Duration{}, t.capture[:t.filled]...)
	ds := append([]time.Duration{}, t.detect[:t.filled]...)
	ts := append([]time.Duration{}, t.tick[:t.filled]...)
	late := t.late
	t.mu.Unlock()

	return TelemetrySnapshot{
		CaptureP50: percentile(cs, 50),
		CaptureP95: percentile(cs, 95),
		CaptureP99: percentile(cs, 99),
		DetectP50:  percentile(ds, 50),
		DetectP95:  percentile(ds, 95),
		DetectP99:  percentile(ds, 99),
		TickP50:    percentile(ts, 50),
		TickP95:    percentile(ts, 95),
		TickP99:    percentile(ts, 99),
		LateTicks:  late,
		Ticks:      ts,
	}
}

// percentile 在 xs 副本上排序计算分位数，不修改入参（保留 Ticks 字段的 time-order）。
func percentile(xs []time.Duration, p int) time.Duration {
	if len(xs) == 0 {
		return 0
	}
	sorted := append([]time.Duration{}, xs...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	idx := (len(sorted) - 1) * p / 100
	return sorted[idx]
}
