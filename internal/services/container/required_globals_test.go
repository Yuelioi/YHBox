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
	containerVars := []VarDecl{
		{Name: "state", Type: "string", Default: "IDLE"},
		{Name: "controlDir", Type: "string"},
		// hookStreak 故意不 declare → 出在 result 里但 Type=""
	}

	got := computeRequiredGlobals(sg, containerVars)
	if len(got) != 3 {
		t.Fatalf("expected 3 unique required globals, got %d: %+v", len(got), got)
	}
	want := []SubgraphRequiredGlobal{
		{Name: "state", Type: "string", Default: "IDLE"},
		{Name: "controlDir", Type: "string"},
		{Name: "hookStreak"}, // Type="" Default=nil
	}
	for i, w := range want {
		if got[i].Name != w.Name {
			t.Errorf("[%d] name: got %q want %q", i, got[i].Name, w.Name)
		}
		if got[i].Type != w.Type {
			t.Errorf("[%d] type: got %q want %q", i, got[i].Type, w.Type)
		}
		if got[i].Default != w.Default {
			t.Errorf("[%d] default: got %v want %v", i, got[i].Default, w.Default)
		}
	}
}

func TestComputeRequiredGlobals_Idempotent(t *testing.T) {
	sg := &Subgraph{Graph: Graph{Nodes: []GraphNode{
		{ID: "1", Kind: "GetVar", Config: map[string]any{"VarName": "x", "Scope": "auto"}},
	}}}
	a := computeRequiredGlobals(sg, nil)
	b := computeRequiredGlobals(sg, nil)
	if len(a) != len(b) || a[0].Name != b[0].Name {
		t.Errorf("expected idempotent result, got a=%+v b=%+v", a, b)
	}
}

func TestComputeRequiredGlobals_NilSafe(t *testing.T) {
	if got := computeRequiredGlobals(nil, nil); got != nil {
		t.Errorf("nil sg should yield nil, got %+v", got)
	}
}

// Integration: Container.Normalize 应自动填充 sg.RequiredGlobals.
func TestContainerNormalize_FillsRequiredGlobals(t *testing.T) {
	c := &Container{
		Vars: []VarDecl{{Name: "state", Type: "string", Default: "IDLE"}},
		Subgraphs: []Subgraph{
			{
				ID:    "sg1",
				Label: "test",
				Graph: Graph{Nodes: []GraphNode{
					{ID: "n1", Kind: "GetVar", Config: map[string]any{"VarName": "state"}},
				}, Edges: []GraphEdge{}},
				OutputPins: []SubgraphOutputDecl{{ID: "p1", Name: "done"}},
			},
		},
	}
	c.Normalize()
	if len(c.Subgraphs[0].RequiredGlobals) != 1 {
		t.Fatalf("expected 1 required global after Normalize, got %d", len(c.Subgraphs[0].RequiredGlobals))
	}
	if c.Subgraphs[0].RequiredGlobals[0].Name != "state" {
		t.Errorf("unexpected name: %s", c.Subgraphs[0].RequiredGlobals[0].Name)
	}
	if c.Subgraphs[0].RequiredGlobals[0].Type != "string" {
		t.Errorf("unexpected type: %s (expected merged from container.Vars)", c.Subgraphs[0].RequiredGlobals[0].Type)
	}
}
