package container

import (
	"context"
	"testing"
)

func TestResolveWindowTarget_NoNode(t *testing.T) {
	c := &Container{}
	_, err := ResolveWindowTarget(context.Background(), c, 0, 0)
	if err != ErrNoWindowTarget {
		t.Fatalf("expected ErrNoWindowTarget, got %v", err)
	}
}

func TestReadWindowTargetMatchSpec(t *testing.T) {
	n := &GraphNode{
		ID:   "wt1",
		Kind: "WindowTarget",
		Config: map[string]any{
			"literal": map[string]any{
				"Title":       "MyGame",
				"Class":       "UnrealWindow",
				"ProcessName": "game.exe",
				"TitleMatch":  "exact",
			},
		},
	}
	spec := ReadWindowTargetMatchSpec(n)
	if spec.Title != "MyGame" {
		t.Errorf("Title: want %q, got %q", "MyGame", spec.Title)
	}
	if spec.Class != "UnrealWindow" {
		t.Errorf("Class: want %q, got %q", "UnrealWindow", spec.Class)
	}
	if spec.ProcessName != "game.exe" {
		t.Errorf("ProcessName: want %q, got %q", "game.exe", spec.ProcessName)
	}
	if spec.TitleMatch != "exact" {
		t.Errorf("TitleMatch: want %q, got %q", "exact", spec.TitleMatch)
	}
}

func TestReadWindowTargetMatchSpec_NilConfig(t *testing.T) {
	n := &GraphNode{Kind: "WindowTarget"}
	spec := ReadWindowTargetMatchSpec(n)
	if spec.Title != "" || spec.Class != "" || spec.ProcessName != "" || spec.TitleMatch != "" {
		t.Errorf("expected empty MatchSpec for nil Config, got %+v", spec)
	}
}

func TestReadWindowTargetMatchSpec_NilNode(t *testing.T) {
	spec := ReadWindowTargetMatchSpec(nil)
	if spec.Title != "" || spec.Class != "" || spec.ProcessName != "" || spec.TitleMatch != "" {
		t.Errorf("expected empty MatchSpec for nil node, got %+v", spec)
	}
}

func TestReadWindowTargetScaleTolerance(t *testing.T) {
	mk := func(cfg map[string]any) *Container {
		nodes := []GraphNode{{ID: "wt", Kind: "WindowTarget", Config: cfg}}
		return &Container{Graph: Graph{Nodes: nodes}}
	}
	cases := []struct {
		name string
		c    *Container
		want float64
	}{
		{"无 WindowTarget 节点", &Container{Graph: Graph{Nodes: []GraphNode{{ID: "s", Kind: "Start"}}}}, DefaultScaleTolerance},
		{"无 pin → 默认", mk(map[string]any{}), DefaultScaleTolerance},
		{"显式 1.5", mk(map[string]any{"ScaleTolerance": 1.5}), 1.5},
		{"非法 <1 → 默认", mk(map[string]any{"ScaleTolerance": 0.3}), DefaultScaleTolerance},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ReadWindowTargetScaleTolerance(c.c); got != c.want {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}
