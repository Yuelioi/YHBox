// internal/node/registry_test.go
package node

import "testing"

type stubNode struct{ kind string }

func (n stubNode) Spec() Spec { return Spec{Kind: n.kind} }

func (n stubNode) Run(ctx Ctx, in Inputs) (Outputs, error) { return nil, nil }

func TestRegister_HappyPath(t *testing.T) {
	ResetRegistryForTest()
	Register(stubNode{kind: "Test1"})

	rn, ok := Get("Test1")
	if !ok {
		t.Fatal("Get failed")
	}
	if rn.Spec.Kind != "Test1" {
		t.Errorf("kind = %q, want Test1", rn.Spec.Kind)
	}
}

func TestRegister_Duplicate_Panics(t *testing.T) {
	ResetRegistryForTest()
	Register(stubNode{kind: "Dup"})
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on duplicate Register")
		}
	}()
	Register(stubNode{kind: "Dup"})
}

func TestFreeze_BlocksRegister(t *testing.T) {
	ResetRegistryForTest()
	Register(stubNode{kind: "A"})
	Freeze()
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on Register after Freeze")
		}
	}()
	Register(stubNode{kind: "B"})
}

func TestAll_ReturnsSnapshot(t *testing.T) {
	ResetRegistryForTest()
	Register(stubNode{kind: "X"})
	Register(stubNode{kind: "Y"})
	all := All()
	if len(all) != 2 {
		t.Errorf("All len = %d, want 2", len(all))
	}
}

// ============================================================================
// C5b: Register strict capability invariant — exactly-one + IsPureData⟹Evaluator.
// ============================================================================

type zeroCapNode struct{}

func (zeroCapNode) Spec() Spec { return Spec{Kind: "ZeroCap"} }

type multiCapNode struct{}

func (multiCapNode) Spec() Spec                            { return Spec{Kind: "MultiCap"} }
func (multiCapNode) Run(_ Ctx, _ Inputs) (Outputs, error)  { return nil, nil }
func (multiCapNode) Evaluate(_ Ctx, _ Inputs) (any, error) { return nil, nil }

type pureNoEvalNode struct{}

func (pureNoEvalNode) Spec() Spec                           { return Spec{Kind: "PureNoEval", IsPureData: true} }
func (pureNoEvalNode) Run(_ Ctx, _ Inputs) (Outputs, error) { return nil, nil }

type markerNode struct{}

func (markerNode) Spec() Spec { return Spec{Kind: "Marker", IsGraphMarker: true} }

type visualNode struct{}

func (visualNode) Spec() Spec { return Spec{Kind: "Visual", IsVisualOnly: true} }

func TestRegister_ZeroCapability_NonMarker_Panics(t *testing.T) {
	ResetRegistryForTest()
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on zero-capability non-marker node, got none")
		}
	}()
	Register(zeroCapNode{})
}

func TestRegister_MultipleCapabilities_Panics(t *testing.T) {
	ResetRegistryForTest()
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on multi-capability node, got none")
		}
	}()
	Register(multiCapNode{})
}

func TestRegister_IsPureDataWithoutEvaluator_Panics(t *testing.T) {
	ResetRegistryForTest()
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on IsPureData node without Evaluator, got none")
		}
	}()
	Register(pureNoEvalNode{})
}

func TestRegister_IsGraphMarker_AllowsZero_OK(t *testing.T) {
	ResetRegistryForTest()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("expected no panic for IsGraphMarker zero-cap node, got %v", r)
		}
	}()
	Register(markerNode{})
}

func TestRegister_IsVisualOnly_AllowsZero_OK(t *testing.T) {
	ResetRegistryForTest()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("expected no panic for IsVisualOnly zero-cap node, got %v", r)
		}
	}()
	Register(visualNode{})
}

// ============================================================================
// C1b: Register invariant — NeedsWindow ⟹ has Window input.
// ============================================================================

type badNeedsWindowNode struct{}

func (badNeedsWindowNode) Spec() Spec {
	return Spec{Kind: "BadNeedsWindow", NeedsWindow: true,
		Inputs: []InputSpec{{Name: "In", Type: "Exec"}}}
}
func (badNeedsWindowNode) Run(_ Ctx, _ Inputs) (Outputs, error) { return nil, nil }

func TestRegister_NeedsWindowRequiresWindowInput(t *testing.T) {
	ResetRegistryForTest()
	defer ResetRegistryForTest()
	defer func() {
		if recover() == nil {
			t.Fatal("NeedsWindow 无 Window 输入应 panic")
		}
	}()
	Register(&badNeedsWindowNode{})
}

type runtimeCapabilityNode struct{ capabilities []RuntimeCapability }

func (n runtimeCapabilityNode) Spec() Spec {
	return Spec{
		Kind:                "RuntimeCapabilityNode",
		Outputs:             []OutputSpec{{Name: "Done", Type: TypeExec}},
		RuntimeCapabilities: n.capabilities,
	}
}

func (runtimeCapabilityNode) Run(ctx Ctx, _ Inputs) (Outputs, error) {
	return ctx.Out("Done").Fire(), nil
}

func TestRegister_RejectsUnknownRuntimeCapability(t *testing.T) {
	ResetRegistryForTest()
	defer ResetRegistryForTest()
	defer func() {
		if recover() == nil {
			t.Fatal("expected unknown runtime capability panic")
		}
	}()
	Register(runtimeCapabilityNode{capabilities: []RuntimeCapability{"unknown"}})
}

func TestRegister_RejectsDuplicateRuntimeCapability(t *testing.T) {
	ResetRegistryForTest()
	defer ResetRegistryForTest()
	defer func() {
		if recover() == nil {
			t.Fatal("expected duplicate runtime capability panic")
		}
	}()
	Register(runtimeCapabilityNode{capabilities: []RuntimeCapability{RuntimeCapabilityLog, RuntimeCapabilityLog}})
}
