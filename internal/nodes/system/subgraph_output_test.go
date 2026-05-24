package system

import (
	"context"
	"errors"
	"testing"

	"yhbox/internal/node"
)

func TestSubgraphOutput_RunReturnsSentinel(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&SubgraphOutput{})
	rn, _ := node.Get("SubgraphOutput")

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{soInDeclID: "done"},
		nil, node.StubServices())
	if !errors.Is(r.Error, errSubgraphNodeStub) {
		t.Errorf("error = %v, want errSubgraphNodeStub", r.Error)
	}
}

func TestSubgraphOutput_NoOutputs(t *testing.T) {
	so := SubgraphOutput{}
	if len(so.Spec().Outputs) != 0 {
		t.Errorf("SubgraphOutput.Spec.Outputs = %d, want 0 (terminal)", len(so.Spec().Outputs))
	}
}
