package services

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func TestParseSystemLogEntryPromotesWorkflowAttribution(t *testing.T) {
	entry := parseSystemLogEntry(`{"time":"2026-07-16T00:00:00Z","level":"info","tag":"WORKFLOW","message":"node completed","graphId":"g1","nodeId":"n1","invocationId":"i1","attempt":2,"durationMs":12,"failure":{"code":"ai.generation_failed","sourceNodeId":"ai-generate"}}`)
	if entry.Source != "WF" || entry.GraphID != "g1" || entry.NodeID != "n1" || entry.InvocationID != "i1" || entry.Attempt != 2 {
		t.Fatalf("workflow entry = %#v", entry)
	}
	fields, ok := entry.Fields.(map[string]any)
	if !ok || fields["durationMs"] != float64(12) {
		t.Fatalf("workflow fields = %#v", entry.Fields)
	}
	failure, ok := fields["failure"].(map[string]any)
	if !ok || failure["code"] != "ai.generation_failed" || failure["sourceNodeId"] != "ai-generate" {
		t.Fatalf("workflow failure fields = %#v", fields["failure"])
	}
}

func TestLogSink_DebounceFlush(t *testing.T) {
	var got []LogBatchEvent
	var mu sync.Mutex
	sink := NewLogSink(func(e LogBatchEvent) {
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
	if len(got[0].Entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(got[0].Entries))
	}
	if got[0].Seq != 1 {
		t.Errorf("expected seq=1, got %d", got[0].Seq)
	}
}

func TestLogSink_SeqMonotonic(t *testing.T) {
	seqs := make(chan uint64, 3)
	sink := NewLogSink(func(e LogBatchEvent) {
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
	sink := NewLogSink(func(e LogBatchEvent) {
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
	sink = NewLogSink(func(LogBatchEvent) {
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

func TestLogSinkCloseClosesFileAndRejectsLateWrites(t *testing.T) {
	dir := t.TempDir()
	sink := NewLogSink(nil)
	sink.SetFileWriter(dir)
	if _, err := sink.Write([]byte("before close\n")); err != nil {
		t.Fatal(err)
	}
	if err := sink.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := sink.Write([]byte("after close\n")); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("late Write error = %v", err)
	}
	files, err := filepath.Glob(filepath.Join(dir, "yotta-*.log"))
	if err != nil || len(files) != 1 {
		t.Fatalf("log files = %v, err = %v", files, err)
	}
	if err := os.Remove(files[0]); err != nil {
		t.Fatalf("log file remained open after Close: %v", err)
	}
}

func TestLogSink_BoundsQueuedLinesBehindSlowEmitter(t *testing.T) {
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	delivered := make(chan LogBatchEvent, 2)
	sink := NewLogSink(func(event LogBatchEvent) {
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
	queuedLines := len(sink.deliveries[0].event.Entries)
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
			if wantSeq == 2 && event.Dropped == 0 {
				t.Fatal("overflow delivery missing dropped count")
			}
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for seq=%d", wantSeq)
		}
	}
}

func TestLogSink_MaxBatchTriggersImmediateFlush(t *testing.T) {
	var emitCount atomic.Int32
	sink := NewLogSink(func(e LogBatchEvent) {
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
	var got []LogBatchEvent
	sink := NewLogSink(func(e LogBatchEvent) {
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
	if got[0].Entries[0].Message != "incomplete continued" {
		t.Errorf("partial concat wrong: %q", got[0].Entries[0].Message)
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
	var got []LogBatchEvent
	sink := NewLogSink(func(e LogBatchEvent) {
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

func TestLogRuntime_MasterOffStopsSourceAndAllDestinations(t *testing.T) {
	defer zerolog.SetGlobalLevel(zerolog.InfoLevel)
	dir := t.TempDir()
	emitted := 0
	sink := NewLogSink(func(LogBatchEvent) { emitted++ })
	runtime := NewLogRuntime(sink)
	settings := defaultSettings().UI.Logger
	settings.Enabled = false
	settings.FileDir = dir
	runtime.Configure(settings)

	logger := zerolog.New(sink)
	logger.Error().Str("expensive", strings.Repeat("x", 128)).Msg("must not be built")
	sink.Flush()

	if emitted != 0 || sink.Snapshot() != "" {
		t.Fatalf("disabled logger produced output: emitted=%d snapshot=%q", emitted, sink.Snapshot())
	}
	files, err := filepath.Glob(filepath.Join(dir, "yotta-*.log"))
	if err != nil || len(files) != 0 {
		t.Fatalf("disabled logger created files: files=%v err=%v", files, err)
	}
}

func TestLogRuntime_LiveAndFileDestinationsAreIndependent(t *testing.T) {
	defer zerolog.SetGlobalLevel(zerolog.InfoLevel)
	dir := t.TempDir()
	emitted := 0
	sink := NewLogSink(func(LogBatchEvent) { emitted++ })
	runtime := NewLogRuntime(sink)
	settings := defaultSettings().UI.Logger
	settings.LiveView = false
	settings.WriteFile = true
	settings.FileDir = dir
	runtime.Configure(settings)

	logger := zerolog.New(sink)
	logger.Info().Msg("file only")
	sink.Flush()
	if emitted != 0 || sink.Snapshot() != "" {
		t.Fatalf("file-only logger reached presentation: emitted=%d snapshot=%q", emitted, sink.Snapshot())
	}
	files, _ := filepath.Glob(filepath.Join(dir, "yotta-*.log"))
	if len(files) != 1 {
		t.Fatalf("file-only logger files=%v", files)
	}
	data, _ := os.ReadFile(files[0])
	if !strings.Contains(string(data), "file only") {
		t.Fatalf("file-only logger did not persist message: %s", data)
	}
	_ = sink.Close()
}

func TestLogRuntime_MinimumLevelFiltersBeforeSink(t *testing.T) {
	defer zerolog.SetGlobalLevel(zerolog.InfoLevel)
	var got LogBatchEvent
	sink := NewLogSink(func(event LogBatchEvent) { got = event })
	runtime := NewLogRuntime(sink)
	settings := defaultSettings().UI.Logger
	settings.Level = "warn"
	settings.WriteFile = false
	runtime.Configure(settings)

	logger := zerolog.New(sink)
	logger.Info().Msg("filtered")
	logger.Warn().Msg("kept")
	sink.drain()
	if len(got.Entries) != 1 || got.Entries[0].Message != "kept" {
		t.Fatalf("minimum level entries=%#v", got.Entries)
	}
}

func BenchmarkLogSinkWrite(b *testing.B) {
	line := []byte(`{"time":"2026-07-14T00:00:00Z","level":"info","message":"benchmark"}` + "\n")
	b.Run("stream-disabled", func(b *testing.B) {
		sink := NewLogSink(nil)
		sink.SetStreamEnabled(false)
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = sink.Write(line)
		}
	})
	b.Run("stream-enabled", func(b *testing.B) {
		sink := NewLogSink(nil)
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = sink.Write(line)
		}
	})
}

func BenchmarkLoggerSourceDisabled(b *testing.B) {
	defer zerolog.SetGlobalLevel(zerolog.InfoLevel)
	sink := NewLogSink(nil)
	runtime := NewLogRuntime(sink)
	settings := defaultSettings().UI.Logger
	settings.Enabled = false
	runtime.Configure(settings)
	logger := zerolog.New(sink)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info().Str("field", "value").Msg("disabled")
	}
}
