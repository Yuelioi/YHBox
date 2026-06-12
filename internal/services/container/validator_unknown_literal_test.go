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

// Expr 节点 inputs 是动态的(config.Inputs[]), 不在 Spec.Inputs — 不可误报。
func TestUnknownLiteral_ExprSkipped(t *testing.T) {
	c := reqContainer([]GraphNode{
		{ID: "e", Kind: "Expr", Config: map[string]any{"literal": map[string]any{"foo": 1}}},
	}, nil)
	if errs := validateUnknownLiteralPins(c, nil); hasCodeForNode(errs, CodeUnknownLiteralPin, "e") {
		t.Fatalf("Expr dynamic inputs must be skipped, got %+v", errs)
	}
}
