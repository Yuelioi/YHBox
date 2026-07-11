// internal/node/registry_test.go
package node

import (
	"testing"
	"time"
)

type stubNode struct{ kind string }

func (n stubNode) Spec() Spec { return Spec{Kind: n.kind} }

func (n stubNode) Run(ctx Ctx, in Inputs) (Outputs, error) { return nil, nil }

type retainedSpecNode struct{ spec Spec }

func (n *retainedSpecNode) Spec() Spec                     { return n.spec }
func (*retainedSpecNode) Run(Ctx, Inputs) (Outputs, error) { return nil, nil }

func TestRegister_HappyPath(t *testing.T) {
	registry := NewRegistry()
	registry.Register(stubNode{kind: "Test1"})

	rn, ok := registry.Get("Test1")
	if !ok {
		t.Fatal("Get failed")
	}
	if rn.Spec.Kind != "Test1" {
		t.Errorf("kind = %q, want Test1", rn.Spec.Kind)
	}
}

func TestRegister_Duplicate_Panics(t *testing.T) {
	registry := NewRegistry()
	registry.Register(stubNode{kind: "Dup"})
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on duplicate Register")
		}
	}()
	registry.Register(stubNode{kind: "Dup"})
}

func TestFreeze_BlocksRegister(t *testing.T) {
	registry := NewRegistry()
	registry.Register(stubNode{kind: "A"})
	registry.Freeze()
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on Register after Freeze")
		}
	}()
	registry.Register(stubNode{kind: "B"})
}

func TestAll_ReturnsSnapshot(t *testing.T) {
	registry := NewRegistry()
	registry.Register(stubNode{kind: "Y"})
	registry.Register(stubNode{kind: "X"})
	all := registry.All()
	if len(all) != 2 {
		t.Errorf("All len = %d, want 2", len(all))
	}
	if all[0].Spec.Kind != "X" || all[1].Spec.Kind != "Y" {
		t.Fatalf("All order = %s,%s, want X,Y", all[0].Spec.Kind, all[1].Spec.Kind)
	}
}

func TestRegistryInstancesAreParallelAndSnapshotIsolated(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{"ParallelA", "ParallelB"} {
		kind := kind
		t.Run(kind, func(t *testing.T) {
			t.Parallel()
			registry := NewRegistry()
			registry.Register(stubNode{kind: "SharedKind"})
			snapshot := registry.Snapshot()
			registry.Register(stubNode{kind: kind})
			if _, ok := snapshot.Get(kind); ok {
				t.Fatalf("snapshot observed later registration %s", kind)
			}
			if registered, ok := snapshot.Get("SharedKind"); !ok || registered.Spec.Kind != "SharedKind" {
				t.Fatal("snapshot lost its committed entry")
			}
		})
	}
}

func TestRegistryDefensivelyClonesMetadata(t *testing.T) {
	registry := NewRegistry()
	source := &retainedSpecNode{spec: Spec{
		Kind:   "RetainedSpec",
		Inputs: []InputSpec{{Name: "Value", Type: "JSON", Default: map[string]any{"nested": "original"}}},
	}}
	registry.Register(source)
	snapshot := registry.Snapshot()

	source.spec.Inputs[0].Name = "mutated-source"
	first, _ := snapshot.Get("RetainedSpec")
	first.Spec.Inputs[0].Name = "mutated-reader"
	first.Defaults["Value"].(map[string]any)["nested"] = "mutated-reader"

	second, _ := snapshot.Get("RetainedSpec")
	if second.Spec.Inputs[0].Name != "Value" {
		t.Fatalf("snapshot metadata escaped: %+v", second.Spec.Inputs)
	}
	if got := second.Defaults["Value"].(map[string]any)["nested"]; got != "original" {
		t.Fatalf("snapshot defaults escaped: %v", got)
	}
	live, _ := registry.Get("RetainedSpec")
	if live.Spec.Inputs[0].Name != "Value" {
		t.Fatalf("live registry metadata escaped: %+v", live.Spec.Inputs)
	}
}

type mixedVisibilityMetadata struct {
	Values []string
	hidden string
}

