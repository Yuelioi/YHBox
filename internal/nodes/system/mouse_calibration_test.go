package system

import (
	"context"
	"testing"

	"github.com/yottaapp/yotta/internal/node"
)

func TestMouseCalibration_Passthrough(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&MouseCalibration{})
	rn, _ := node.Get("MouseCalibration")

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{mcInCounts360: 800.0},
		nil, node.StubServices(), false)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != mcOutDone {
		t.Errorf("exit = %q, want %q", r.ExitName, mcOutDone)
	}
}

func TestMouseCalibration_DefaultCounts(t *testing.T) {
	// counts360 Default = 0, Optional — 没传也应走 Fire (declarative no-op).
	node.ResetRegistryForTest()
	node.Register(&MouseCalibration{})
	rn, _ := node.Get("MouseCalibration")

	r := node.RunNode(context.Background(), rn, nil, nil, nil, node.StubServices(), false)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != mcOutDone {
		t.Errorf("exit = %q, want %q", r.ExitName, mcOutDone)
	}
}
