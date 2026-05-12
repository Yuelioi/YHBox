package gui

import (
	"strings"
	"sync"
	"testing"
	"time"
)

func TestLogSink_WriteAndSnapshot(t *testing.T) {
	s := NewLogSink(3, nil) // capacity=3, no walk update callback
	s.Write([]byte("line1\n"))
	s.Write([]byte("line2\n"))
	s.Write([]byte("line3\n"))

	out := s.Snapshot()
	if !strings.Contains(out, "line1") || !strings.Contains(out, "line3") {
		t.Errorf("snapshot missing lines: %q", out)
	}
}

func TestLogSink_RingBufferOverflow(t *testing.T) {
	s := NewLogSink(2, nil)
	s.Write([]byte("a\n"))
	s.Write([]byte("b\n"))
	s.Write([]byte("c\n"))

	out := s.Snapshot()
	if strings.Contains(out, "a") {
		t.Errorf("oldest line 'a' should be dropped: %q", out)
	}
	if !strings.Contains(out, "b") || !strings.Contains(out, "c") {
		t.Errorf("snapshot should keep last 2: %q", out)
	}
}

func TestLogSink_PartialLineBuffered(t *testing.T) {
	s := NewLogSink(10, nil)
	s.Write([]byte("part1"))
	s.Write([]byte("part2\nline2\n"))

	out := s.Snapshot()
	if !strings.Contains(out, "part1part2") {
		t.Errorf("fragments should be joined: %q", out)
	}
	if !strings.Contains(out, "line2") {
		t.Errorf("line2 missing: %q", out)
	}
}

func TestLogSink_CallbackTriggered(t *testing.T) {
	var (
		mu     sync.Mutex
		called int
	)
	s := NewLogSink(10, func(lines []string) {
		mu.Lock()
		called++
		mu.Unlock()
	})
	s.Write([]byte("a\n"))
	s.Write([]byte("b\n"))
	// 多次 Write 经 trailing-edge debounce 合并为一次 callback
	time.Sleep(flushDebounce + 50*time.Millisecond)
	mu.Lock()
	got := called
	mu.Unlock()
	if got < 1 {
		t.Errorf("callback should fire at least once after debounce, got %d", got)
	}
}
