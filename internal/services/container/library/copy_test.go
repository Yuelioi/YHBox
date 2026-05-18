package library

import (
	"fmt"
	"testing"
	"time"

	"yhbox/internal/services/container"
)

func TestCopySubgraph_FreshContainer_NoConflict(t *testing.T) {
	lib := &LibrarySubgraph{
		Subgraph: container.Subgraph{
			ID:    "lib-sg",
			Label: "上钩等待",
			Graph: container.Graph{
				ID:      "g-libsg",
				Version: container.GraphSchemaVersion,
				Nodes: []container.GraphNode{
					{ID: "old-id-1", Kind: "SubgraphInput", CreatedAt: time.Now().UTC()},
					{ID: "old-id-2", Kind: "WaitTemplate", Config: map[string]any{"template": "fish/onhook"}, CreatedAt: time.Now().UTC()},
				},
				Edges: []container.GraphEdge{{From: "old-id-1.out", To: "old-id-2.in"}},
			},
			OutputPins: []container.SubgraphOutputDecl{{ID: "d", Name: "done"}},
		},
	}
	target := &container.Container{ID: "tgt", Subgraphs: nil}
	existingKeys := map[string]struct{}{}

	result, err := CopySubgraph(lib, target, existingKeys, nil, func(b []byte) string { return "newidxxxxxxxx" })
	if err != nil {
		t.Fatalf("copy: %v", err)
	}
	for _, n := range result.NewSubgraph.Graph.Nodes {
		if n.ID == "old-id-1" || n.ID == "old-id-2" {
			t.Errorf("node id not rewritten: %s", n.ID)
		}
	}
	if len(result.TemplateKeyMap) != 0 {
		t.Errorf("expected no template rename, got %+v", result.TemplateKeyMap)
	}
}

func TestCopySubgraph_TemplateKeyConflict(t *testing.T) {
	lib := &LibrarySubgraph{
		Subgraph: container.Subgraph{
			ID:    "lib-sg",
			Label: "x",
			Graph: container.Graph{
				ID: "g", Version: container.GraphSchemaVersion,
				Nodes: []container.GraphNode{
					{ID: "n", Kind: "WaitTemplate", Config: map[string]any{"template": "fish/onhook"}, CreatedAt: time.Now().UTC()},
				},
			},
			OutputPins: []container.SubgraphOutputDecl{{ID: "d", Name: "done"}},
		},
	}
	target := &container.Container{ID: "tgt"}
	existingKeys := map[string]struct{}{"fish/onhook": {}}

	result, err := CopySubgraph(lib, target, existingKeys, nil, func(b []byte) string { return "newidxxxxxxxx" })
	if err != nil {
		t.Fatalf("copy: %v", err)
	}
	if result.TemplateKeyMap["fish/onhook"] != "fish/onhook_2" {
		t.Errorf("expected fish/onhook → fish/onhook_2, got %+v", result.TemplateKeyMap)
	}
	n := result.NewSubgraph.Graph.Nodes[0]
	if n.Config["template"] != "fish/onhook_2" {
		t.Errorf("node config.template not patched: %v", n.Config["template"])
	}
}

