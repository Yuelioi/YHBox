package container

import (
	"testing"
)

func TestLiteralTypeMismatch_StringForNumberPin(t *testing.T) {
	c := &Container{
		SchemaVersion: 4,
		Graph: Graph{
			ID: "g", Version: 1,
			Nodes: []GraphNode{
				{ID: "s", Kind: "Start"},
				{ID: "wt", Kind: "WindowTarget"},
				{ID: "sl", Kind: "Sleep", Config: map[string]any{
					"literal": map[string]any{"Duration": "abc"}, // should be number/duration
				}},
			},
		},
	}
	errs := ValidateContainer(c)
	found := false
	for _, e := range errs {
		if e.Code == CodeLiteralTypeMismatch {
			if pin, _ := e.Params["pin"].(string); pin == "Duration" {
				found = true
			}
		}
	}
	if !found {
		t.Errorf("expected LITERAL_TYPE_MISMATCH for Sleep.Duration = \"abc\", got: %v", errs)
	}
}

func TestLiteralTypeMismatch_BoolForStringPin(t *testing.T) {
	c := &Container{
		SchemaVersion: 4,
		Graph: Graph{
			ID: "g", Version: 1,
			Nodes: []GraphNode{
				{ID: "s", Kind: "Start"},
				{ID: "wt", Kind: "WindowTarget"},
				{ID: "sw", Kind: "Switch", Config: map[string]any{
					"Case1Value": "a",
					"literal":    map[string]any{"Value": true}, // should be string
				}},
			},
		},
	}
	errs := ValidateContainer(c)
	found := false
	for _, e := range errs {
		if e.Code == CodeLiteralTypeMismatch {
			if pin, _ := e.Params["pin"].(string); pin == "Value" {
				found = true
			}
		}
	}
	if !found {
		t.Errorf("expected LITERAL_TYPE_MISMATCH for Switch.Value = true, got: %v", errs)
	}
}

func TestLiteralTypeMatch_CorrectTypes(t *testing.T) {
	// Sanity: correct types don't trigger the error.
	c := &Container{
		SchemaVersion: 4,
		Graph: Graph{
			ID: "g", Version: 1,
			Nodes: []GraphNode{
				{ID: "s", Kind: "Start"},
				{ID: "wt", Kind: "WindowTarget"},
				{ID: "sl", Kind: "Sleep", Config: map[string]any{
					"literal": map[string]any{"Duration": float64(500)},
				}},
				{ID: "if", Kind: "If", Config: map[string]any{
					"literal": map[string]any{"Condition": true},
				}},
			},
		},
	}
	errs := ValidateContainer(c)
	for _, e := range errs {
		if e.Code == CodeLiteralTypeMismatch {
			t.Errorf("unexpected LITERAL_TYPE_MISMATCH on valid literals: %+v", e)
		}
	}
}
