package control

import (
	"context"
	"testing"

	"github.com/yottaapp/yotta/internal/node"
)

func TestStart_HappyPath(t *testing.T) {
	registry := node.NewRegistry()
	registry.Register(&Start{})
	rn, _ := registry.Get("Start")

	r := node.RunNode(context.Background(), rn, nil, nil, nil, node.StubServices(), false)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != startOutOut {
		t.Errorf("exit = %q, want %q", r.ExitName, startOutOut)
	}
}
