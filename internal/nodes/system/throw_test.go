package system

import (
	"context"
	"errors"
	"strings"
	"testing"

	"yotta/internal/node"
)

func TestThrow_ReturnsThrowErrorWithMessage(t *testing.T) {
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

func TestThrow_ErrorIsTypedThrowError(t *testing.T) {
	// errors.As 抽 typed ThrowError + 拿 Message 字段.
	node.ResetRegistryForTest()
	node.Register(&Throw{})
	rn, _ := node.Get("Throw")

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{thInMessage: "fish escaped"},
		nil, node.StubServices())

	var te *ThrowError
	if !errors.As(r.Error, &te) {
		t.Fatalf("error %v not *ThrowError", r.Error)
	}
	if te.Message != "fish escaped" {
		t.Errorf("Message = %q, want 'fish escaped'", te.Message)
	}
}

func TestThrow_ErrorIsBaseThrowSentinel(t *testing.T) {
	// errors.Is(err, errThrow) — 调用方仅判断"是否 throw" 时用, 不关心 message.
	node.ResetRegistryForTest()
	node.Register(&Throw{})
	rn, _ := node.Get("Throw")

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{thInMessage: "any"},
		nil, node.StubServices())

	if !errors.Is(r.Error, errThrow) {
		t.Errorf("errors.Is(%v, errThrow) = false, want true", r.Error)
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
	var te *ThrowError
	if !errors.As(r.Error, &te) {
		t.Errorf("empty-message error %v not *ThrowError", r.Error)
	}
}

func TestThrow_NoOutputs(t *testing.T) {
	th := Throw{}
	if len(th.Spec().Outputs) != 0 {
		t.Errorf("Throw.Spec.Outputs = %d entries, want 0 (terminal)", len(th.Spec().Outputs))
	}
}
