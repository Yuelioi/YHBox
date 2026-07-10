package system

import (
	"context"
	"errors"
	"testing"

	"github.com/yottaapp/yotta/internal/node"
)

func TestSubgraph_Spec_DynamicOutputsWithOnlyFailStatic(t *testing.T) {
	sp := (Subgraph{}).Spec()
	if !sp.DynamicOutputs {
		t.Error("Subgraph.Spec.DynamicOutputs should be true (出口 = callee OutputPins decl ID)")
	}
	if len(sp.Outputs) != 1 || sp.Outputs[0].Name != "Fail" {
		t.Errorf("Outputs = %+v, want only static Fail", sp.Outputs)
	}
}

func TestSubgraph_RunRegion_FiresBodyReachedExit(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Subgraph{})
	rn, _ := node.Get("Subgraph")

	for _, declID := range []string{"done", "failed", "0b1e4f7a-uuid-decl"} {
		calls := 0
		body := func(_ node.Ctx) (string, error) {
			calls++
			return declID, nil
		}

		r := node.RunNodeAsRegion(context.Background(), rn, nil,
			map[string]any{sgInSubgraphID: "sg_foo"},
			nil, node.StubServices(), false, body)

		if r.Error != nil {
			t.Fatalf("error = %v", r.Error)
		}
		if r.Panic != nil {
			t.Fatalf("panic: %v\n%s", r.Panic, r.PanicStack)
		}
		if calls != 1 {
			t.Errorf("body calls = %d, want 1", calls)
		}
		if r.ExitName != declID {
			t.Errorf("exit = %q, want %q", r.ExitName, declID)
		}
	}
}

func TestSubgraph_RunRegion_EmptyExitMeansNoFire(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Subgraph{})
	rn, _ := node.Get("Subgraph")

	body := func(_ node.Ctx) (string, error) { return "", nil }
	r := node.RunNodeAsRegion(context.Background(), rn, nil,
		map[string]any{sgInSubgraphID: "sg_foo"},
		nil, node.StubServices(), false, body)

	if r.Error != nil || r.Panic != nil {
		t.Fatalf("unexpected error/panic: %v / %v", r.Error, r.Panic)
	}
	if r.ExitName != "" {
		t.Errorf("exit = %q, want empty (body 未到达任何出口)", r.ExitName)
	}
}

func TestSubgraph_RunRegion_PropagatesBodyError(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Subgraph{})
	rn, _ := node.Get("Subgraph")

	boom := errors.New("body boom")
	body := func(_ node.Ctx) (string, error) { return "", boom }

	r := node.RunNodeAsRegion(context.Background(), rn, nil,
		map[string]any{sgInSubgraphID: "sg_foo"},
		nil, node.StubServices(), false, body)

	if !errors.Is(r.Error, boom) {
		t.Errorf("error = %v, want boom", r.Error)
	}
	if r.ExitName != "" {
		t.Errorf("exit = %q, want empty (error path)", r.ExitName)
	}
}

func TestSubgraph_RequiredSubgraphIDMissing(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Subgraph{})
	rn, _ := node.Get("Subgraph")

	// Subgraph 是 RegionRunner — 用 RunNodeAsRegion 走 Required gate.
	r := node.RunNodeAsRegion(context.Background(), rn, nil, nil, nil,
		node.StubServices(), false, func(node.Ctx) (string, error) { return "", nil })
	if len(r.Validation) == 0 {
		t.Errorf("expected REQUIRED_FIELD_MISSING for SubgraphID, got %+v", r)
	}
}

func TestSubgraph_Dependencies_RegisteredViaInterface(t *testing.T) {
	// registry 自动探测 Dependencer 接口 → rn.Dependencies != nil.
	node.ResetRegistryForTest()
	node.Register(&Subgraph{})
	rn, _ := node.Get("Subgraph")

	if rn.Dependencies == nil {
		t.Fatal("Subgraph should register as Dependencer (rn.Dependencies nil)")
	}
}
