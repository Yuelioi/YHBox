package control

import (
	"context"
	"errors"
	"testing"

	"github.com/yottaapp/yotta/internal/node"
)

func TestContinue_ReturnsSentinel(t *testing.T) {
	registry := node.NewRegistry()
	registry.Register(&Continue{})
	rn, _ := registry.Get("Continue")
	r := node.RunNode(context.Background(), rn, nil, nil, nil, node.StubServices(), false)
	if !errors.Is(r.Error, errContinueRequested) {
		t.Errorf("error = %v, want errContinueRequested", r.Error)
	}
}
