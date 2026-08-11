package platform

import (
	"os"
	"testing"
	"time"
)

func TestHighResTimer_BeginEndIdempotent(t *testing.T) {
	// 多次 Begin/End 不 panic 是基本要求
	tm := NewHighResTimer()
	if err := tm.Begin(); err != nil {
		t.Fatalf("Begin: %v", err)
	}
	tm.End()
	tm.End() // 重复 End 安全
	if err := tm.Begin(); err != nil {
		t.Fatalf("second Begin: %v", err)
	}
	tm.End()
}

func TestHighResTimer_TickerAccuracy(t *testing.T) {
	if os.Getenv("YOTTA_WINDOWS_TIMER_SMOKE") != "1" {
		t.Skip("timing smoke runs in an isolated release gate")
	}
	// 没开 high-res 前，8ms ticker 实际 >=15ms
	// 开了后，8ms ticker 实际接近 8ms
	if testing.Short() {
		t.Skip("timing-sensitive, skip in -short")
	}
	tm := NewHighResTimer()
	if err := tm.Begin(); err != nil {
		t.Fatalf("Begin: %v", err)
	}
	defer tm.End()

	ticker := time.NewTicker(8 * time.Millisecond)
	defer ticker.Stop()
	t0 := time.Now()
	for i := 0; i < 50; i++ {
		<-ticker.C
	}
	elapsed := time.Since(t0)
	// 50 ticks × 8ms = 400ms 理想。允许 50% 抖动余量（CI 慢）
	if elapsed > 600*time.Millisecond {
		t.Errorf("50 ticks of 8ms took %v, expected < 600ms (timeBeginPeriod(1) not effective?)", elapsed)
	}
}
