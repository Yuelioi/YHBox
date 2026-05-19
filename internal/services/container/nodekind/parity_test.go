package nodekind_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"yhbox/internal/services/container/nodekind"
	_ "yhbox/internal/services/container/nodekind/specs"
)

// TestRegistryParity loads the frontend registry dump (produced by
// nodeRegistry/parity.test.ts via `npm test`) and asserts the backend's
// ExportJSON matches it field-by-field, kind-by-kind.
//
// If `testdata/fe-registry.json` is missing the test fails loudly with a
// reproduction recipe — this is intentional, the parity check is the whole
// point of D2 and silently skipping defeats the cross-check.
//
// To regenerate the FE dump after a frontend spec change:
//   cd frontend && npm test -- parity
func TestRegistryParity(t *testing.T) {
	feBytes, err := os.ReadFile(filepath.Join("testdata", "fe-registry.json"))
	if err != nil {
		t.Fatalf("read fe-registry.json: %v\n\nRegenerate via:\n  cd frontend && npm test -- parity", err)
	}
	var fe []nodekind.ExportedSpec
	if err := json.Unmarshal(feBytes, &fe); err != nil {
		t.Fatalf("unmarshal fe dump: %v", err)
	}

	beBytes, err := nodekind.ExportJSON()
	if err != nil {
		t.Fatalf("ExportJSON: %v", err)
	}
	var be []nodekind.ExportedSpec
	if err := json.Unmarshal(beBytes, &be); err != nil {
		t.Fatalf("unmarshal be dump: %v", err)
	}

	feByKind := indexByKind(fe)
	beByKind := indexByKind(be)

	// 1. Kind set equality.
	feKinds := sortedKeys(feByKind)
	beKinds := sortedKeys(beByKind)
	if !reflect.DeepEqual(feKinds, beKinds) {
		t.Errorf("kind set mismatch:\n  FE-only: %v\n  BE-only: %v",
			diffStrings(feKinds, beKinds), diffStrings(beKinds, feKinds))
		return // further per-kind diffs are noise once the sets differ
	}

	// 2. Per-kind field-by-field.
	for _, kind := range feKinds {
		fe, be := feByKind[kind], beByKind[kind]
		if fe.Group != be.Group {
			t.Errorf("kind %q: group FE=%q BE=%q", kind, fe.Group, be.Group)
		}
		if !equalStringSlices(fe.ExecIn, be.ExecIn) {
			t.Errorf("kind %q: execIn FE=%v BE=%v", kind, fe.ExecIn, be.ExecIn)
		}
		if !equalStringSlices(fe.ExecOut, be.ExecOut) {
			t.Errorf("kind %q: execOut FE=%v BE=%v", kind, fe.ExecOut, be.ExecOut)
		}
		if fe.HasExecOutFn != be.HasExecOutFn {
			t.Errorf("kind %q: hasExecOutFn FE=%v BE=%v", kind, fe.HasExecOutFn, be.HasExecOutFn)
		}
		if !reflect.DeepEqual(fe.DataIn, be.DataIn) {
			t.Errorf("kind %q: dataIn FE=%v BE=%v", kind, fe.DataIn, be.DataIn)
		}
		if fe.HasDataInDynamicFn != be.HasDataInDynamicFn {
			t.Errorf("kind %q: hasDataInDynamicFn FE=%v BE=%v", kind, fe.HasDataInDynamicFn, be.HasDataInDynamicFn)
		}
		if !reflect.DeepEqual(fe.DataOut, be.DataOut) {
			t.Errorf("kind %q: dataOut FE=%v BE=%v", kind, fe.DataOut, be.DataOut)
		}
		if fe.IsPureData != be.IsPureData {
			t.Errorf("kind %q: isPureData FE=%v BE=%v", kind, fe.IsPureData, be.IsPureData)
		}
		if fe.IsVisualOnly != be.IsVisualOnly {
			t.Errorf("kind %q: isVisualOnly FE=%v BE=%v", kind, fe.IsVisualOnly, be.IsVisualOnly)
		}
	}
}

func indexByKind(specs []nodekind.ExportedSpec) map[string]nodekind.ExportedSpec {
	out := make(map[string]nodekind.ExportedSpec, len(specs))
	for _, s := range specs {
		out[s.Kind] = s
	}
	return out
}

func sortedKeys(m map[string]nodekind.ExportedSpec) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func diffStrings(a, b []string) []string {
	bs := make(map[string]bool, len(b))
	for _, x := range b {
		bs[x] = true
	}
	var out []string
	for _, x := range a {
		if !bs[x] {
			out = append(out, x)
		}
	}
	return out
}

// equalStringSlices treats nil and empty as equal — both sides normalize empty
// slices to nil in their ExportedSpec.
func equalStringSlices(a, b []string) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	return reflect.DeepEqual(a, b)
}
