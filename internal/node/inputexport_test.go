package node

import "testing"

func TestExportDeclaredInputs_OnlyDeclaredPins(t *testing.T) {
	spec := &Spec{
		Kind: "X",
		Inputs: []InputSpec{
			{Name: "key", Type: "string"},
			{Name: "thr", Type: "number"},
		},
	}
	in := &inputsImpl{
		merged:  map[string]any{"key": "hook", "thr": 0.9, "_internal": 123},
		present: map[string]bool{"key": true, "thr": true, "_internal": true},
	}
	got := ExportDeclaredInputs(spec, in)
	if len(got) != 2 || got["key"] != "hook" || got["thr"] != 0.9 {
		t.Fatalf("ExportDeclaredInputs = %#v", got)
	}
	if _, leaked := got["_internal"]; leaked {
		t.Fatalf("undeclared key leaked: %#v", got)
	}
}

func TestExportDeclaredInputs_MissingPinOmitted(t *testing.T) {
	spec := &Spec{Inputs: []InputSpec{{Name: "a"}, {Name: "b"}}}
	in := &inputsImpl{merged: map[string]any{"a": 1}, present: map[string]bool{"a": true}}
	got := ExportDeclaredInputs(spec, in)
	if _, ok := got["b"]; ok {
		t.Fatalf("unpresent pin b should be omitted: %#v", got)
	}
}
