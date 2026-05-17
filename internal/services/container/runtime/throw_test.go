package runtime

import (
	"context"
	"errors"
	"testing"

	"yhbox/internal/services/container"
)

// Throw 节点在 execNode 直接 return errThrow sentinel, 不碰 edges/nodesByID — 所以
// 零值 ContainerRunner 足够测试 switch 分支挂上 + 出错 payload.

func TestThrowReturnsErrThrow(t *testing.T) {
	r := &ContainerRunner{}
	node := &container.GraphNode{
		ID:     "th1",
		Kind:   "Throw",
		Config: map[string]any{"message": "no_currency"},
	}
	_, err := r.execNode(context.Background(), node, ExecToken{InPin: "in"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var te *errThrow
	if !errors.As(err, &te) {
		t.Fatalf("expected errThrow, got %T: %v", err, err)
	}
	if te.message != "no_currency" {
		t.Fatalf("expected message no_currency, got %q", te.message)
	}
}

func TestThrowEmptyMessageStillErrThrow(t *testing.T) {
	r := &ContainerRunner{}
	node := &container.GraphNode{
		ID:     "th2",
		Kind:   "Throw",
		Config: map[string]any{},
	}
	_, err := r.execNode(context.Background(), node, ExecToken{InPin: "in"})
	if err == nil {
		t.Fatal("expected error even with empty message")
	}
	var te *errThrow
	if !errors.As(err, &te) {
		t.Fatalf("expected errThrow even on empty message, got %T", err)
	}
}
