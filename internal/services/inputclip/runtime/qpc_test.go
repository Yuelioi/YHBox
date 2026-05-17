package runtime

import (
	"testing"
	"time"
)

func TestQPCMicrosMonotonic(t *testing.T) {
	a := QPCMicros()
	time.Sleep(10 * time.Millisecond)
	b := QPCMicros()
	if b <= a {
		t.Errorf("QPC 不单调: a=%d b=%d", a, b)
	}
	delta := b - a
	if delta < 8000 || delta > 30000 {
		t.Errorf("QPC delta = %d us, 期望 10ms ±20ms", delta)
	}
}
