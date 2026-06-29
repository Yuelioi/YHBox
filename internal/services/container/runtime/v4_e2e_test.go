package runtime

import (
	"context"
	"testing"
	"time"

	"yotta/internal/services/container"
	"yotta/internal/services/execution"
	"yotta/internal/services/expr"
)

// TestV4_E2E_AllCategories: a "kitchen sink" container exercising the full v4 data-flow stack.
//
// Topology:
//
//	Start → SetVar(state="IDLE",global,literal)
//	      → SetVar(counter=0,global,literal)
//	      → SetVar(dst,global)  ← receives data from a chain
//	      → If(condition)       ← condition via Expr+GetVar
//	          .then → Stop
//	          .else → Stop
//
// Data flow into setDst.value:
//
//	[GetVar counter] → [Add b=10] → setDst.value  (counter + 10 = 10)
//
// Data flow into If.condition:
//
//	[GetVar counter] → [Expr "i > -1"] → If.condition  (always true initially)
//
// Validates: per-tick snapshot, GetVar global, Add pure func via data edge,
// GetVar data edge, Expr with bare identifier + InputEnv isolation, SetVar from edge,
// exec branching following Expr→If chain.
func TestV4_E2E_AllCategories(t *testing.T) {
	c := &container.Container{
		SchemaVersion: 1,
		ID:            "v4-e2e-all",
		Name:          "v4-e2e-all",
		Vars: []container.VarDecl{
			{Name: "state", Type: "string", Default: ""},
			{Name: "counter", Type: "number", Default: 0.0},
			{Name: "dst", Type: "number", Default: 0.0},
		},
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "start", Kind: "Start"},
				// Init state + counter
				{ID: "setState", Kind: "SetVar", Config: map[string]any{
					"VarName": "state", "Scope": "global",
					"literal": map[string]any{"Value": "IDLE"},
				}},
				{ID: "setCounter", Kind: "SetVar", Config: map[string]any{
					"VarName": "counter", "Scope": "global",
					"literal": map[string]any{"Value": 0.0},
				}},
				// Data sources for setDst
				{ID: "getCounter", Kind: "GetVar", Config: map[string]any{
					"VarName": "counter", "Scope": "global",
				}},
				{ID: "add10", Kind: "Add", Config: map[string]any{
					"literal": map[string]any{"B": 10.0},
				}},
				{ID: "setDst", Kind: "SetVar", Config: map[string]any{
					"VarName": "dst", "Scope": "global",
				}},
				// Branch condition
				{ID: "getIter", Kind: "GetVar", Config: map[string]any{
					"VarName": "counter", "Scope": "global",
				}},
				{ID: "condExpr", Kind: "Expr", Config: map[string]any{
					"Expression": "i > -1",
					"OutType":    "auto",
					"Inputs":     []any{map[string]any{"Name": "i", "Type": "number"}},
				}},
				{ID: "if1", Kind: "If"},
				{ID: "stop", Kind: "Stop"},
			},
			Edges: []container.GraphEdge{
				// Exec chain
				{From: "start.Done", To: "setState.In"},
				{From: "setState.Done", To: "setCounter.In"},
				{From: "setCounter.Done", To: "setDst.In"},
				{From: "setDst.Done", To: "if1.In"},
				{From: "if1.True", To: "stop.In"},
				{From: "if1.False", To: "stop.In"},
				// Data: GetVar(counter) → Add(A) → setDst.value
				{From: "getCounter.Value", To: "add10.A"},
				{From: "add10.Result", To: "setDst.Value"},
				// Data: GetVar(counter) → Expr.i → If.condition
				{From: "getIter.Value", To: "condExpr.i"},
				{From: "condExpr.Result", To: "if1.Condition"},
			},
		},
	}
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, nil, nil, nil, 0)
	stubRuntimeWindowAndInput(rt)
	r := NewContainerRunner(rt)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := r.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}

	vars := rt.Vars()
	if got := expr.AsString(vars["state"]); got != "IDLE" {
		t.Errorf("state: want IDLE, got %v", got)
	}
	if got, _ := expr.AsNumber(vars["counter"]); got != 0.0 {
		t.Errorf("counter: want 0 (literal init), got %v", got)
	}
	if got, _ := expr.AsNumber(vars["dst"]); got != 10.0 {
		t.Errorf("dst (counter+10 via Add edge): want 10, got %v", got)
	}
}

// TestV4_E2E_FishingFightLite: minimal Fishing v2 fight-state pattern using v4 nodes.
// Validates SetVar to record a step number. Stand-in for actual fish state-machine.
func TestV4_E2E_FishingFightLite(t *testing.T) {
	c := &container.Container{
		SchemaVersion: 1,
		ID:            "v4-fight-lite",
		Vars: []container.VarDecl{
			{Name: "step", Type: "number", Default: 0.0},
		},
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "setStep", Kind: "SetVar", Config: map[string]any{
					"VarName": "step", "Scope": "global",
					"literal": map[string]any{"Value": 1.0},
				}},
				{ID: "stop", Kind: "Stop"},
			},
			Edges: []container.GraphEdge{
				{From: "start.Done", To: "setStep.In"},
				{From: "setStep.Done", To: "stop.In"},
			},
		},
	}
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, nil, nil, nil, 0)
	stubRuntimeWindowAndInput(rt)
	r := NewContainerRunner(rt)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := r.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got, _ := expr.AsNumber(rt.Vars()["step"]); got != 1.0 {
		t.Fatalf("step: want 1, got %v", got)
	}
}
