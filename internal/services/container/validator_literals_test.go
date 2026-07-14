package container

import (
	"testing"

	"github.com/yottaapp/yotta/internal/node"
)

func TestLiteralTypeMismatch_StringForNumberPin(t *testing.T) {
	c := &Container{
		SchemaVersion: 4,
		Graph: Graph{
			ID: "g", SchemaVersion: 1,
			Nodes: []GraphNode{
				{ID: "s", Kind: "Start"},
				{ID: "wt", Kind: "Win32WindowTarget"},
				{ID: "sl", Kind: "Sleep", Config: map[string]any{
					"literal": map[string]any{"Duration": "abc"}, // should be number/duration
				}},
			},
		},
	}
	errs := ValidateContainer(c, nil)
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
			ID: "g", SchemaVersion: 1,
			Nodes: []GraphNode{
				{ID: "s", Kind: "Start"},
				{ID: "wt", Kind: "Win32WindowTarget"},
				{ID: "sw", Kind: "Switch", Config: map[string]any{
					"cases":   []any{"a"},
					"literal": map[string]any{"Value": true}, // should be string
				}},
			},
		},
	}
	errs := ValidateContainer(c, nil)
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
			ID: "g", SchemaVersion: 1,
			Nodes: []GraphNode{
				{ID: "s", Kind: "Start"},
				{ID: "wt", Kind: "Win32WindowTarget"},
				{ID: "w", Kind: "WaitTemplate", Config: map[string]any{
					"literal": map[string]any{"Templates": []any{"ns.icon"}},
				}},
			},
		},
	}
	errs := ValidateContainer(c, nil)
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
			ID: "g", SchemaVersion: 1,
			Nodes: []GraphNode{
				{ID: "s", Kind: "Start"},
				{ID: "wt", Kind: "Win32WindowTarget"},
				{ID: "w", Kind: "WaitTemplate", Config: map[string]any{
					"literal": map[string]any{"Templates": []any{1.1}},
				}},
			},
		},
	}
	errs := ValidateContainer(c, nil)
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
			ID: "g", SchemaVersion: 1,
			Nodes: []GraphNode{
				{ID: "s", Kind: "Start"},
				{ID: "wt", Kind: "Win32WindowTarget"},
				{ID: "sl", Kind: "Sleep", Config: map[string]any{
					"literal": map[string]any{"Duration": float64(500)},
				}},
				{ID: "if", Kind: "If", Config: map[string]any{
					"literal": map[string]any{"Condition": true},
				}},
			},
		},
	}
	errs := ValidateContainer(c, nil)
	for _, e := range errs {
		if e.Code == CodeLiteralTypeMismatch {
			t.Errorf("unexpected LITERAL_TYPE_MISMATCH on valid literals: %+v", e)
		}
	}
}

func TestLiteralTypeMismatch_DynamicInput(t *testing.T) {
	c := reqContainer([]GraphNode{
		{ID: "expr", Kind: "Expr", Config: map[string]any{
			"Inputs":  []any{map[string]any{"Name": "count", "Type": "Number"}},
			"literal": map[string]any{"count": "not-a-number"},
		}},
	}, nil)
	errs := validateLiteralTypes(node.DefaultRegistrySnapshot(), c, nil)
	if !hasCodeForNode(errs, CodeLiteralTypeMismatch, "expr") {
		t.Fatalf("dynamic input literal type must be checked, got %+v", errs)
	}
}

func TestLiteralTypeMatch_DynamicIntegerInputUsesCanonicalNumberType(t *testing.T) {
	c := reqContainer([]GraphNode{
		{ID: "ai", Kind: "AI", Config: map[string]any{
			"Inputs":  []any{map[string]any{"Name": "count", "Type": "Integer"}},
			"literal": map[string]any{"count": float64(2)},
		}},
	}, nil)
	errs := validateLiteralTypes(node.DefaultRegistrySnapshot(), c, nil)
	if hasCodeForNode(errs, CodeLiteralTypeMismatch, "ai") {
		t.Fatalf("dynamic Integer must use the canonical number type, got %+v", errs)
	}
}