// 2026-05-19 行为反转: OutputPins ID 不再 copy 时重写, SubgraphOutput.declID 跟随保留.
// 理由: pin ID 只在 subgraph 内部 scope, 不跨 subgraph 冲突. 重写成 UUID 反而导致编辑器
// 默认 pin 名 "out" 跟实际 UUID 对不上 → INVALID_PIN. 保留库定义 (例如 "out") 方便用户拼图.
func TestCopySubgraph_PreservesOutputPinsAndDeclID(t *testing.T) {
	lib := &LibrarySubgraph{
		Subgraph: container.Subgraph{
			ID:    "lib-sg",
			Label: "上钩等待",
			Graph: container.Graph{
				ID: "g-libsg", Version: container.GraphSchemaVersion,
				Nodes: []container.GraphNode{
					{ID: "in-node", Kind: "SubgraphInput", CreatedAt: time.Now().UTC()},
					{ID: "out-found", Kind: "SubgraphOutput",
						Config:    map[string]any{"declID": "found"},
						CreatedAt: time.Now().UTC()},
					{ID: "out-timeout", Kind: "SubgraphOutput",
						Config:    map[string]any{"declID": "timeout"},
						CreatedAt: time.Now().UTC()},
				},
				Edges: []container.GraphEdge{
					{From: "in-node.out", To: "out-found.in"},
					{From: "in-node.out", To: "out-timeout.in"},
				},
			},
			OutputPins: []container.SubgraphOutputDecl{
				{ID: "found", Name: "found"},
				{ID: "timeout", Name: "timeout"},
			},
		},
	}
	target := &container.Container{ID: "tgt"}
	existingKeys := map[string]struct{}{}

	var counter int
	idGen := func(b []byte) string {
		counter++
		return fmt.Sprintf("newid-%08d", counter)
	}
	result, err := CopySubgraph(lib, target, existingKeys, nil, idGen)
	if err != nil {
		t.Fatalf("copy: %v", err)
	}

	// OutputPins ID 必须保留 (不被 UUID 化)
	gotIDs := map[string]bool{}
	for _, p := range result.NewSubgraph.OutputPins {
		gotIDs[p.ID] = true
	}
	if !gotIDs["found"] || !gotIDs["timeout"] {
		t.Errorf("OutputPins ID 应保留 'found' + 'timeout', got %+v", result.NewSubgraph.OutputPins)
	}

	// SubgraphOutput.declID 也应保留 (因为 outputPins.ID 没变, 内部 declID 引用天然一致)
	for _, n := range result.NewSubgraph.Graph.Nodes {
		if n.Kind != "SubgraphOutput" {
			continue
		}
		gotDecl, _ := n.Config["declID"].(string)
		if gotDecl != "found" && gotDecl != "timeout" {
			t.Errorf("SubgraphOutput %s declID=%q, 应是 'found' 或 'timeout' (保留库定义)", n.ID, gotDecl)
		}
	}
}

func TestCopySubgraph_DoubleConflict(t *testing.T) {
	lib := &LibrarySubgraph{
		Subgraph: container.Subgraph{
			ID: "x", Label: "x",
			Graph: container.Graph{
				ID: "g", Version: container.GraphSchemaVersion,
				Nodes: []container.GraphNode{
					{ID: "n", Kind: "WaitTemplate", Config: map[string]any{"template": "k"}, CreatedAt: time.Now().UTC()},
				},
			},
			OutputPins: []container.SubgraphOutputDecl{{ID: "d", Name: "done"}},
		},
	}
	target := &container.Container{ID: "tgt"}
	existingKeys := map[string]struct{}{"k": {}, "k_2": {}}

	result, err := CopySubgraph(lib, target, existingKeys, nil, func(b []byte) string { return "newidxxxxxxxx" })
	if err != nil {
		t.Fatal(err)
	}
	if result.TemplateKeyMap["k"] != "k_3" {
		t.Errorf("expected k → k_3, got %+v", result.TemplateKeyMap)
	}
}

// mockContainerAccess 实现 containerStoreAccess 接口，供 CopyToContainer 集成测试用。
type mockContainerAccess struct {
	cont container.Container
}

func (m *mockContainerAccess) Get(id string) (container.Container, bool) {
	if m.cont.ID == id {
		return m.cont, true
	}
	return container.Container{}, false
}

func (m *mockContainerAccess) SaveSubgraph(_ string, sg *container.Subgraph) error {
	// 追加到内存容器（幂等：同 ID 覆盖）
	for i, s := range m.cont.Subgraphs {
		if s.ID == sg.ID {
			m.cont.Subgraphs[i] = *sg
			return nil
		}
	}
	m.cont.Subgraphs = append(m.cont.Subgraphs, *sg)
	return nil
}

