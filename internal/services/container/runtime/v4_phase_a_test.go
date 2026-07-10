package runtime

import (
	"context"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/services/container"
	"github.com/yottaapp/yotta/internal/services/execution"
	"github.com/yottaapp/yotta/internal/services/expr"
)

// TestPhaseA_E2E_SetVarGlobal: end-to-end exec → SetVar (literal pin) → tick boundary
// → next exec sees new global var. Validates:
//   - per-exec-tick snapshot
//   - data-in pin pullDataPin + literal config storage
//   - SetVar reading via pull instead of expr config
//   - global scope writing rt.vars
func TestPhaseA_E2E_SetVarGlobal(t *testing.T) {
	c := &container.Container{
		SchemaVersion: 1,
		ID:            "v4-phaseA-e2e",
		Name:          "v4-phaseA-e2e",
		Vars:          []container.VarDecl{{Name: "state", Type: "string", Default: ""}},
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "set", Kind: "SetVar",
					Config: map[string]any{
						"VarName": "state",
						"Scope":   "global",
						"literal": map[string]any{"Value": "FISHING"},
					}},
				{ID: "stop", Kind: "Stop"},
			},
			Edges: []container.GraphEdge{
				{From: "start.Done", To: "set.In"},
				{From: "set.Done", To: "stop.In"},
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

	got := expr.AsString(rt.Vars()["state"])
	if got != "FISHING" {
		t.Fatalf("global state after SetVar: want FISHING, got %v", got)
	}
}

// TestPhaseA_E2E_IncVarDeltaLiteral: SetVar(counter=10) → IncVar(delta=5) → counter==15.
// Validates IncVar pulling delta from literal pin.
func TestPhaseA_E2E_IncVarDeltaLiteral(t *testing.T) {
	c := &container.Container{
		SchemaVersion: 1,
		ID:            "v4-incvar-e2e",
		Name:          "v4-incvar-e2e",
		Vars:          []container.VarDecl{{Name: "counter", Type: "number", Default: 0.0}},
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "set", Kind: "SetVar",
					Config: map[string]any{
						"VarName": "counter", "Scope": "global",
						"literal": map[string]any{"Value": 10.0},
					}},
				{ID: "inc", Kind: "IncVar",
					Config: map[string]any{
						"VarName": "counter", "Scope": "global",
						"literal": map[string]any{"Delta": 5.0},
					}},
				{ID: "stop", Kind: "Stop"},
			},
			Edges: []container.GraphEdge{
				{From: "start.Done", To: "set.In"},
				{From: "set.Done", To: "inc.In"},
				{From: "inc.Done", To: "stop.In"},
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

	got, _ := expr.AsNumber(rt.Vars()["counter"])
	if got != 15.0 {
		t.Fatalf("counter after SetVar(10)+IncVar(+5): want 15, got %v", got)
	}
}

// TestPhaseA_E2E_GetVarViaDataEdge: SetVar(src="hello") → SetVar(dst pulls from GetVar(src)) → dst=="hello"
// Validates GetVar + dataEdgeIndex + cross-tick snapshot consistency.
//
// Graph:
//
//	Start → SetVar(src global "hello")
//	      → SetVar(dst global) [data: GetVar(src,global).value → dst.value]
//	      → Stop
func TestPhaseA_E2E_GetVarViaDataEdge(t *testing.T) {
	c := &container.Container{
		SchemaVersion: 1,
		ID:            "v4-getvar-e2e",
		Name:          "v4-getvar-e2e",
		Vars: []container.VarDecl{
			{Name: "src", Type: "string", Default: ""},
			{Name: "dst", Type: "string", Default: ""},
		},
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "setSrc", Kind: "SetVar",
					Config: map[string]any{
						"VarName": "src", "Scope": "global",
						"literal": map[string]any{"Value": "hello"},
					}},
				{ID: "gv", Kind: "GetVar",
					Config: map[string]any{"VarName": "src", "Scope": "global"}},
				{ID: "setDst", Kind: "SetVar",
					Config: map[string]any{"VarName": "dst", "Scope": "global"}},
				{ID: "stop", Kind: "Stop"},
			},
			Edges: []container.GraphEdge{
				// Exec chain
				{From: "start.Done", To: "setSrc.In"},
				{From: "setSrc.Done", To: "setDst.In"},
				{From: "setDst.Done", To: "stop.In"},
				// Data flow: GetVar(src).value → SetVar(dst).value
				{From: "gv.Value", To: "setDst.Value"},
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

	got := expr.AsString(rt.Vars()["dst"])
	if got != "hello" {
		t.Fatalf("dst via GetVar(src) edge: want 'hello', got %v", got)
	}
}

// TestPhaseA_E2E_LocalScopeIsolation: SetVar(scope=local) does NOT pollute rt.vars.
// IncVar in same frame can still increment the local. Validates scope semantics end-to-end.
func TestPhaseA_E2E_LocalScopeIsolation(t *testing.T) {
	c := &container.Container{
		SchemaVersion: 1,
		ID:            "v4-localscope-e2e",
		Name:          "v4-localscope-e2e",
		Vars:          []container.VarDecl{{Name: "i", Type: "number", Default: 0.0}},
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "set", Kind: "SetVar",
					Config: map[string]any{
						"VarName": "i", "Scope": "local",
						"literal": map[string]any{"Value": 1.0},
					}},
				{ID: "inc", Kind: "IncVar",
					Config: map[string]any{
						"VarName": "i", "Scope": "local",
						"literal": map[string]any{"Delta": 2.0},
					}},
				{ID: "stop", Kind: "Stop"},
			},
			Edges: []container.GraphEdge{
				{From: "start.Done", To: "set.In"},
				{From: "set.Done", To: "inc.In"},
				{From: "inc.Done", To: "stop.In"},
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

	// local scope: rt.vars[i] should still be its Default (0), not 3
	if got := rt.Vars()["i"]; got != 0.0 {
		t.Fatalf("local scope must NOT touch rt.vars: want 0 (default), got %v", got)
	}
}
