package container

import "testing"

func TestWindowTargetForNode(t *testing.T) {
	// 构造 Start → WT_A(Title=A) → n1(ClickAt) → WT_B(Title=B) → n2(ClickAt)
	// 用 exec 边 "<id>.<outpin>"→"<id>.In" 串联.
	c := &Container{Graph: Graph{
		Nodes: []GraphNode{
			{ID: "start", Kind: "Start"},
			{ID: "wta", Kind: "WindowTarget", Config: map[string]any{"Title": "A"}},
			{ID: "n1", Kind: "ClickAt"},
			{ID: "wtb", Kind: "WindowTarget", Config: map[string]any{"Title": "B"}},
			{ID: "n2", Kind: "ClickAt"},
		},
		Edges: []GraphEdge{
			{From: "start.Done", To: "wta.In"},
			{From: "wta.Done", To: "n1.In"},
			{From: "n1.Done", To: "wtb.In"},
			{From: "wtb.Done", To: "n2.In"},
		},
	}}
	if wt := windowTargetForNode(c, "n2"); wt == nil || PinString(wt, "Title") != "B" {
		t.Fatalf("n2 应回溯到 WT_B, got %v", wt)
	}
	if wt := windowTargetForNode(c, "n1"); wt == nil || PinString(wt, "Title") != "A" {
		t.Fatalf("n1 应回溯到 WT_A, got %v", wt)
	}
	if wt := windowTargetForNode(c, ""); wt == nil || PinString(wt, "Title") != "A" {
		t.Fatalf("空 nodeID 应回落主窗口 WT_A, got %v", wt)
	}
}
