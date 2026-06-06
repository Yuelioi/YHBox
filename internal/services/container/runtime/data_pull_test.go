package runtime

import (
	"context"
	"strings"
	"testing"

	"yotta/internal/services/container"
	"yotta/internal/services/expr"
)

func TestPullDataPin_Literal(t *testing.T) {
	_, r := newTestRunner(t)
	n := &container.GraphNode{
		ID: "n1", Kind: "Sleep",
		Config: map[string]any{
			"literal": map[string]any{
				"Duration": 500.0,
			},
		},
	}
	r.nodesByID = map[string]*container.GraphNode{"n1": n}

	v, err := r.pullDataPin(context.Background(), "n1", "Duration")
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

	v, _ := r.pullDataPin(context.Background(), "n1", "Duration")
	if v != nil {
		t.Fatalf("no edge no literal: want nil, got %v", v)
	}
}

// Verifies that pullDataPin follows a data edge to a GetVar source and resolves it.
func TestPullDataPin_FromGetVarEdge(t *testing.T) {
	_, r := newTestRunner(t)
	ctx := withTickSnapshot(context.Background(), CaptureSnapshot(map[string]expr.Value{"hp": float64(0.8)}))

	src := &container.GraphNode{
		ID: "gv", Kind: "GetVar",
		Config: map[string]any{"VarName": "hp", "Scope": "global"},
	}
	dst := &container.GraphNode{ID: "sleep", Kind: "Sleep", Config: map[string]any{}}
	r.nodesByID = map[string]*container.GraphNode{"gv": src, "sleep": dst}
	r.dataEdges = buildDataEdgeIndex(container.Graph{
		Nodes: []container.GraphNode{*src, *dst},
		Edges: []container.GraphEdge{
			{From: "gv.Value", To: "sleep.Duration"},
		},
	})

	v, err := r.pullDataPin(ctx, "sleep", "Duration")
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
	ctx := withTickSnapshot(context.Background(), CaptureSnapshot(map[string]expr.Value{"hp": float64(0.8)}))

	src := &container.GraphNode{
		ID: "gv", Kind: "GetVar",
		Config: map[string]any{"VarName": "hp", "Scope": "global"},
	}
	dst := &container.GraphNode{
		ID: "sleep", Kind: "Sleep",
		Config: map[string]any{
			"literal": map[string]any{"Duration": 999.0}, // should be ignored
		},
	}
	r.nodesByID = map[string]*container.GraphNode{"gv": src, "sleep": dst}
	r.dataEdges = buildDataEdgeIndex(container.Graph{
		Nodes: []container.GraphNode{*src, *dst},
		Edges: []container.GraphEdge{
			{From: "gv.Value", To: "sleep.Duration"},
		},
	})

	v, _ := r.pullDataPin(ctx, "sleep", "Duration")
	if got, _ := expr.AsNumber(v); got != 0.8 {
		t.Fatalf("edge must win over literal: want 0.8, got %v", v)
	}
}

func TestDataEdgeIndex_IgnoresExecEdges(t *testing.T) {
	// edge type derived from (from-node.kind, from-pin).
	// Sleep.out is exec-out (not in DataOut) → filtered out.
	// GetVar.value is data-out → kept.
	idx := buildDataEdgeIndex(container.Graph{
		Nodes: []container.GraphNode{
			{ID: "a", Kind: "Sleep"},
			{ID: "b", Kind: "Sleep"},
			{ID: "gv", Kind: "GetVar", Config: map[string]any{"VarName": "x", "Scope": "global"}},
			{ID: "c", Kind: "SetVar", Config: map[string]any{"VarName": "x", "Scope": "global"}},
		},
		Edges: []container.GraphEdge{
			{From: "a.Done", To: "b.in"},      // Sleep.out is exec-out → not data
			{From: "gv.Value", To: "c.x"},    // GetVar.value is data-out → data
		},
	})
	if src, pin := idx.Source("b", "in"); src != "" || pin != "" {
		t.Fatalf("exec edge leaked into data index: %s.%s", src, pin)
	}
	if src, pin := idx.Source("c", "x"); src != "gv" || pin != "Value" {
		t.Fatalf("data edge: want gv.Value, got %s.%s", src, pin)
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

// evalDataSource must reject exec kinds as data-edge sources.
// Catches drift where a new exec kind's data-out is mistakenly pulled
// instead of read from the sys snapshot.
func TestEvalDataSourceRejectsExecKind(t *testing.T) {
	_, r := newTestRunner(t)
	r.nodesByID = map[string]*container.GraphNode{
		"n1": {ID: "n1", Kind: "Sleep"}, // exec kind, not pure-data
	}
	_, err := r.evalDataSource(context.Background(), "n1", "out")
	if err == nil {
		t.Fatal("expected error for exec-kind source, got nil")
	}
	if !strings.Contains(err.Error(), "not pure-data") {
		t.Errorf("wrong error msg: %v", err)
	}
}

// evalDataSource must reject unknown kinds (registry miss).
func TestEvalDataSourceRejectsUnknownKind(t *testing.T) {
	_, r := newTestRunner(t)
	r.nodesByID = map[string]*container.GraphNode{
		"n1": {ID: "n1", Kind: "Bogus"},
	}
	_, err := r.evalDataSource(context.Background(), "n1", "out")
	if err == nil {
		t.Fatal("expected error for unknown kind, got nil")
	}
	if !strings.Contains(err.Error(), "unknown kind") {
		t.Errorf("wrong error msg: %v", err)
	}
}
