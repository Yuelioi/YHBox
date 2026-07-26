package schema

import (
	"strings"
	"testing"
)

func FuzzParseSource(f *testing.F) {
	f.Add([]byte(validSource31ForTest()))
	f.Add([]byte(strings.Repeat("[", 140) + "0" + strings.Repeat("]", 140)))
	f.Add([]byte(`{"format":"yotta.workflow","version":3}`))
	f.Add([]byte(`{"format":"yotta.workflow","format":"yotta.workflow","version":3}`))
	f.Add([]byte(`{"schemaVersion":2}`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		source, diagnostics := ParseSource(raw)
		if len(diagnostics) == 0 && (source.Format != Format || source.Version != Version) {
			t.Fatalf("accepted wrong epoch: %#v", source)
		}
	})
}
