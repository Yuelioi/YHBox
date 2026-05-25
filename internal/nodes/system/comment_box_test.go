package system

import (
	"testing"
)

func TestCommentBox_VisualOnlyFlag(t *testing.T) {
	cb := CommentBox{}
	if !cb.Spec().IsVisualOnly {
		t.Errorf("CommentBox.Spec.IsVisualOnly = false, want true")
	}
}

func TestCommentBox_NoOutputs(t *testing.T) {
	// render-only 节点不应有 exec-out.
	cb := CommentBox{}
	if len(cb.Spec().Outputs) != 0 {
		t.Errorf("CommentBox.Spec.Outputs = %d entries, want 0 (render-only)", len(cb.Spec().Outputs))
	}
}
