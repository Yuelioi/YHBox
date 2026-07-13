package schema

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseSourceAcceptsStrictV3Example(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "contracts", "workflow", "v3", "examples", "minimal.json"))
	if err != nil {
		t.Fatal(err)
	}
	source, diagnostics := ParseSource(raw)
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}
	if source.Format != Format || source.Version != Version || source.EntryGraph != "main" {
		t.Fatalf("source = %#v", source)
	}
}

func TestParseSourceRejectsUnsupportedEpochs(t *testing.T) {
	for _, raw := range []string{
		`{"format":"yotta.workflow","version":2}`,
		`{"format":"yotta.workflow","version":0}`,
		`{"format":"yotta.workflow"}`,
		`{"version":3}`,
	} {
		_, diagnostics := ParseSource([]byte(raw))
		if len(diagnostics) != 1 || diagnostics[0].Code != CodeUnsupportedWorkflowFormat {
			t.Fatalf("%s: %#v", raw, diagnostics)
		}
	}
}

func TestParseSourceRejectsUnknownDuplicateAndTrailingFields(t *testing.T) {
	base := `{"format":"yotta.workflow","version":3,"workflow":{"id":"w","name":"W"},"revision":0,"entryGraph":"main","graphs":[{"id":"main","kind":"main","nodes":[],"edges":[],"inputs":[],"outputs":[]}],"variables":[],"secretRefs":[],"requestedCapabilities":[]}`
	tests := []struct{ name, raw, code string }{
		{"unknown", base[:len(base)-1] + `,"legacy":true}`, CodeUnknownField},
		{"duplicate", `{"format":"yotta.workflow","format":"yotta.workflow","version":3}`, CodeDuplicateField},
		{"trailing", base + `{}`, CodeInvalidWorkflowJSON},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, diagnostics := ParseSource([]byte(test.raw))
			if len(diagnostics) != 1 || diagnostics[0].Code != test.code {
				t.Fatalf("diagnostics = %#v", diagnostics)
			}
		})
	}
}

func TestParseSourceRejectsMissingRequiredCollections(t *testing.T) {
	raw := []byte(`{"format":"yotta.workflow","version":3,"workflow":{"id":"w","name":"W"},"revision":0,"entryGraph":"main","graphs":[{"id":"main","kind":"main","nodes":[],"edges":[],"inputs":[],"outputs":[]}]}`)
	_, diagnostics := ParseSource(raw)
	for _, field := range []string{"variables", "secretRefs", "requestedCapabilities"} {
		if !hasDiagnosticAt(diagnostics, CodeMissingRequiredField, field) {
			t.Fatalf("missing diagnostic for %s: %#v", field, diagnostics)
		}
	}
}

func TestParseSourceRejectsNullRequiredCollections(t *testing.T) {
	raw := []byte(`{"format":"yotta.workflow","version":3,"workflow":{"id":"w","name":"W"},"revision":0,"entryGraph":"main","graphs":[{"id":"main","kind":"main","nodes":[],"edges":[],"inputs":[],"outputs":[]}],"variables":null,"secretRefs":null,"requestedCapabilities":null}`)
	_, diagnostics := ParseSource(raw)
	for _, field := range []string{"variables", "secretRefs", "requestedCapabilities"} {
		if !hasDiagnosticAt(diagnostics, CodeMissingRequiredField, field) {
			t.Fatalf("missing diagnostic for %s: %#v", field, diagnostics)
		}
	}
}

func hasDiagnosticAt(diagnostics []Diagnostic, code, field string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code && len(diagnostic.FieldPath) > 0 && diagnostic.FieldPath[len(diagnostic.FieldPath)-1] == field {
			return true
		}
	}
	return false
}
