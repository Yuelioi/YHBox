package container

import "testing"

func TestValidateTemplateKey_InvalidKey(t *testing.T) {
	sg := Subgraph{
		ID: "sg1",
		Graph: Graph{Nodes: []GraphNode{
			{ID: "n1", Kind: "CheckTemplate", Config: map[string]any{"template": "no_dot"}},
		}},
	}
	c := &Container{Subgraphs: []Subgraph{sg}}
	errs := ValidateContainer(c)
	found := false
	for _, e := range errs {
		if e.Code == "INVALID_TEMPLATE_KEY" && e.NodeID == "n1" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected INVALID_TEMPLATE_KEY error, got %v", errs)
	}
}

func TestValidateTemplateKey_Valid(t *testing.T) {
	sg := Subgraph{
		ID: "sg1",
		Graph: Graph{Nodes: []GraphNode{
			{ID: "n1", Kind: "CheckTemplate", Config: map[string]any{"template": "fishing.hook_icon"}},
		}},
	}
	c := &Container{Subgraphs: []Subgraph{sg}}
	errs := ValidateContainer(c)
	for _, e := range errs {
		if e.Code == "INVALID_TEMPLATE_KEY" {
			t.Errorf("unexpected INVALID_TEMPLATE_KEY for valid key: %v", e)
		}
	}
}

func TestValidateContainerWithDeps_TemplateNotFound(t *testing.T) {
	sg := Subgraph{
		ID: "sg1",
		Graph: Graph{Nodes: []GraphNode{
			{ID: "n1", Kind: "CheckTemplate", Config: map[string]any{"template": "fishing.hook_icon"}},
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
			{ID: "n1", Kind: "CheckTemplate", Config: map[string]any{"template": "fishing.hook_icon"}},
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
