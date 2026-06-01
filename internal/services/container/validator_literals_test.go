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
					"cases":   []any{"a"},
					"literal": map[string]any{"Value": true}, // should be string
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

func TestLiteralTypeMatch_TemplateKeyList(t *testing.T) {
	// TemplateKey 语义 pin (WaitTemplate.Templates) 实为字符串列表, 数组 literal 不该报 mismatch.
	c := &Container{
		SchemaVersion: 4,
		Graph: Graph{
			ID: "g", Version: 1,
			Nodes: []GraphNode{
				{ID: "s", Kind: "Start"},
				{ID: "wt", Kind: "WindowTarget"},
				{ID: "w", Kind: "WaitTemplate", Config: map[string]any{
					"literal": map[string]any{"Templates": []any{"ns.icon"}},
				}},
			},
		},
	}
	errs := ValidateContainer(c)
	for _, e := range errs {
		if e.Code == CodeLiteralTypeMismatch {
			if pin, _ := e.Params["pin"].(string); pin == "Templates" {
				t.Errorf("unexpected LITERAL_TYPE_MISMATCH for TemplateKey list: %+v", e)
			}
		}
	}
}

func TestLiteralTypeMismatch_TemplateKeyNonString(t *testing.T) {
	// 列表元素非 string (数字) 仍该报 mismatch — 模板 key 必须是字符串.
	c := &Container{
		SchemaVersion: 4,
		Graph: Graph{
			ID: "g", Version: 1,
			Nodes: []GraphNode{
				{ID: "s", Kind: "Start"},
				{ID: "wt", Kind: "WindowTarget"},
				{ID: "w", Kind: "WaitTemplate", Config: map[string]any{
					"literal": map[string]any{"Templates": []any{1.1}},
				}},
			},
		},
	}
	errs := ValidateContainer(c)
	found := false
	for _, e := range errs {
		if e.Code == CodeLiteralTypeMismatch {
			if pin, _ := e.Params["pin"].(string); pin == "Templates" {
				found = true
			}
		}
	}
	if !found {
		t.Errorf("expected LITERAL_TYPE_MISMATCH for TemplateKey non-string element, got: %v", errs)
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
