package container

import (
	"strings"
	"testing"
)

func TestValidate_OK(t *testing.T) {
	c := &Container{
		SchemaVersion: 1, ID: "x", Name: "ok",
		Graph: Graph{Nodes: []GraphNode{{ID: "n1", Kind: "Start"}}},
	}
	if err := c.Validate(); err != nil {
		t.Errorf("expected valid, got %v", err)
	}
}

func TestValidate_MissingName(t *testing.T) {
	c := &Container{SchemaVersion: 1, ID: "x"}
	err := c.Validate()
	if err == nil || !strings.Contains(err.Error(), "name") {
		t.Errorf("expected name error, got %v", err)
	}
}

func TestValidate_BadHotkey(t *testing.T) {
	c := &Container{SchemaVersion: 1, ID: "x", Name: "ok", Hotkey: "  "}
	if err := c.Validate(); err != nil {
		t.Errorf("空白 hotkey 应允许，got %v", err)
	}
}

func TestValidate_DuplicateVarName(t *testing.T) {
	c := &Container{
		SchemaVersion: 1, ID: "x", Name: "ok",
		Vars: []VarDecl{{Name: "foo", Type: "number"}, {Name: "foo", Type: "string"}},
	}
	err := c.Validate()
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Errorf("expected duplicate var error, got %v", err)
	}
}

func TestValidate_BadVarName(t *testing.T) {
	c := &Container{
		SchemaVersion: 1, ID: "x", Name: "ok",
		Vars: []VarDecl{{Name: "1invalid", Type: "number"}},
	}
	err := c.Validate()
	if err == nil || !strings.Contains(err.Error(), "name") {
		t.Errorf("expected var name error, got %v", err)
	}
}

func TestValidate_DuplicateNodeID(t *testing.T) {
	c := &Container{
		SchemaVersion: 1, ID: "x", Name: "ok",
		Graph: Graph{Nodes: []GraphNode{
			{ID: "n1", Kind: "Start"}, {ID: "n1", Kind: "Sleep"},
		}},
	}
	err := c.Validate()
	if err == nil || !strings.Contains(err.Error(), "duplicate node") {
		t.Errorf("expected duplicate node error, got %v", err)
	}
}

func TestValidate_UnknownNodeKind(t *testing.T) {
	c := &Container{
		SchemaVersion: 1, ID: "x", Name: "ok",
		Graph: Graph{Nodes: []GraphNode{{ID: "n1", Kind: "WeirdKind"}}},
	}
	err := c.Validate()
	if err == nil || !strings.Contains(err.Error(), "unknown node kind") {
		t.Errorf("expected unknown kind error, got %v", err)
	}
}

func TestValidate_EdgeFormat(t *testing.T) {
	c := &Container{
		SchemaVersion: 1, ID: "x", Name: "ok",
		Graph: Graph{
			Nodes: []GraphNode{{ID: "n1", Kind: "Start"}, {ID: "n2", Kind: "Sleep"}},
			Edges: []GraphEdge{{From: "n1", To: "n2.in"}}, // bad: no pin
		},
	}
	err := c.Validate()
	if err == nil || !strings.Contains(err.Error(), "edge") {
		t.Errorf("expected edge format error, got %v", err)
	}
}

// ---- graph-level rules ----

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

func TestValidate_PinExists(t *testing.T) {
	c := &Container{
		SchemaVersion: 1, Name: "x",
		Graph: Graph{
			Nodes: []GraphNode{{ID: "s", Kind: "Start"}, {ID: "sl", Kind: "Sleep"}},
			Edges: []GraphEdge{{From: "s.bogus", To: "sl.in"}},
		},
	}
	if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "pin") {
		t.Errorf("bad pin must error, got %v", err)
	}
}

func TestValidate_InPinDuplicateRejected(t *testing.T) {
	c := &Container{
		SchemaVersion: 1, Name: "x",
		Graph: Graph{
			Nodes: []GraphNode{
				{ID: "s", Kind: "Start"},
				{ID: "a", Kind: "SetVar", Config: map[string]any{"varName": "x", "value": "1"}},
				{ID: "b", Kind: "SetVar", Config: map[string]any{"varName": "y", "value": "2"}},
				{ID: "t", Kind: "Sleep", Config: map[string]any{"durationMs": "1"}},
			},
			Edges: []GraphEdge{
				{From: "s.out", To: "a.in"},
				{From: "a.out", To: "t.in"},
				{From: "b.out", To: "t.in"}, // 第二条入边 → 应被拒
			},
		},
		Vars: []VarDecl{{Name: "x", Type: "number"}, {Name: "y", Type: "number"}},
	}
	if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "入边") {
		t.Errorf("duplicate in edge must error, got %v", err)
	}
}

func TestValidate_LoopBodyMustYield(t *testing.T) {
	c := &Container{
		SchemaVersion: 1, Name: "x",
		Graph: Graph{
			Nodes: []GraphNode{
				{ID: "s", Kind: "Start"},
				{ID: "lp", Kind: "Loop", Config: map[string]any{"mode": "forever"}},
				{ID: "sv", Kind: "SetVar", Config: map[string]any{"varName": "i", "value": "0"}},
			},
			Edges: []GraphEdge{
				{From: "s.out", To: "lp.in"},
				{From: "lp.body", To: "sv.in"},
				{From: "sv.out", To: "lp.loopback"},
			},
		},
		Vars: []VarDecl{{Name: "i", Type: "number"}},
	}
	if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "yield") {
		t.Errorf("loop without yield must error, got %v", err)
	}
}

func TestValidate_BreakOutsideLoop(t *testing.T) {
	c := &Container{
		SchemaVersion: 1, Name: "x",
		Graph: Graph{
			Nodes: []GraphNode{
				{ID: "s", Kind: "Start"},
				{ID: "br", Kind: "Break"},
			},
			Edges: []GraphEdge{{From: "s.out", To: "br.in"}},
		},
	}
	if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "Loop") {
		t.Errorf("Break outside Loop must error, got %v", err)
	}
}

func TestValidate_BadExpression(t *testing.T) {
	c := &Container{
		SchemaVersion: 1, Name: "x",
		Graph: Graph{
			Nodes: []GraphNode{
				{ID: "s", Kind: "Start"},
				{ID: "sl", Kind: "Sleep", Config: map[string]any{"durationMs": "1 + "}},
			},
			Edges: []GraphEdge{{From: "s.out", To: "sl.in"}},
		},
	}
	if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "parse") {
		t.Errorf("bad expr must error, got %v", err)
	}
}

func TestValidate_EmptyGraphAllowed(t *testing.T) {
	c := &Container{SchemaVersion: 1, Name: "x", Graph: Graph{}}
	if err := c.Validate(); err != nil {
		t.Errorf("empty graph should be valid (just created), got %v", err)
	}
}
