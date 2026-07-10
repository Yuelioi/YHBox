package runtime

import "testing"

func TestWaitUntilDoesNotReturnEarly(t *testing.T) {
	policy := DefaultPlaybackPolicy()
	start := QPCMicros()
	target := start + 50000 // 50ms 后
	WaitUntil(target, policy)
	got := QPCMicros()
	if got < target {
		t.Errorf("早返: got=%d target=%d (差 %d us)", got, target, target-got)
	}
}
