package runtime

import (
	"testing"
	"time"
)

func TestStopwatchStartStopRead(t *testing.T) {
	sw := newStopwatchTable()

	sw.start("foo")
	time.Sleep(50 * time.Millisecond)
	sw.stop("foo")
	elapsed := sw.read("foo")

	if elapsed < 40 || elapsed > 200 {
		t.Fatalf("expected ~50ms elapsed, got %d", elapsed)
	}
}

func TestStopwatchReadRunning(t *testing.T) {
	sw := newStopwatchTable()
	sw.start("running")
	time.Sleep(30 * time.Millisecond)
	elapsed := sw.read("running")
	if elapsed < 20 || elapsed > 200 {
		t.Fatalf("expected ~30ms running elapsed, got %d", elapsed)
	}
}

func TestStopwatchRestart(t *testing.T) {
	sw := newStopwatchTable()
	sw.start("k")
	time.Sleep(30 * time.Millisecond)
	sw.start("k") // restart, 不报错
	time.Sleep(20 * time.Millisecond)
	elapsed := sw.read("k")
	if elapsed >= 40 {
		t.Fatalf("restart should reset, got %d", elapsed)
	}
}

func TestStopwatchMissingKey(t *testing.T) {
	sw := newStopwatchTable()
	elapsed := sw.read("never-started")
	if elapsed != 0 {
		t.Fatalf("missing key should return 0, got %d", elapsed)
	}
}
