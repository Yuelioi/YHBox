package control

import (
	"context"
	"testing"

	"yotta/internal/node"
)

func TestStart_HappyPath(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Start{})
	rn, _ := node.Get("Start")

	r := node.RunNode(context.Background(), rn, nil, nil, nil, node.StubServices())
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != startOutOut {
		t.Errorf("exit = %q, want %q", r.ExitName, startOutOut)
	}
}
