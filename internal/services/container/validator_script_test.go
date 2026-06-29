package container

import "testing"

// 一个 Script 节点同时含语法错 Code + 重名 Inputs — 两个码都要报。
func TestValidateScriptNodes(t *testing.T) {
	c := &Container{Graph: Graph{Nodes: []GraphNode{
		{ID: "s1", Kind: "Script", Config: map[string]any{
			"literal": map[string]any{"Code": "let a = ;"},
			"Inputs":  []any{map[string]any{"Name": "x", "Type": "number"}, map[string]any{"Name": "x", "Type": "number"}},
		}},
	}}}
	errs := validateScriptNodes(c, nil)
	want := map[string]bool{"SCRIPT_PARSE_ERROR": false, "SCRIPT_DUPLICATE_INPUT": false}
	for _, e := range errs {
		want[e.Code] = true
	}
	for code, hit := range want {
		if !hit {
			t.Fatalf("missing %s, got %+v", code, errs)
		}
	}
}
