package system

import (
	"context"
	"errors"
	"testing"

	"yhbox/internal/node"
)

func TestSubgraph_StubReturnsError(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Subgraph{})
	rn, _ := node.Get("Subgraph")

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{sgInSubgraphID: "any_id"},
		nil, node.StubServices())

	if r.Error == nil {
		t.Fatal("expected stub error")
	}
	if !errors.Is(r.Error, errSubgraphNotImplemented) {
		t.Errorf("error = %v, want errSubgraphNotImplemented", r.Error)
	}
}

func TestSubgraph_RequiredSubgraphIDMissing(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Subgraph{})
	rn, _ := node.Get("Subgraph")

	r := node.RunNode(context.Background(), rn, nil, nil, nil, node.StubServices())
	if len(r.Validation) == 0 {
		t.Error("expected REQUIRED_FIELD_MISSING for SubgraphID")
	}
}
