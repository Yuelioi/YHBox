package runtime

import (
	"context"
	"errors"
	"testing"
	"time"

	nodepkg "yotta/internal/node"
	"yotta/internal/services/container"
	"yotta/internal/services/execution"
)

// CallSubgraph (脚本调子图入口) 测试: 同步跑 callee 返到达出口的 decl Name,
// params seed + 声明 default 补全, 递归深度防护.

func newCallerRunner(t *testing.T, c *container.Container, sgs []container.Subgraph) (*ContainerRunner, *RuntimeContext) {
	t.Helper()
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, nil, nil, nil, 0)
	rt.Subgraphs = sgs
	stubRuntimeWindowAndInput(rt)
	return NewContainerRunner(rt), rt
}

func paramEchoContainer() (*container.Container, []container.Subgraph) {
	sgs := []container.Subgraph{{
		ID:    "sg_p",
		Entry: container.SubgraphMarker{NodeID: "sgin"},
		OutputPins: []container.SubgraphOutputDecl{
			{ID: "d-ok", Name: "ok", NodeID: "out_ok"},
		},
		InputParams: []container.SubgraphInputParam{
			{Name: "msg", Type: "string", Default: "dv"},
		},
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "gp", Kind: "GetParam", Config: map[string]any{
					"literal": map[string]any{"ParamName": "msg"},
				}},
				{ID: "sv", Kind: "SetVar", Config: map[string]any{"VarName": "got"}},
			},
			Edges: []container.GraphEdge{
				{From: "sgin.Done", To: "sv.In"},
				{From: "sv.Done", To: "out_ok.In"},
				{From: "gp.Value", To: "sv.Value"},
			},
		},
	}}
	c := &container.Container{
		SchemaVersion: 1,
		ID:            "call_sg_test",
		Vars:          []container.VarDecl{{Name: "got", Type: "string", Default: ""}},
		Graph: container.Graph{
			Nodes: []container.GraphNode{{ID: "start", Kind: "Start"}},
		},
	}
	return c, sgs
}

func TestCallSubgraph_ParamSeedAndExitName(t *testing.T) {
	c, sgs := paramEchoContainer()
	r, rt := newCallerRunner(t, c, sgs)
	exit, err := r.CallSubgraph(context.Background(), "sg_p", map[string]any{"msg": "hello"})
	if err != nil {
		t.Fatalf("CallSubgraph: %v", err)
	}
	if exit != "ok" {
		t.Errorf("exit = %q, want %q (decl Name 而非 decl ID)", exit, "ok")
	}
	if got := rt.Vars()["got"]; got != "hello" {
		t.Errorf("got = %v, want hello (params seed 进 callee LocalParams)", got)
	}
}

func TestCallSubgraph_MissingParamFallsBackToDefault(t *testing.T) {
	c, sgs := paramEchoContainer()
	r, rt := newCallerRunner(t, c, sgs)
	if _, err := r.CallSubgraph(context.Background(), "sg_p", map[string]any{}); err != nil {
		t.Fatalf("CallSubgraph: %v", err)
	}
	if got := rt.Vars()["got"]; got != "dv" {
		t.Errorf("got = %v, want dv (声明入参缺省补 Default)", got)
	}
}

func TestCallSubgraph_MultiExitReturnsReachedName(t *testing.T) {
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
		ID:            "call_sg_multi",
		Graph:         container.Graph{Nodes: []container.GraphNode{{ID: "start", Kind: "Start"}}},
	}
	r, _ := newCallerRunner(t, c, sgs)
	exit, err := r.CallSubgraph(context.Background(), "sg_m", nil)
	if err != nil {
		t.Fatalf("CallSubgraph: %v", err)
	}
	if exit != "bad" {
		t.Errorf("exit = %q, want bad", exit)
	}
}

func TestCallSubgraph_UnknownSubgraph_Error(t *testing.T) {
	c, sgs := paramEchoContainer()
	r, _ := newCallerRunner(t, c, sgs)
	if _, err := r.CallSubgraph(context.Background(), "ghost", nil); err == nil {
		t.Fatal("expected error for unknown subgraph id")
	}
}

func TestScriptCallsSubgraph_EndToEnd(t *testing.T) {
	c, sgs := paramEchoContainer()
	c.Vars = append(c.Vars, container.VarDecl{Name: "exitName", Type: "string", Default: ""})
	c.Graph = container.Graph{
		Nodes: []container.GraphNode{
			{ID: "start", Kind: "Start"},
			{ID: "sc", Kind: "Script", Config: map[string]any{
				"literal": map[string]any{
					"Code": `let r = Subgraph({SubgraphID: "sg_p", msg: "yo"});
SetVar({VarName: "exitName", Value: r.exit});
return r.exit`,
				},
			}},
		},
		Edges: []container.GraphEdge{{From: "start.Done", To: "sc.In"}},
	}
	r, rt := newCallerRunner(t, c, sgs)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := r.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := rt.Vars()["got"]; got != "yo" {
		t.Errorf("got = %v, want yo (脚本入参传进 callee)", got)
	}
	if got := rt.Vars()["exitName"]; got != "ok" {
		t.Errorf("exitName = %v, want ok (脚本拿到出口名)", got)
	}
}

func TestScriptCallsSubgraph_CancelUnblocksCalleeSleep(t *testing.T) {
	sgs := []container.Subgraph{{
		ID:    "sg_sleep",
		Entry: container.SubgraphMarker{NodeID: "sgin"},
		OutputPins: []container.SubgraphOutputDecl{
			{ID: "d-ok", Name: "ok", NodeID: "out_ok"},
		},
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "zz", Kind: "Sleep", Config: map[string]any{
					"literal": map[string]any{"Duration": 60000.0},
				}},
			},
			Edges: []container.GraphEdge{
				{From: "sgin.Done", To: "zz.In"},
				{From: "zz.Done", To: "out_ok.In"},
			},
		},
	}}
	c := &container.Container{
		SchemaVersion: 1,
		ID:            "script_sg_cancel",
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "sc", Kind: "Script", Config: map[string]any{
					"literal": map[string]any{"Code": `Subgraph({SubgraphID: "sg_sleep"})`},
				}},
			},
			Edges: []container.GraphEdge{{From: "start.Done", To: "sc.In"}},
		},
	}
	r, _ := newCallerRunner(t, c, sgs)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()
	start := time.Now()
	_ = r.Run(ctx)
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("cancel 后 %v 才返回 — 子图内阻塞节点没随 ctx 取消", elapsed)
	}
}

func TestCallSubgraph_DepthGuard(t *testing.T) {
	c, sgs := paramEchoContainer()
	r, _ := newCallerRunner(t, c, sgs)
	// 人工压满 frame 栈模拟深递归 (脚本里子图套 Script 再调回自己).
	sg := &sgs[0]
	for range 64 {
		r.state.PushFrame(container.MainGraphRef(), sg, "fake")
	}
	_, err := r.CallSubgraph(context.Background(), "sg_p", nil)
	if err == nil {
		t.Fatal("expected depth-guard error, got nil")
	}
	var coded nodepkg.Coded
	if !errors.As(err, &coded) || coded.ErrCode() != nodepkg.CodeSubgraphRecursion {
		t.Errorf("err = %v, want Coded subgraph_recursion", err)
	}
}
