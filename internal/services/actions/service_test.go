package actions

import (
	"errors"
	"path/filepath"
	"sync"
	"testing"
)

var errBoom = errors.New("boom")

// ---- fakes ----

type fakeRunner struct {
	mu         sync.Mutex
	started    int
	stopped    int
	hwndSet    uintptr
	startErr   error
	lastAction *Action
}

func (f *fakeRunner) Start(a *Action) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.startErr != nil {
		return f.startErr
	}
	f.started++
	f.lastAction = a
	return nil
}
func (f *fakeRunner) Stop() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stopped++
	return nil
}
func (f *fakeRunner) SetHWND(h uintptr) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.hwndSet = h
}

type fakeLease struct {
	released bool
	mu       sync.Mutex
}

func (l *fakeLease) Release() {
	l.mu.Lock()
	l.released = true
	l.mu.Unlock()
}

type fakeBotGate struct {
	acquireErr error
	current    *fakeLease
}

func (g *fakeBotGate) AcquireBot(name string) (BotLease, error) {
	if g.acquireErr != nil {
		return nil, g.acquireErr
	}
	l := &fakeLease{}
	g.current = l
	return l, nil
}

type fakeGame struct {
	hwnd uintptr
	ok   bool
}

func (g *fakeGame) HWND() (uintptr, bool)       { return g.hwnd, g.ok }
func (g *fakeGame) BringToForeground(_ uintptr) {}

func newTestService(t *testing.T) (*Service, *fakeRunner, *fakeBotGate, *fakeGame) {
	t.Helper()
	store := NewStore(filepath.Join(t.TempDir(), "actions"))
	runner := &fakeRunner{}
	gate := &fakeBotGate{}
	game := &fakeGame{hwnd: 0xCAFE, ok: true}
	svc := NewService(store, runner, gate, game)
	return svc, runner, gate, game
}

// ---- tests ----

func TestService_CreateAssignsIDAndTimestamp(t *testing.T) {
	s, _, _, _ := newTestService(t)
	a, err := s.Create("x")
	if err != nil {
		t.Fatal(err)
	}
	if a.ID == "" {
		t.Error("ID 应自动分配")
	}
	if a.CreatedAt.IsZero() {
		t.Error("CreatedAt 应填")
	}
}

func TestService_UpdatePatchRunsNormalize(t *testing.T) {
	s, _, _, _ := newTestService(t)
	a, _ := s.Create("x")
	patch := `{"steps":[{"kind":"click_left","xRatio":0.5,"yRatio":0.5,"durationMs":50}]}`
	if err := s.Update(a.ID, patch); err != nil {
		t.Fatal(err)
	}
	got, _ := s.store.Get(a.ID)
	if got.Steps[0].ID == "" {
		t.Error("应自动填 Step.ID")
	}
}

func TestService_UpdateRejectsInvalidPatch(t *testing.T) {
	s, _, _, _ := newTestService(t)
	a, _ := s.Create("x")
	// 非法 step.kind 应被 Validate 拒绝
	patch := `{"steps":[{"kind":"bogus"}]}`
	if err := s.Update(a.ID, patch); err == nil {
		t.Error("应拒绝无效 step kind")
	}
}

func TestService_DeleteRemovesFromList(t *testing.T) {
	s, _, _, _ := newTestService(t)
	a, _ := s.Create("x")
	if err := s.Delete(a.ID); err != nil {
		t.Fatal(err)
	}
	if len(s.List()) != 0 {
		t.Error("Delete 后 List 应空")
	}
}

// ---- RunOnce / Stop ----

func TestService_RunOnce_NoGameWindow(t *testing.T) {
	s, runner, _, game := newTestService(t)
	game.ok = false
	a, _ := s.Create("x")
	if err := s.RunOnce(a.ID); err == nil {
		t.Error("无游戏窗口应返 error")
	}
	if runner.started != 0 {
		t.Error("无游戏窗口时不应 Start runner")
	}
}

func TestService_RunOnce_HappyPath(t *testing.T) {
	s, runner, gate, _ := newTestService(t)
	a, _ := s.Create("x")
	if err := s.RunOnce(a.ID); err != nil {
		t.Fatal(err)
	}
	if runner.started != 1 {
		t.Errorf("Runner.Start 应被调 1 次，got %d", runner.started)
	}
	if runner.hwndSet != 0xCAFE {
		t.Errorf("应 SetHWND 到 game.HWND，got 0x%x", runner.hwndSet)
	}
	if gate.current == nil || gate.current.released {
		t.Error("Start 成功时 lease 应被持有")
	}
	// emit "idle" → 应释放 lease
	s.OnRunnerEvent(a.ID, "idle")
	if !gate.current.released {
		t.Error("idle 后 lease 应释放")
	}
}

func TestService_RunOnce_StartFailureReleasesLease(t *testing.T) {
	s, runner, gate, _ := newTestService(t)
	runner.startErr = errBoom
	a, _ := s.Create("x")
	if err := s.RunOnce(a.ID); err == nil {
		t.Error("Runner.Start 失败应传播 error")
	}
	if gate.current == nil || !gate.current.released {
		t.Error("Start 失败 lease 必须立刻释放")
	}
}

func TestService_StopRunning_Delegates(t *testing.T) {
	s, runner, _, _ := newTestService(t)
	if err := s.StopRunning(); err != nil {
		t.Fatal(err)
	}
	if runner.stopped != 1 {
		t.Errorf("Runner.Stop 应被调 1 次，got %d", runner.stopped)
	}
}
