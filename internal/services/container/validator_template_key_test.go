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
	c := &Container{}
	// hasTemplate always returns false → expect TEMPLATE_NOT_FOUND
	errs := ValidateContainerWithDeps(c, []Subgraph{sg}, func(string) bool { return false })
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

// 回归: template 节点直接放在主图 (c.Graph.Nodes) 也要查存在性.
// 曾漏扫主图 → 主图上引用不存在 GUID 的节点校验绿灯、运行时静默 miss.
func TestValidateContainerWithDeps_TemplateNotFound_MainGraph(t *testing.T) {
	c := &Container{
		Graph: Graph{Nodes: []GraphNode{
			{ID: "main1", Kind: "WaitTemplate", Config: map[string]any{"literal": map[string]any{"Templates": []any{"missing-guid"}}}},
		}},
	}
	errs := ValidateContainerWithDeps(c, nil, func(string) bool { return false })
	found := false
	for _, e := range errs {
		if e.Code == "TEMPLATE_NOT_FOUND" && e.NodeID == "main1" {
			if len(e.GraphPath) == 0 || e.GraphPath[0] != "main" {
				t.Errorf("expected GraphPath [main], got %v", e.GraphPath)
			}
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected TEMPLATE_NOT_FOUND on main graph, got %v", errs)
	}
}

func TestValidateContainerWithDeps_TemplateFound(t *testing.T) {
	sg := Subgraph{
		ID: "sg1",
		Graph: Graph{Nodes: []GraphNode{
			{ID: "n1", Kind: "CheckTemplate", Config: map[string]any{"literal": map[string]any{"Templates": []any{"some-guid"}}}},
		}},
	}
	c := &Container{}
	errs := ValidateContainerWithDeps(c, []Subgraph{sg}, func(string) bool { return true })
	for _, e := range errs {
		if e.Code == "TEMPLATE_NOT_FOUND" {
			t.Errorf("unexpected TEMPLATE_NOT_FOUND when hasTemplate=true: %v", e)
		}
	}
}
