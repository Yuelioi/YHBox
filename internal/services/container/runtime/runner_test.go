package runtime

import (
	"context"
	"testing"

	"yhbox/internal/services/container"
	"yhbox/internal/services/execution"
)

// 构造小图工具：Start → SetVar → Stop。
func newTestContainer(nodes []container.GraphNode, edges []container.GraphEdge, vars []container.VarDecl) *container.Container {
	return &container.Container{
		SchemaVersion: 1,
		ID:            "test",
		Name:          "test",
		Vars:          vars,
		Graph: container.Graph{
			Nodes: nodes,
			Edges: edges,
		},
	}
}

func TestRunner_StartSleep(t *testing.T) {
	c := newTestContainer(
		[]container.GraphNode{
			{ID: "start", Kind: "Start"},
			{ID: "s1", Kind: "Sleep", Config: map[string]any{"durationMs": "10"}},
		},
		[]container.GraphEdge{
			{From: "start.out", To: "s1.in"},
		},
		nil,
	)
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, NoopInvoker{}, NoopInputDriver{}, NoopColorDetector{}, nil)
	r := NewContainerRunner(rt)
	if err := r.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestRunner_SetVarThenIncVar(t *testing.T) {
	c := newTestContainer(
		[]container.GraphNode{
			{ID: "start", Kind: "Start"},
			{ID: "set", Kind: "SetVar", Config: map[string]any{"varName": "x", "value": "10"}},
			{ID: "inc", Kind: "IncVar", Config: map[string]any{"varName": "x", "delta": "5"}},
		},
		[]container.GraphEdge{
			{From: "start.out", To: "set.in"},
			{From: "set.out", To: "inc.in"},
		},
		[]container.VarDecl{{Name: "x", Type: "number", Default: 0.0}},
	)
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, NoopInvoker{}, NoopInputDriver{}, NoopColorDetector{}, nil)
	r := NewContainerRunner(rt)
	if err := r.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	vars := rt.Vars()
	if vars["x"] != 15.0 {
		t.Errorf("x = %v, want 15", vars["x"])
	}
}

func TestRunner_IfBranch(t *testing.T) {
	c := newTestContainer(
		[]container.GraphNode{
			{ID: "start", Kind: "Start"},
			{ID: "set1", Kind: "SetVar", Config: map[string]any{"varName": "y", "value": "1"}},
			{ID: "if", Kind: "If", Config: map[string]any{"condition": "$vars.y == 1"}},
			{ID: "setThen", Kind: "SetVar", Config: map[string]any{"varName": "branch", "value": "\"then\""}},
			{ID: "setElse", Kind: "SetVar", Config: map[string]any{"varName": "branch", "value": "\"else\""}},
		},
		[]container.GraphEdge{
			{From: "start.out", To: "set1.in"},
			{From: "set1.out", To: "if.in"},
			{From: "if.then", To: "setThen.in"},
			{From: "if.else", To: "setElse.in"},
		},
		[]container.VarDecl{
			{Name: "y", Type: "number", Default: 0.0},
			{Name: "branch", Type: "string", Default: ""},
		},
	)
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, NoopInvoker{}, NoopInputDriver{}, NoopColorDetector{}, nil)
	r := NewContainerRunner(rt)
	if err := r.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := rt.Vars()["branch"]; got != "then" {
		t.Errorf("branch = %v, want then", got)
	}
}

func TestRunner_LoopCount(t *testing.T) {
	c := newTestContainer(
		[]container.GraphNode{
			{ID: "start", Kind: "Start"},
			{ID: "loop", Kind: "Loop", Config: map[string]any{"mode": "count", "count": "3"}},
			{ID: "inc", Kind: "IncVar", Config: map[string]any{"varName": "i", "delta": "1"}},
		},
		[]container.GraphEdge{
			{From: "start.out", To: "loop.in"},
			{From: "loop.body", To: "inc.in"},
			{From: "inc.out", To: "loop.loopback"},
		},
		[]container.VarDecl{{Name: "i", Type: "number", Default: 0.0}},
	)
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, NoopInvoker{}, NoopInputDriver{}, NoopColorDetector{}, nil)
	r := NewContainerRunner(rt)
	if err := r.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := rt.Vars()["i"]; got != 3.0 {
		t.Errorf("i = %v, want 3", got)
	}
}

func TestRunner_BreakExitsLoop(t *testing.T) {
	c := newTestContainer(
		[]container.GraphNode{
			{ID: "start", Kind: "Start"},
			{ID: "loop", Kind: "Loop", Config: map[string]any{"mode": "forever"}},
			{ID: "inc", Kind: "IncVar", Config: map[string]any{"varName": "i", "delta": "1"}},
			{ID: "if", Kind: "If", Config: map[string]any{"condition": "$vars.i >= 2"}},
			{ID: "br", Kind: "Break"},
		},
		[]container.GraphEdge{
			{From: "start.out", To: "loop.in"},
			{From: "loop.body", To: "inc.in"},
			{From: "inc.out", To: "if.in"},
			{From: "if.then", To: "br.in"},
			{From: "if.else", To: "loop.loopback"},
		},
		[]container.VarDecl{{Name: "i", Type: "number", Default: 0.0}},
	)
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, NoopInvoker{}, NoopInputDriver{}, NoopColorDetector{}, nil)
	r := NewContainerRunner(rt)
	if err := r.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := rt.Vars()["i"]; got != 2.0 {
		t.Errorf("i = %v, want 2", got)
	}
}

func TestRunner_StopHalts(t *testing.T) {
	c := newTestContainer(
		[]container.GraphNode{
			{ID: "start", Kind: "Start"},
			{ID: "set1", Kind: "SetVar", Config: map[string]any{"varName": "a", "value": "1"}},
			{ID: "stop", Kind: "Stop"},
			{ID: "set2", Kind: "SetVar", Config: map[string]any{"varName": "a", "value": "2"}},
		},
		[]container.GraphEdge{
			{From: "start.out", To: "set1.in"},
			{From: "set1.out", To: "stop.in"},
			{From: "stop.out", To: "set2.in"},
		},
		[]container.VarDecl{{Name: "a", Type: "number", Default: 0.0}},
	)
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, NoopInvoker{}, NoopInputDriver{}, NoopColorDetector{}, nil)
	r := NewContainerRunner(rt)
	if err := r.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := rt.Vars()["a"]; got != 1.0 {
		t.Errorf("Stop 后 a 应停留在 1，实际 %v", got)
	}
}

func TestRunner_NoStartNode(t *testing.T) {
	c := newTestContainer(
		[]container.GraphNode{{ID: "s", Kind: "Sleep"}}, nil, nil,
	)
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, NoopInvoker{}, NoopInputDriver{}, NoopColorDetector{}, nil)
	r := NewContainerRunner(rt)
	if err := r.Run(context.Background()); err == nil {
		t.Error("expected error for no Start node")
	}
}
