package container

import (
	"encoding/json"
	"testing"
	"time"
)

func TestContainer_JSONRoundTrip(t *testing.T) {
	src := Container{
		SchemaVersion: 1, ID: "uuid-1", Name: "战斗主循环",
		Description: "test", Category: "战斗", Tags: []string{"副本", "自动"},
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
	if got.Name != src.Name || got.Hotkey != src.Hotkey || got.Category != src.Category {
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
