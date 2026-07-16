package container

import "testing"

// AI 的可绑字段包括 Done.Text。

func TestValidateCaptureRefs_DeclaredOK(t *testing.T) {
	c := &Container{
		Vars: []VarDecl{{Name: "n", Type: "string"}},
		Graph: Graph{Nodes: []GraphNode{
			{ID: "d1", Kind: "AI", Config: map[string]any{
				"capture": map[string]any{"Text": "n"},
			}},
		}},
	}
	if errs := validateCaptureRefs(c, nil); len(errs) != 0 {
		t.Fatalf("declared var + valid field should not error, got: %+v", errs)
	}
}

func TestValidateCaptureRefs_UndeclaredVar(t *testing.T) {
	c := &Container{
		Graph: Graph{Nodes: []GraphNode{
			{ID: "d1", Kind: "AI", Config: map[string]any{
				"capture": map[string]any{"Text": "ghost"},
			}},
		}},
	}
	errs := validateCaptureRefs(c, nil)
	if len(errs) != 1 || errs[0].Code != CodeInvalidVarRef {
		t.Fatalf("undeclared capture var should fire INVALID_VAR_REF, got: %+v", errs)
	}
}

func TestValidateCaptureRefs_InvalidField(t *testing.T) {
	c := &Container{
		Vars: []VarDecl{{Name: "n", Type: "string"}},
		Graph: Graph{Nodes: []GraphNode{
			{ID: "d1", Kind: "AI", Config: map[string]any{
				"capture": map[string]any{"Nonexistent": "n"},
			}},
		}},
	}
	errs := validateCaptureRefs(c, nil)
	if len(errs) != 1 || errs[0].Code != CodeInvalidPin {
		t.Fatalf("capture on non-bindable field should fire INVALID_PIN, got: %+v", errs)
	}
}

func TestValidateCaptureRefs_EmptyBindingSkipped(t *testing.T) {
	c := &Container{
		Graph: Graph{Nodes: []GraphNode{
			{ID: "d1", Kind: "AI", Config: map[string]any{
				"capture": map[string]any{"Text": ""}, // 空 = 未配
			}},
		}},
	}
	if errs := validateCaptureRefs(c, nil); len(errs) != 0 {
		t.Fatalf("empty binding should be skipped, got: %+v", errs)
	}
}

func TestValidateCaptureRefs_SubgraphRequiredGlobalsWhitelist(t *testing.T) {
	sgs := []Subgraph{{
		ID:              "sg1",
		RequiredGlobals: []string{"shared"},
		Graph: Graph{Nodes: []GraphNode{
			{ID: "d1", Kind: "AI", Config: map[string]any{
				"capture": map[string]any{"Text": "shared"},
			}},
		}},
	}}
	c := &Container{Graph: Graph{}}
	if errs := validateCaptureRefs(c, sgs); len(errs) != 0 {
		t.Fatalf("subgraph RequiredGlobals 白名单应允许捕获绑定, got: %+v", errs)
	}
}
