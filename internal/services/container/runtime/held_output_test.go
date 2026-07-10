package runtime

import (
	"context"
	"testing"

	"github.com/yottaapp/yotta/internal/services/container"
	"github.com/yottaapp/yotta/internal/services/execution"
)

// held output 语义全覆盖 (spec 2026-06-23-held-exec-outputs §6/§9). 写/读/跨跳的基本盘在
// dispatch_v5_test.go (TestCaptureExecOutputs_WritesPerRunCache / TestHeldOutput_CrossHopReadsCache);
// 这里补 fan-out / 未fire / 稀疏 / loop 覆盖写 / 子图作用域键唯一。

// TestHeldOutput_FanOut — 源一次 fire, 两个并联消费者各自读同一缓存值.
func TestHeldOutput_FanOut(t *testing.T) {
	resetTdEcho()
	c := &container.Container{
		SchemaVersion: 1, ID: "test-heldoutput-fanout",
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "n1", Kind: tkFailf},
				{ID: "a", Kind: tkEcho},
				{ID: "b", Kind: tkEcho},
			},
			Edges: []container.GraphEdge{
				{From: "n1.Fail", To: "a.in"},
				{From: "n1.Code", To: "a.Value"},
				{From: "n1.Code", To: "b.Value"},
			},
		},
	}
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, nil, nil, nil, 0)
	r := NewContainerRunner(rt)
	if _, err := r.execNodeViaFramework(context.Background(), r.nodesByID["n1"], ExecToken{NodeID: "n1", InPin: "in"}); err != nil {
		t.Fatalf("n1: %v", err)
	}
	if _, err := r.execNodeViaFramework(context.Background(), r.nodesByID["a"], ExecToken{NodeID: "a", InPin: "in"}); err != nil {
		t.Fatalf("a: %v", err)
	}
	if got := getTdEchoLast(); got != "capture_failed" {
		t.Errorf("a.Value = %v, want capture_failed", got)
	}
	if _, err := r.execNodeViaFramework(context.Background(), r.nodesByID["b"], ExecToken{NodeID: "b", InPin: "in"}); err != nil {
		t.Fatalf("b: %v", err)
	}
	if got := getTdEchoLast(); got != "capture_failed" {
		t.Errorf("b.Value = %v, want capture_failed (fan-out 同缓存值)", got)
	}
}

// TestHeldOutput_NotFired — 源从未 fire → 缓存无键 → 消费方走默认 (nil), 不 panic.
func TestHeldOutput_NotFired(t *testing.T) {
	resetTdEcho()
	c := &container.Container{
		SchemaVersion: 1, ID: "test-heldoutput-notfired",
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "n1", Kind: tkFailf},
				{ID: "sink", Kind: tkEcho},
			},
			Edges: []container.GraphEdge{
				{From: "n1.Code", To: "sink.Value"},
			},
		},
	}
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, nil, nil, nil, 0)
	r := NewContainerRunner(rt)
	// 不跑 n1 → 缓存无键. 直接跑 sink.
	if _, err := r.execNodeViaFramework(context.Background(), r.nodesByID["sink"], ExecToken{NodeID: "sink", InPin: "in"}); err != nil {
		t.Fatalf("sink: %v", err)
	}
	if got := getTdEchoLast(); got != nil {
		t.Errorf("sink.Value = %v, want nil (源未 fire → 缓存无键 → 默认)", got)
	}
}

// TestHeldOutput_Sparse — 某次 fire 未带某字段 → 缓存留旧值 (稀疏写, captureExecOutputs 只写本次带的字段).
func TestHeldOutput_Sparse(t *testing.T) {
	c := &container.Container{
		SchemaVersion: 1, ID: "test-heldoutput-sparse",
		Graph: container.Graph{Nodes: []container.GraphNode{{ID: "n1", Kind: tkFailf}}},
	}
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, nil, nil, nil, 0)
	r := NewContainerRunner(rt)
	n1 := r.nodesByID["n1"]
	r.captureExecOutputs(n1, map[string]any{"Code": "v1"})  // 第一次 fire 带 Code
	r.captureExecOutputs(n1, map[string]any{"Other": "v2"}) // 第二次 fire 走别的出口、不带 Code
	if got := r.execOutputs["n1.Code"]; got != "v1" {
		t.Errorf("execOutputs[n1.Code] = %v, want v1 (稀疏: 第二次未带 Code → 留旧值)", got)
	}
	if got := r.execOutputs["n1.Other"]; got != "v2" {
		t.Errorf("execOutputs[n1.Other] = %v, want v2", got)
	}
}

// TestHeldOutput_LoopOverwriteLastValue — 循环体内同节点多轮 fire: 缓存覆盖写, 循环外读最后一轮.
// (region body 内的 fire 也经 routeResult → captureExecOutputs, 端到端 region 路径由
// TestHeldOutput_SubgraphCrossHop 证; 此处直证覆盖写=最新一轮的核心语义.)
func TestHeldOutput_LoopOverwriteLastValue(t *testing.T) {
	c := &container.Container{
		SchemaVersion: 1, ID: "test-heldoutput-loop",
		Graph: container.Graph{Nodes: []container.GraphNode{{ID: "n1", Kind: tkFailf}}},
	}
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, nil, nil, nil, 0)
	r := NewContainerRunner(rt)
	n1 := r.nodesByID["n1"]
	for i := 1; i <= 3; i++ {
		r.captureExecOutputs(n1, map[string]any{"Code": i})
	}
	if got := r.execOutputs["n1.Code"]; got != 3 {
		t.Errorf("execOutputs[n1.Code] = %v, want 3 (loop 覆盖写 → 最后一轮值)", got)
	}
}

// TestHeldOutput_SubgraphCrossHop — 子图内 exec 出口字段跨跳直连: 缓存键随子图节点 ID (全局唯一)
// 写/读一致, 切表 (nodesByID/dataEdges swap) 不破缓存. 端到端跑一次子图.
func TestHeldOutput_SubgraphCrossHop(t *testing.T) {
	resetTdEcho()
	sgs := []container.Subgraph{{
		ID:    "sg_h",
		Entry: container.SubgraphMarker{NodeID: "sgin"},
		OutputPins: []container.SubgraphOutputDecl{
			{ID: "d-ok", Name: "ok", NodeID: "out_ok"},
		},
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "failer", Kind: tkFailf},
				{ID: "sink", Kind: tkEcho},
			},
			Edges: []container.GraphEdge{
				{From: "sgin.Done", To: "failer.in"},
				{From: "failer.Fail", To: "sink.in"},    // exec: failer → sink
				{From: "failer.Code", To: "sink.Value"}, // data: 子图内 exec 出口字段直连
				{From: "sink.Out", To: "out_ok.In"},
			},
		},
	}}
	c := &container.Container{
		SchemaVersion: 1, ID: "test-heldoutput-subgraph",
		Graph: container.Graph{Nodes: []container.GraphNode{{ID: "start", Kind: "Start"}}},
	}
	r, _ := newCallerRunner(t, c, sgs)
	exit, err := r.CallSubgraph(context.Background(), "sg_h", nil)
	if err != nil {
		t.Fatalf("CallSubgraph: %v", err)
	}
	if exit != "ok" {
		t.Errorf("exit = %q, want ok", exit)
	}
	if got := getTdEchoLast(); got != "capture_failed" {
		t.Errorf("sink.Value = %v, want capture_failed (子图内 held output 直连, 键随子图节点 ID 唯一)", got)
	}
}
