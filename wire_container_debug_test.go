package main

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/yottaapp/yotta/internal/nodes/all"
	"github.com/yottaapp/yotta/internal/services/container"
	containerruntime "github.com/yottaapp/yotta/internal/services/container/runtime"
	"github.com/yottaapp/yotta/internal/services/execution"
)

type lifecycleDebugRunner struct {
	startStarted chan struct{}
	blockStart   bool
	stepStarted  chan struct{}
	stopStarted  chan struct{}
	stopRelease  chan struct{}
	stops        atomic.Int32
}

type terminalDebugRunner struct{ *lifecycleDebugRunner }

func (*terminalDebugRunner) StepOnce(context.Context) (containerruntime.DebugStepResult, error) {
	return containerruntime.DebugStepResult{Finished: true}, nil
}

func (r *lifecycleDebugRunner) StartRuntime(ctx context.Context) error {
	if r.startStarted != nil {
		select {
		case r.startStarted <- struct{}{}:
		default:
		}
	}
	if r.blockStart {
		<-ctx.Done()
		return ctx.Err()
	}
	return nil
}
func (r *lifecycleDebugRunner) StopRuntime() {
	r.stops.Add(1)
	if r.stopStarted != nil {
		select {
		case r.stopStarted <- struct{}{}:
		default:
		}
	}
	if r.stopRelease != nil {
		<-r.stopRelease
	}
}
func (*lifecycleDebugRunner) SeedFromEntry() error      { return nil }
func (*lifecycleDebugRunner) SeedFromNode(string) error { return nil }
func (r *lifecycleDebugRunner) StepOnce(ctx context.Context) (containerruntime.DebugStepResult, error) {
	select {
	case r.stepStarted <- struct{}{}:
	default:
	}
	<-ctx.Done()
	return containerruntime.DebugStepResult{}, ctx.Err()
}
func (*lifecycleDebugRunner) QueueSnapshot() []containerruntime.ExecToken {
	return []containerruntime.ExecToken{{NodeID: "node", InPin: "In"}}
}
func (*lifecycleDebugRunner) NodeKind(string) string      { return "Sleep" }
func (*lifecycleDebugRunner) VarSnapshot() map[string]any { return nil }

