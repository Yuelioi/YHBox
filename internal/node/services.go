// internal/node/services.go
// stub services. 真实 backend 由 main.go 注入 (wire from wire_container.go).
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

func (stubVisionService) Match(_ context.Context, _ []string, _ float64, _ Geometry) (MatchHit, error) {
	return MatchHit{}, nil
}

func (stubVisionService) WaitMatch(_ context.Context, _ []string, _ float64, _ Geometry, _ time.Duration) (MatchHit, error) {
	return MatchHit{}, nil
}

func (stubVisionService) DualBarTrack(roi Geometry, inner, outer HSVRange, opts DualBarOptions) (DualColorBarResult, error) {
	return DualColorBarResult{}, nil
}

func (stubVisionService) DetectColor(roi Geometry, mode string, rng [6]int) (int, float64, float64, error) {
	return 0, 0, 0, nil
}

func (stubVisionService) DetectColorHSV(roi Geometry, hsv HSVRange) (int, float64, error) {
	return 0, 0, nil
}

func (stubVisionService) ROIColorScan(roi Geometry, hsv HSVRange, axis string, minPx, maxPx int) ([]ClusterEntry, error) {
	return nil, nil
}

func (stubVisionService) DetectColorBlobs(roi Geometry, mode string, rng [6]int, minArea int) ([]BlobEntry, error) {
	return nil, nil
}

func (stubVisionService) GridSignature(roi Geometry, gridSize int) ([]uint8, error) {
	return nil, nil
}

func (stubVisionService) FindColorSignature(_ Geometry, _ ColorSignature, _ int) (bool, Point, error) {
	return false, Point{}, nil
}

func (stubVisionService) DecodeQR(_ Geometry) ([]QRResult, error) {
	return nil, nil
}

func (stubVisionService) MatchAll(_ context.Context, _ []string, _ float64, _ int, _ Geometry) ([]TemplateMatch, error) {
	return nil, nil
}

// StubVisionService — test 用. main.go 注入真 wire_container.go::templateMatcherAdapter.
func StubVisionService() VisionService { return stubVisionService{} }

// ---- InputService ----

// stubInputService 测试用 no-op. 所有方法吞输入返 nil error.
type stubInputService struct{}

func (stubInputService) KeyDown(vk string) error                                           { return nil }
func (stubInputService) KeyUp(vk string) error                                             { return nil }
func (stubInputService) Click(x, y float64, button string, durationMs int) error           { return nil }
func (stubInputService) MouseMoveRel(dx, dy, durationMs int) error                         { return nil }
func (stubInputService) MoveTo(x, y float64) error                                         { return nil }
func (stubInputService) CursorRatio() (float64, float64, error)                            { return 0, 0, nil }
func (stubInputService) Scroll(x, y float64, notches int, horizontal bool) error           { return nil }
func (stubInputService) MouseDown(x, y float64, button string) error                       { return nil }
func (stubInputService) MouseUp(button string) error                                       { return nil }
func (stubInputService) Drag(x1, y1, x2, y2 float64, button string, durationMs int) error { return nil }
func (stubInputService) TypeText(s string) error                                           { return nil }

// StubInputService — test 用. main.go 注入 pkg/input.Backend 适配.
func StubInputService() InputService { return stubInputService{} }

// ---- VarStore ----

// stubVarStore in-memory map, mutex-safe — 测试 + 单 graph 实例够用.
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
// scope-aware 变种忽略 scope (stub 是单层 map, 无 frame). 真 wire 在 RuntimeContext adapter.
func NewStubVarStore() VarStore { return &stubVarStore{m: map[string]any{}} }

func (s *stubVarStore) GetScoped(name, _ string) (any, bool) { return s.Get(name) }
func (s *stubVarStore) SetScoped(name, _ string, v any)      { s.Set(name, v) }
func (s *stubVarStore) IncScoped(name, _ string, d float64) float64 {
	return s.Inc(name, d)
}

// LastChange — stub 不跟踪时间戳, 恒返 0.
func (s *stubVarStore) LastChange(_ string) int64 { return 0 }

// Names — 可选能力 (services/script.varNamer): Script $getter 注入用.
func (s *stubVarStore) Names() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, 0, len(s.m))
	for k := range s.m {
		out = append(out, k)
	}
	return out
}

// ---- ParamStore ----

// stubParamStore 测试用 no-op. Get 总返 (nil, false).
type stubParamStore struct{}

func (stubParamStore) Get(string) (any, bool) { return nil, false }

// NewStubParamStore — 测试用 no-op ParamStore.
func NewStubParamStore() ParamStore { return stubParamStore{} }

// ---- WindowService ----

type stubWindowService struct{}

func (stubWindowService) BringForeground() error                                                          { return nil }
func (stubWindowService) HWND() uintptr                                                                   { return 0 }
func (stubWindowService) ClientSize() (int, int, error)                                                   { return 0, 0, nil }
func (stubWindowService) SetActive(_ context.Context, _, _, _, _ string) error                            { return nil }

// StubWindowService — 测试用 no-op.
func StubWindowService() WindowService { return stubWindowService{} }

// ---- CaptureService ----

type stubCaptureService struct{}

func (stubCaptureService) Capture() ([]byte, error)                    { return nil, nil }
func (stubCaptureService) CaptureROI(roi Geometry) ([]byte, error)     { return nil, nil }

// StubCaptureService — 测试用. 返 nil bytes, 节点应当能处理 (skip write) 或自己报错.
func StubCaptureService() CaptureService { return stubCaptureService{} }

// ---- StopwatchStore ----

type stubStopwatchEntry struct {
	startAt time.Time
	stopAt  time.Time
	running bool
}

// stubStopwatchStore in-memory, mutex-safe.
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
		return // no-op
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
// main.go 不用这个, 直接 new ServiceBundle 塞真 backend.
//
// Snapshot 默认 nil — EvaluatePureData 检查 nil 跳过 wrap, 现有 PureData Evaluator
// 测试 (Expr 等) 走 stub Vars/Sys, 不需要 snapshot 行为.
func StubServices() ServiceBundle {
	return ServiceBundle{
		Vision:      StubVisionService(),
		Log:         DefaultLogService(),
		Input:       StubInputService(),
		Vars:        NewStubVarStore(),
		Params:      NewStubParamStore(),
		Window:      StubWindowService(),
		Capture:     StubCaptureService(),
		Stopwatches: NewStubStopwatchStore(),
		Snapshot:    nil,
	}
}
