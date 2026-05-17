package runtime

import (
	"sync"
	"time"
)

// stopwatchState 单 key 时序记录. running=true 时 stoppedAt 无意义.
type stopwatchState struct {
	startAt   time.Time
	stoppedAt time.Time
	running   bool
}

// stopwatchTable per-ContainerRunner 多 stopwatch 存储, key 用户命名空间.
// 与 SetVar 的 $vars.* 表独立 (即同名 key 不冲突), 见 spec v3 Phase C.
//
// 并发: ContainerRunner 在 Parallel/Race 子分支下会并发 dispatch 节点, stopwatch 操作
// 必须互斥. mu 粒度足够 — 节点只调用 start/stop/read, 单次操作均为 O(1).
type stopwatchTable struct {
	mu sync.Mutex
	m  map[string]*stopwatchState
}

func newStopwatchTable() *stopwatchTable {
	return &stopwatchTable{m: make(map[string]*stopwatchState)}
}

// start 已存在同 key (无论 running 与否) → reset. 不报错 (用户预期"重新开始").
func (s *stopwatchTable) start(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[key] = &stopwatchState{startAt: time.Now(), running: true}
}

// stop key 不存在 → return false (调用方决定是否 warn). 存在 → 记 stoppedAt.
// 已停止的 key 再次 stop: 视为 no-op (保持首次 stoppedAt, 不刷). 返 true.
func (s *stopwatchTable) stop(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, ok := s.m[key]
	if !ok {
		return false
	}
	if !st.running {
		return true
	}
	st.stoppedAt = time.Now()
	st.running = false
	return true
}

// read 不存在 → 0. running → now-start. stopped → stoppedAt-start. 返毫秒.
func (s *stopwatchTable) read(key string) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, ok := s.m[key]
	if !ok {
		return 0
	}
	if st.running {
		return time.Since(st.startAt).Milliseconds()
	}
	return st.stoppedAt.Sub(st.startAt).Milliseconds()
}
