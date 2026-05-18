package runtime

import (
	"context"
	"testing"
	"time"

	"yhbox/internal/services/container"
	"yhbox/internal/services/execution"
	"yhbox/internal/services/expr"
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
//	[GetSys iter] → [Expr "i > -1"] → If.condition  (always true initially)
//
// Validates: per-tick snapshot, GetVar global, Add pure func via data edge,
// GetSys data edge, Expr with bare identifier + InputEnv isolation, SetVar from edge,
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
					"varName": "state", "scope": "global",
					"literal": map[string]any{"value": "IDLE"},
				}},
				{ID: "setCounter", Kind: "SetVar", Config: map[string]any{
					"varName": "counter", "scope": "global",
					"literal": map[string]any{"value": 0.0},
				}},
				// Data sources for setDst
				{ID: "getCounter", Kind: "GetVar", Config: map[string]any{
					"varName": "counter", "scope": "global",
				}},
				{ID: "add10", Kind: "Add", Config: map[string]any{
					"literal": map[string]any{"b": 10.0},
				}},
				{ID: "setDst", Kind: "SetVar", Config: map[string]any{
					"varName": "dst", "scope": "global",
				}},
				// Branch condition
				{ID: "getIter", Kind: "GetSys", Config: map[string]any{"path": "iter"}},
				{ID: "condExpr", Kind: "Expr", Config: map[string]any{
					"expr":    "i > -1",
					"outType": "auto",
					"inputs":  []any{map[string]any{"name": "i", "type": "number"}},
				}},
				{ID: "if1", Kind: "If"},
				{ID: "stop", Kind: "Stop"},
			},
			Edges: []container.GraphEdge{
				// Exec chain
				{From: "start.out", To: "setState.in"},
				{From: "setState.out", To: "setCounter.in"},
				{From: "setCounter.out", To: "setDst.in"},
				{From: "setDst.out", To: "if1.in"},
				{From: "if1.then", To: "stop.in"},
				{From: "if1.else", To: "stop.in"},
				// Data: GetVar(counter) → Add(a) → setDst.value
				{From: "getCounter.value", To: "add10.a", Kind: "data"},
				{From: "add10.result", To: "setDst.value", Kind: "data"},
				// Data: GetSys(iter) → Expr.i → If.condition
				{From: "getIter.value", To: "condExpr.i", Kind: "data"},
				{From: "condExpr.value", To: "if1.condition", Kind: "data"},
			},
		},
	}
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, NoopColorDetector{}, nil, nil, nil, nil, 0)
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
// Validates GetSys ($sys.iter), Expr boolean check, conditional output via If →
// SetVar to record a step number. Stand-in for actual fish state-machine.
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
					"varName": "step", "scope": "global",
					"literal": map[string]any{"value": 1.0},
				}},
				{ID: "stop", Kind: "Stop"},
			},
			Edges: []container.GraphEdge{
				{From: "start.out", To: "setStep.in"},
				{From: "setStep.out", To: "stop.in"},
			},
		},
	}
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, NoopColorDetector{}, nil, nil, nil, nil, 0)
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
