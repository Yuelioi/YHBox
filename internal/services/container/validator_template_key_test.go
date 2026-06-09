package container

import "testing"

// 资产改 GUID 后无格式校验 (合法性 = 存在性). 这里只验存在性 (hasTemplate 注入).

func TestValidateContainerWithDeps_TemplateNotFound(t *testing.T) {
	sg := Subgraph{
		ID: "sg1",
		Graph: Graph{Nodes: []GraphNode{
			{ID: "n1", Kind: "CheckTemplate", Config: map[string]any{"literal": map[string]any{"Templates": []any{"some-guid"}}}},
		}},
	}
	c := &Container{Subgraphs: []Subgraph{sg}}
	// hasTemplate always returns false → expect TEMPLATE_NOT_FOUND
	errs := ValidateContainerWithDeps(c, func(string) bool { return false }, nil)
	found := false
	for _, e := range errs {
		if e.Code == "TEMPLATE_NOT_FOUND" && e.NodeID == "n1" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected TEMPLATE_NOT_FOUND, got %v", errs)
	}
}

func TestValidateContainerWithDeps_TemplateFound(t *testing.T) {
	sg := Subgraph{
		ID: "sg1",
		Graph: Graph{Nodes: []GraphNode{
			{ID: "n1", Kind: "CheckTemplate", Config: map[string]any{"literal": map[string]any{"Templates": []any{"some-guid"}}}},
		}},
	}
	c := &Container{Subgraphs: []Subgraph{sg}}
	errs := ValidateContainerWithDeps(c, func(string) bool { return true }, nil)
	for _, e := range errs {
		if e.Code == "TEMPLATE_NOT_FOUND" {
			t.Errorf("unexpected TEMPLATE_NOT_FOUND when hasTemplate=true: %v", e)
		}
	}
}
