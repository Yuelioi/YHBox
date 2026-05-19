package container

import "testing"

func TestValidateDataGraph_DetectsCycle(t *testing.T) {
	c := &Container{
		SchemaVersion: 1,
		Graph: Graph{
			Nodes: []GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "e1", Kind: "Expr", Config: map[string]any{
					"expr":   "a + 1",
					"inputs": []any{map[string]any{"name": "a", "type": "number"}},
				}},
				{ID: "e2", Kind: "Expr", Config: map[string]any{
					"expr":   "b * 2",
					"inputs": []any{map[string]any{"name": "b", "type": "number"}},
				}},
			},
			Edges: []GraphEdge{
				{From: "e1.value", To: "e2.b"},
				{From: "e2.value", To: "e1.a"}, // cycle e1→e2→e1 (data edges derived from Expr.value out-pin)
			},
		},
	}
	errs := ValidateContainer(c)
	found := false
	for _, e := range errs {
		if e.Code == CodeDataGraphCycle {
			found = true
		}
	}
	if !found {
		t.Fatalf("want DATA_GRAPH_CYCLE, got %+v", errs)
	}
}

func TestValidateDataGraph_AcceptsDAG(t *testing.T) {
	c := &Container{
		SchemaVersion: 1,
		Vars:          []VarDecl{{Name: "x", Type: "number", Default: 0.0}},
		Graph: Graph{
			Nodes: []GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "gv", Kind: "GetVar", Config: map[string]any{"varName": "x", "scope": "global"}},
				{ID: "add", Kind: "Add"},
				{ID: "sv", Kind: "SetVar", Config: map[string]any{"varName": "x", "scope": "global"}},
			},
			Edges: []GraphEdge{
				{From: "start.out", To: "sv.in"},
				{From: "gv.value", To: "add.a"},
				{From: "add.result", To: "sv.value"},
			},
		},
	}
	errs := ValidateContainer(c)
	for _, e := range errs {
		if e.Code == CodeDataGraphCycle {
			t.Errorf("DAG falsely flagged as cycle: %+v", e)
		}
	}
}

// Exec edges (Kind="" or "exec") must NOT count as cycles even when there's a loopback.
func TestValidateDataGraph_IgnoresExecLoopback(t *testing.T) {
	c := &Container{
		SchemaVersion: 1,
		Graph: Graph{
			Nodes: []GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "loop", Kind: "Loop"},
				{ID: "stop", Kind: "Stop"},
			},
			Edges: []GraphEdge{
				{From: "start.out", To: "loop.in"},
				{From: "loop.body", To: "loop.loopback"}, // exec self-loop (Loop's nature)
				{From: "loop.complete", To: "stop.in"},
			},
		},
	}
	errs := ValidateContainer(c)
	for _, e := range errs {
		if e.Code == CodeDataGraphCycle {
			t.Errorf("exec self-loop falsely flagged as data cycle: %+v", e)
		}
	}
}
