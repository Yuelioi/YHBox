package container

import (
	"strings"
	"testing"
)

func TestValidate_OK(t *testing.T) {
	c := &Container{
		SchemaVersion: 1, ID: "x", Name: "ok",
		Graph: Graph{Nodes: []GraphNode{
			{ID: "n1", Kind: "Start"},
			{ID: "w", Kind: "WindowTarget", Config: map[string]any{
				"match": map[string]any{"title": "异环"},
			}},
		}},
	}
	if err := c.Validate(); err != nil {
		t.Errorf("expected valid, got %v", err)
	}
}

func TestValidate_BadHotkey(t *testing.T) {
	c := &Container{SchemaVersion: 1, ID: "x", Name: "ok", Hotkey: "  "}
	if err := c.Validate(); err != nil {
		t.Errorf("空白 hotkey 应允许，got %v", err)
	}
}

func TestValidate_StartCount(t *testing.T) {
	twoStarts := &Container{
		SchemaVersion: 1, Name: "x",
		Graph: Graph{Nodes: []GraphNode{{ID: "s1", Kind: "Start"}, {ID: "s2", Kind: "Start"}}},
	}
	if err := twoStarts.Validate(); err == nil || !strings.Contains(err.Error(), "Start") {
		t.Errorf("two starts must error, got %v", err)
	}
	zeroStart := &Container{
		SchemaVersion: 1, Name: "x",
		Graph: Graph{Nodes: []GraphNode{{ID: "s1", Kind: "Sleep"}}},
	}
	if err := zeroStart.Validate(); err == nil || !strings.Contains(err.Error(), "Start") {
		t.Errorf("zero start must error, got %v", err)
	}
}

func TestValidate_EmptyGraphAllowed(t *testing.T) {
	c := &Container{SchemaVersion: 1, Name: "x", Graph: Graph{}}
	if err := c.Validate(); err != nil {
		t.Errorf("empty graph should be valid (just created), got %v", err)
	}
}
