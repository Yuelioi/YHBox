package control

import (
	"context"
	"testing"

	"yotta/internal/node"
)

func TestIf_TrueBranch(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&If{})
	rn, _ := node.Get("If")

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{ifInCond: true},
		nil, node.StubServices())
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != ifOutTrue {
		t.Errorf("exit = %q, want %q", r.ExitName, ifOutTrue)
	}
}

func TestIf_FalseBranch(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&If{})
	rn, _ := node.Get("If")

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{ifInCond: false},
		nil, node.StubServices())
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != ifOutFalse {
		t.Errorf("exit = %q, want %q", r.ExitName, ifOutFalse)
	}
}

func TestIf_DefaultIsTrue(t *testing.T) {
	// Spec Default = true → 没传 Condition 应走 True.
	node.ResetRegistryForTest()
	node.Register(&If{})
	rn, _ := node.Get("If")

	r := node.RunNode(context.Background(), rn, nil, nil, nil, node.StubServices())
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != ifOutTrue {
		t.Errorf("exit = %q, want %q (default Condition=true)", r.ExitName, ifOutTrue)
	}
}