// TestCopyToContainer_RecursivelyImportsNestedSubgraphs 验证：
// 导入含 Subgraph/Try 节点引用的主图时，依赖子图自动递归导入，
// 且主图内的 config.subgraphId 改写为本地新 ID。
func TestCopyToContainer_RecursivelyImportsNestedSubgraphs(t *testing.T) {
	st, _ := newStore(t)

	// inner 子图：无依赖
	inner := &LibrarySubgraph{
		Subgraph: container.Subgraph{
			ID:    "inner",
			Label: "inner-sg",
			Graph: container.Graph{
				ID:      "g-inner",
				Version: container.GraphSchemaVersion,
				Nodes: []container.GraphNode{
					{ID: "in-node", Kind: "SubgraphInput", CreatedAt: time.Now().UTC()},
					{ID: "out-node", Kind: "SubgraphOutput",
						Config:    map[string]any{"declID": "done"},
						CreatedAt: time.Now().UTC()},
				},
				Edges: []container.GraphEdge{{From: "in-node.out", To: "out-node.in"}},
			},
			OutputPins: []container.SubgraphOutputDecl{{ID: "done", Name: "done"}},
		},
	}
	if err := st.SaveSubgraph(inner); err != nil {
		t.Fatalf("save inner: %v", err)
	}

	// outer 子图：含一个 Subgraph 节点引用 inner
	outer := &LibrarySubgraph{
		Subgraph: container.Subgraph{
			ID:    "outer",
			Label: "outer-sg",
			Graph: container.Graph{
				ID:      "g-outer",
				Version: container.GraphSchemaVersion,
				Nodes: []container.GraphNode{
					{ID: "start", Kind: "SubgraphInput", CreatedAt: time.Now().UTC()},
					{ID: "call-inner", Kind: "Subgraph",
						Config:    map[string]any{"subgraphId": "inner"},
						CreatedAt: time.Now().UTC()},
					{ID: "end", Kind: "SubgraphOutput",
						Config:    map[string]any{"declID": "done"},
						CreatedAt: time.Now().UTC()},
				},
				Edges: []container.GraphEdge{
					{From: "start.out", To: "call-inner.in"},
					{From: "call-inner.done", To: "end.in"},
				},
			},
			OutputPins: []container.SubgraphOutputDecl{{ID: "done", Name: "done"}},
		},
	}
	if err := st.SaveSubgraph(outer); err != nil {
		t.Fatalf("save outer: %v", err)
	}

	mock := &mockContainerAccess{cont: container.Container{ID: "tgt"}}
	svc := NewService(st)
	svc.SetContainerStoreAccess(mock)

	result, err := svc.CopyToContainer("outer", "tgt")
	if err != nil {
		t.Fatalf("CopyToContainer: %v", err)
	}

	// 容器应有 2 个子图：outer + inner（dep 先存入，主图后存入）
	if len(mock.cont.Subgraphs) != 2 {
		t.Fatalf("expected 2 subgraphs in container, got %d", len(mock.cont.Subgraphs))
	}

	// 找到本地 inner 副本（ID 以 sg- 开头，不是原来的 "inner"）
	var localInnerID string
	for _, sg := range mock.cont.Subgraphs {
		if sg.ID != result.ID {
			localInnerID = sg.ID
		}
	}
	if localInnerID == "inner" || localInnerID == "" {
		t.Errorf("inner dep ID should be a new sg-xxxxxxxx ID, got %q", localInnerID)
	}

	// 主图内的 Subgraph 节点 config.subgraphId 应指向本地 inner 副本 ID
	var callerNode *container.GraphNode
	for i := range result.Graph.Nodes {
		if result.Graph.Nodes[i].Kind == "Subgraph" {
			callerNode = &result.Graph.Nodes[i]
			break
		}
	}
	if callerNode == nil {
		t.Fatal("outer result has no Subgraph node")
	}
	gotSubgraphID, _ := callerNode.Config["subgraphId"].(string)
	if gotSubgraphID != localInnerID {
		t.Errorf("config.subgraphId = %q, want %q (local inner copy)", gotSubgraphID, localInnerID)
	}
}
