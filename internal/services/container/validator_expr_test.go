package container

import "testing"

func TestValidate_Expr_DuplicateInput(t *testing.T) {
	c := &Container{
		SchemaVersion: 1,
		Graph: Graph{
			Nodes: []GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "e1", Kind: "Expr", Config: map[string]any{
					"Expression": "i + 1",
					"Inputs": []any{
						map[string]any{"Name": "i", "Type": "number"},
						map[string]any{"Name": "i", "Type": "string"}, // dup
					},
				}},
			},
		},
	}
	errs := ValidateContainer(c, nil)
	found := false
	for _, e := range errs {
		if e.Code == CodeExprDuplicateInput {
			found = true
		}
	}
	if !found {
		t.Fatalf("want EXPR_DUPLICATE_INPUT, got %+v", errs)
	}
}

func TestValidate_Expr_ParseError(t *testing.T) {
	c := &Container{
		SchemaVersion: 1,
		Graph: Graph{
			Nodes: []GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "e1", Kind: "Expr", Config: map[string]any{
					"Expression": "1 +",
					"Inputs":     []any{},
				}},
			},
		},
	}
	errs := ValidateContainer(c, nil)
	found := false
	for _, e := range errs {
		if e.Code == CodeExprParseError {
			found = true
		}
	}
	if !found {
		t.Fatalf("want EXPR_PARSE_ERROR, got %+v", errs)
	}
}

func TestValidate_Expr_UnknownInput(t *testing.T) {
	c := &Container{
		SchemaVersion: 1,
		Graph: Graph{
			Nodes: []GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "e1", Kind: "Expr", Config: map[string]any{
					"Expression": "x + missing",
					"Inputs":     []any{map[string]any{"Name": "x", "Type": "number"}},
				}},
			},
		},
	}
	errs := ValidateContainer(c, nil)
	found := false
	for _, e := range errs {
		if e.Code == CodeExprUnknownInput && e.Params["name"] == "missing" {
			found = true
		}
	}
	if !found {
		t.Fatalf("want EXPR_UNKNOWN_INPUT (name=missing), got %+v", errs)
	}
}

func TestValidate_Expr_UnknownFunction(t *testing.T) {
	c := &Container{
		SchemaVersion: 1,
		Graph: Graph{
			Nodes: []GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "e1", Kind: "Expr", Config: map[string]any{
					"Expression": "clmap(1, 2, 3)", // clamp 手滑
					"Inputs":     []any{},
				}},
			},
		},
	}
	errs := ValidateContainer(c, nil)
	found := false
	for _, e := range errs {
		if e.Code == CodeExprUnknownFunction && e.Params["name"] == "clmap" {
			found = true
		}
	}
	if !found {
		t.Fatalf("want EXPR_UNKNOWN_FUNCTION (name=clmap), got %+v", errs)
	}
}

func TestValidate_Expr_FnArity(t *testing.T) {
	c := &Container{
		SchemaVersion: 1,
		Graph: Graph{
			Nodes: []GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "e1", Kind: "Expr", Config: map[string]any{
					"Expression": "clamp(1)",
					"Inputs":     []any{},
				}},
			},
		},
	}
	errs := ValidateContainer(c, nil)
	found := false
	for _, e := range errs {
		if e.Code == CodeExprFnArity && e.Params["name"] == "clamp" && e.Params["got"] == 1 {
			found = true
		}
	}
	if !found {
		t.Fatalf("want EXPR_FN_ARITY (name=clamp got=1), got %+v", errs)
	}
}

func TestValidate_Expr_AllValid(t *testing.T) {
	c := &Container{
		SchemaVersion: 1,
		Graph: Graph{
			Nodes: []GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "e1", Kind: "Expr", Config: map[string]any{
					"Expression": `s == "FISHING" && round(hp, 2) > 0.5`,
					"Inputs": []any{
						map[string]any{"Name": "s", "Type": "string"},
						map[string]any{"Name": "hp", "Type": "number"},
					},
				}},
			},
		},
	}
	errs := ValidateContainer(c, nil)
	for _, e := range errs {
		switch e.Code {
		case CodeExprDuplicateInput, CodeExprParseError, CodeExprUnknownInput,
			CodeExprUnknownFunction, CodeExprFnArity:
			t.Errorf("valid expr triggered unexpected: %+v", e)
		}
	}
}

// $name 引用对照容器 Vars: 未声明 → EXPR_UNKNOWN_VAR; 已声明不报。
func TestValidate_Expr_UnknownVar(t *testing.T) {
	c := &Container{
		Vars: []VarDecl{{Name: "hp", Type: "number"}},
		Graph: Graph{Nodes: []GraphNode{
			{ID: "e1", Kind: "Expr", Config: map[string]any{
				"Expression": "$hp + $ghost",
				"Inputs":     []any{},
			}},
		}},
	}
	errs := validateExprNodes(c, nil)
	var hit int
	for _, e := range errs {
		if e.Code == CodeExprUnknownVar {
			hit++
			if e.Params["name"] != "ghost" {
				t.Fatalf("报错对象不对 (声明过的 hp 不该报): %+v", e)
			}
		}
	}
	if hit != 1 {
		t.Fatalf("want exactly 1 EXPR_UNKNOWN_VAR, got %+v", errs)
	}
}
