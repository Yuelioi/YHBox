package container

import "testing"

// config.literal 写了 KeyPress 不存在的 pin "Message" → UNKNOWN_LITERAL_PIN warning。
func TestUnknownLiteral_BogusPinWarns(t *testing.T) {
	c := reqContainer([]GraphNode{
		{ID: "k", Kind: "KeyPress", Config: map[string]any{"literal": map[string]any{"Message": "hi"}}},
	}, nil)
	errs := validateUnknownLiteralPins(c, nil)
	if !hasCodeForNode(errs, CodeUnknownLiteralPin, "k") {
		t.Fatalf("expected UNKNOWN_LITERAL_PIN for bogus pin, got %+v", errs)
	}
	for _, e := range errs {
		if e.Code == CodeUnknownLiteralPin && e.Severity != SeverityWarning {
			t.Fatalf("UNKNOWN_LITERAL_PIN must be warning, got %s", e.Severity)
		}
	}
}

// config.literal 全是合法 pin → 不报。
func TestUnknownLiteral_ValidPinNoWarn(t *testing.T) {
	c := reqContainer([]GraphNode{
		{ID: "k", Kind: "KeyPress", Config: map[string]any{"literal": map[string]any{"VK": "F"}}},
	}, nil)
	if errs := validateUnknownLiteralPins(c, nil); hasCodeForNode(errs, CodeUnknownLiteralPin, "k") {
		t.Fatalf("valid literal pin must not warn, got %+v", errs)
	}
}

func TestUnknownLiteral_DynamicDeclaredPinNoWarn(t *testing.T) {
	c := reqContainer([]GraphNode{
		{ID: "e", Kind: "Expr", Config: map[string]any{
			"Inputs":  []any{map[string]any{"Name": "foo", "Type": "Number"}},
			"literal": map[string]any{"foo": 1},
		}},
	}, nil)
	if errs := validateUnknownLiteralPins(c, nil); hasCodeForNode(errs, CodeUnknownLiteralPin, "e") {
		t.Fatalf("declared dynamic input must be accepted, got %+v", errs)
	}
}

func TestUnknownLiteral_DynamicUndeclaredPinWarns(t *testing.T) {
	c := reqContainer([]GraphNode{
		{ID: "e", Kind: "Expr", Config: map[string]any{
			"Inputs":  []any{map[string]any{"Name": "foo", "Type": "Number"}},
			"literal": map[string]any{"foo": 1, "fooo": 2},
		}},
	}, nil)
	if errs := validateUnknownLiteralPins(c, nil); !hasCodeForNode(errs, CodeUnknownLiteralPin, "e") {
		t.Fatalf("undeclared dynamic input must warn, got %+v", errs)
	}
}

func TestUnknownLiteral_PortableMetadataAllowedButBogusPinWarns(t *testing.T) {
	for _, kind := range []string{"Win32WindowTarget", "AndroidTarget"} {
		t.Run(kind, func(t *testing.T) {
			c := reqContainer([]GraphNode{
				{ID: "target", Kind: kind, Config: map[string]any{
					"literal": map[string]any{"Target": "game", "Bogus": true},
				}},
			}, nil)
			errs := validateUnknownLiteralPins(c, nil)
			if !hasUnknownLiteralPin(errs, "target", "Bogus") {
				t.Fatalf("bogus pin must still warn, got %+v", errs)
			}
			if hasUnknownLiteralPin(errs, "target", "Target") {
				t.Fatalf("portable binding metadata must not warn, got %+v", errs)
			}
		})
	}
}

func hasUnknownLiteralPin(errs []ValidationError, nodeID, pin string) bool {
	for _, issue := range errs {
		if issue.Code == CodeUnknownLiteralPin && issue.NodeID == nodeID && issue.Params["pin"] == pin {
			return true
		}
	}
	return false
}
