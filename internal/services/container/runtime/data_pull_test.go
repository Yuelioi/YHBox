package runtime

import (
	"testing"

	"yhbox/internal/services/container"
	"yhbox/internal/services/expr"
)

func TestPullDataPin_Literal(t *testing.T) {
	_, r := newTestRunner(t)
	n := &container.GraphNode{
		ID: "n1", Kind: "Sleep",
		Config: map[string]any{
			"literal": map[string]any{
				"durationMs": 500.0,
			},
		},
	}
	r.nodesByID = map[string]*container.GraphNode{"n1": n}

	v, err := r.pullDataPin("n1", "durationMs")
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := expr.AsNumber(v); got != 500.0 {
		t.Fatalf("literal: want 500, got %v", v)
	}
}

func TestPullDataPin_NoEdgeNoLiteral_ReturnsNil(t *testing.T) {
	_, r := newTestRunner(t)
	n := &container.GraphNode{ID: "n1", Kind: "Sleep", Config: map[string]any{}}
	r.nodesByID = map[string]*container.GraphNode{"n1": n}

	v, _ := r.pullDataPin("n1", "durationMs")
	if v != nil {
		t.Fatalf("no edge no literal: want nil, got %v", v)
	}
}

// Verifies that pullDataPin follows a data edge to a GetVar source and resolves it.
func TestPullDataPin_FromGetVarEdge(t *testing.T) {
	_, r := newTestRunner(t)
	r.currentTick = CaptureSnapshot(map[string]expr.Value{"hp": float64(0.8)}, SysState{})

	src := &container.GraphNode{
		ID: "gv", Kind: "GetVar",
		Config: map[string]any{"varName": "hp", "scope": "global"},
	}
	dst := &container.GraphNode{ID: "sleep", Kind: "Sleep", Config: map[string]any{}}
	r.nodesByID = map[string]*container.GraphNode{"gv": src, "sleep": dst}
	r.dataEdges = buildDataEdgeIndex(container.Graph{
		Edges: []container.GraphEdge{
			{From: "gv.value", To: "sleep.durationMs", Kind: "data"},
		},
	})

	v, err := r.pullDataPin("sleep", "durationMs")
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := expr.AsNumber(v); got != 0.8 {
		t.Fatalf("pulled via edge: want 0.8, got %v", v)
	}
}

// Literal under "literal" key is ignored when edge is present (edge wins).
func TestPullDataPin_EdgeWinsOverLiteral(t *testing.T) {
	_, r := newTestRunner(t)
	r.currentTick = CaptureSnapshot(map[string]expr.Value{"hp": float64(0.8)}, SysState{})

	src := &container.GraphNode{
		ID: "gv", Kind: "GetVar",
		Config: map[string]any{"varName": "hp", "scope": "global"},
	}
	dst := &container.GraphNode{
		ID: "sleep", Kind: "Sleep",
		Config: map[string]any{
			"literal": map[string]any{"durationMs": 999.0}, // should be ignored
		},
	}
	r.nodesByID = map[string]*container.GraphNode{"gv": src, "sleep": dst}
	r.dataEdges = buildDataEdgeIndex(container.Graph{
		Edges: []container.GraphEdge{
			{From: "gv.value", To: "sleep.durationMs", Kind: "data"},
		},
	})

	v, _ := r.pullDataPin("sleep", "durationMs")
	if got, _ := expr.AsNumber(v); got != 0.8 {
		t.Fatalf("edge must win over literal: want 0.8, got %v", v)
	}
}

func TestDataEdgeIndex_IgnoresExecEdges(t *testing.T) {
	idx := buildDataEdgeIndex(container.Graph{
		Edges: []container.GraphEdge{
			{From: "a.out", To: "b.in"},                  // no Kind = exec
			{From: "a.out", To: "b.in", Kind: "exec"},    // explicit exec
			{From: "gv.value", To: "c.x", Kind: "data"},  // data
		},
	})
	if src, pin := idx.Source("b", "in"); src != "" || pin != "" {
		t.Fatalf("exec edge leaked into data index: %s.%s", src, pin)
	}
	if src, pin := idx.Source("c", "x"); src != "gv" || pin != "value" {
		t.Fatalf("data edge: want gv.value, got %s.%s", src, pin)
	}
}

func TestToExprValue_Point(t *testing.T) {
	v := toExprValue(map[string]any{"x": 0.5, "y": 0.6})
	p, ok := v.(expr.Point)
	if !ok {
		t.Fatalf("want expr.Point, got %T", v)
	}
	if p.X != 0.5 || p.Y != 0.6 {
		t.Fatalf("point coords: got %+v", p)
	}
}

func TestToExprValue_IntPromotedToFloat64(t *testing.T) {
	if v := toExprValue(int(5)); v != float64(5) {
		t.Fatalf("int → float64: got %T(%v)", v, v)
	}
	if v := toExprValue(int64(7)); v != float64(7) {
		t.Fatalf("int64 → float64: got %T(%v)", v, v)
	}
}
