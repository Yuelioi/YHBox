package container

import (
	"testing"
)

func TestResolveWindowTarget_NoNode(t *testing.T) {
	c := &Container{}
	_, err := ResolveWindowTarget(c, 0, 0)
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
