package container

import (
	"testing"
)

func TestComputeRequiredGlobals_BasicDerive(t *testing.T) {
	sg := &Subgraph{
		Graph: Graph{Nodes: []GraphNode{
			{ID: "1", Kind: "GetVar", Config: map[string]any{"VarName": "state", "Scope": "auto"}},
			{ID: "2", Kind: "SetVar", Config: map[string]any{"VarName": "controlDir", "Scope": "global"}},
			{ID: "3", Kind: "IncVar", Config: map[string]any{"VarName": "hookStreak"}}, // 缺 Scope → 默认 auto
			{ID: "4", Kind: "GetVar", Config: map[string]any{"VarName": "tmp", "Scope": "local"}}, // 跳
			{ID: "5", Kind: "GetVar", Config: map[string]any{"VarName": "state"}}, // dedup
			{ID: "6", Kind: "Start"}, // 非 var kind 跳
		}},
	}

	got := computeRequiredGlobals(sg)
	want := []string{"state", "controlDir", "hookStreak"}
	if len(got) != len(want) {
		t.Fatalf("expected %d unique required globals, got %d: %+v", len(want), len(got), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("[%d] name: got %q want %q", i, got[i], w)
		}
	}
}

func TestComputeRequiredGlobals_Idempotent(t *testing.T) {
	sg := &Subgraph{Graph: Graph{Nodes: []GraphNode{
		{ID: "1", Kind: "GetVar", Config: map[string]any{"VarName": "x", "Scope": "auto"}},
	}}}
	a := computeRequiredGlobals(sg)
	b := computeRequiredGlobals(sg)
	if len(a) != len(b) || a[0] != b[0] {
		t.Errorf("expected idempotent result, got a=%+v b=%+v", a, b)
	}
}

func TestComputeRequiredGlobals_NilSafe(t *testing.T) {
	if got := computeRequiredGlobals(nil); got != nil {
		t.Errorf("nil sg should yield nil, got %+v", got)
	}
}
