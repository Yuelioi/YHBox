package runtime

import (
	"context"
	"testing"

	nodepkg "yhbox/internal/node"
	"yhbox/internal/services/container"
	"yhbox/internal/services/expr"
)

// Expr 现在走 framework EvaluatePureData. dispatch_v5.buildDataWireFor 对 Expr 节点特例
// 按 config.Inputs[] pull dynamic input pin 进 dataWire, Expr.Evaluate 再 in.Keys() 遍历
// 进 expr.InputEnv.

// evalExprViaFramework 走当前生产 path (resolveDataPinV5/evalDataSource 转 EvaluatePureData),
// 测试只调一次 helper, 不重复 dispatch 拼装. ctx 携 withTickSnapshot 的 tick 给 bundle.Snapshot.
func (r *ContainerRunner) evalExprViaFramework(t *testing.T, ctx context.Context, n *container.GraphNode) (any, error) {
	t.Helper()
	rn, ok := nodepkg.Get("Expr")
	if !ok {
		t.Fatalf("Expr not registered")
	}
	dw := r.buildDataWireFor(ctx, n, rn)
	cfg := r.buildConfigFor(n)
	return nodepkg.EvaluatePureData(ctx, rn, dw, cfg, r.bundle)
}

// TestExpr_LiteralInputs: Expr "i + 1" with input i (literal pin) → 6 when i=5.
func TestExpr_LiteralInputs(t *testing.T) {
	_, r := newTestRunner(t)
	ctx := withTickSnapshot(context.Background(), CaptureSnapshot(map[string]expr.Value{}, SysState{}))

	n := &container.GraphNode{
		ID: "e1", Kind: "Expr",
		Config: map[string]any{
			"Expression": "i + 1",
			"OutType":    "auto",
			"Inputs":     []any{map[string]any{"Name": "i", "Type": "number"}},
			"literal":    map[string]any{"i": 5.0},
		},
	}
	r.nodesByID = map[string]*container.GraphNode{"e1": n}

	v, err := r.evalExprViaFramework(t, ctx, n)
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
	ctx := withTickSnapshot(context.Background(), CaptureSnapshot(map[string]expr.Value{}, SysState{}))

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

	v, _ := r.evalExprViaFramework(t, ctx, n)
	if v != true {
		t.Fatalf("complex bool expr: want true, got %v", v)
	}
}

// TestExpr_DollarPathsAreIsolated: Expr cannot access $vars (InputEnv design).
// Even if rt.vars has the value, Expr returns nil for $vars.X.
func TestExpr_DollarPathsAreIsolated(t *testing.T) {
	_, r := newTestRunner(t)
	ctx := withTickSnapshot(context.Background(), CaptureSnapshot(map[string]expr.Value{"hp": float64(99)}, SysState{}))

	n := &container.GraphNode{
		ID: "e1", Kind: "Expr",
		Config: map[string]any{
			"Expression": "$vars.hp",
			"Inputs":     []any{},
		},
	}
	r.nodesByID = map[string]*container.GraphNode{"e1": n}

	v, _ := r.evalExprViaFramework(t, ctx, n)
	if v != nil {
		t.Fatalf("$vars in Expr must be isolated: want nil, got %v", v)
	}
}

// TestExpr_PullFromGetVarEdge: input value comes via data edge from GetVar.
func TestExpr_PullFromGetVarEdge(t *testing.T) {
	_, r := newTestRunner(t)
	ctx := withTickSnapshot(context.Background(), CaptureSnapshot(map[string]expr.Value{"counter": float64(10)}, SysState{}))

	gv := &container.GraphNode{
		ID: "gv", Kind: "GetVar",
		Config: map[string]any{"VarName": "counter", "Scope": "global"},
	}
	ex := &container.GraphNode{
		ID: "ex", Kind: "Expr",
		Config: map[string]any{
			"Expression": "c * 2",
			"Inputs":     []any{map[string]any{"Name": "c", "Type": "number"}},
		},
	}
	r.nodesByID = map[string]*container.GraphNode{"gv": gv, "ex": ex}
	r.dataEdges = buildDataEdgeIndex(container.Graph{
		Nodes: []container.GraphNode{*gv, *ex},
		Edges: []container.GraphEdge{
			{From: "gv.Value", To: "ex.c"},
		},
	})

	v, err := r.evalExprViaFramework(t, ctx, ex)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := expr.AsNumber(v)
	if got != 20.0 {
		t.Fatalf("Expr c*2 with c from GetVar(counter=10): want 20, got %v", v)
	}
}

// TestExpr_EmptyExprReturnsNil: draft node with empty expr is non-fatal.
//
// 注意: Expression 在 Spec 里是 Required, framework 在 validateRequired 阶段
// 检查 in.Has(name) — config 给 "" (空串) 时, present[Expression]=true, Has 通过.
// 所以空串走到 Evaluate, defensive 返 nil (而非 ValidationError).
func TestExpr_EmptyExprReturnsNil(t *testing.T) {
	_, r := newTestRunner(t)
	ctx := withTickSnapshot(context.Background(), CaptureSnapshot(map[string]expr.Value{}, SysState{}))

	n := &container.GraphNode{
		ID: "e1", Kind: "Expr",
		Config: map[string]any{"Expression": "", "Inputs": []any{}},
	}
	r.nodesByID = map[string]*container.GraphNode{"e1": n}

	v, err := r.evalExprViaFramework(t, ctx, n)
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
	ctx := withTickSnapshot(context.Background(), CaptureSnapshot(map[string]expr.Value{}, SysState{}))

	n := &container.GraphNode{
		ID: "e1", Kind: "Expr",
		Config: map[string]any{"Expression": "1 +", "Inputs": []any{}},
	}
	r.nodesByID = map[string]*container.GraphNode{"e1": n}

	_, err := r.evalExprViaFramework(t, ctx, n)
	if err == nil {
		t.Fatal("malformed expr should error")
	}
}
