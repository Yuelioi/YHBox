package control

import (
	"context"
	"errors"
	"testing"

	"github.com/yottaapp/yotta/internal/node"
)

func TestStop_ReturnsSentinel(t *testing.T) {
	registry := node.NewRegistry()
	registry.Register(&Stop{})
	rn, _ := registry.Get("Stop")

	r := node.RunNode(context.Background(), rn, nil, nil, nil, node.StubServices(), false)
	if !errors.Is(r.Error, errStopRun) {
		t.Fatalf("Error = %v, want errStopRun", r.Error)
	}
	if r.ExitName != "" {
		t.Errorf("ExitName = %q, want empty (terminal node)", r.ExitName)
	}
}
