package runtime

import (
	"context"
	"errors"
	"testing"
	"time"

	nodepkg "github.com/yottaapp/yotta/internal/node"
	"github.com/yottaapp/yotta/internal/services/container"
	"github.com/yottaapp/yotta/internal/services/execution"
)

// 多出口路由测试: 父图边以 callee OutputPins decl ID 为 pin, 调用节点按 body
// 实际到达的出口 marker 路由 — 不是无脑 "Done".

func runRoutingContainer(t *testing.T, c *container.Container, sgs []container.Subgraph) (*RuntimeContext, error) {
	t.Helper()
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, nil, nil, nil, 0)
	rt.Subgraphs = sgs
	stubRuntimeWindowAndInput(rt)
	r := NewContainerRunner(rt)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return rt, r.Run(ctx)
}

func setVarNode(id, varName string, value any) container.GraphNode {
	return container.GraphNode{ID: id, Kind: "SetVar", Config: map[string]any{
		"VarName": varName, "Value": value,
	}}
}

func TestSubgraphCall_RoutesByReachedDecl(t *testing.T) {
	sgs := []container.Subgraph{{
		ID:    "sg_m",
		Entry: container.SubgraphMarker{NodeID: "sgin"},
		OutputPins: []container.SubgraphOutputDecl{
			{ID: "d-ok", Name: "ok", NodeID: "out_ok"},
			{ID: "d-bad", Name: "bad", NodeID: "out_bad"},
		},
		Graph: container.Graph{
			Edges: []container.GraphEdge{{From: "sgin.Done", To: "out_bad.In"}},
		},
	}}
	c := &container.Container{
		SchemaVersion: 1,
		ID:            "route_test",
		Vars:          []container.VarDecl{{Name: "reached", Type: "string", Default: ""}},
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "call", Kind: "Subgraph", Config: map[string]any{"SubgraphID": "sg_m"}},
				setVarNode("okNode", "reached", "ok"),
				setVarNode("badNode", "reached", "bad"),
			},
			Edges: []container.GraphEdge{
				{From: "start.Done", To: "call.In"},
				{From: "call.d-ok", To: "okNode.In"},
				{From: "call.d-bad", To: "badNode.In"},
			},
		},
	}
	rt, err := runRoutingContainer(t, c, sgs)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := rt.Vars()["reached"]; got != "bad" {
		t.Errorf("reached = %v, want %q (body 到达 out_bad marker → 走 call.d-bad 边)", got, "bad")
	}
}

func TestSubgraphCall_DrainedSingleExit_FallsBackToOnlyDecl(t *testing.T) {
	// 子图体没接出口 marker (跑干) + 单出口 → 视同到达唯一出口, 父图继续.
	sgs := []container.Subgraph{{
		ID:    "sg_s",
		Entry: container.SubgraphMarker{NodeID: "sgin"},
		OutputPins: []container.SubgraphOutputDecl{
			{ID: "d-only", Name: "done", NodeID: "out_only"},
		},
		Graph: container.Graph{
			Nodes: []container.GraphNode{setVarNode("mid", "inner", true)},
			Edges: []container.GraphEdge{{From: "sgin.Done", To: "mid.In"}},
		},
	}}
	c := &container.Container{
		SchemaVersion: 1,
		ID:            "drain_single_test",
		Vars: []container.VarDecl{
			{Name: "inner", Type: "bool", Default: false},
			{Name: "after", Type: "bool", Default: false},
		},
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "call", Kind: "Subgraph", Config: map[string]any{"SubgraphID": "sg_s"}},
				setVarNode("afterNode", "after", true),
			},
			Edges: []container.GraphEdge{
				{From: "start.Done", To: "call.In"},
				{From: "call.d-only", To: "afterNode.In"},
			},
		},
	}
	rt, err := runRoutingContainer(t, c, sgs)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := rt.Vars()["inner"]; got != true {
		t.Errorf("inner = %v, want true (子图体跑过)", got)
	}
	if got := rt.Vars()["after"]; got != true {
		t.Errorf("after = %v, want true (单出口跑干 → fallback 唯一 decl)", got)
	}
}

func TestSubgraphCall_DrainedMultiExit_CodedError(t *testing.T) {
	// 多出口子图体跑干 (没到达任何出口 marker) → 出口歧义, Coded 错误冒泡.
	sgs := []container.Subgraph{{
		ID:    "sg_d",
		Entry: container.SubgraphMarker{NodeID: "sgin"},
		OutputPins: []container.SubgraphOutputDecl{
			{ID: "d-a", Name: "a", NodeID: "out_a"},
			{ID: "d-b", Name: "b", NodeID: "out_b"},
		},
		Graph: container.Graph{},
	}}
	c := &container.Container{
		SchemaVersion: 1,
		ID:            "drain_multi_test",
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "call", Kind: "Subgraph", Config: map[string]any{"SubgraphID": "sg_d"}},
			},
			Edges: []container.GraphEdge{{From: "start.Done", To: "call.In"}},
		},
	}
	_, err := runRoutingContainer(t, c, sgs)
	if err == nil {
		t.Fatal("expected subgraph_no_exit error, got nil")
	}
	var coded nodepkg.Coded
	if !errors.As(err, &coded) || coded.ErrCode() != nodepkg.CodeSubgraphNoExit {
		t.Errorf("err = %v, want Coded subgraph_no_exit", err)
	}
}
