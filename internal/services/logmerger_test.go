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
