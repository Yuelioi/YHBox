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
