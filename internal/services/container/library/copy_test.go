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
	result, err := CopySubgraph(lib, target, existingKeys, idGen)
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

	result, err := CopySubgraph(lib, target, existingKeys, func(b []byte) string { return "newidxxxxxxxx" })
	if err != nil {
		t.Fatal(err)
	}
	if result.TemplateKeyMap["k"] != "k_3" {
		t.Errorf("expected k → k_3, got %+v", result.TemplateKeyMap)
	}
}
