package nodeauthoring

import (
	"encoding/json"
	"testing"
)

func TestGeneratedProjectionKeepsCompilerInstructionAsExactTaggedUnion(t *testing.T) {
	raw, err := GenerateSchema()
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]any
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatal(err)
	}
	definitions := schema["$defs"].(map[string]any)
	instruction := definitions["InstructionSpec"].(map[string]any)
	variants, ok := instruction["oneOf"].([]any)
	if !ok || len(variants) != 5 {
		t.Fatalf("projection instruction schema = %#v", instruction)
	}
	for _, rawVariant := range variants {
		variant := rawVariant.(map[string]any)
		if variant["additionalProperties"] != false || len(variant["required"].([]any)) != 2 {
			t.Fatalf("projection instruction variant is not exact: %#v", variant)
		}
	}
}
