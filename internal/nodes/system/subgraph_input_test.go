package system

import (
	"context"
	"errors"
	"testing"

	"yhbox/internal/node"
)

func TestSubgraphInput_RunReturnsSentinel(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&SubgraphInput{})
	rn, _ := node.Get("SubgraphInput")

	r := node.RunNode(context.Background(), rn, nil, nil, nil, node.StubServices())
	if !errors.Is(r.Error, errSubgraphNodeStub) {
		t.Errorf("error = %v, want errSubgraphNodeStub", r.Error)
	}
}

func TestSubgraphInput_NoInputs(t *testing.T) {
	si := SubgraphInput{}
	if len(si.Spec().Inputs) != 0 {
		t.Errorf("SubgraphInput.Spec.Inputs = %d, want 0 (entry point)", len(si.Spec().Inputs))
	}
}
