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

	result, err := CopySubgraph(lib, target, existingKeys, func(b []byte) string { return "newidxxxxxxxx" })
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

	result, err := CopySubgraph(lib, target, existingKeys, func(b []byte) string { return "newidxxxxxxxx" })
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

// B-3 regression: 多 OutputPins 的库子图被 copy 时，内部 SubgraphOutput 节点
// config.declID 必须随 OutputPins 的 ID 重发同步改写；否则父图调用边引用旧 ID 找不到下游。
func TestCopySubgraph_RewritesSubgraphOutputDeclID(t *testing.T) {
	lib := &LibrarySubgraph{
		Subgraph: container.Subgraph{
			ID:    "lib-sg",
			Label: "上钩等待",
			Graph: container.Graph{
				ID: "g-libsg", Version: container.GraphSchemaVersion,
				Nodes: []container.GraphNode{
					{ID: "in-node", Kind: "SubgraphInput", CreatedAt: time.Now().UTC()},
					{ID: "out-found", Kind: "SubgraphOutput",
						Config:    map[string]any{"declID": "decl-found-OLD"},
						CreatedAt: time.Now().UTC()},
					{ID: "out-timeout", Kind: "SubgraphOutput",
						Config:    map[string]any{"declID": "decl-timeout-OLD"},
						CreatedAt: time.Now().UTC()},
				},
				Edges: []container.GraphEdge{
					{From: "in-node.out", To: "out-found.in"},
					{From: "in-node.out", To: "out-timeout.in"},
				},
			},
			OutputPins: []container.SubgraphOutputDecl{
				{ID: "decl-found-OLD", Name: "found"},
				{ID: "decl-timeout-OLD", Name: "timeout"},
			},
		},
	}
	target := &container.Container{ID: "tgt"}
	existingKeys := map[string]struct{}{}

	// 用 counter-based 生成器拿可断言的 id（需 ≥ 8 字符，copy.go 会 [:8] 截）
	var counter int
	idGen := func(b []byte) string {
		counter++
		return fmt.Sprintf("newid-%08d", counter)
	}
	result, err := CopySubgraph(lib, target, existingKeys, idGen)
	if err != nil {
		t.Fatalf("copy: %v", err)
	}

	// 验证 OutputPins ID 被重发（不再是 OLD）
	for _, p := range result.NewSubgraph.OutputPins {
		if p.ID == "decl-found-OLD" || p.ID == "decl-timeout-OLD" {
			t.Errorf("OutputPin ID not rewritten: %+v", p)
		}
	}

	// 关键断言：内部 SubgraphOutput 节点的 config.declID 必须跟新 OutputPins.ID 对齐
	newDeclByName := map[string]string{}
	for _, p := range result.NewSubgraph.OutputPins {
		newDeclByName[p.Name] = p.ID
	}
	for _, n := range result.NewSubgraph.Graph.Nodes {
		if n.Kind != "SubgraphOutput" {
			continue
		}
		gotDecl, _ := n.Config["declID"].(string)
		if gotDecl == "decl-found-OLD" || gotDecl == "decl-timeout-OLD" {
			t.Errorf("SubgraphOutput node %s 仍引用旧 declID %q（应改成新 OutputPins ID）", n.ID, gotDecl)
		}
		// 应该是某个新 OutputPins ID
		matched := false
		for _, newID := range newDeclByName {
			if gotDecl == newID {
				matched = true
				break
			}
		}
		if !matched {
			t.Errorf("SubgraphOutput node %s declID=%q 不在新 OutputPins 列表里", n.ID, gotDecl)
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

	result, err := CopySubgraph(lib, target, existingKeys, func(b []byte) string { return "newidxxxxxxxx" })
	if err != nil {
		t.Fatal(err)
	}
	if result.TemplateKeyMap["k"] != "k_3" {
		t.Errorf("expected k → k_3, got %+v", result.TemplateKeyMap)
	}
}
