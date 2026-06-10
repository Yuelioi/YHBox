package container

import "testing"

// TestValidate_PinTypeMismatch_StringToNumber:
// A GetVar(string) connected to a number-typed pin (IncVar.delta) must error.
func TestValidate_PinTypeMismatch_StringToNumber(t *testing.T) {
	c := &Container{
		SchemaVersion: 1,
		ID:            "v4-pin-mismatch",
		Vars:          []VarDecl{{Name: "msg", Type: "string", Default: ""}},
		Graph: Graph{
			Nodes: []GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "gv", Kind: "GetVar", Config: map[string]any{"VarName": "msg", "Scope": "global"}},
				{ID: "inc", Kind: "IncVar", Config: map[string]any{"VarName": "msg", "Scope": "global"}},
			},
			Edges: []GraphEdge{
				{From: "start.Done", To: "inc.in"},
				{From: "gv.Value", To: "inc.Delta"}, // string → number = mismatch (data edge derived from GetVar.value out-pin)
			},
		},
	}
	errs := ValidateContainer(c)
	found := false
	for _, e := range errs {
		if e.Code == CodePinTypeMismatch {
			found = true
			if e.Params["from"] != "string" || e.Params["to"] != "number" {
				t.Errorf("params mismatch: %+v", e.Params)
			}
		}
	}
	if !found {
		t.Fatalf("expected PIN_TYPE_MISMATCH, got %+v", errs)
	}
}

// TestValidate_PinTypeCoercion_NumberToBool_Warning:
// number → bool is allowed but produces a warning.
func TestValidate_PinTypeCoercion_NumberToBool_Warning(t *testing.T) {
	// We need a node with a bool-typed data-in pin; no node ships one yet
	// (SetVar/IncVar are any/number).
	t.Skip("no bool data-in pin in Phase A (If migration is Phase C task C3)")
}

// TestValidate_AnyAcceptsEverything: SetVar.value is type "any", accepts string/number/bool source.
func TestValidate_AnyAcceptsEverything(t *testing.T) {
	c := &Container{
		SchemaVersion: 1,
		Vars: []VarDecl{
			{Name: "src", Type: "string", Default: ""},
			{Name: "dst", Type: "string", Default: ""},
		},
		Graph: Graph{
			Nodes: []GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "gv", Kind: "GetVar", Config: map[string]any{"VarName": "src", "Scope": "global"}},
				{ID: "sv", Kind: "SetVar", Config: map[string]any{"VarName": "dst", "Scope": "global"}},
			},
			Edges: []GraphEdge{
				{From: "start.Done", To: "sv.in"},
				{From: "gv.Value", To: "sv.Value"}, // string → any = OK (data edge derived)
			},
		},
	}
	errs := ValidateContainer(c)
	for _, e := range errs {
		if e.Code == CodePinTypeMismatch || e.Code == CodePinTypeCoercionWarning {
			t.Errorf("any-typed pin should accept string with no warning: %+v", e)
		}
	}
}

// TestValidate_ListVarToListPin: GetVar(list) → ListContains.List 同型直连, 无错无 warn.
func TestValidate_ListVarToListPin(t *testing.T) {
	c := &Container{
		SchemaVersion: 1,
		Vars:          []VarDecl{{Name: "items", Type: "list", Default: []any{1.0, 2.0}}},
		Graph: Graph{
			Nodes: []GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "gv", Kind: "GetVar", Config: map[string]any{"VarName": "items", "Scope": "global"}},
				{ID: "lc", Kind: "ListContains", Config: map[string]any{"Value": "1"}},
			},
			Edges: []GraphEdge{
				{From: "gv.Value", To: "lc.List"}, // list → list = OK
			},
		},
	}
	errs := ValidateContainer(c)
	for _, e := range errs {
		if e.Code == CodePinTypeMismatch || e.Code == CodePinTypeCoercionWarning {
			t.Errorf("list var → List pin should connect cleanly: %+v", e)
		}
	}
}

// TestValidate_ListVarToNumberPin_Mismatch: GetVar(list) → IncVar.Delta (number) must error.
func TestValidate_ListVarToNumberPin_Mismatch(t *testing.T) {
	c := &Container{
		SchemaVersion: 1,
		Vars: []VarDecl{
			{Name: "items", Type: "list", Default: []any{}},
			{Name: "n", Type: "number", Default: 0.0},
		},
		Graph: Graph{
			Nodes: []GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "gv", Kind: "GetVar", Config: map[string]any{"VarName": "items", "Scope": "global"}},
				{ID: "inc", Kind: "IncVar", Config: map[string]any{"VarName": "n", "Scope": "global"}},
			},
			Edges: []GraphEdge{
				{From: "start.Done", To: "inc.in"},
				{From: "gv.Value", To: "inc.Delta"}, // list → number = mismatch
			},
		},
	}
	errs := ValidateContainer(c)
	found := false
	for _, e := range errs {
		if e.Code == CodePinTypeMismatch {
			found = true
			if e.Params["from"] != "list" || e.Params["to"] != "number" {
				t.Errorf("params mismatch: %+v", e.Params)
			}
		}
	}
	if !found {
		t.Fatalf("expected PIN_TYPE_MISMATCH, got %+v", errs)
	}
}

// TestValidate_ExecEdgeNotPinTypeChecked: Kind="" (exec) edges are ignored by validateDataPinTypes.
func TestValidate_ExecEdgeNotPinTypeChecked(t *testing.T) {
	c := &Container{
		SchemaVersion: 1,
		Vars:          []VarDecl{{Name: "x", Type: "string", Default: ""}},
		Graph: Graph{
			Nodes: []GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "inc", Kind: "IncVar", Config: map[string]any{"VarName": "x", "Scope": "global"}},
			},
			Edges: []GraphEdge{
				{From: "start.Done", To: "inc.in"}, // exec edge — Kind unset
			},
		},
	}
	errs := ValidateContainer(c)
	for _, e := range errs {
		if e.Code == CodePinTypeMismatch {
			t.Errorf("exec edge should not trigger pin-type check: %+v", e)
		}
	}
}
