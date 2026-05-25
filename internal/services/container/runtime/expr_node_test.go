package runtime

import (
	"testing"

	"yhbox/internal/services/container"
	"yhbox/internal/services/expr"
)

// TestExpr_LiteralInputs: Expr "i + 1" with input i (literal pin) → 6 when i=5.
func TestExpr_LiteralInputs(t *testing.T) {
	_, r := newTestRunner(t)
	r.currentTick = CaptureSnapshot(map[string]expr.Value{}, SysState{})

	n := &container.GraphNode{
		ID: "e1", Kind: "Expr",
		Config: map[string]any{
			"Expression": "i + 1",
			"OutType": "auto",
			"Inputs": []any{map[string]any{"Name": "i", "Type": "number"}},
			"literal": map[string]any{"i": 5.0},
		},
	}
	r.nodesByID = map[string]*container.GraphNode{"e1": n}

	v, err := r.evalExpr(n)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := expr.AsNumber(v)
	if got != 6.0 {
		t.Fatalf("Expr i+1 with i=5: want 6, got %v", v)
	}
}

// TestExpr_BoolExpression: complex boolean expression with multiple inputs.
func TestExpr_BoolExpression(t *testing.T) {
	_, r := newTestRunner(t)
	r.currentTick = CaptureSnapshot(map[string]expr.Value{}, SysState{})

	n := &container.GraphNode{
		ID: "e1", Kind: "Expr",
		Config: map[string]any{
			"Expression": `s == "FISHING" && hp > 0.5`,
			"Inputs": []any{
				map[string]any{"Name": "s", "Type": "string"},
				map[string]any{"Name": "hp", "Type": "number"},
			},
			"literal": map[string]any{"s": "FISHING", "hp": 0.8},
		},
	}
	r.nodesByID = map[string]*container.GraphNode{"e1": n}

	v, _ := r.evalExpr(n)
	if v != true {
		t.Fatalf("complex bool expr: want true, got %v", v)
	}
}

// TestExpr_DollarPathsAreIsolated: Expr cannot access $vars (InputEnv design).
// Even if rt.vars has the value, Expr returns nil for $vars.X.
func TestExpr_DollarPathsAreIsolated(t *testing.T) {
	_, r := newTestRunner(t)
	r.currentTick = CaptureSnapshot(map[string]expr.Value{"hp": float64(99)}, SysState{})

	n := &container.GraphNode{
		ID: "e1", Kind: "Expr",
		Config: map[string]any{
			"Expression": "$vars.hp",
			"Inputs": []any{},
		},
	}
	r.nodesByID = map[string]*container.GraphNode{"e1": n}

	v, _ := r.evalExpr(n)
	if v != nil {
		t.Fatalf("$vars in Expr must be isolated: want nil, got %v", v)
	}
}

// TestExpr_PullFromGetVarEdge: input value comes via data edge from GetVar.
func TestExpr_PullFromGetVarEdge(t *testing.T) {
	_, r := newTestRunner(t)
	r.currentTick = CaptureSnapshot(map[string]expr.Value{"counter": float64(10)}, SysState{})

	gv := &container.GraphNode{
		ID: "gv", Kind: "GetVar",
		Config: map[string]any{"VarName": "counter", "Scope": "global"},
	}
	ex := &container.GraphNode{
		ID: "ex", Kind: "Expr",
		Config: map[string]any{
			"Expression": "c * 2",
			"Inputs": []any{map[string]any{"Name": "c", "Type": "number"}},
		},
	}
	r.nodesByID = map[string]*container.GraphNode{"gv": gv, "ex": ex}
	r.dataEdges = buildDataEdgeIndex(container.Graph{
		Nodes: []container.GraphNode{*gv, *ex},
		Edges: []container.GraphEdge{
			{From: "gv.Value", To: "ex.c"},
		},
	})

	v, err := r.evalExpr(ex)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := expr.AsNumber(v)
	if got != 20.0 {
		t.Fatalf("Expr c*2 with c from GetVar(counter=10): want 20, got %v", v)
	}
}

// TestExpr_EmptyExprReturnsNil: draft node with empty expr is non-fatal.
func TestExpr_EmptyExprReturnsNil(t *testing.T) {
	_, r := newTestRunner(t)
	r.currentTick = CaptureSnapshot(map[string]expr.Value{}, SysState{})

	n := &container.GraphNode{
		ID: "e1", Kind: "Expr",
		Config: map[string]any{"Expression": "", "Inputs": []any{}},
	}
	r.nodesByID = map[string]*container.GraphNode{"e1": n}

	v, err := r.evalExpr(n)
	if err != nil {
		t.Fatal(err)
	}
	if v != nil {
		t.Fatalf("empty expr: want nil, got %v", v)
	}
}

// TestExpr_ParseError: malformed expr returns parse error.
func TestExpr_ParseError(t *testing.T) {
	_, r := newTestRunner(t)
	r.currentTick = CaptureSnapshot(map[string]expr.Value{}, SysState{})

	n := &container.GraphNode{
		ID: "e1", Kind: "Expr",
		Config: map[string]any{"Expression": "1 +", "Inputs": []any{}},
	}
	r.nodesByID = map[string]*container.GraphNode{"e1": n}

	_, err := r.evalExpr(n)
	if err == nil {
		t.Fatal("malformed expr should error")
	}
}
