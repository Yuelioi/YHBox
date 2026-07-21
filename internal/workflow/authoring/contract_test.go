package authoring_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/yottaapp/yotta/internal/workflow/authoring"
)

func TestGeneratedPatchSchemaUsesExactTaggedUnion(t *testing.T) {
	raw, err := authoring.GenerateSchema()
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		Definitions map[string]struct {
			OneOf []json.RawMessage `json:"oneOf"`
		} `json:"$defs"`
	}
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	if got := len(document.Definitions["Command"].OneOf); got != 33 {
		t.Fatalf("command variants = %d", got)
	}
	if !bytes.Contains(raw, []byte(`"additionalProperties": false`)) ||
		!bytes.Contains(raw, []byte(`"const": "connect"`)) ||
		!bytes.Contains(raw, []byte(`"connect"`)) {
		t.Fatalf("schema is not an exact tagged union: %s", raw)
	}
}
