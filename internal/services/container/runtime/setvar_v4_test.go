package runtime

import (
	"context"
	"testing"

	"yhbox/internal/services/container"
	"yhbox/internal/services/expr"
)

// TestSetVar_FromLiteralPin: SetVar reads value from inline literal pin (v4).
func TestSetVar_FromLiteralPin(t *testing.T) {
	_, r := newTestRunner(t)
	r.currentTick = CaptureSnapshot(map[string]expr.Value{}, SysState{})

	n := &container.GraphNode{
		ID: "s1", Kind: "SetVar",
		Config: map[string]any{
			"varName": "state",
			"scope":   "global",
			"literal": map[string]any{"value": "FISHING"},
		},
	}
	r.nodesByID = map[string]*container.GraphNode{"s1": n}
	r.edges = &edgeIndex{out: map[string][]string{}} // empty exec edges (Stop node)

	_, err := r.execSetVar(context.Background(), n, ExecToken{NodeID: "s1"})
	if err != nil {
		t.Fatal(err)
	}
	if got := expr.AsString(r.rt.Vars()["state"]); got != "FISHING" {
		t.Fatalf("global state: want FISHING, got %v", r.rt.Vars()["state"])
	}
}

// TestSetVar_LocalScopeWritesFrameVars: SetVar scope=local writes frame.LocalVars,
// does NOT touch rt.vars.
func TestSetVar_LocalScopeWritesFrameVars(t *testing.T) {
	_, r := newTestRunner(t)
	r.currentTick = CaptureSnapshot(map[string]expr.Value{}, SysState{})

	n := &container.GraphNode{
		ID: "s1", Kind: "SetVar",
		Config: map[string]any{
			"varName": "iter",
			"scope":   "local",
			"literal": map[string]any{"value": 7.0},
		},
	}
	r.nodesByID = map[string]*container.GraphNode{"s1": n}
	r.edges = &edgeIndex{out: map[string][]string{}}

	_, err := r.execSetVar(context.Background(), n, ExecToken{NodeID: "s1"})
	if err != nil {
		t.Fatal(err)
	}

	v, ok := r.state.GetLocalVarHere("iter")
	if !ok {
		t.Fatal("local iter not set")
	}
	if v.(float64) != 7.0 {
		t.Fatalf("local iter: want 7, got %v", v)
	}
	// global rt.vars should NOT have it
	if _, ok := r.rt.Vars()["iter"]; ok {
		t.Fatal("local scope must not leak to rt.vars")
	}
}

// TestSetVar_PullFromGetVarEdge: SetVar value pulled via data edge from a GetVar node.
func TestSetVar_PullFromGetVarEdge(t *testing.T) {
	_, r := newTestRunner(t)
	r.currentTick = CaptureSnapshot(map[string]expr.Value{"src": "from-getvar"}, SysState{})

	gv := &container.GraphNode{
		ID: "gv", Kind: "GetVar",
		Config: map[string]any{"varName": "src", "scope": "global"},
	}
	sv := &container.GraphNode{
		ID: "sv", Kind: "SetVar",
		Config: map[string]any{"varName": "dst", "scope": "global"},
	}
	r.nodesByID = map[string]*container.GraphNode{"gv": gv, "sv": sv}
	r.edges = &edgeIndex{out: map[string][]string{}}
	r.dataEdges = buildDataEdgeIndex(container.Graph{
		Edges: []container.GraphEdge{
			{From: "gv.value", To: "sv.value", Kind: "data"},
		},
	})

	_, err := r.execSetVar(context.Background(), sv, ExecToken{NodeID: "sv"})
	if err != nil {
		t.Fatal(err)
	}
	if got := expr.AsString(r.rt.Vars()["dst"]); got != "from-getvar" {
		t.Fatalf("dst from GetVar edge: want 'from-getvar', got %v", r.rt.Vars()["dst"])
	}
}
