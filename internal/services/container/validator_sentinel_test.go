package container

import "testing"

// P1.2: validateSentinelScope 测试 — Break/Continue/Throw 作用域校验.

func TestValidate_BreakInsideLoop_OK(t *testing.T) {
	c := &Container{
		Graph: Graph{
			Nodes: []GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "loop", Kind: "Loop"},
				{ID: "brk", Kind: "Break"},
			},
			Edges: []GraphEdge{
				{From: "start.out", To: "loop.in"},
				{From: "loop.body", To: "brk.in"},
			},
		},
	}
	errs := validateSentinelScope(c)
	for _, e := range errs {
		if e.Code == CodeBreakOutsideLoop {
			t.Errorf("Break exec-reachable from loop.body should NOT emit BREAK_OUTSIDE_LOOP, got %+v", e)
		}
	}
}

func TestValidate_BreakInMainNoLoop_ERROR(t *testing.T) {
	c := &Container{
		Graph: Graph{
			Nodes: []GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "brk", Kind: "Break"},
			},
			Edges: []GraphEdge{
				{From: "start.out", To: "brk.in"},
			},
		},
	}
	errs := validateSentinelScope(c)
	found := false
	for _, e := range errs {
		if e.Code == CodeBreakOutsideLoop && e.NodeID == "brk" {
			found = true
		}
	}
	if !found {
		t.Errorf("Break in main without Loop should emit BREAK_OUTSIDE_LOOP, got %+v", errs)
	}
}

func TestValidate_ContinueInMainNoLoop_ERROR(t *testing.T) {
	c := &Container{
		Graph: Graph{
			Nodes: []GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "cont", Kind: "Continue"},
			},
			Edges: []GraphEdge{
				{From: "start.out", To: "cont.in"},
			},
		},
	}
	errs := validateSentinelScope(c)
	found := false
	for _, e := range errs {
		if e.Code == CodeContinueOutsideLoop && e.NodeID == "cont" {
			found = true
		}
	}
	if !found {
		t.Errorf("Continue in main without Loop should emit CONTINUE_OUTSIDE_LOOP, got %+v", errs)
	}
}

func TestValidate_ThrowInMainNoTryBody_ERROR(t *testing.T) {
	c := &Container{
		Graph: Graph{
			Nodes: []GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "thr", Kind: "Throw"},
			},
			Edges: []GraphEdge{
				{From: "start.out", To: "thr.in"},
			},
		},
	}
	errs := validateSentinelScope(c)
	found := false
	for _, e := range errs {
		if e.Code == CodeThrowOutsideTry && e.NodeID == "thr" {
			found = true
		}
	}
	if !found {
		t.Errorf("Throw in main without Try should emit THROW_OUTSIDE_TRY, got %+v", errs)
	}
}

func TestValidate_ThrowInTryBodySubgraph_OK(t *testing.T) {
	c := &Container{
		Graph: Graph{
			Nodes: []GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "try", Kind: "Try", Config: map[string]any{"SubgraphID": "sg-try-body"}},
			},
			Edges: []GraphEdge{
				{From: "start.out", To: "try.in"},
			},
		},
		Subgraphs: []Subgraph{
			{
				ID: "sg-try-body",
				Graph: Graph{
					Nodes: []GraphNode{
						{ID: "in", Kind: "SubgraphInput"},
						{ID: "thr", Kind: "Throw"},
					},
					Edges: []GraphEdge{
						{From: "in.out", To: "thr.in"},
					},
				},
			},
		},
	}
	errs := validateSentinelScope(c)
	for _, e := range errs {
		if e.Code == CodeThrowOutsideTry {
			t.Errorf("Throw inside Try body subgraph should NOT emit THROW_OUTSIDE_TRY, got %+v", e)
		}
	}
}
