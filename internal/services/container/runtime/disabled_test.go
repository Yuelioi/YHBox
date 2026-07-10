package runtime

import (
	"context"
	"testing"

	"github.com/yottaapp/yotta/internal/services/container"
	"github.com/yottaapp/yotta/internal/services/expr"
)

// TestDisabled_LinearPassthrough: Disabled Sleep skips to .out
func TestDisabled_LinearPassthrough(t *testing.T) {
	_, r := newTestRunner(t)
	n := &container.GraphNode{
		ID: "sleep1", Kind: "Sleep", Disabled: true,
		Config: map[string]any{"literal": map[string]any{"Duration": 9999.0}},
	}
	r.nodesByID = map[string]*container.GraphNode{"sleep1": n}
	r.edges = &edgeIndex{out: map[string][]string{"sleep1.Done": {"target.in"}}}
	tokens, err := r.execNode(context.Background(), n, ExecToken{InPin: "in"})
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 1 || tokens[0].NodeID != "target" {
		t.Fatalf("disabled Sleep should pass through to .out → target, got: %+v", tokens)
	}
}

// TestDisabled_Loop_GoesToComplete: Disabled Loop routes to .complete (skip body)
func TestDisabled_Loop_GoesToComplete(t *testing.T) {
	_, r := newTestRunner(t)
	n := &container.GraphNode{ID: "loop1", Kind: "Loop", Disabled: true}
	r.nodesByID = map[string]*container.GraphNode{"loop1": n}
	r.edges = &edgeIndex{out: map[string][]string{"loop1.done": {"target.in"}}}
	tokens, err := r.execNode(context.Background(), n, ExecToken{InPin: "in"})
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 1 || tokens[0].NodeID != "target" {
		t.Fatalf("disabled Loop should pass through to .complete, got: %+v", tokens)
	}
}

// TestDisabled_If_GoesToThen
func TestDisabled_If_GoesToThen(t *testing.T) {
	_, r := newTestRunner(t)
	n := &container.GraphNode{ID: "if1", Kind: "If", Disabled: true}
	r.nodesByID = map[string]*container.GraphNode{"if1": n}
	r.edges = &edgeIndex{out: map[string][]string{"if1.True": {"target.in"}}}
	tokens, err := r.execNode(context.Background(), n, ExecToken{InPin: "in"})
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 1 || tokens[0].NodeID != "target" {
		t.Fatalf("disabled If should pass through to .then, got: %+v", tokens)
	}
}

// TestDisabled_Throw_Noop
func TestDisabled_Throw_Noop(t *testing.T) {
	_, r := newTestRunner(t)
	n := &container.GraphNode{ID: "throw1", Kind: "Throw", Disabled: true}
	r.nodesByID = map[string]*container.GraphNode{"throw1": n}
	r.edges = &edgeIndex{out: map[string][]string{}}
	tokens, err := r.execNode(context.Background(), n, ExecToken{InPin: "in"})
	if err != nil {
		t.Fatal(err)
	}
	if tokens != nil {
		t.Fatalf("disabled Throw should noop (nil tokens), got: %+v", tokens)
	}
}

// TestDisabled_PureData_GetVarReturnsNil: disabled GetVar's data-out returns nil
// (caller's pullNumber fallback fires instead of evaluating the var).
func TestDisabled_PureData_GetVarReturnsNil(t *testing.T) {
	rt, r := newTestRunner(t)
	rt.SetVar("x", expr.Value(42.0))
	ctx := withTickSnapshot(context.Background(), CaptureSnapshot(map[string]expr.Value{"x": 42.0}))

	disabledGV := &container.GraphNode{
		ID: "gv1", Kind: "GetVar", Disabled: true,
		Config: map[string]any{"VarName": "x", "Scope": "global"},
	}
	r.nodesByID = map[string]*container.GraphNode{"gv1": disabledGV}

	val, err := r.evalDataSource(ctx, "gv1", "value")
	if err != nil {
		t.Fatal(err)
	}
	if val != nil {
		t.Fatalf("disabled GetVar should return nil, got %v", val)
	}
}
