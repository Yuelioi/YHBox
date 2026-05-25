package system

import (
	"context"
	"errors"
	"testing"

	"yhbox/internal/node"
)

func TestSubgraph_Run_ReturnsMustUseRegionSentinel(t *testing.T) {
	// Run (非 RunRegion) 被框架直调 → 防御 sentinel.
	node.ResetRegistryForTest()
	node.Register(&Subgraph{})
	rn, _ := node.Get("Subgraph")

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{sgInSubgraphID: "sg_foo"},
		nil, node.StubServices())

	if !errors.Is(r.Error, node.ErrMustUseRegion) {
		t.Errorf("Run error = %v, want node.ErrMustUseRegion", r.Error)
	}
}

func TestSubgraph_RunRegion_InvokesBodyOnceThenDone(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Subgraph{})
	rn, _ := node.Get("Subgraph")

	calls := 0
	body := func(_ node.Ctx) error {
		calls++
		return nil
	}

	r := node.RunNodeAsRegion(context.Background(), rn, nil,
		map[string]any{sgInSubgraphID: "sg_foo"},
		nil, node.StubServices(), body)

	if r.Error != nil {
		t.Fatalf("error = %v", r.Error)
	}
	if r.Panic != nil {
		t.Fatalf("panic: %v\n%s", r.Panic, r.PanicStack)
	}
	if calls != 1 {
		t.Errorf("body calls = %d, want 1", calls)
	}
	if r.ExitName != sgOutDone {
		t.Errorf("exit = %q, want %q", r.ExitName, sgOutDone)
	}
}

func TestSubgraph_RunRegion_PropagatesBodyError(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Subgraph{})
	rn, _ := node.Get("Subgraph")

	boom := errors.New("body boom")
	body := func(_ node.Ctx) error { return boom }

	r := node.RunNodeAsRegion(context.Background(), rn, nil,
		map[string]any{sgInSubgraphID: "sg_foo"},
		nil, node.StubServices(), body)

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

	r := node.RunNode(context.Background(), rn, nil, nil, nil, node.StubServices())
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
