package system

import (
	"context"
	"strings"
	"testing"

	"yhbox/internal/node"
)

func TestThrow_ReturnsErrorWithMessage(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Throw{})
	rn, _ := node.Get("Throw")

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{thInMessage: "boom"},
		nil, node.StubServices())
	if r.Error == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(r.Error.Error(), "throw: boom") {
		t.Errorf("error = %q, want substring 'throw: boom'", r.Error.Error())
	}
}

func TestThrow_EmptyMessage(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Throw{})
	rn, _ := node.Get("Throw")

	r := node.RunNode(context.Background(), rn, nil, nil, nil, node.StubServices())
	if r.Error == nil {
		t.Fatal("expected error even with empty message")
	}
	if !strings.HasPrefix(r.Error.Error(), "throw: ") {
		t.Errorf("error = %q, want prefix 'throw: '", r.Error.Error())
	}
}

func TestThrow_NoOutputs(t *testing.T) {
	th := Throw{}
	if len(th.Spec().Outputs) != 0 {
		t.Errorf("Throw.Spec.Outputs = %d entries, want 0 (terminal)", len(th.Spec().Outputs))
	}
}
