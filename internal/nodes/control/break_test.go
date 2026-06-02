package control

import (
	"context"
	"errors"
	"testing"

	"yotta/internal/node"
)

func TestBreak_ReturnsSentinel(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Break{})
	rn, _ := node.Get("Break")
	r := node.RunNode(context.Background(), rn, nil, nil, nil, node.StubServices())
	if !errors.Is(r.Error, errBreakRequested) {
		t.Errorf("error = %v, want errBreakRequested", r.Error)
	}
}
