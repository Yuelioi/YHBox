package variable

import (
	"context"
	"testing"

	"github.com/yottaapp/yotta/internal/node"
)

// IsPureData spec flag 校验.
func TestPureData_Flag(t *testing.T) {
	for _, n := range []node.Node{&GetVar{}, &GetParam{}} {
		s := n.Spec()
		if !s.IsPureData {
			t.Errorf("%s.IsPureData = false, want true", s.Kind)
		}
	}
}

// ============================================================================
// C5b: Get* framework-path tests — exercise EvaluatePureData with snapshot wrap.
// ============================================================================

// fixedParamStore — test helper, returns canned LocalParams values.
type fixedParamStore map[string]any

func (f fixedParamStore) Get(name string) (any, bool) {
	v, ok := f[name]
	return v, ok
}

func TestGetVar_EvaluateViaFramework_GlobalReadsSnapshot(t *testing.T) {
	registry := node.NewRegistry()
	registry.Register(&GetVar{})
	rn, _ := registry.Get("GetVar")

	services := node.StubServices()
	// Snapshot stub: 提供 frozen Vars. global scope 应走 snapshot.
	services.Snapshot = func(_ context.Context) node.Snapshot {
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

func TestGetParam_EvaluateViaFramework(t *testing.T) {
	registry := node.NewRegistry()
	registry.Register(&GetParam{})
	rn, _ := registry.Get("GetParam")

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
