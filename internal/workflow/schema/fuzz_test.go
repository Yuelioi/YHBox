package schema

import "testing"

func FuzzParseSource(f *testing.F) {
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
