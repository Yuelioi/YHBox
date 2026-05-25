package variable

import (
	"context"
	"strings"
	"testing"

	"yhbox/internal/node"
)

// IsPureData spec flag 校验.
func TestPureData_Flag(t *testing.T) {
	for _, n := range []node.Node{&GetVar{}, &GetSys{}, &GetParam{}} {
		s := n.Spec()
		if !s.IsPureData {
			t.Errorf("%s.IsPureData = false, want true", s.Kind)
		}
	}
}

// ============================================================================
// C5b: Get* framework-path tests — exercise EvaluatePureData with snapshot wrap.
// ============================================================================

// fixedSysStore — test helper, returns canned values without schema validation.
type fixedSysStore map[string]any

func (f fixedSysStore) Get(path string) (any, bool) {
	v, ok := f[path]
	return v, ok
}

// fixedParamStore — test helper, returns canned LocalParams values.
type fixedParamStore map[string]any

func (f fixedParamStore) Get(name string) (any, bool) {
	v, ok := f[name]
	return v, ok
}

func TestGetVar_EvaluateViaFramework_GlobalReadsSnapshot(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&GetVar{})
	rn, _ := node.Get("GetVar")

	services := node.StubServices()
	// Snapshot stub: 提供 frozen Vars. global scope 应走 snapshot.
	services.Snapshot = func() node.Snapshot {
		return node.Snapshot{Vars: map[string]any{"counter": float64(42)}}
	}

	v, err := node.EvaluatePureData(context.Background(), rn,
		nil, // dataWire
		map[string]any{
			"VarName": "counter",
			"Scope":   "global",
		},
		services,
	)
	if err != nil {
		t.Fatal(err)
	}
	if v != float64(42) {
		t.Errorf("expected 42 from snapshot, got %v", v)
	}
}

func TestGetSys_EvaluateViaFramework_FrozenPath(t *testing.T) {
	// 用 schema 内合法 path. lastBarTrack.cursorX 在 sys.PathSchema, 走 frozen sys store.
	node.ResetRegistryForTest()
	node.Register(&GetSys{})
	rn, _ := node.Get("GetSys")

	services := node.StubServices()
	services.Snapshot = func() node.Snapshot {
		return node.Snapshot{
			Sys: fixedSysStore{"lastBarTrack.cursorX": float64(123)},
		}
	}

	v, err := node.EvaluatePureData(context.Background(), rn,
		nil,
		map[string]any{"Path": "lastBarTrack.cursorX"},
		services,
	)
	if err != nil {
		t.Fatal(err)
	}
	if v != float64(123) {
		t.Errorf("expected 123 from frozen sys, got %v", v)
	}
}

func TestGetSys_Evaluate_UnknownPath_Errors(t *testing.T) {
	// schema 校验保留: bogus path 应返 error, 不 silent nil (legacy evalGetSys parity).
	node.ResetRegistryForTest()
	node.Register(&GetSys{})
	rn, _ := node.Get("GetSys")

	services := node.StubServices()
	services.Snapshot = func() node.Snapshot { return node.Snapshot{} }

	_, err := node.EvaluatePureData(context.Background(), rn,
		nil,
		map[string]any{"Path": "bogus.field"},
		services,
	)
	if err == nil {
		t.Fatal("expected error on unknown sys path, got nil")
	}
	if !strings.Contains(err.Error(), "unknown sys path") {
		t.Errorf("expected 'unknown sys path' in error, got: %v", err)
	}
}

func TestGetParam_EvaluateViaFramework(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&GetParam{})
	rn, _ := node.Get("GetParam")

	services := node.StubServices()
	// Stub ParamStore with canned value.
	services.Params = fixedParamStore{"input1": "paramVal"}

	v, err := node.EvaluatePureData(context.Background(), rn,
		nil,
		map[string]any{"ParamName": "input1"},
		services,
	)
	if err != nil {
		t.Fatal(err)
	}
	if v != "paramVal" {
		t.Errorf("expected paramVal, got %v", v)
	}
}
