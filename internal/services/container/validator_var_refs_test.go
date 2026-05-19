package container

import (
	"testing"
)

func TestValidateVarRefs_DeclaredOK(t *testing.T) {
	c := &Container{
		Vars: []VarDecl{{Name: "x", Type: "number", Default: 0.0}},
		Graph: Graph{Nodes: []GraphNode{
			{ID: "gv", Kind: "GetVar", Config: map[string]any{"varName": "x", "scope": "auto"}},
		}},
	}
	errs := validateVarRefs(c)
	if len(errs) != 0 {
		t.Fatalf("declared var should not error, got: %+v", errs)
	}
}

func TestValidateVarRefs_UndeclaredAutoScope(t *testing.T) {
	c := &Container{
		Vars: nil,
		Graph: Graph{Nodes: []GraphNode{
			{ID: "gv", Kind: "GetVar", Config: map[string]any{"varName": "ghost", "scope": "auto"}},
		}},
	}
	errs := validateVarRefs(c)
	if len(errs) != 1 || errs[0].Code != CodeInvalidVarRef {
		t.Fatalf("undeclared scope=auto should fire INVALID_VAR_REF, got: %+v", errs)
	}
	if errs[0].NodeID != "gv" {
		t.Errorf("wrong NodeID")
	}
}

func TestValidateVarRefs_LocalScopeSkipped(t *testing.T) {
	c := &Container{
		Vars: nil,
		Graph: Graph{Nodes: []GraphNode{
			{ID: "gv", Kind: "GetVar", Config: map[string]any{"varName": "tmp", "scope": "local"}},
		}},
	}
	errs := validateVarRefs(c)
	if len(errs) != 0 {
		t.Fatalf("scope=local should skip declared check, got: %+v", errs)
	}
}

func TestValidateVarRefs_SubgraphAlsoChecked(t *testing.T) {
	c := &Container{
		Vars: []VarDecl{{Name: "x", Type: "number"}},
		Graph: Graph{Nodes: []GraphNode{}},
		Subgraphs: []Subgraph{{
			ID: "sg1",
			Graph: Graph{Nodes: []GraphNode{
				{ID: "sv", Kind: "SetVar", Config: map[string]any{"varName": "ghost", "scope": "global"}},
			}},
		}},
	}
	errs := validateVarRefs(c)
	if len(errs) != 1 || errs[0].Code != CodeInvalidVarRef {
		t.Fatalf("undeclared in subgraph should fire INVALID_VAR_REF, got: %+v", errs)
	}
	if errs[0].NodeID != "sv" {
		t.Errorf("wrong NodeID")
	}
}
