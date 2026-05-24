package system

import (
	"context"
	"errors"
	"testing"

	"yhbox/internal/node"
)

func TestCommentBox_VisualOnlyFlag(t *testing.T) {
	cb := CommentBox{}
	if !cb.Spec().IsVisualOnly {
		t.Errorf("CommentBox.Spec.IsVisualOnly = false, want true")
	}
}

func TestCommentBox_RunReturnsSentinel(t *testing.T) {
	// 防御性 — framework gatekeep 应阻止 Run, 但万一调进来必须返 sentinel.
	node.ResetRegistryForTest()
	node.Register(&CommentBox{})
	rn, _ := node.Get("CommentBox")

	r := node.RunNode(context.Background(), rn, nil, nil, nil, node.StubServices())
	if !errors.Is(r.Error, errVisualOnlyNotRunnable) {
		t.Errorf("error = %v, want errVisualOnlyNotRunnable", r.Error)
	}
}

func TestCommentBox_NoOutputs(t *testing.T) {
	// render-only 节点不应有 exec-out.
	cb := CommentBox{}
	if len(cb.Spec().Outputs) != 0 {
		t.Errorf("CommentBox.Spec.Outputs = %d entries, want 0 (render-only)", len(cb.Spec().Outputs))
	}
}
