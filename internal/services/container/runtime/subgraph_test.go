package runtime

import (
	"context"
	"strings"
	"testing"
	"time"

	"yotta/internal/services/container"
	"yotta/internal/services/execution"
	"yotta/internal/services/expr"
)

func TestResolveSubgraphCall_OK(t *testing.T) {
	c := &container.Container{
		ID: "c1",
		Subgraphs: []container.Subgraph{
			{ID: "sg-A", Label: "A"},
		},
	}
	callNode := &container.GraphNode{
		ID:        "call",
		Kind:      "Subgraph",
		Config:    map[string]any{"SubgraphID": "sg-A"},
		CreatedAt: time.Now().UTC(),
	}
	sg, err := ResolveSubgraphCall(c, callNode)
	if err != nil {
		t.Fatalf("expected ok, got %v", err)
	}
	if sg.ID != "sg-A" {
		t.Errorf("got %q", sg.ID)
	}
}

func TestResolveSubgraphCall_MissingConfig(t *testing.T) {
	c := &container.Container{ID: "c1"}
	callNode := &container.GraphNode{ID: "call", Kind: "Subgraph", Config: map[string]any{}, CreatedAt: time.Now().UTC()}
	_, err := ResolveSubgraphCall(c, callNode)
	if err == nil || !strings.Contains(err.Error(), "SubgraphID") {
		t.Errorf("expected SubgraphID err, got %v", err)
	}
}

func TestResolveSubgraphCall_NotFound(t *testing.T) {
	c := &container.Container{ID: "c1", Subgraphs: nil}
	callNode := &container.GraphNode{ID: "call", Kind: "Subgraph", Config: map[string]any{"SubgraphID": "ghost"}, CreatedAt: time.Now().UTC()}
	_, err := ResolveSubgraphCall(c, callNode)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected not found err, got %v", err)
	}
}

// TestSubgraph_MultiCallSiteRouting 防回归: 同 parent graph 调用同 subgraph 两次,
// 两次 Done 必须分别路由回各自 call-site 下游 (callA.Done → markA, callB.Done → markB),
// 不能把第二次的 Done 也错路由回第一次的 markA.
func TestSubgraph_MultiCallSiteRouting(t *testing.T) {
	sg := container.Subgraph{
		ID:         "sub_inc",
		Label:      "sub_inc",
		Entry:      container.SubgraphMarker{NodeID: "sin"},
		OutputPins: []container.SubgraphOutputDecl{{ID: "done", Name: "done", NodeID: "sout"}},
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "inc", Kind: "IncVar", Config: map[string]any{
					"VarName": "visited", "Scope": "global",
					"literal": map[string]any{"Delta": 1.0},
				}},
			},
			Edges: []container.GraphEdge{
				{From: "sin.Done", To: "inc.In"},
				{From: "inc.Done", To: "sout.In"},
			},
		},
	}
	c := &container.Container{
		SchemaVersion: 1,
		ID:            "multi-call-test",
		Name:          "multi-call-test",
		Vars: []container.VarDecl{
			{Name: "visited", Type: "number", Default: 0.0},
			{Name: "markA", Type: "string", Default: ""},
			{Name: "markB", Type: "string", Default: ""},
		},
		Subgraphs: []container.Subgraph{sg},
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "callA", Kind: "Subgraph", Config: map[string]any{"SubgraphID": "sub_inc"}},
				{ID: "setA", Kind: "SetVar", Config: map[string]any{
					"VarName": "markA", "Scope": "global",
					"literal": map[string]any{"Value": "A"},
				}},
				{ID: "callB", Kind: "Subgraph", Config: map[string]any{"SubgraphID": "sub_inc"}},
				{ID: "setB", Kind: "SetVar", Config: map[string]any{
					"VarName": "markB", "Scope": "global",
					"literal": map[string]any{"Value": "B"},
				}},
				{ID: "stop", Kind: "Stop"},
			},
			Edges: []container.GraphEdge{
				{From: "start.Done", To: "callA.in"},
				{From: "callA.done", To: "setA.In"},
				{From: "setA.Done", To: "callB.in"},
				{From: "callB.done", To: "setB.In"},
				{From: "setB.Done", To: "stop.in"},
			},
		},
	}
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, nil, nil, nil, 0)
	stubRuntimeWindowAndInput(rt)
	r := NewContainerRunner(rt)
	if err := r.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got, _ := expr.AsNumber(rt.Vars()["visited"]); got != 2.0 {
		t.Errorf("visited = %v, want 2 (subgraph body 应每次 call 跑一遍)", got)
	}
	if rt.Vars()["markA"] != "A" {
		t.Errorf("markA = %v, want \"A\" (callA.Done 应路由到 setA)", rt.Vars()["markA"])
	}
	if rt.Vars()["markB"] != "B" {
		t.Errorf("markB = %v, want \"B\" (callB.Done 应路由到 setB 而非 setA — call-site routing 回归点)", rt.Vars()["markB"])
	}
}
