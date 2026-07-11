package services

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestLogSink_AppendDumpLine_FileOnly(t *testing.T) {
	dir := t.TempDir()
	emitted := 0
	s := NewLogSink(func(LogLinesEvent) { emitted++ })
	s.SetFileWriter(dir)
	s.AppendDumpLine(`CheckTemplate(n1) in{a=1} ×3`)

	files, _ := os.ReadDir(dir)
	if len(files) == 0 {
		t.Fatal("no log file written")
	}
	data, _ := os.ReadFile(filepath.Join(dir, files[0].Name()))
	if !strings.Contains(string(data), "×3") || !strings.Contains(string(data), "node-dump") {
		t.Fatalf("dump line not in file: %s", data)
	}
	if emitted != 0 {
		t.Fatalf("AppendDumpLine must not emit log:lines, emitted=%d", emitted)
	}
	s.SetFileWriter("") // Windows: release file handle so t.TempDir cleanup can delete
}

func TestLogSink_AppendActionTrace_FileOnlyRedacted(t *testing.T) {
	dir := t.TempDir()
	emitted := 0
	s := NewLogSink(func(LogLinesEvent) { emitted++ })
	s.SetFileWriter(dir)
	s.AppendActionTrace(map[string]any{
		"containerId": "container-1",
		"action":      "click",
		"source": map[string]any{
			"NodeID":   "click-1",
			"NodeKind": "ClickAt",
			"InPin":    "In",
		},
		"target": map[string]any{
			"ID":   "win32:100",
			"Kind": "win32-window",
			"Ref":  map[string]any{"HWND": 100},
		},
		"backend":         "win32",
		"status":          "success",
		"request":         map[string]any{"secret": "raw request"},
		"result":          map[string]any{"secret": "raw result"},
		"coordinateSteps": []any{map[string]any{"Input": "raw coords"}},
		"durationMs":      12,
	})

	files, _ := filepath.Glob(filepath.Join(dir, "yotta-*.log"))
	if len(files) != 1 {
		t.Fatalf("expected 1 log file, got %d", len(files))
	}
	data, _ := os.ReadFile(files[0])
	raw := string(data)
	if strings.Contains(raw, "raw request") || strings.Contains(raw, "raw result") || strings.Contains(raw, "HWND") {
		t.Fatalf("action trace log leaked raw payload: %s", raw)
	}
	var line map[string]any
	if err := json.Unmarshal(data[:len(data)-1], &line); err != nil {
		t.Fatalf("invalid action trace JSON line: %v\n%s", err, data)
	}
	if line["event"] != "action-trace" || line["action"] != "click" || line["coordinateStepCount"] != float64(1) {
		t.Fatalf("unexpected action trace line: %#v", line)
	}
	target, _ := line["target"].(map[string]any)
	if target["id"] != "win32:100" || target["kind"] != "win32-window" {
		t.Fatalf("target not sanitized as expected: %#v", target)
	}
	if emitted != 0 {
		t.Fatalf("AppendActionTrace must not emit log:lines, emitted=%d", emitted)
	}
	s.SetFileWriter("") // Windows: release file handle so t.TempDir cleanup can delete
}

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
	seqs := make(chan uint64, 3)
	sink := NewLogSink(func(e LogLinesEvent) {
		seqs <- e.Seq
	})

	for i := 0; i < 3; i++ {
		sink.Write([]byte("x\n"))
		sink.drain()
	}

	for i := 0; i < 3; i++ {
		select {
		case got := <-seqs:
			if got != uint64(i+1) {
				t.Errorf("seq[%d] expected %d, got %d", i, i+1, got)
			}
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for seq=%d", i+1)
		}
	}
}

func TestLogSink_DoesNotDeliverLaterBatchBeforeBlockedCallback(t *testing.T) {
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	delivered := make(chan uint64, 2)
	sink := NewLogSink(func(e LogLinesEvent) {
		if e.Seq == 1 {
			close(firstStarted)
			<-releaseFirst
		}
		delivered <- e.Seq
	})

	sink.Write([]byte("first\n"))
	sink.Flush()
	select {
	case <-firstStarted:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for first callback")
	}
	sink.Write([]byte("second\n"))
	sink.Flush()

	sink.mu.Lock()
	queuedBatches := len(sink.deliveries)
	queuedSeq := uint64(0)
	if queuedBatches > 0 {
		queuedSeq = sink.deliveries[0].event.Seq
	}
	sink.mu.Unlock()
	if queuedBatches != 1 || queuedSeq != 2 {
		close(releaseFirst)
		t.Fatalf("second delivery entered callback early: queued batches=%d seq=%d", queuedBatches, queuedSeq)
	}
	close(releaseFirst)

	for want := uint64(1); want <= 2; want++ {
		select {
		case got := <-delivered:
			if got != want {
				t.Fatalf("delivery order: got seq=%d, want %d", got, want)
			}
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for seq=%d", want)
		}
	}
}

