package container

import "testing"

// 绑定项校验: 未声明变量 → BOUND_VAR_UNKNOWN (error); 类型不合 → BOUND_VAR_TYPE_MISMATCH
// (warning); 类型 any / 同型 / 连线项不报。
func TestValidateBoundInputs(t *testing.T) {
	c := &Container{
		Vars: []VarDecl{
			{Name: "hp", Type: "number"},
			{Name: "msg", Type: "string"},
		},
		Graph: Graph{Nodes: []GraphNode{
			{ID: "e1", Kind: "Expr", Config: map[string]any{
				"Inputs": []any{
					map[string]any{"Name": "a", "Type": "number", "Var": "hp"},      // OK
					map[string]any{"Name": "b", "Type": "number", "Var": "missing"}, // unknown
					map[string]any{"Name": "c", "Type": "number", "Var": "msg"},     // type mismatch
					map[string]any{"Name": "d", "Type": "any", "Var": "msg"},        // any 不报
					map[string]any{"Name": "w", "Type": "number"},                   // 连线项不碰
				},
			}},
		}},
	}
	errs := validateBoundInputs(c)
	var unknown, mismatch int
	for _, e := range errs {
		switch e.Code {
		case CodeBoundVarUnknown:
			unknown++
			if e.Params["var"] != "missing" {
				t.Fatalf("unknown 报错对象不对: %+v", e)
			}
		case CodeBoundVarTypeMismatch:
			mismatch++
			if e.Params["var"] != "msg" {
				t.Fatalf("mismatch 报错对象不对: %+v", e)
			}
		}
	}
	if unknown != 1 || mismatch != 1 || len(errs) != 2 {
		t.Fatalf("want 1 unknown + 1 mismatch, got %+v", errs)
	}
}
