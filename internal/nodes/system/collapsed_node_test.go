package system

import (
	"context"
	"testing"

	"github.com/yottaapp/yotta/internal/node"
)

func TestCollapsedNode_Spec_DynamicOutputsWithOnlyFailStatic(t *testing.T) {
	sp := (CollapsedNode{}).Spec()
	if !node.HasDynamicPortRole(&sp, node.DynamicPortOutput) {
		t.Error("CollapsedNode.Spec should declare graph-interface outputs")
	}
	if len(sp.Outputs) != 1 || sp.Outputs[0].Name != "Fail" {
		t.Errorf("Outputs = %+v, want only static Fail", sp.Outputs)
	}
}

func TestCollapsedNode_RunRegion_FiresBodyReachedExit(t *testing.T) {
	registry := node.NewRegistry()
	registry.Register(&CollapsedNode{})
	rn, _ := registry.Get("CollapsedNode")

	body := func(_ node.Ctx) (string, error) { return "decl-abc", nil }
	r := node.RunNodeAsRegion(context.Background(), rn, nil,
		map[string]any{cnInSubgraphID: "sg_foo"},
		nil, node.StubServices(), false, body)

	if r.Error != nil || r.Panic != nil {
		t.Fatalf("unexpected error/panic: %v / %v", r.Error, r.Panic)
	}
	if r.ExitName != "decl-abc" {
		t.Errorf("exit = %q, want %q", r.ExitName, "decl-abc")
	}
}

func TestCollapsedNode_RequiredSubgraphIDMissing(t *testing.T) {
	registry := node.NewRegistry()
	registry.Register(&CollapsedNode{})
	rn, _ := registry.Get("CollapsedNode")

	// CollapsedNode 是 RegionRunner — 用 RunNodeAsRegion 走 Required gate.
	r := node.RunNodeAsRegion(context.Background(), rn, nil, nil, nil,
		node.StubServices(), false, func(node.Ctx) (string, error) { return "", nil })
	if len(r.Validation) == 0 {
		t.Error("expected REQUIRED_FIELD_MISSING for subgraphId")
	}
}
