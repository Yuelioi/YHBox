// internal/node/services.go
// Phase 1-4 stub services. Phase 5 由 main.go 注入真实 backend (wire from wire_container.go).
//
// 全部 stub 实现 no-op / zero value, 不 panic. 测试用; production 真节点应 main.go
// inject 真 backend, 节点拿到 nil service 调方法才 panic.
package node

import (
	"context"
	"log"
	"sync"
	"time"
)

// ---- LogService ----

type stdoutLogService struct{}

func (stdoutLogService) Debug(format string, args ...any) { log.Printf("[DEBUG] "+format, args...) }
func (stdoutLogService) Info(format string, args ...any)  { log.Printf("[INFO] "+format, args...) }
func (stdoutLogService) Warn(format string, args ...any)  { log.Printf("[WARN] "+format, args...) }

// DefaultLogService stdout-based, test 用. main.go 注入 zerolog-based 替换.
func DefaultLogService() LogService { return stdoutLogService{} }

// ---- VisionService ----

// stubVisionService 测试用. 任何 key 都返 nil/0/nil (always miss).
type stubVisionService struct{}

func (stubVisionService) Match(key string, threshold float64) (*Point, float64, error) {
	return nil, 0, nil
}

func (stubVisionService) WaitMatch(ctx context.Context, key string, threshold float64, timeout time.Duration) (*Point, float64, error) {
	return nil, 0, nil
}

func (stubVisionService) BarTrack(roi Rect) (BarTrackResult, error) {
	return BarTrackResult{}, nil
}

// StubVisionService — Phase 1-4 test 用. Phase 5 main.go 注入真 wire_container.go::templateMatcherAdapter.
func StubVisionService() VisionService { return stubVisionService{} }

// ---- InputService ----

// stubInputService 测试用 no-op. 所有方法吞输入返 nil error.
type stubInputService struct{}

func (stubInputService) KeyPress(vk string, durationMs int) error                          { return nil }
func (stubInputService) KeyDown(vk string) error                                           { return nil }
func (stubInputService) KeyUp(vk string) error                                             { return nil }
func (stubInputService) Click(x, y float64, button string, durationMs int) error           { return nil }
func (stubInputService) MouseMoveRel(dx, dy, durationMs int) error                         { return nil }
func (stubInputService) Scroll(x, y float64, notches int) error                            { return nil }
func (stubInputService) MouseDown(x, y float64, button string) error                       { return nil }
func (stubInputService) MouseUp(button string) error                                       { return nil }

// StubInputService — Phase 4 test 用. Phase 5 main.go 注入 pkg/input.Backend 适配.
func StubInputService() InputService { return stubInputService{} }

// ---- VarStore ----

// stubVarStore in-memory map, mutex-safe — 测试 + 单 graph 实例够用.
// Phase 5 wire 真 RuntimeContext.SetVar/Vars 时这个 stub 报废.
type stubVarStore struct {
	mu sync.RWMutex
	m  map[string]any
}

func (s *stubVarStore) Get(name string) (any, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.m[name]
	return v, ok
}

func (s *stubVarStore) Set(name string, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.m == nil {
		s.m = map[string]any{}
	}
	s.m[name] = value
}

func (s *stubVarStore) Inc(name string, delta float64) float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.m == nil {
		s.m = map[string]any{}
	}
	cur := 0.0
	switch v := s.m[name].(type) {
	case float64:
		cur = v
	case int:
		cur = float64(v)
	case int64:
		cur = float64(v)
	}
	newV := cur + delta
	s.m[name] = newV
	return newV
}

// NewStubVarStore — 测试用 in-memory VarStore. 每次 new 一个独立实例.
func NewStubVarStore() VarStore { return &stubVarStore{m: map[string]any{}} }

// ---- SysStore ----

// StubSysStore in-memory key→any. Phase 5 wire SysState resolveSysPath 时这 stub
// 仍可能被测试直接复用 (preset 几个 path 值 via SetForTest).
type StubSysStore struct {
	mu sync.RWMutex
	m  map[string]any
}

func (s *StubSysStore) Get(path string) (any, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.m[path]
	return v, ok
}

// SetForTest 仅 test 用 — production wire 不该有 Set, sys 只读.
func (s *StubSysStore) SetForTest(path string, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.m == nil {
		s.m = map[string]any{}
	}
	s.m[path] = value
}

// NewStubSysStore — 测试用. 返 *StubSysStore 露 SetForTest 给 test preset 字段.
func NewStubSysStore() *StubSysStore { return &StubSysStore{m: map[string]any{}} }

// ---- WindowService ----

type stubWindowService struct{}

func (stubWindowService) BringForeground() error          { return nil }
func (stubWindowService) HWND() uintptr                   { return 0 }
func (stubWindowService) ClientSize() (int, int, error)   { return 0, 0, nil }

// StubWindowService — 测试用 no-op.
func StubWindowService() WindowService { return stubWindowService{} }

// ---- CaptureService ----

type stubCaptureService struct{}

func (stubCaptureService) Capture() ([]byte, error)                          { return nil, nil }
func (stubCaptureService) CaptureROI(x, y, w, h int) ([]byte, error)         { return nil, nil }

// StubCaptureService — 测试用. 返 nil bytes, 节点应当能处理 (skip write) 或自己报错.
func StubCaptureService() CaptureService { return stubCaptureService{} }

// ---- StopwatchStore ----

type stubStopwatchEntry struct {
	startAt time.Time
	stopAt  time.Time
	running bool
}

// stubStopwatchStore in-memory, mutex-safe. 镜像老 stopwatchTable 语义.
type stubStopwatchStore struct {
	mu sync.Mutex
	m  map[string]*stubStopwatchEntry
}

func (s *stubStopwatchStore) Start(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.m == nil {
		s.m = map[string]*stubStopwatchEntry{}
	}
	s.m[key] = &stubStopwatchEntry{startAt: time.Now(), running: true}
}

func (s *stubStopwatchStore) Stop(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, ok := s.m[key]
	if !ok {
		return // no-op, 镜像老语义
	}
	if !st.running {
		return // 已停, 保留首次 stopAt
	}
	st.stopAt = time.Now()
	st.running = false
}

func (s *stubStopwatchStore) Read(key string) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, ok := s.m[key]
	if !ok {
		return 0
	}
	if st.running {
		return time.Since(st.startAt).Milliseconds()
	}
	return st.stopAt.Sub(st.startAt).Milliseconds()
}

// NewStubStopwatchStore — 测试用 in-memory store.
func NewStubStopwatchStore() StopwatchStore { return &stubStopwatchStore{m: map[string]*stubStopwatchEntry{}} }

// ---- ServiceBundle helpers ----

// StubServices 返一个全 stub 填充的 ServiceBundle, test 用.
// Phase 5 main.go 不用这个, 直接 new ServiceBundle 塞真 backend.
func StubServices() ServiceBundle {
	return ServiceBundle{
		Vision:      StubVisionService(),
		Log:         DefaultLogService(),
		Input:       StubInputService(),
		Vars:        NewStubVarStore(),
		Sys:         NewStubSysStore(),
		Window:      StubWindowService(),
		Capture:     StubCaptureService(),
		Stopwatches: NewStubStopwatchStore(),
	}
}
