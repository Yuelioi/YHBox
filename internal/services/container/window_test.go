package container

import (
	"context"
	"testing"
)

func TestResolveWin32WindowTarget_NoNode(t *testing.T) {
	c := &Container{}
	_, err := ResolveWin32WindowTarget(context.Background(), c, 0, 0)
	if err != ErrNoWin32WindowTarget {
		t.Fatalf("expected ErrNoWin32WindowTarget, got %v", err)
	}
}

func TestReadWin32WindowTargetMatchSpec(t *testing.T) {
	n := &GraphNode{
		ID:   "wt1",
		Kind: "Win32WindowTarget",
		Config: map[string]any{
			"literal": map[string]any{
				"Title":       "MyGame",
				"Class":       "UnrealWindow",
				"ProcessName": "game.exe",
				"TitleMatch":  "exact",
			},
		},
	}
	spec := ReadWin32WindowTargetMatchSpec(n)
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

func TestReadWin32WindowTargetMatchSpec_NilConfig(t *testing.T) {
	n := &GraphNode{Kind: "Win32WindowTarget"}
	spec := ReadWin32WindowTargetMatchSpec(n)
	if spec.Title != "" || spec.Class != "" || spec.ProcessName != "" || spec.TitleMatch != "" {
		t.Errorf("expected empty MatchSpec for nil Config, got %+v", spec)
	}
}

func TestReadWin32WindowTargetMatchSpec_NilNode(t *testing.T) {
	spec := ReadWin32WindowTargetMatchSpec(nil)
	if spec.Title != "" || spec.Class != "" || spec.ProcessName != "" || spec.TitleMatch != "" {
		t.Errorf("expected empty MatchSpec for nil node, got %+v", spec)
	}
}

func TestReadWin32WindowTargetScaleTolerance(t *testing.T) {
	cases := []struct {
		name string
		c    *Container
		want float64
	}{
		{"未填 → 默认", &Container{}, DefaultScaleTolerance},
		{"显式 1.5", &Container{ScaleTolerance: 1.5}, 1.5},
		{"非法 <1 → 默认", &Container{ScaleTolerance: 0.3}, DefaultScaleTolerance},
		{"正好 1.0 → 1.0", &Container{ScaleTolerance: 1.0}, 1.0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ReadWin32WindowTargetScaleTolerance(c.c); got != c.want {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}

func TestReadWin32WindowTargetInputBackend(t *testing.T) {
	cases := []struct {
		name string
		c    *Container
		want string
	}{
		{"未填 → 前台默认", &Container{}, "sendinput"},
		{"显式后台", &Container{InputBackend: "postmessage"}, "postmessage"},
		{"显式前台", &Container{InputBackend: "sendinput"}, "sendinput"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ReadWin32WindowTargetInputBackend(c.c); got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}
