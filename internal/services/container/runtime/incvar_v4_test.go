package runtime

import (
	"context"
	"testing"

	"yhbox/internal/services/container"
	"yhbox/internal/services/expr"
)

// TestIncVar_LiteralDelta: IncVar delta from inline literal pin (v4).
func TestIncVar_LiteralDelta(t *testing.T) {
	_, r := newTestRunner(t)
	r.currentTick = CaptureSnapshot(map[string]expr.Value{"counter": float64(0)}, SysState{})
	r.rt.SetVar("counter", float64(0))

	n := &container.GraphNode{
		ID: "i1", Kind: "IncVar",
		Config: map[string]any{
			"varName": "counter",
			"scope":   "global",
			"literal": map[string]any{"delta": 5.0},
		},
	}
	r.nodesByID = map[string]*container.GraphNode{"i1": n}
	r.edges = &edgeIndex{out: map[string][]string{}}

	_, err := r.execIncVar(context.Background(), n, ExecToken{NodeID: "i1"})
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := expr.AsNumber(r.rt.Vars()["counter"]); got != 5.0 {
		t.Fatalf("counter += 5: want 5, got %v", got)
	}
}

// TestIncVar_DefaultDelta: missing literal → default delta=1 (pin schema default).
func TestIncVar_DefaultDelta(t *testing.T) {
	_, r := newTestRunner(t)
	r.currentTick = CaptureSnapshot(map[string]expr.Value{"counter": float64(10)}, SysState{})
	r.rt.SetVar("counter", float64(10))

	n := &container.GraphNode{
		ID: "i1", Kind: "IncVar",
		Config: map[string]any{
			"varName": "counter",
			"scope":   "global",
			// no literal, no edge
		},
	}
	r.nodesByID = map[string]*container.GraphNode{"i1": n}
	r.edges = &edgeIndex{out: map[string][]string{}}

	_, _ = r.execIncVar(context.Background(), n, ExecToken{NodeID: "i1"})
	if got, _ := expr.AsNumber(r.rt.Vars()["counter"]); got != 11.0 {
		t.Fatalf("default delta=1: want 11, got %v", got)
	}
}

// TestIncVar_LocalScope: increments frame.LocalVars.
func TestIncVar_LocalScope(t *testing.T) {
	_, r := newTestRunner(t)
	r.currentTick = CaptureSnapshot(map[string]expr.Value{}, SysState{})
	r.state.SetLocalVar("counter", float64(3))

	n := &container.GraphNode{
		ID: "i1", Kind: "IncVar",
		Config: map[string]any{
			"varName": "counter",
			"scope":   "local",
			"literal": map[string]any{"delta": 2.0},
		},
	}
	r.nodesByID = map[string]*container.GraphNode{"i1": n}
	r.edges = &edgeIndex{out: map[string][]string{}}

	_, _ = r.execIncVar(context.Background(), n, ExecToken{NodeID: "i1"})
	v, _ := r.state.GetLocalVarHere("counter")
	if v.(float64) != 5.0 {
		t.Fatalf("local counter +=2: want 5, got %v", v)
	}
}
