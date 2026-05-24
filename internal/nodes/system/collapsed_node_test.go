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
	if !errors.Is(r.Error, errSubgraphNodeStub) {
		t.Errorf("error = %v, want errSubgraphNodeStub", r.Error)
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
