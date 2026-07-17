package compiler

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestDebugSnapshotCollectionsMarshalAsEmptyCollections(t *testing.T) {
	snapshot := cloneDebugSnapshot(DebugSnapshot{})
	raw, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{`"queue":null`, `"inputs":null`, `"outputs":null`, `"state":null`} {
		if strings.Contains(string(raw), field) {
			t.Fatalf("debug snapshot leaked nullable collection %s: %s", field, raw)
		}
	}
}

func TestDebugControllerStepContinuePauseAndCancellation(t *testing.T) {
	control, err := NewDebugController(DebugControllerOptions{StartPaused: true})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	first := checkpointAsync(control, ctx, "first")
	waitDebugSnapshot(t, control, func(snapshot DebugSnapshot) bool {
		return snapshot.Status == DebugPaused && snapshot.NodeID == "first"
	})
	if err := control.Step(); err != nil {
		t.Fatal(err)
	}
	if err := <-first; err != nil {
		t.Fatal(err)
	}

	second := checkpointAsync(control, ctx, "second")
	waitDebugSnapshot(t, control, func(snapshot DebugSnapshot) bool {
		return snapshot.Status == DebugPaused && snapshot.NodeID == "second"
	})
	if err := control.Continue(); err != nil {
		t.Fatal(err)
	}
	if err := <-second; err != nil {
		t.Fatal(err)
	}

	if err := control.Pause(); err != nil {
		t.Fatal(err)
	}
	third := checkpointAsync(control, ctx, "third")
	waitDebugSnapshot(t, control, func(snapshot DebugSnapshot) bool {
		return snapshot.Status == DebugPaused && snapshot.NodeID == "third"
	})
	cancel()
	if err := <-third; err == nil {
		t.Fatal("cancelled checkpoint returned no error")
	}
}

func TestDebugBreakpointCanTargetOneSubgraphCallPath(t *testing.T) {
	control, err := NewDebugController(DebugControllerOptions{Breakpoints: []DebugBreakpoint{{
		GraphPath: []string{"main", "left-call", "child"}, GraphID: "child", NodeID: "step",
	}}})
	if err != nil {
		t.Fatal(err)
	}
	base := DebugSnapshot{
		GraphID: "child", NodeID: "step", Attempt: 1, Queue: []DebugQueueEntry{},
		Inputs: map[string]DebugValueView{}, Outputs: map[string]map[string]DebugValueView{}, State: map[string]DebugStateView{},
	}
	wrong := base
	wrong.GraphPath = []string{"main", "right-call", "child"}
	if err := control.checkpoint(context.Background(), wrong); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		exact := base
		exact.GraphPath = []string{"main", "left-call", "child"}
		done <- control.checkpoint(ctx, exact)
	}()
	snapshot := waitDebugSnapshot(t, control, func(snapshot DebugSnapshot) bool {
		return snapshot.Status == DebugPaused && slices.Equal(snapshot.GraphPath, []string{"main", "left-call", "child"})
	})
	snapshot.GraphPath[0] = "mutated"
	if control.Snapshot().GraphPath[0] != "main" {
		t.Fatal("debug snapshot graph path was not cloned")
	}
	if err := control.Continue(); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestExecutorDebugUsesTheOrdinarySchedulerAndJournal(t *testing.T) {
	catalog, contracts, locks := schedulerCatalogForTest(t)
	program := compileSchedulerProgram(t, catalog, contracts, true)
	now := time.Date(2026, 7, 17, 5, 0, 0, 0, time.UTC)
	owner, journal := admittedSchedulerExecution(t, catalog, program, now)
	t.Cleanup(func() { _ = owner.Close(context.Background()) })
	var calls atomic.Int32
	adapters := schedulerAdapters(locks, map[string]Adapter{
		"source": func(context.Context, Invocation) (AdapterResult, error) {
			calls.Add(1)
			return AdapterResult{ExecOutputs: []string{"right"}}, nil
		},
		"left": emptyAdapter,
		"right": func(context.Context, Invocation) (AdapterResult, error) {
			calls.Add(1)
			return AdapterResult{}, nil
		},
		"handler": emptyAdapter,
	})
	control, err := NewDebugController(DebugControllerOptions{StartPaused: true})
	if err != nil {
		t.Fatal(err)
	}
	executor := NewExecutor(catalog, adapters, ExecutorOptions{Now: func() time.Time { return now.Add(time.Second) }})
	done := make(chan error, 1)
	go func() {
		_, runErr := executor.RunDebug(context.Background(), program, owner, journal, control)
		done <- runErr
	}()

	waitDebugSnapshot(t, control, func(snapshot DebugSnapshot) bool {
		return snapshot.Status == DebugPaused && snapshot.NodeID == "source"
	})
	if calls.Load() != 0 {
		t.Fatalf("adapter ran before pause: %d", calls.Load())
	}
	if err := control.Step(); err != nil {
		t.Fatal(err)
	}
	waitDebugSnapshot(t, control, func(snapshot DebugSnapshot) bool {
		return snapshot.Status == DebugPaused && snapshot.NodeID == "right"
	})
	if calls.Load() != 1 {
		t.Fatalf("step ran %d adapters", calls.Load())
	}
	if err := control.Continue(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("debug Run did not complete")
	}
	if calls.Load() != 2 || len(journal.Current().Journal()) != 4 {
		t.Fatalf("calls=%d journal=%#v", calls.Load(), journal.Current().Journal())
	}
}

func checkpointAsync(control *DebugController, ctx context.Context, nodeID string) <-chan error {
	done := make(chan error, 1)
	go func() {
		done <- control.checkpoint(ctx, DebugSnapshot{
			GraphID: "main", NodeID: nodeID, Attempt: 1,
			Queue: []DebugQueueEntry{}, Inputs: map[string]DebugValueView{},
			Outputs: map[string]map[string]DebugValueView{}, State: map[string]DebugStateView{},
		})
	}()
	return done
}

func waitDebugSnapshot(t *testing.T, control *DebugController, matches func(DebugSnapshot) bool) DebugSnapshot {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		snapshot := control.Snapshot()
		if matches(snapshot) {
			return snapshot
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("debug snapshot did not reach expected state: %#v", control.Snapshot())
	return DebugSnapshot{}
}
