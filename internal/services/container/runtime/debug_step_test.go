package runtime

import (
	"context"
	"strings"
	"testing"

	"yotta/internal/services/container"
	"yotta/internal/services/execution"
)

func newDebugStepRunner(t *testing.T, c *container.Container, sgs []container.Subgraph) *ContainerRunner {
	t.Helper()
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, nil, nil, nil, 0)
	rt.Subgraphs = sgs
	stubRuntimeWindowAndInput(rt)
	return NewContainerRunner(rt)
}

func TestDebugStepOnceFromEntryExecutesOneToken(t *testing.T) {
	c := newTestContainer(
		[]container.GraphNode{
			{ID: "start", Kind: "Start"},
			{ID: "set", Kind: "SetVar", Config: map[string]any{
				"VarName": "x",
				"Scope":   "global",
				"literal": map[string]any{"Value": 42.0},
			}},
			{ID: "log", Kind: "Log", Config: map[string]any{
				"literal": map[string]any{"Message": "done", "Level": "info"},
			}},
		},
		[]container.GraphEdge{
			{From: "start.Done", To: "set.In"},
			{From: "set.Done", To: "log.In"},
		},
		[]container.VarDecl{{Name: "x", Type: "number", Default: 0.0}},
	)
	r := newDebugStepRunner(t, c, nil)
	ctx := context.Background()

	if err := r.StartRuntime(ctx); err != nil {
		t.Fatalf("StartRuntime: %v", err)
	}
	defer r.StopRuntime()
	if err := r.SeedFromEntry(); err != nil {
		t.Fatalf("SeedFromEntry: %v", err)
	}

	res, err := r.StepOnce(ctx)
	if err != nil {
		t.Fatalf("StepOnce: %v", err)
	}
	if res.NodeID != "set" || res.NodeKind != "SetVar" || res.InPin != "In" {
		t.Fatalf("step result = %+v, want set SetVar In", res)
	}
	if res.Exit != "Done" {
		t.Fatalf("exit = %q, want Done", res.Exit)
	}
	if got := r.rt.Vars()["x"]; got != 42.0 {
		t.Fatalf("x = %v, want 42", got)
	}
	q := r.QueueSnapshot()
	if len(q) != 1 || q[0].NodeID != "log" || q[0].InPin != "In" {
		t.Fatalf("queue = %+v, want next log.In", q)
	}
}

func TestDebugSeedFromNodeQueuesSelectedNode(t *testing.T) {
	c := newTestContainer(
		[]container.GraphNode{
			{ID: "target", Kind: tkHappy},
			{ID: "done", Kind: "Stop"},
		},
		[]container.GraphEdge{{From: "target.Out", To: "done.In"}},
		nil,
	)
	r := newDebugStepRunner(t, c, nil)

	if err := r.SeedFromNode("target"); err != nil {
		t.Fatalf("SeedFromNode: %v", err)
	}
	q := r.QueueSnapshot()
	if len(q) != 1 || q[0].NodeID != "target" || q[0].InPin != "in" {
		t.Fatalf("queue = %+v, want target.in", q)
	}
}

func TestDebugSeedFromNodeRejectsMissingOrNonExecutableNode(t *testing.T) {
	c := newTestContainer(
		[]container.GraphNode{
			{ID: "start", Kind: "Start"},
			{ID: "data", Kind: "GetVar", Config: map[string]any{"VarName": "x"}},
		},
		nil,
		nil,
	)
	r := newDebugStepRunner(t, c, nil)

	if err := r.SeedFromNode("missing"); err == nil || !strings.Contains(err.Error(), "debug_invalid_start_node") {
		t.Fatalf("missing error = %v, want debug_invalid_start_node", err)
	}
	if err := r.SeedFromNode("data"); err == nil || !strings.Contains(err.Error(), "debug_start_node_not_executable") {
		t.Fatalf("data node error = %v, want debug_start_node_not_executable", err)
	}
}

func TestDebugStepSubgraphIsAtomic(t *testing.T) {
	resetTdHappyCounter()
	sg := container.Subgraph{
		ID:    "sg_atomic",
		Label: "Atomic",
		Entry: container.SubgraphMarker{NodeID: "sub_in"},
		OutputPins: []container.SubgraphOutputDecl{
			{ID: "ok", Name: "ok", NodeID: "sub_out"},
		},
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "body", Kind: tkHappyCounted},
			},
			Edges: []container.GraphEdge{
				{From: "sub_in.Done", To: "body.In"},
				{From: "body.Out", To: "sub_out.In"},
			},
		},
	}
	c := newTestContainer(
		[]container.GraphNode{
			{ID: "start", Kind: "Start"},
			{ID: "sg_call", Kind: "Subgraph", Config: map[string]any{"SubgraphID": "sg_atomic"}},
			{ID: "after", Kind: "Stop"},
		},
		[]container.GraphEdge{
			{From: "start.Done", To: "sg_call.In"},
			{From: "sg_call.ok", To: "after.In"},
		},
		nil,
	)
	r := newDebugStepRunner(t, c, []container.Subgraph{sg})
	ctx := context.Background()
	if err := r.StartRuntime(ctx); err != nil {
		t.Fatalf("StartRuntime: %v", err)
	}
	defer r.StopRuntime()
	if err := r.SeedFromEntry(); err != nil {
		t.Fatalf("SeedFromEntry: %v", err)
	}

	res, err := r.StepOnce(ctx)
	if err != nil {
		t.Fatalf("StepOnce: %v", err)
	}
	if res.NodeID != "sg_call" || res.Exit != "ok" {
		t.Fatalf("step result = %+v, want sg_call ok", res)
	}
	if got := tdHappyCounter.Load(); got != 1 {
		t.Fatalf("subgraph body runs = %d, want 1", got)
	}
	q := r.QueueSnapshot()
	if len(q) != 1 || q[0].NodeID != "after" {
		t.Fatalf("queue = %+v, want after", q)
	}
}
