package container

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

func TestContainer_JSONRoundTrip(t *testing.T) {
	src := Container{
		SchemaVersion: 1, ID: "uuid-1", Name: "战斗主循环",
		Description: "test", Tags: []string{"副本", "自动"},
		Hotkey: "Ctrl+Shift+1",
		Vars:   []VarDecl{{Name: "enemyHp", Type: "number", Default: float64(100)}},
		Graph: Graph{
			Nodes: []GraphNode{{ID: "n1", Kind: "Start", X: 100, Y: 100, Config: map[string]any{}}},
			Edges: []GraphEdge{{From: "n1.out", To: "n2.in"}},
		},
		CreatedAt: time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC),
	}
	b, err := json.Marshal(src)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got Container
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Name != src.Name || got.Hotkey != src.Hotkey {
		t.Errorf("string fields mismatch")
	}
	if len(got.Tags) != 2 || got.Tags[0] != "副本" {
		t.Errorf("tags mismatch: %v", got.Tags)
	}
	if len(got.Vars) != 1 || got.Vars[0].Name != "enemyHp" {
		t.Errorf("vars mismatch: %v", got.Vars)
	}
	if len(got.Graph.Nodes) != 1 || got.Graph.Nodes[0].Kind != "Start" {
		t.Errorf("graph nodes mismatch")
	}
	if len(got.Graph.Edges) != 1 || got.Graph.Edges[0].From != "n1.out" {
		t.Errorf("graph edges mismatch")
	}
}

// 验证 v2 schema：Graph 必须有 ID + Version；Container 不再有 Category；GraphNode 有 CreatedAt
func TestContainerSchemaV2Fields(t *testing.T) {
	c := Container{
		SchemaVersion: CurrentSchemaVersion,
		ID:            "x",
		Name:          "test",
		Graph: Graph{
			ID:      "g-main",
			Version: GraphSchemaVersion,
			Nodes: []GraphNode{
				{ID: "n1", Kind: "Start", CreatedAt: time.Now().UTC()},
			},
			Edges: nil,
		},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	b, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if bytes.Contains(b, []byte(`"category"`)) {
		t.Errorf("expected no 'category' field in JSON, got: %s", string(b))
	}
	if !bytes.Contains(b, []byte(`"id":"g-main"`)) {
		t.Errorf("expected Graph.ID in JSON, got: %s", string(b))
	}
	if !bytes.Contains(b, []byte(`"version":1`)) {
		t.Errorf("expected Graph.Version=1 in JSON, got: %s", string(b))
	}
	if !bytes.Contains(b, []byte(`"createdAt"`)) {
		t.Errorf("expected node.createdAt in JSON, got: %s", string(b))
	}

	var c2 Container
	if err := json.Unmarshal(b, &c2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if c2.Graph.ID != "g-main" {
		t.Errorf("Graph.ID lost: %q", c2.Graph.ID)
	}
	if c2.Graph.Version != GraphSchemaVersion {
		t.Errorf("Graph.Version lost: %d", c2.Graph.Version)
	}
	if c2.Graph.Nodes[0].CreatedAt.IsZero() {
		t.Errorf("node.CreatedAt lost")
	}
}
