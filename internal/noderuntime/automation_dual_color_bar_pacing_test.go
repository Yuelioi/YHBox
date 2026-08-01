package noderuntime

import (
	"testing"
	"time"
)

func TestDualColorBarPacingDelayCompletesTheMinimumCycle(t *testing.T) {
	startedAt := time.Date(2026, 8, 1, 1, 0, 0, 0, time.UTC)
	if got := dualColorBarPacingDelay(startedAt, startedAt.Add(35*time.Millisecond), 80*time.Millisecond); got != 45*time.Millisecond {
		t.Fatalf("pacing delay = %s, want 45ms", got)
	}
	if got := dualColorBarPacingDelay(startedAt, startedAt.Add(95*time.Millisecond), 80*time.Millisecond); got != 0 {
		t.Fatalf("slow-cycle pacing delay = %s, want 0", got)
	}
}