func TestDebugManagerCloseStopsPausedRunnerAndRejectsRestart(t *testing.T) {
	runner := &lifecycleDebugRunner{stepStarted: make(chan struct{}, 1)}
	mgr := newContainerDebugManagerWithRunner(func(string) (debugRunner, error) { return runner, nil }, nil, nil)
	if _, err := mgr.DebugStart("container", container.DebugStartOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := mgr.CloseContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := runner.stops.Load(); got != 1 {
		t.Fatalf("StopRuntime calls = %d", got)
	}
	if _, err := mgr.DebugStart("container", container.DebugStartOptions{}); !errors.Is(err, errDebugManagerClosed) {
		t.Fatalf("DebugStart after close error = %v", err)
	}
}

func TestDebugManagerCloseCancelsAndWaitsForActiveStep(t *testing.T) {
	runner := &lifecycleDebugRunner{stepStarted: make(chan struct{}, 1)}
	mgr := newContainerDebugManagerWithRunner(func(string) (debugRunner, error) { return runner, nil }, nil, nil)
	state, err := mgr.DebugStart("container", container.DebugStartOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.DebugStep(state.SessionID); err != nil {
		t.Fatal(err)
	}
	select {
	case <-runner.stepStarted:
	case <-time.After(time.Second):
		t.Fatal("debug step did not start")
	}
	if err := mgr.CloseContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := runner.stops.Load(); got != 1 {
		t.Fatalf("StopRuntime calls = %d", got)
	}
}

func TestDebugManagerCloseWaitForStartingObeysContext(t *testing.T) {
	runner := &lifecycleDebugRunner{stepStarted: make(chan struct{}, 1)}
	started := make(chan struct{})
	release := make(chan struct{})
	mgr := newContainerDebugManagerWithRunner(func(string) (debugRunner, error) {
		close(started)
		<-release
		return runner, nil
	}, nil, nil)
	startResult := make(chan error, 1)
	go func() {
		_, err := mgr.DebugStart("container", container.DebugStartOptions{})
		startResult <- err
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("DebugStart did not enter factory")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if err := mgr.CloseContext(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("CloseContext error = %v", err)
	}
	close(release)
	select {
	case err := <-startResult:
		if !errors.Is(err, errDebugManagerClosed) {
			t.Fatalf("in-flight DebugStart error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("in-flight DebugStart did not finish cleanup")
	}
	if err := mgr.CloseContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := runner.stops.Load(); got != 1 {
		t.Fatalf("StopRuntime calls = %d", got)
	}
}

func TestDebugManagerCloseWaitForRunnerCleanupObeysContext(t *testing.T) {
	runner := &lifecycleDebugRunner{
		stepStarted: make(chan struct{}, 1),
		stopStarted: make(chan struct{}, 1),
		stopRelease: make(chan struct{}),
	}
	mgr := newContainerDebugManagerWithRunner(func(string) (debugRunner, error) { return runner, nil }, nil, nil)
	if _, err := mgr.DebugStart("container", container.DebugStartOptions{}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if err := mgr.CloseContext(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("CloseContext error = %v", err)
	}
	select {
	case <-runner.stopStarted:
	case <-time.After(time.Second):
		t.Fatal("runner cleanup did not start")
	}
	close(runner.stopRelease)
	ctx2, cancel2 := context.WithTimeout(context.Background(), time.Second)
	defer cancel2()
	if err := mgr.CloseContext(ctx2); err != nil {
		t.Fatal(err)
	}
}

func TestDebugManagerCloseCancelsStartingRuntime(t *testing.T) {
	runner := &lifecycleDebugRunner{
		startStarted: make(chan struct{}, 1),
		stepStarted:  make(chan struct{}, 1),
		blockStart:   true,
	}
	mgr := newContainerDebugManagerWithRunner(func(string) (debugRunner, error) { return runner, nil }, nil, nil)
	startResult := make(chan error, 1)
	go func() {
		_, err := mgr.DebugStart("container", container.DebugStartOptions{})
		startResult <- err
	}()
	select {
	case <-runner.startStarted:
	case <-time.After(time.Second):
		t.Fatal("StartRuntime did not start")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := mgr.CloseContext(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-startResult:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("DebugStart error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("DebugStart did not observe shutdown cancellation")
	}
	if got := runner.stops.Load(); got != 1 {
		t.Fatalf("StopRuntime calls = %d", got)
	}
}

func TestDebugManagerCloseWaitsForConcurrentDebugStopCleanup(t *testing.T) {
	runner := &lifecycleDebugRunner{
		stepStarted: make(chan struct{}, 1),
		stopStarted: make(chan struct{}, 1),
		stopRelease: make(chan struct{}),
	}
	mgr := newContainerDebugManagerWithRunner(func(string) (debugRunner, error) { return runner, nil }, nil, nil)
	state, err := mgr.DebugStart("container", container.DebugStartOptions{})
	if err != nil {
		t.Fatal(err)
	}
	stopResult := make(chan error, 1)
	go func() {
		_, err := mgr.DebugStop(state.SessionID)
		stopResult <- err
	}()
	select {
	case <-runner.stopStarted:
	case <-time.After(time.Second):
		t.Fatal("DebugStop cleanup did not start")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if err := mgr.CloseContext(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("CloseContext error = %v", err)
	}
	close(runner.stopRelease)
	select {
	case err := <-stopResult:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("DebugStop cleanup did not finish")
	}
	ctx2, cancel2 := context.WithTimeout(context.Background(), time.Second)
	defer cancel2()
	if err := mgr.CloseContext(ctx2); err != nil {
		t.Fatal(err)
	}
}

func TestDebugManagerCloseDuringTerminalCleanupRemovesSession(t *testing.T) {
	base := &lifecycleDebugRunner{
		stepStarted: make(chan struct{}, 1),
		stopStarted: make(chan struct{}, 1),
		stopRelease: make(chan struct{}),
	}
	runner := &terminalDebugRunner{lifecycleDebugRunner: base}
	mgr := newContainerDebugManagerWithRunner(func(string) (debugRunner, error) { return runner, nil }, nil, nil)
	state, err := mgr.DebugStart("container", container.DebugStartOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.DebugStep(state.SessionID); err != nil {
		t.Fatal(err)
	}
	select {
	case <-base.stopStarted:
	case <-time.After(time.Second):
		t.Fatal("terminal cleanup did not start")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if err := mgr.CloseContext(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("CloseContext error = %v", err)
	}
	close(base.stopRelease)
	ctx2, cancel2 := context.WithTimeout(context.Background(), time.Second)
	defer cancel2()
	if err := mgr.CloseContext(ctx2); err != nil {
		t.Fatal(err)
	}
}

func newWireDebugTestRunner(c *container.Container) func(string) (*containerruntime.ContainerRunner, error) {
	return func(id string) (*containerruntime.ContainerRunner, error) {
		if id != c.ID {
			return nil, errContainerNotFoundForDebug
		}
		rt := containerruntime.NewRuntimeContext(c, execution.NewInputBus(), containerruntime.NoopMatcher{}, nil, nil, nil, 0)
		return containerruntime.NewContainerRunner(rt), nil
	}
}

func TestDebugManagerStartRejectsWorkerBusy(t *testing.T) {
	c := &container.Container{ID: "c1", Name: "c1", Graph: container.Graph{Nodes: []container.GraphNode{{ID: "start", Kind: "Start"}}}}
	mgr := newContainerDebugManager(newWireDebugTestRunner(c), nil, func() bool { return true })

	_, err := mgr.DebugStart("c1", container.DebugStartOptions{})
	if err == nil || err.Error() != "debug_run_busy" {
		t.Fatalf("DebugStart error = %v, want debug_run_busy", err)
	}
}

func TestDebugManagerStepEmitsPausedState(t *testing.T) {
	c := &container.Container{
		ID:   "c1",
		Name: "c1",
		Vars: []container.VarDecl{{Name: "x", Type: "number", Default: 0.0}},
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "set", Kind: "SetVar", Config: map[string]any{
					"VarName": "x",
					"Scope":   "global",
					"literal": map[string]any{"Value": 7.0},
				}},
				{ID: "stop", Kind: "Stop"},
			},
			Edges: []container.GraphEdge{
				{From: "start.Done", To: "set.In"},
				{From: "set.Done", To: "stop.In"},
			},
		},
	}
	var (
		mu     sync.Mutex
		states []container.DebugSessionState
	)
	emit := func(name string, data any) {
		if name != "debug:state" {
			return
		}
		state, ok := data.(container.DebugSessionState)
		if !ok {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		states = append(states, state)
	}
	mgr := newContainerDebugManager(newWireDebugTestRunner(c), emit, func() bool { return false })

	start, err := mgr.DebugStart("c1", container.DebugStartOptions{})
	if err != nil {
		t.Fatalf("DebugStart: %v", err)
	}
	if start.Status != container.DebugStatusPaused || start.CurrentNodeID != "set" {
		t.Fatalf("start state = %+v", start)
	}
	step, err := mgr.DebugStep(start.SessionID)
	if err != nil {
		t.Fatalf("DebugStep: %v", err)
	}
	if step.Status != container.DebugStatusStepping {
		t.Fatalf("step state = %+v, want stepping", step)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		state, _ := mgr.DebugState(start.SessionID)
		if state.Status == container.DebugStatusPaused && state.LastNodeID == "set" {
			if state.CurrentNodeID != "stop" || state.LastExit != "Done" {
				t.Fatalf("paused state = %+v", state)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	mu.Lock()
	defer mu.Unlock()
	t.Fatalf("timed out waiting for paused debug state; emitted=%+v", states)
}

func TestDebugManagerStepSkipsDisabledDownstreamNode(t *testing.T) {
	c := &container.Container{
		ID:   "c1",
		Name: "c1",
		Vars: []container.VarDecl{{Name: "x", Type: "number", Default: 0.0}},
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "set", Kind: "SetVar", Config: map[string]any{
					"VarName": "x",
					"Scope":   "global",
					"literal": map[string]any{"Value": 7.0},
				}},
				{ID: "disabled", Kind: "Sleep", Disabled: true, Config: map[string]any{
					"literal": map[string]any{"Duration": 9999.0},
				}},
				{ID: "stop", Kind: "Stop"},
			},
			Edges: []container.GraphEdge{
				{From: "start.Done", To: "set.In"},
				{From: "set.Done", To: "disabled.In"},
				{From: "disabled.Done", To: "stop.In"},
			},
		},
	}
	mgr := newContainerDebugManager(newWireDebugTestRunner(c), nil, func() bool { return false })

	state, err := mgr.DebugStart("c1", container.DebugStartOptions{})
	if err != nil {
		t.Fatalf("DebugStart: %v", err)
	}
	if state.Status != container.DebugStatusPaused || state.CurrentNodeID != "set" {
		t.Fatalf("start state = %+v, want paused at set", state)
	}
	if _, err := mgr.DebugStep(state.SessionID); err != nil {
		t.Fatalf("DebugStep: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		got, err := mgr.DebugState(state.SessionID)
		if err != nil {
			t.Fatalf("DebugState: %v", err)
		}
		if got.Status == container.DebugStatusPaused && got.LastNodeID == "set" {
			if got.CurrentNodeID != "stop" || len(got.Queue) != 1 || got.Queue[0].NodeID == "disabled" {
				t.Fatalf("paused state = %+v, want disabled skipped to stop", got)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timed out waiting for disabled downstream node to be skipped")
}

func TestDebugManagerStepAndroidTargetPausesAtNextNode(t *testing.T) {
	c := &container.Container{
		ID:   "c1",
		Name: "c1",
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "android-target", Kind: "AndroidTarget", Config: map[string]any{
					"literal": map[string]any{
						"Serial": "127.0.0.1:16384",
						"Name":   "SDY AN00",
						"Width":  1280.0,
						"Height": 720.0,
					},
				}},
				{ID: "start-app", Kind: "AndroidStartApp", Config: map[string]any{
					"literal": map[string]any{"Package": "com.example.app"},
				}},
			},
			Edges: []container.GraphEdge{
				{From: "start.Done", To: "android-target.In"},
				{From: "android-target.Done", To: "start-app.In"},
			},
		},
	}
	mgr := newContainerDebugManager(newWireDebugTestRunner(c), nil, func() bool { return false })

	state, err := mgr.DebugStart("c1", container.DebugStartOptions{})
	if err != nil {
		t.Fatalf("DebugStart: %v", err)
	}
	if state.Status != container.DebugStatusPaused || state.CurrentNodeID != "android-target" {
		t.Fatalf("start state = %+v, want paused at android target", state)
	}
	step, err := mgr.DebugStep(state.SessionID)
	if err != nil {
		t.Fatalf("DebugStep: %v", err)
	}
	if step.Status != container.DebugStatusStepping {
		t.Fatalf("step state = %+v, want stepping", step)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		got, err := mgr.DebugState(state.SessionID)
		if err != nil {
			t.Fatalf("DebugState: %v", err)
		}
		if got.Status == container.DebugStatusPaused {
			if got.LastNodeID != "android-target" || got.LastExit != "Done" || got.CurrentNodeID != "start-app" {
				t.Fatalf("paused state = %+v, want android-target Done then start-app", got)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timed out waiting for AndroidTarget step to return to paused")
}

func TestDebugManagerSecondStartReturnsBusy(t *testing.T) {
	c := &container.Container{
		ID:   "c1",
		Name: "c1",
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "stop", Kind: "Stop"},
			},
			Edges: []container.GraphEdge{{From: "start.Done", To: "stop.In"}},
		},
	}
	mgr := newContainerDebugManager(newWireDebugTestRunner(c), nil, func() bool { return false })

	state, err := mgr.DebugStart("c1", container.DebugStartOptions{})
	if err != nil {
		t.Fatalf("DebugStart: %v", err)
	}
	if _, err := mgr.DebugStart("c1", container.DebugStartOptions{}); err == nil || err.Error() != "debug_session_busy" {
		t.Fatalf("second DebugStart error = %v, want debug_session_busy", err)
	}
	if _, err := mgr.DebugStop(state.SessionID); err != nil {
		t.Fatalf("DebugStop: %v", err)
	}
}

func TestDebugManagerStepWhileSteppingReturnsBusy(t *testing.T) {
	c := &container.Container{
		ID:   "c1",
		Name: "c1",
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "sleep", Kind: "Sleep"},
				{ID: "stop", Kind: "Stop"},
			},
			Edges: []container.GraphEdge{
				{From: "start.Done", To: "sleep.In"},
				{From: "sleep.Done", To: "stop.In"},
			},
		},
	}
	mgr := newContainerDebugManager(newWireDebugTestRunner(c), nil, func() bool { return false })

	state, err := mgr.DebugStart("c1", container.DebugStartOptions{})
	if err != nil {
		t.Fatalf("DebugStart: %v", err)
	}
	if _, err := mgr.DebugStep(state.SessionID); err != nil {
		t.Fatalf("DebugStep: %v", err)
	}
	if _, err := mgr.DebugStep(state.SessionID); err == nil || err.Error() != "debug_session_busy" {
		t.Fatalf("second DebugStep error = %v, want debug_session_busy", err)
	}
	if _, err := mgr.DebugStop(state.SessionID); err != nil {
		t.Fatalf("DebugStop: %v", err)
	}
}

func TestDebugManagerStopDuringLongStepReleasesSession(t *testing.T) {
	c := &container.Container{
		ID:   "c1",
		Name: "c1",
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "sleep", Kind: "Sleep"},
				{ID: "stop", Kind: "Stop"},
			},
			Edges: []container.GraphEdge{
				{From: "start.Done", To: "sleep.In"},
				{From: "sleep.Done", To: "stop.In"},
			},
		},
	}
	mgr := newContainerDebugManager(newWireDebugTestRunner(c), nil, func() bool { return false })

	state, err := mgr.DebugStart("c1", container.DebugStartOptions{})
	if err != nil {
		t.Fatalf("DebugStart: %v", err)
	}
	step, err := mgr.DebugStep(state.SessionID)
	if err != nil {
		t.Fatalf("DebugStep: %v", err)
	}
	if step.Status != container.DebugStatusStepping {
		t.Fatalf("step state = %+v, want stepping", step)
	}
	stopped, err := mgr.DebugStop(state.SessionID)
	if err != nil {
		t.Fatalf("DebugStop: %v", err)
	}
	if stopped.Status != container.DebugStatusStopped {
		t.Fatalf("stop state = %+v, want stopped", stopped)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !mgr.IsActive() {
			next, err := mgr.DebugStart("c1", container.DebugStartOptions{})
			if err != nil {
				t.Fatalf("DebugStart after long-step stop: %v", err)
			}
			if _, err := mgr.DebugStop(next.SessionID); err != nil {
				t.Fatalf("cleanup DebugStop: %v", err)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("debug manager remained active after stopping long step")
}

func TestDebugManagerStopReleasesSession(t *testing.T) {
	c := &container.Container{ID: "c1", Name: "c1", Graph: container.Graph{Nodes: []container.GraphNode{{ID: "start", Kind: "Start"}}}}
	mgr := newContainerDebugManager(newWireDebugTestRunner(c), nil, func() bool { return false })

	state, err := mgr.DebugStart("c1", container.DebugStartOptions{})
	if err != nil {
		t.Fatalf("DebugStart: %v", err)
	}
	if _, err := mgr.DebugStop(state.SessionID); err != nil {
		t.Fatalf("DebugStop: %v", err)
	}
	if _, err := mgr.DebugStart("c1", container.DebugStartOptions{}); err != nil {
		t.Fatalf("DebugStart after stop: %v", err)
	}
}

var errContainerNotFoundForDebug = context.Canceled
