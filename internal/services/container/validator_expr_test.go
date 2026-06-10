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
	errs := ValidateContainer(c)
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
					"Inputs": []any{},
				}},
			},
		},
	}
	errs := ValidateContainer(c)
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
					"Inputs": []any{map[string]any{"Name": "x", "Type": "number"}},
				}},
			},
		},
	}
	errs := ValidateContainer(c)
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
	errs := ValidateContainer(c)
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
	errs := ValidateContainer(c)
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
	errs := ValidateContainer(c)
	for _, e := range errs {
		switch e.Code {
		case CodeExprDuplicateInput, CodeExprParseError, CodeExprUnknownInput,
			CodeExprUnknownFunction, CodeExprFnArity:
			t.Errorf("valid expr triggered unexpected: %+v", e)
		}
	}
}
