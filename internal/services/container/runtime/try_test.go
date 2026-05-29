package runtime

import (
	"context"
	"testing"

	"yhbox/internal/services/container"
)

// Try done: 子图正常完成 → done 出口
// 注意: 节点 ID 不能含 "." — edgeIndex 用 SplitN(to, ".", 2) 解析 "<nodeId>.<pin>",
// 含点的 ID 会被错误分割 (已知 runtime 设计限制).
func TestTryDonePath(t *testing.T) {
	_, r := newTestRunnerWithSubgraph(t, "sg1", []*container.GraphNode{
		{ID: "sg1in", Kind: "SubgraphInput"},
		{ID: "sg1out", Kind: "SubgraphOutput", Config: map[string]any{"DeclID": "done"}},
	}, []container.GraphEdge{
		{From: "sg1in.Done", To: "sg1out.In"},
	})
	node := &container.GraphNode{
		ID:     "try1",
		Kind:   "Try",
		Config: map[string]any{"SubgraphID": "sg1", "literal": map[string]any{"TimeoutMs": 5000.0}}, // literal pin
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

// Try 行为由 dispatch_v5_test.go (TryDone/TryCatch/TryThrow/TryMissingSubgraphID) 覆盖.
