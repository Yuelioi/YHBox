package container

import "testing"

// DetectColor 的可绑字段 = Count, Center (Found/NotFound 出口 Data)。

func TestValidateCaptureRefs_DeclaredOK(t *testing.T) {
	c := &Container{
		Vars: []VarDecl{{Name: "n", Type: "number"}},
		Graph: Graph{Nodes: []GraphNode{
			{ID: "d1", Kind: "DetectColor", Config: map[string]any{
				"capture": map[string]any{"Count": "n"},
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
			{ID: "d1", Kind: "DetectColor", Config: map[string]any{
				"capture": map[string]any{"Count": "ghost"},
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
		Vars: []VarDecl{{Name: "n", Type: "number"}},
		Graph: Graph{Nodes: []GraphNode{
			{ID: "d1", Kind: "DetectColor", Config: map[string]any{
				"capture": map[string]any{"Nonexistent": "n"}, // 非 DetectColor 可绑字段
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
			{ID: "d1", Kind: "DetectColor", Config: map[string]any{
				"capture": map[string]any{"Count": ""}, // 空 = 未配
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
			{ID: "d1", Kind: "DetectColor", Config: map[string]any{
				"capture": map[string]any{"Count": "shared"},
			}},
		}},
	}}
	c := &Container{Graph: Graph{}}
	if errs := validateCaptureRefs(c, sgs); len(errs) != 0 {
		t.Fatalf("subgraph RequiredGlobals 白名单应允许捕获绑定, got: %+v", errs)
	}
}
