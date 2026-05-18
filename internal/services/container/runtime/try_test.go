package runtime

import (
	"context"
	"testing"
	"time"

	"yhbox/internal/services/container"
)

// Try done: 子图正常完成 → done 出口
// 注意: 节点 ID 不能含 "." — edgeIndex 用 SplitN(to, ".", 2) 解析 "<nodeId>.<pin>",
// 含点的 ID 会被错误分割 (已知 runtime 设计限制).
func TestTryDonePath(t *testing.T) {
	_, r := newTestRunnerWithSubgraph(t, "sg1", []*container.GraphNode{
		{ID: "sg1in", Kind: "SubgraphInput"},
		{ID: "sg1out", Kind: "SubgraphOutput", Config: map[string]any{"declID": "done"}},
	}, []container.GraphEdge{
		{From: "sg1in.out", To: "sg1out.in"},
	})
	node := &container.GraphNode{
		ID:     "try1",
		Kind:   "Try",
		Config: map[string]any{"subgraphId": "sg1", "literal": map[string]any{"timeoutMs": 5000.0}}, // v4 literal pin
	}
	if r.nodesByID == nil {
		r.nodesByID = map[string]*container.GraphNode{}
	}
	r.nodesByID[node.ID] = node
	toks, err := r.execNode(context.Background(), node, ExecToken{InPin: "in"})
	if err != nil {
		t.Fatalf("done path err: %v", err)
	}
	// done 出口无边连出 → toks 为空 (edges.next 返 nil)
	// 关键是没报错且走 done pin (不是 error/timeout)
	_ = toks
	// errorMsg 应为空 (done 路径)
	if got := r.rt.Sys().LastTry.ErrorMsg; got != "" {
		t.Fatalf("done path: expected empty errorMsg, got %q", got)
	}
}

// Try timeout: 子图含 Sleep > timeoutMs → timeout 出口, 提前返回
func TestTryTimeoutPath(t *testing.T) {
	_, r := newTestRunnerWithSubgraph(t, "sg2", []*container.GraphNode{
		{ID: "sg2in", Kind: "SubgraphInput"},
		{ID: "sg2sleep", Kind: "Sleep", Config: map[string]any{"literal": map[string]any{"durationMs": 500.0}}}, // v4 literal
		{ID: "sg2out", Kind: "SubgraphOutput", Config: map[string]any{"declID": "done"}},
	}, []container.GraphEdge{
		{From: "sg2in.out", To: "sg2sleep.in"},
		{From: "sg2sleep.out", To: "sg2out.in"},
	})
	node := &container.GraphNode{
		ID:     "try2",
		Kind:   "Try",
		Config: map[string]any{"subgraphId": "sg2", "literal": map[string]any{"timeoutMs": 100.0}}, // v4 literal pin
	}
	if r.nodesByID == nil {
		r.nodesByID = map[string]*container.GraphNode{}
	}
	r.nodesByID[node.ID] = node // v4: pullDataPin lookup
	start := time.Now()
	_, err := r.execNode(context.Background(), node, ExecToken{InPin: "in"})
	if err != nil {
		t.Fatalf("timeout path should set out pin not return err, got: %v", err)
	}
	elapsed := time.Since(start)
	if elapsed >= 400*time.Millisecond {
		t.Fatalf("timeout should fire at ~100ms, took %v", elapsed)
	}
	// errorMsg 应含 "timeout"
	if got := r.rt.Sys().LastTry.ErrorMsg; got == "" {
		t.Fatalf("timeout path: expected non-empty errorMsg, got empty")
	}
}

// Try error: 子图含 Throw → error 出口, errorMsg 数据传出
func TestTryErrorPath(t *testing.T) {
	_, r := newTestRunnerWithSubgraph(t, "sg3", []*container.GraphNode{
		{ID: "sg3in", Kind: "SubgraphInput"},
		{ID: "sg3throw", Kind: "Throw", Config: map[string]any{"literal": map[string]any{"message": "boom"}}}, // v4 literal
	}, []container.GraphEdge{
		{From: "sg3in.out", To: "sg3throw.in"},
	})
	node := &container.GraphNode{
		ID:     "try3",
		Kind:   "Try",
		Config: map[string]any{"subgraphId": "sg3", "literal": map[string]any{"timeoutMs": 5000.0}}, // v4 literal pin
	}
	if r.nodesByID == nil {
		r.nodesByID = map[string]*container.GraphNode{}
	}
	r.nodesByID[node.ID] = node
	_, err := r.execNode(context.Background(), node, ExecToken{InPin: "in"})
	if err != nil {
		t.Fatalf("error path should not bubble err (Try catches), got: %v", err)
	}
	// errorMsg 应填 "boom"
	if got := r.rt.Sys().LastTry.ErrorMsg; got != "boom" {
		t.Fatalf("expected errorMsg 'boom', got %q", got)
	}
}
