package system

import (
	"context"
	"errors"
	"testing"

	"yhbox/internal/node"
)

func TestCollapsedNode_RunReturnsSentinel(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&CollapsedNode{})
	rn, _ := node.Get("CollapsedNode")

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{cnInSubgraphID: "sg_anon_123"},
		nil, node.StubServices())
	// Phase 5.5b: CollapsedNode 升级成 RegionRunner, Run 防御性返 node.ErrMustUseRegion.
	if !errors.Is(r.Error, node.ErrMustUseRegion) {
		t.Errorf("error = %v, want node.ErrMustUseRegion", r.Error)
	}
}

func TestCollapsedNode_RequiredSubgraphIDMissing(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&CollapsedNode{})
	rn, _ := node.Get("CollapsedNode")

	r := node.RunNode(context.Background(), rn, nil, nil, nil, node.StubServices())
	if len(r.Validation) == 0 {
		t.Error("expected REQUIRED_FIELD_MISSING for subgraphId")
	}
}
