package container

import "testing"

// validateSentinelScope 测试 — Break/Continue 作用域校验.

func TestValidate_BreakInsideLoop_OK(t *testing.T) {
	c := &Container{
		Graph: Graph{
			Nodes: []GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "loop", Kind: "Loop"},
				{ID: "brk", Kind: "Break"},
			},
			Edges: []GraphEdge{
				{From: "start.Done", To: "loop.In"},
				{From: "loop.Body", To: "brk.In"},
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
				{From: "start.Done", To: "brk.In"},
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
				{From: "start.Done", To: "cont.In"},
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

// Try 删除后 Throw 不再受 scope 限制 — 由 region Fail 出口截获或冒泡, 任意位置合法.
// validateSentinelScope 对 Throw 不应产生任何 sentinel-scope 错误.
func TestValidate_ThrowAnywhere_NoScopeError(t *testing.T) {
	c := &Container{
		Graph: Graph{
			Nodes: []GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "thr", Kind: "Throw"},
			},
			Edges: []GraphEdge{
				{From: "start.Done", To: "thr.In"},
			},
		},
	}
	errs := validateSentinelScope(c)
	for _, e := range errs {
		if e.NodeID == "thr" {
			t.Errorf("Throw in main should produce no sentinel-scope error, got %+v", e)
		}
	}
}
