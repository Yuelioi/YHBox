package nodekind

import (
	"encoding/json"
	"testing"

	_ "yhbox/internal/services/container/nodekind/specs"
)

// TestExportJSONStable asserts that ExportJSON is deterministic (sorted by Kind)
// and contains all registered kinds with the expected static fields.
// D2 parity test depends on this being byte-stable across two calls.
func TestExportJSONStable(t *testing.T) {
	data, err := ExportJSON()
	if err != nil {
		t.Fatalf("ExportJSON: %v", err)
	}
	var specs []ExportedSpec
	if err := json.Unmarshal(data, &specs); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(specs) < 60 {
		t.Errorf("expected >= 60 registered specs, got %d", len(specs))
	}

	// Idempotent: second call produces byte-identical output.
	data2, err := ExportJSON()
	if err != nil {
		t.Fatalf("ExportJSON (2nd call): %v", err)
	}
	if string(data) != string(data2) {
		t.Errorf("ExportJSON not stable across calls")
	}

	// Verify sort: Kinds are ascending.
	for i := 1; i < len(specs); i++ {
		if specs[i-1].Kind > specs[i].Kind {
			t.Errorf("specs not sorted by Kind: %q > %q at index %d", specs[i-1].Kind, specs[i].Kind, i)
		}
	}
}

// TestExportedSpecShape spot-checks a few kinds to ensure the exported view
// correctly reflects backend Spec fields.
func TestExportedSpecShape(t *testing.T) {
	data, _ := ExportJSON()
	var specs []ExportedSpec
	_ = json.Unmarshal(data, &specs)
	byKind := map[string]ExportedSpec{}
	for _, s := range specs {
		byKind[s.Kind] = s
	}

	// Sleep: has data-in pin durationMs (number); exec node.
	sleep, ok := byKind["Sleep"]
	if !ok {
		t.Fatal("Sleep not in export")
	}
	if sleep.DataIn["durationMs"] != PinNumber {
		t.Errorf("Sleep.DataIn.durationMs = %v, want number", sleep.DataIn["durationMs"])
	}
	if sleep.IsPureData {
		t.Errorf("Sleep.IsPureData = true, want false")
	}
	if !contains(sleep.ExecIn, "in") || !contains(sleep.ExecOut, "out") {
		t.Errorf("Sleep exec pins wrong: %+v", sleep)
	}

	// GetVar: pure-data, value:any data-out, no exec pins.
	gv, ok := byKind["GetVar"]
	if !ok {
		t.Fatal("GetVar not in export")
	}
	if !gv.IsPureData {
		t.Errorf("GetVar.IsPureData = false, want true")
	}
	if len(gv.ExecIn) != 0 || len(gv.ExecOut) != 0 {
		t.Errorf("GetVar exec pins should be empty: %+v", gv)
	}
	if gv.DataOut["value"] != PinAny {
		t.Errorf("GetVar.DataOut.value = %v, want any", gv.DataOut["value"])
	}

	// Switch: has ExecOutFn.
	sw, ok := byKind["Switch"]
	if !ok {
		t.Fatal("Switch not in export")
	}
	if !sw.HasExecOutFn {
		t.Errorf("Switch.HasExecOutFn = false, want true")
	}

	// Expr: has DataInDynamicFn.
	expr, ok := byKind["Expr"]
	if !ok {
		t.Fatal("Expr not in export")
	}
	if !expr.HasDataInDynamicFn {
		t.Errorf("Expr.HasDataInDynamicFn = false, want true")
	}

	// CommentBox: isVisualOnly.
	cb, ok := byKind["CommentBox"]
	if !ok {
		t.Fatal("CommentBox not in export")
	}
	if !cb.IsVisualOnly {
		t.Errorf("CommentBox.IsVisualOnly = false, want true")
	}
}

func contains(s []string, want string) bool {
	for _, v := range s {
		if v == want {
			return true
		}
	}
	return false
}