func TestLogSink_CloseIsSafeFromEmitCallback(t *testing.T) {
	done := make(chan error, 1)
	var sink *LogSink
	sink = NewLogSink(func(LogLinesEvent) {
		done <- sink.Close()
	})
	sink.Write([]byte("line\n"))
	sink.Flush()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Close from callback: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Close deadlocked inside emit callback")
	}
}

func TestLogSink_BoundsQueuedLinesBehindSlowEmitter(t *testing.T) {
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	delivered := make(chan LogLinesEvent, 2)
	sink := NewLogSink(func(event LogLinesEvent) {
		if event.Seq == 1 {
			close(firstStarted)
			<-releaseFirst
		}
		delivered <- event
	})

	sink.Write([]byte("first\n"))
	sink.Flush()
	select {
	case <-firstStarted:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for blocked callback")
	}
	sink.Write([]byte(strings.Repeat("queued\n", maxQueuedDeliveryLines+100)))

	sink.mu.Lock()
	queuedBatches := len(sink.deliveries)
	queuedLines := len(sink.deliveries[0].event.Lines)
	sink.mu.Unlock()
	if queuedBatches != 1 {
		t.Fatalf("queued batches=%d, want 1 coalesced batch", queuedBatches)
	}
	if queuedLines > maxQueuedDeliveryLines {
		t.Fatalf("queued lines=%d, limit=%d", queuedLines, maxQueuedDeliveryLines)
	}

	close(releaseFirst)
	sink.drain()
	for wantSeq := uint64(1); wantSeq <= 2; wantSeq++ {
		select {
		case event := <-delivered:
			if event.Seq != wantSeq {
				t.Fatalf("got seq=%d, want %d", event.Seq, wantSeq)
			}
			if wantSeq == 2 && !strings.Contains(event.Lines[0], "log-delivery-overflow") {
				t.Fatalf("overflow delivery missing warning: %q", event.Lines[0])
			}
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for seq=%d", wantSeq)
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
	sink.drain()

	mu.Lock()
	defer mu.Unlock()
	if len(got) != 1 {
		t.Fatalf("expected immediate flush, got %d events", len(got))
	}
}

func TestLogSink_SetFileWriter_OnOff(t *testing.T) {
	dir := t.TempDir()
	sink := NewLogSink(nil)
	sink.SetFileWriter(dir)
	_, _ = sink.Write([]byte("line1\n"))
	sink.Flush()
	files, _ := filepath.Glob(filepath.Join(dir, "yotta-*.log"))
	if len(files) != 1 {
		t.Fatalf("expected 1 log file, got %d", len(files))
	}

	// 关写文件 → 后续 line 不入新文件
	sink.SetFileWriter("")
	_, _ = sink.Write([]byte("line2\n"))
	sink.Flush()
	data, _ := os.ReadFile(files[0])
	if strings.Contains(string(data), "line2") {
		t.Fatalf("line2 should not be in file after SetFileWriter(\"\")")
	}
	// (file already closed by SetFileWriter("") above — no extra cleanup needed)
}

func TestLogSink_SetFileWriter_Reopen(t *testing.T) {
	dir := t.TempDir()
	sink := NewLogSink(nil)
	sink.SetFileWriter(dir)
	_, _ = sink.Write([]byte("a\n"))
	sink.Flush()
	sink.SetFileWriter("")
	sink.SetFileWriter(dir)
	_, _ = sink.Write([]byte("b\n"))
	sink.Flush()
	files, _ := filepath.Glob(filepath.Join(dir, "yotta-*.log"))
	data, _ := os.ReadFile(files[0])
	if !strings.Contains(string(data), "b") {
		t.Fatalf("line b should be in file after re-enable, got: %s", string(data))
	}
	sink.SetFileWriter("") // Windows: release file handle so t.TempDir cleanup can delete
}
