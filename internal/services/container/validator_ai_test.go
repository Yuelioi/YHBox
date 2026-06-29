package container

import "testing"

func aiNode(literal map[string]any, inputs []any) *GraphNode {
	cfg := map[string]any{}
	if literal != nil {
		cfg["literal"] = literal
	}
	if inputs != nil {
		cfg["Inputs"] = inputs
	}
	return &GraphNode{ID: "n", Kind: "AI", Config: cfg}
}

func hasValidationCode(errs []ValidationError, code string) bool {
	for _, e := range errs {
		if e.Code == code {
			return true
		}
	}
	return false
}

func TestValidateAI_EmptyUser(t *testing.T) {
	errs := validateAI(aiNode(nil, nil))
	if !hasValidationCode(errs, "AI_EMPTY_USER") {
		t.Errorf("expected AI_EMPTY_USER, got %+v", errs)
	}
}

func TestValidateAI_UndeclaredPlaceholder(t *testing.T) {
	errs := validateAI(aiNode(map[string]any{"User": "{{ghost}}"}, nil))
	if !hasValidationCode(errs, "AI_UNDECLARED_PLACEHOLDER") {
		t.Errorf("expected AI_UNDECLARED_PLACEHOLDER, got %+v", errs)
	}
}

func TestValidateAI_DeclaredPlaceholderOK(t *testing.T) {
	errs := validateAI(aiNode(
		map[string]any{"User": "hi {{x}}"},
		[]any{map[string]any{"Name": "x", "Type": "String"}},
	))
	if hasValidationCode(errs, "AI_UNDECLARED_PLACEHOLDER") {
		t.Errorf("declared {{x}} must not flag: %+v", errs)
	}
	if hasValidationCode(errs, "AI_EMPTY_USER") {
		t.Errorf("User non-empty must not flag: %+v", errs)
	}
}

func TestValidateAI_ImageInText(t *testing.T) {
	errs := validateAI(aiNode(
		map[string]any{"User": "see {{pic}}"},
		[]any{map[string]any{"Name": "pic", "Type": "Image"}},
	))
	if !hasValidationCode(errs, "AI_IMAGE_IN_TEXT") {
		t.Errorf("expected AI_IMAGE_IN_TEXT warn, got %+v", errs)
	}
}

func TestValidateAI_EscapedPlaceholderNotFlagged(t *testing.T) {
	errs := validateAI(aiNode(map[string]any{"User": `lit \{{x}}`}, nil))
	if hasValidationCode(errs, "AI_UNDECLARED_PLACEHOLDER") {
		t.Errorf("escaped \\{{ must not be treated as a reference: %+v", errs)
	}
}
