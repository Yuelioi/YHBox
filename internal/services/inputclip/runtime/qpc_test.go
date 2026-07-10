package runtime

import (
	"testing"
	"time"
)

func TestQPCMicrosMonotonic(t *testing.T) {
	wallStart := time.Now()
	a := QPCMicros()
	time.Sleep(10 * time.Millisecond)
	b := QPCMicros()
	wallDelta := uint64(time.Since(wallStart).Microseconds())
	if b <= a {
		t.Errorf("QPC 不单调: a=%d b=%d", a, b)
	}
	qpcDelta := b - a
	diff := qpcDelta
	if wallDelta > qpcDelta {
		diff = wallDelta - qpcDelta
	} else {
		diff = qpcDelta - wallDelta
	}
	tolerance := wallDelta/10 + 2_000
	if diff > tolerance {
		t.Errorf("QPC delta = %d us, wall delta = %d us, diff = %d us", qpcDelta, wallDelta, diff)
	}
}
