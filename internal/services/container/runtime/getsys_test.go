package runtime

import (
	"testing"

	"yhbox/internal/services/container"
	"yhbox/internal/services/expr"
)

func TestGetSys_LastTemplateFound(t *testing.T) {
	_, r := newTestRunner(t)
	r.currentTick = CaptureSnapshot(map[string]expr.Value{}, SysState{LastFound: true})

	n := &container.GraphNode{
		ID: "gs1", Kind: "GetSys",
		Config: map[string]any{"path": "lastTemplate.found"},
	}
	v, err := r.evalGetSys(n)
	if err != nil {
		t.Fatal(err)
	}
	if v != true {
		t.Fatalf("lastTemplate.found: want true, got %v", v)
	}
}

func TestGetSys_Iter(t *testing.T) {
	_, r := newTestRunner(t)
	r.currentTick = CaptureSnapshot(map[string]expr.Value{}, SysState{Iter: 7})

	n := &container.GraphNode{
		ID: "gs1", Kind: "GetSys",
		Config: map[string]any{"path": "iter"},
	}
	v, _ := r.evalGetSys(n)
	got, _ := expr.AsNumber(v)
	if got != 7.0 {
		t.Fatalf("iter: want 7, got %v", v)
	}
}

func TestGetSys_UnknownPath(t *testing.T) {
	_, r := newTestRunner(t)
	r.currentTick = CaptureSnapshot(map[string]expr.Value{}, SysState{})

	n := &container.GraphNode{
		ID: "gs1", Kind: "GetSys",
		Config: map[string]any{"path": "bogus.path"},
	}
	_, err := r.evalGetSys(n)
	if err == nil {
		t.Fatal("want err for unknown sys path")
	}
}

// Ensure SysPathSchema (runtime) and validator's sysPathSchemaCopy stay in sync.
// (Mirror check; runs in runtime package since SysPathSchema lives there.)
func TestSysPathSchema_AllPathsResolve(t *testing.T) {
	_, r := newTestRunner(t)
	r.currentTick = CaptureSnapshot(map[string]expr.Value{}, SysState{})

	for path := range SysPathSchema {
		n := &container.GraphNode{
			ID: "gs1", Kind: "GetSys",
			Config: map[string]any{"path": path},
		}
		_, err := r.evalGetSys(n)
		if err != nil {
			t.Errorf("SysPathSchema lists %q but resolveSysPath rejects: %v", path, err)
		}
	}
}
