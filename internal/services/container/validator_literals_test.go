package container

import (
	"strings"
	"testing"

	_ "yhbox/internal/services/container/nodekind/specs"
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
					"literal": map[string]any{"durationMs": "abc"}, // should be number
				}},
			},
		},
	}
	errs := ValidateContainer(c)
	found := false
	for _, e := range errs {
		if e.Code == CodeLiteralTypeMismatch && strings.Contains(e.Message, "durationMs") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected LITERAL_TYPE_MISMATCH for Sleep.durationMs = \"abc\", got: %v", errs)
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
					"cases":   []any{"a"},
					"literal": map[string]any{"value": true}, // should be string
				}},
			},
		},
	}
	errs := ValidateContainer(c)
	found := false
	for _, e := range errs {
		if e.Code == CodeLiteralTypeMismatch && strings.Contains(e.Message, "value") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected LITERAL_TYPE_MISMATCH for Switch.value = true, got: %v", errs)
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
					"literal": map[string]any{"durationMs": float64(500)},
				}},
				{ID: "if", Kind: "If", Config: map[string]any{
					"literal": map[string]any{"condition": true},
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
