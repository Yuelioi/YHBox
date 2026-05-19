package runtime

import (
	"context"
	"testing"

	"yhbox/internal/services/container"
)

// TestExecCollapsedNode_DispatchesViaSameHandlerAsSubgraph: CollapsedNode kind
// 跟 Subgraph 走同 dispatch, 后备 isAnonymous=true subgraph 能正常入帧返出口.
//
// 模拟形态: 主图 Start → CollapsedNode(target=sg_collapsed) → 无后续;
//
//	sg_collapsed (isAnonymous=true): SubgraphInput → SubgraphOutput
func TestExecCollapsedNode_DispatchesViaSameHandlerAsSubgraph(t *testing.T) {
	sgID := "sg_collapsed"
	rt, r := newTestRunnerWithSubgraph(t, sgID, []*container.GraphNode{
		{ID: "sgin", Kind: "SubgraphInput", Config: map[string]any{}},
		{ID: "sgout", Kind: "SubgraphOutput", Config: map[string]any{"declID": "done"}},
	}, []container.GraphEdge{
		{From: "sgin.out", To: "sgout.in"},
	})
	// 给后备 subgraph 设 isAnonymous=true (验证 dispatch 不拒绝)
	for i := range rt.Container.Subgraphs {
		if rt.Container.Subgraphs[i].ID == sgID {
			rt.Container.Subgraphs[i].IsAnonymous = true
		}
	}
	node := &container.GraphNode{
		ID:     "cn1",
		Kind:   "CollapsedNode",
		Config: map[string]any{"subgraphId": sgID},
	}
	// dispatch 不报错就证明 Kind switch case 命中
	_, err := r.execNode(context.Background(), node, ExecToken{InPin: "in"})
	if err != nil {
		t.Fatalf("CollapsedNode dispatch 应跟 Subgraph 一致, err: %v", err)
	}
}
