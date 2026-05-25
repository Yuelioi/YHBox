package system

import (
	"context"
	"testing"

	"yhbox/internal/node"
)

func TestCollapsedNode_RequiredSubgraphIDMissing(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&CollapsedNode{})
	rn, _ := node.Get("CollapsedNode")

	// CollapsedNode 是 RegionRunner — 用 RunNodeAsRegion 走 Required gate.
	r := node.RunNodeAsRegion(context.Background(), rn, nil, nil, nil,
		node.StubServices(), func(node.Ctx) error { return nil })
	if len(r.Validation) == 0 {
		t.Error("expected REQUIRED_FIELD_MISSING for subgraphId")
	}
}
