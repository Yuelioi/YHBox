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

	r := node.RunNode(context.Background(), rn, nil, nil, nil, node.StubServices())
	if len(r.Validation) == 0 {
		t.Error("expected REQUIRED_FIELD_MISSING for subgraphId")
	}
}
