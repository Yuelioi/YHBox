package container

import "testing"

func TestValidate_Expr_DuplicateInput(t *testing.T) {
	c := &Container{
		SchemaVersion: 1,
		Graph: Graph{
			Nodes: []GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "e1", Kind: "Expr", Config: map[string]any{
					"expr":   "i + 1",
					"inputs": []any{
						map[string]any{"name": "i", "type": "number"},
						map[string]any{"name": "i", "type": "string"}, // dup
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
					"expr":   "1 +",
					"inputs": []any{},
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
					"expr":   "x + missing",
					"inputs": []any{map[string]any{"name": "x", "type": "number"}},
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

func TestValidate_Expr_AllValid(t *testing.T) {
	c := &Container{
		SchemaVersion: 1,
		Graph: Graph{
			Nodes: []GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "e1", Kind: "Expr", Config: map[string]any{
					"expr":   `s == "FISHING" && hp > 0.5`,
					"inputs": []any{
						map[string]any{"name": "s", "type": "string"},
						map[string]any{"name": "hp", "type": "number"},
					},
				}},
			},
		},
	}
	errs := ValidateContainer(c)
	for _, e := range errs {
		if e.Code == CodeExprDuplicateInput || e.Code == CodeExprParseError || e.Code == CodeExprUnknownInput {
			t.Errorf("valid expr triggered unexpected: %+v", e)
		}
	}
}
