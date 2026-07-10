package services

import (
	"sync"
	"testing"
	"time"
)

func newTestMerger() (*LogMerger, *[]string, *sync.Mutex) {
	var mu sync.Mutex
	var files []string
	m := NewLogMerger(
		func(name string, data any) {},
		func(line string) { mu.Lock(); files = append(files, line); mu.Unlock() },
	)
	return m, &files, &mu
}

func TestLogMerger_ConsecutiveSameCoalesces(t *testing.T) {
	m, files, mu := newTestMerger()
	defer m.Close()
	for i := 0; i < 42; i++ {
		m.Add("c", "n_a", "DetectColor", "DetectColor(n_a) out{hit=false}", "out{hit=false}", false)
	}
	m.Add("c", "n_a", "DetectColor", "DetectColor(n_a) out{hit=true}", "out{hit=true}", false)
	mu.Lock()
	defer mu.Unlock()
	found := false
	for _, l := range *files {
		if l == "DetectColor(n_a) out{hit=false} ×42" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected '×42' finalized to file, got %#v", *files)
	}
}

func TestLogMerger_InterleavedNodesEachCoalesce(t *testing.T) {
	m, files, mu := newTestMerger()
	defer m.Close()
	for i := 0; i < 5; i++ {
		m.Add("c", "n_a", "A", "A(n_a) out{v=1}", "out{v=1}", false)
		m.Add("c", "n_b", "B", "B(n_b) out{v=2}", "out{v=2}", false)
	}
	m.FlushContainer("c")
	mu.Lock()
	defer mu.Unlock()
	cnt := func(s string) int {
		n := 0
		for _, l := range *files {
			if l == s {
				n++
			}
		}
		return n
	}
	if cnt("A(n_a) out{v=1} ×5") != 1 || cnt("B(n_b) out{v=2} ×5") != 1 {
		t.Fatalf("interleaved nodes must each coalesce to ×5, got %#v", *files)
	}
}

func TestLogMerger_IdleTimeoutDoesNotSplitActive(t *testing.T) {
	m, files, mu := newTestMerger()
	m.idleTimeout = 300 * time.Millisecond
	defer m.Close()
	for i := 0; i < 5; i++ {
		m.Add("c", "n_a", "A", "A(n_a) out{v=1}", "out{v=1}", false)
		time.Sleep(40 * time.Millisecond)
	}
	m.FlushContainer("c")
	mu.Lock()
	defer mu.Unlock()
	for _, l := range *files {
		if l == "A(n_a) out{v=1} ×5" {
			return
		}
	}
	t.Fatalf("active segment (gaps<idle) must stay one ×5, got %#v", *files)
}

// 复现「短图经常没日志」: 容器在 250ms tick 前跑完, dirty 段只有 FlushContainer 收尾.
// FlushContainer 必须把未刷段 emit 到前端 (final=true), 否则 GUI 面板一条都收不到.
func TestLogMerger_FlushEmitsUnflushedSegments(t *testing.T) {
	var mu sync.Mutex
	var emitted []map[string]any
	m := NewLogMerger(
		func(name string, data any) {
			if name != "container:node-dump-batch" {
				return
			}
			d, _ := data.(map[string]any)
			es, _ := d["entries"].([]map[string]any)
			mu.Lock()
			emitted = append(emitted, es...)
			mu.Unlock()
		},
		func(line string) {},
	)
	defer m.Close()
	// 模拟短图: 两节点各 dump 一次, 不等 tick 立刻 Flush.
	m.Add("c", "n_a", "A", "A(n_a) out{v=1}", "out{v=1}", false)
	m.Add("c", "n_b", "B", "B(n_b) out{v=2}", "out{v=2}", false)
	m.FlushContainer("c")
	mu.Lock()
	defer mu.Unlock()
	if len(emitted) != 2 {
		t.Fatalf("FlushContainer 必须 emit 2 个未刷段到前端, got %d: %#v", len(emitted), emitted)
	}
	for _, e := range emitted {
		if e["final"] != true {
			t.Errorf("flush emit 应 final=true (定版), got %#v", e)
		}
	}
}

func TestLogMerger_ErrorLineNotCoalesced(t *testing.T) {
	m, files, mu := newTestMerger()
	defer m.Close()
	m.Add("c", "n_a", "A", "A(n_a) in{x=1} err=boom", "in{x=1} err=boom", true)
	m.Add("c", "n_a", "A", "A(n_a) in{x=1} err=boom", "in{x=1} err=boom", true)
	mu.Lock()
	defer mu.Unlock()
	n := 0
	for _, l := range *files {
		if l == "A(n_a) in{x=1} err=boom" {
			n++
		}
	}
	if n != 2 {
		t.Fatalf("error lines must not coalesce, got %d", n)
	}
}

func TestLogMerger_CloseDrainsPendingSegmentsAndIsIdempotent(t *testing.T) {
	m, files, mu := newTestMerger()
	m.Add("c", "n_a", "A", "A(n_a) out{v=1}", "out{v=1}", false)
	m.Close()
	m.Close()

	mu.Lock()
	defer mu.Unlock()
	if len(*files) != 1 || (*files)[0] != "A(n_a) out{v=1}" {
		t.Fatalf("drained files = %#v", *files)
	}
}
