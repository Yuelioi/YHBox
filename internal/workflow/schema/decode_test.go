package schema

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
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

func TestParseSourceUsesJSONSchemaNumericEqualityForVersion(t *testing.T) {
	raw := []byte(`{"format":"yotta.workflow","version":3.0,"workflow":{"id":"w","name":"W"},"revision":0.0,"entryGraph":"main","graphs":[{"id":"main","kind":"main","nodes":[],"edges":[],"inputs":[],"outputs":[]}],"variables":[],"secretRefs":[],"requestedCapabilities":[]}`)
	_, diagnostics := ParseSource(raw)
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v", diagnostics)
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
		if !hasDiagnosticAt(diagnostics, CodeInvalidField, field) {
			t.Fatalf("missing diagnostic for %s: %#v", field, diagnostics)
		}
	}
}

func TestParseSourceRejectsEmptyCapability(t *testing.T) {
	raw := []byte(`{"format":"yotta.workflow","version":3,"workflow":{"id":"w","name":"W"},"revision":0,"entryGraph":"main","graphs":[{"id":"main","kind":"main","nodes":[],"edges":[],"inputs":[],"outputs":[]}],"variables":[],"secretRefs":[],"requestedCapabilities":[""]}`)
	_, diagnostics := ParseSource(raw)
	if !hasDiagnosticPath(diagnostics, []string{"requestedCapabilities", "0"}) {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}
}

func TestParseSourceEnforcesGeneratedStructuralContract(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		path []string
	}{
		{
			name: "null revision",
			raw:  `{"format":"yotta.workflow","version":3,"workflow":{"id":"w","name":"W"},"revision":null,"entryGraph":"main","graphs":[{"id":"main","kind":"main","nodes":[],"edges":[],"inputs":[],"outputs":[]}],"variables":[],"secretRefs":[],"requestedCapabilities":[]}`,
			path: []string{"revision"},
		},
		{
			name: "unsafe revision",
			raw:  `{"format":"yotta.workflow","version":3,"workflow":{"id":"w","name":"W"},"revision":9007199254740992,"entryGraph":"main","graphs":[{"id":"main","kind":"main","nodes":[],"edges":[],"inputs":[],"outputs":[]}],"variables":[],"secretRefs":[],"requestedCapabilities":[]}`,
			path: []string{"revision"},
		},
		{
			name: "missing node position",
			raw:  `{"format":"yotta.workflow","version":3,"workflow":{"id":"w","name":"W"},"revision":0,"entryGraph":"main","graphs":[{"id":"main","kind":"main","nodes":[{"id":"n","kind":"core.noop","config":{}}],"edges":[],"inputs":[],"outputs":[]}],"variables":[],"secretRefs":[],"requestedCapabilities":[]}`,
			path: []string{"graphs", "0", "nodes", "0", "position"},
		},
		{
			name: "missing position coordinate",
			raw:  `{"format":"yotta.workflow","version":3,"workflow":{"id":"w","name":"W"},"revision":0,"entryGraph":"main","graphs":[{"id":"main","kind":"main","nodes":[{"id":"n","kind":"core.noop","position":{"x":0},"config":{}}],"edges":[],"inputs":[],"outputs":[]}],"variables":[],"secretRefs":[],"requestedCapabilities":[]}`,
			path: []string{"graphs", "0", "nodes", "0", "position", "y"},
		},
		{
			name: "empty edge endpoint",
			raw:  `{"format":"yotta.workflow","version":3,"workflow":{"id":"w","name":"W"},"revision":0,"entryGraph":"main","graphs":[{"id":"main","kind":"main","nodes":[],"edges":[{"from":"","to":"n.in"}],"inputs":[],"outputs":[]}],"variables":[],"secretRefs":[],"requestedCapabilities":[]}`,
			path: []string{"graphs", "0", "edges", "0", "from"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, diagnostics := ParseSource([]byte(test.raw))
			if !hasDiagnosticPath(diagnostics, test.path) {
				t.Fatalf("missing diagnostic at %v: %#v", test.path, diagnostics)
			}
		})
	}
}

func TestParseSourceReportsCompleteNestedUnknownFieldContext(t *testing.T) {
	raw := []byte(`{"format":"yotta.workflow","version":3,"workflow":{"id":"w","name":"W"},"revision":0,"entryGraph":"main","graphs":[{"id":"main","kind":"main","nodes":[{"id":"n","kind":"core.noop","position":{"x":0,"y":0,"legacy":true},"config":{}}],"edges":[],"inputs":[],"outputs":[]}],"variables":[],"secretRefs":[],"requestedCapabilities":[]}`)
	_, diagnostics := ParseSource(raw)
	want := []string{"graphs", "0", "nodes", "0", "position", "legacy"}
	if !hasDiagnosticPath(diagnostics, want) {
		t.Fatalf("missing diagnostic at %v: %#v", want, diagnostics)
	}
	if diagnostics[0].GraphPath[0] != "main" || diagnostics[0].NodeID != "n" {
		t.Fatalf("context = %#v", diagnostics[0])
	}
}

func TestParseSourceRejectsStructuralDiagnosticBombBeforeSchemaValidation(t *testing.T) {
	var raw strings.Builder
	raw.WriteString(`{"format":"yotta.workflow","version":3`)
	for index := 0; index < MaxDiagnostics+20; index++ {
		raw.WriteString(`,"unknown` + strconv.Itoa(index) + `":true`)
	}
	raw.WriteString(`,"workflow":{"id":"w","name":"W"},"revision":0,"entryGraph":"main","graphs":[{"id":"main","kind":"main","nodes":[],"edges":[],"inputs":[],"outputs":[]}],"variables":[],"secretRefs":[],"requestedCapabilities":[]}`)
	_, diagnostics := ParseSource([]byte(raw.String()))
	if len(diagnostics) != 1 || diagnostics[0].Code != CodeDiagnosticBudgetExceeded {
		t.Fatalf("diagnostics = %d %#v", len(diagnostics), diagnostics)
	}
}

func TestParseSourceReportsCollectionLimitWithoutDiagnosticBudgetSentinel(t *testing.T) {
	var graphs strings.Builder
	for index := 0; index < 257; index++ {
		if index > 0 {
			graphs.WriteByte(',')
		}
		kind := "subgraph"
		if index == 0 {
			kind = "main"
		}
		graphs.WriteString(`{"id":"g` + strconv.Itoa(index) + `","kind":"` + kind + `","nodes":[],"edges":[],"inputs":[],"outputs":[]}`)
	}
	raw := `{"format":"yotta.workflow","version":3,"workflow":{"id":"w","name":"W"},"revision":0,"entryGraph":"g0","graphs":[` + graphs.String() + `],"variables":[],"secretRefs":[],"requestedCapabilities":[]}`
	_, diagnostics := ParseSource([]byte(raw))
	if len(diagnostics) == 0 {
		t.Fatal("oversized graph collection was accepted")
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == CodeDiagnosticBudgetExceeded {
			t.Fatalf("collection limit mislabeled as diagnostic exhaustion: %#v", diagnostics)
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

func hasDiagnosticPath(diagnostics []Diagnostic, path []string) bool {
	for _, diagnostic := range diagnostics {
		if len(diagnostic.FieldPath) != len(path) {
			continue
		}
		matches := true
		for index := range path {
			matches = matches && diagnostic.FieldPath[index] == path[index]
		}
		if matches {
			return true
		}
	}
	return false
}
