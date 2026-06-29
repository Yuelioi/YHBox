package main

import (
	"context"
	"sync"
	"testing"
	"time"

	_ "yotta/internal/nodes/all"
	"yotta/internal/services/container"
	containerruntime "yotta/internal/services/container/runtime"
	"yotta/internal/services/execution"
)

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
