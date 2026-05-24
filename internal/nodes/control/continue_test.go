package control

import (
	"context"
	"errors"
	"testing"

	"yhbox/internal/node"
)

func TestContinue_ReturnsSentinel(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Continue{})
	rn, _ := node.Get("Continue")
	r := node.RunNode(context.Background(), rn, nil, nil, nil, node.StubServices())
	if !errors.Is(r.Error, errContinueRequested) {
		t.Errorf("error = %v, want errContinueRequested", r.Error)
	}
}
