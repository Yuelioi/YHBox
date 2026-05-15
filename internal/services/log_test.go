package services

import (
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestLogSink_DebounceFlush(t *testing.T) {
	var got []LogLinesEvent
	var mu sync.Mutex
	sink := NewLogSink(func(e LogLinesEvent) {
		mu.Lock()
		got = append(got, e)
		mu.Unlock()
	})

	sink.Write([]byte("line1\nline2\n"))
	time.Sleep(150 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(got) != 1 {
		t.Fatalf("expected 1 emit, got %d", len(got))
	}
	if len(got[0].Lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(got[0].Lines))
	}
	if got[0].Seq != 1 {
		t.Errorf("expected seq=1, got %d", got[0].Seq)
	}
}

func TestLogSink_SeqMonotonic(t *testing.T) {
	var seqs []uint64
	var mu sync.Mutex
	sink := NewLogSink(func(e LogLinesEvent) {
		mu.Lock()
		seqs = append(seqs, e.Seq)
		mu.Unlock()
	})

	for i := 0; i < 3; i++ {
		sink.Write([]byte("x\n"))
		time.Sleep(120 * time.Millisecond)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(seqs) != 3 {
		t.Fatalf("expected 3 emits, got %d", len(seqs))
	}
	for i := 0; i < 3; i++ {
		if seqs[i] != uint64(i+1) {
			t.Errorf("seq[%d] expected %d, got %d", i, i+1, seqs[i])
		}
	}
}

func TestLogSink_MaxBatchTriggersImmediateFlush(t *testing.T) {
	var emitCount atomic.Int32
	sink := NewLogSink(func(e LogLinesEvent) {
		emitCount.Add(1)
	})

	for i := 0; i < maxBatchLines+10; i++ {
		sink.Write([]byte("line\n"))
	}

	time.Sleep(50 * time.Millisecond)

	count := emitCount.Load()
	if count < 1 {
		t.Fatalf("expected at least 1 immediate flush, got %d", count)
	}
}

func TestLogSink_PartialLineBuffered(t *testing.T) {
	var mu sync.Mutex
	var got []LogLinesEvent
	sink := NewLogSink(func(e LogLinesEvent) {
		mu.Lock()
		got = append(got, e)
		mu.Unlock()
	})

	sink.Write([]byte("incomplete"))
	time.Sleep(150 * time.Millisecond)

	mu.Lock()
	if len(got) != 0 {
		mu.Unlock()
		t.Errorf("partial line should not emit; got %d events", len(got))
		return
	}
	mu.Unlock()

	sink.Write([]byte(" continued\n"))
	time.Sleep(150 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(got) != 1 {
		t.Fatalf("expected 1 emit after newline, got %d", len(got))
	}
	if got[0].Lines[0] != "incomplete continued" {
		t.Errorf("partial concat wrong: %q", got[0].Lines[0])
	}
}

func TestLogSink_SnapshotRingBuffer(t *testing.T) {
	sink := NewLogSink(nil)
	for i := 0; i < ringCapacity+50; i++ {
		sink.Write([]byte("x\n"))
	}
	snap := sink.Snapshot()
	lines := strings.Split(snap, "\n")
	if len(lines) != ringCapacity {
		t.Errorf("ring should cap at %d, got %d lines", ringCapacity, len(lines))
	}
}

func TestLogSink_Flush(t *testing.T) {
	var mu sync.Mutex
	var got []LogLinesEvent
	sink := NewLogSink(func(e LogLinesEvent) {
		mu.Lock()
		got = append(got, e)
		mu.Unlock()
	})

	sink.Write([]byte("line1\nline2\n"))
	sink.Flush()
	time.Sleep(20 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(got) != 1 {
		t.Fatalf("expected immediate flush, got %d events", len(got))
	}
}