func TestRegistryClonesExportedReferencesInStructWithUnexportedFields(t *testing.T) {
	registry := NewRegistry()
	registry.Register(&retainedSpecNode{spec: Spec{Kind: "MixedMetadata", Inputs: []InputSpec{{
		Name: "Value", Type: "Any", Default: mixedVisibilityMetadata{Values: []string{"original"}, hidden: "kept"},
	}}}})

	first, _ := registry.Get("MixedMetadata")
	metadata := first.Defaults["Value"].(mixedVisibilityMetadata)
	metadata.Values[0] = "mutated"
	second, _ := registry.Get("MixedMetadata")
	got := second.Defaults["Value"].(mixedVisibilityMetadata)
	if got.Values[0] != "original" || got.hidden != "kept" {
		t.Fatalf("metadata escaped defensive clone: %+v", got)
	}
}

func TestRegisterRejectsCyclicMetadata(t *testing.T) {
	cycle := map[string]any{}
	cycle["self"] = cycle
	registry := NewRegistry()
	defer func() {
		if recover() == nil {
			t.Fatal("cyclic metadata must fail registration")
		}
	}()
	registry.Register(&retainedSpecNode{spec: Spec{Kind: "CyclicMetadata", Inputs: []InputSpec{{Name: "Value", Type: "Any", Default: cycle}}}})
}

type reentrantSpecNode struct{ registry *Registry }

func (n reentrantSpecNode) Spec() Spec {
	n.registry.Get("Existing")
	return Spec{Kind: "ReentrantSpec"}
}
func (reentrantSpecNode) Run(Ctx, Inputs) (Outputs, error) { return nil, nil }

func TestRegisterDoesNotHoldLockWhileCallingSpec(t *testing.T) {
	registry := NewRegistry()
	registry.Register(stubNode{kind: "Existing"})
	done := make(chan struct{})
	go func() {
		registry.Register(reentrantSpecNode{registry: registry})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Register deadlocked in a re-entrant Spec call")
	}
}

func TestSnapshotRegistryRejectsNil(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("nil registry must fail fast")
		}
	}()
	SnapshotRegistry(nil)
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
	registry := NewRegistry()
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on zero-capability non-marker node, got none")
		}
	}()
	registry.Register(zeroCapNode{})
}

func TestRegister_MultipleCapabilities_Panics(t *testing.T) {
	registry := NewRegistry()
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on multi-capability node, got none")
		}
	}()
	registry.Register(multiCapNode{})
}

func TestRegister_IsPureDataWithoutEvaluator_Panics(t *testing.T) {
	registry := NewRegistry()
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on IsPureData node without Evaluator, got none")
		}
	}()
	registry.Register(pureNoEvalNode{})
}

func TestRegister_IsGraphMarker_AllowsZero_OK(t *testing.T) {
	registry := NewRegistry()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("expected no panic for IsGraphMarker zero-cap node, got %v", r)
		}
	}()
	registry.Register(markerNode{})
}

func TestRegister_IsVisualOnly_AllowsZero_OK(t *testing.T) {
	registry := NewRegistry()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("expected no panic for IsVisualOnly zero-cap node, got %v", r)
		}
	}()
	registry.Register(visualNode{})
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
	registry := NewRegistry()
	defer func() {
		if recover() == nil {
			t.Fatal("NeedsWindow 无 Window 输入应 panic")
		}
	}()
	registry.Register(&badNeedsWindowNode{})
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
	registry := NewRegistry()
	defer func() {
		if recover() == nil {
			t.Fatal("expected unknown runtime capability panic")
		}
	}()
	registry.Register(runtimeCapabilityNode{capabilities: []RuntimeCapability{"unknown"}})
}

func TestRegister_RejectsDuplicateRuntimeCapability(t *testing.T) {
	registry := NewRegistry()
	defer func() {
		if recover() == nil {
			t.Fatal("expected duplicate runtime capability panic")
		}
	}()
	registry.Register(runtimeCapabilityNode{capabilities: []RuntimeCapability{RuntimeCapabilityLog, RuntimeCapabilityLog}})
}
