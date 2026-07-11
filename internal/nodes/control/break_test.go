package control

import (
	"context"
	"errors"
	"testing"

	"github.com/yottaapp/yotta/internal/node"
)

func TestBreak_ReturnsSentinel(t *testing.T) {
	registry := node.NewRegistry()
	registry.Register(&Break{})
	rn, _ := registry.Get("Break")
	r := node.RunNode(context.Background(), rn, nil, nil, nil, node.StubServices(), false)
	if !errors.Is(r.Error, errBreakRequested) {
		t.Errorf("error = %v, want errBreakRequested", r.Error)
	}
}
