package container

import (
	"testing"
)

func TestValidateVarRefs_DeclaredOK(t *testing.T) {
	c := &Container{
		Vars: []VarDecl{{Name: "x", Type: "number", Default: 0.0}},
		Graph: Graph{Nodes: []GraphNode{
			{ID: "gv", Kind: "GetVar", Config: map[string]any{"VarName": "x", "Scope": "auto"}},
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
			{ID: "gv", Kind: "GetVar", Config: map[string]any{"VarName": "ghost", "Scope": "auto"}},
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
			{ID: "gv", Kind: "GetVar", Config: map[string]any{"VarName": "tmp", "Scope": "local"}},
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
				{ID: "sv", Kind: "SetVar", Config: map[string]any{"VarName": "ghost", "Scope": "global"}},
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

// B11: subgraph 内 var ref 若在 sg.RequiredGlobals 白名单 — 视作 caller 会 provide.
func TestValidateVarRefs_SubgraphRequiredGlobalWhitelisted(t *testing.T) {
	c := &Container{
		Vars:  nil, // container 故意没声明
		Graph: Graph{Nodes: []GraphNode{}},
		Subgraphs: []Subgraph{{
			ID:              "sg1",
			RequiredGlobals: []SubgraphRequiredGlobal{{Name: "state", Type: "string"}},
			Graph: Graph{Nodes: []GraphNode{
				{ID: "gv", Kind: "GetVar", Config: map[string]any{"VarName": "state", "Scope": "auto"}},
			}},
		}},
	}
	errs := validateVarRefs(c)
	if len(errs) != 0 {
		t.Fatalf("subgraph RequiredGlobals 白名单应允许引用, got: %+v", errs)
	}
}

// B11 regression: subgraph 引用既不在 container.Vars 也不在 RequiredGlobals 的 var, 仍报错.
func TestValidateVarRefs_SubgraphUnknownVarStillErrors(t *testing.T) {
	c := &Container{
		Vars:  nil,
		Graph: Graph{Nodes: []GraphNode{}},
		Subgraphs: []Subgraph{{
			ID:              "sg1",
			RequiredGlobals: []SubgraphRequiredGlobal{{Name: "state"}}, // 只声明 state
			Graph: Graph{Nodes: []GraphNode{
				{ID: "gv", Kind: "GetVar", Config: map[string]any{"VarName": "unknown_var", "Scope": "auto"}},
			}},
		}},
	}
	errs := validateVarRefs(c)
	if len(errs) != 1 || errs[0].Code != CodeInvalidVarRef {
		t.Fatalf("未在 Vars 也未在 RequiredGlobals 应仍报错, got: %+v", errs)
	}
}
