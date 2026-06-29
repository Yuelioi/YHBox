package system

import "testing"

func TestCommentBox_VisualOnlyFlag(t *testing.T) {
	if !(CommentBox{}).Spec().IsVisualOnly {
		t.Errorf("CommentBox.Spec.IsVisualOnly = false, want true")
	}
}

func TestCommentBox_NoOutputs(t *testing.T) {
	if n := len((CommentBox{}).Spec().Outputs); n != 0 {
		t.Errorf("CommentBox.Spec.Outputs = %d entries, want 0 (render-only)", n)
	}
}

func TestCommentBox_InputsAreTitleContentColor(t *testing.T) {
	got := map[string]bool{}
	for _, in := range (CommentBox{}).Spec().Inputs {
		got[in.Name] = true
	}
	for _, want := range []string{"Title", "Content", "Color", "Icon", "Width"} {
		if !got[want] {
			t.Errorf("missing input %q; got %+v", want, got)
		}
	}
}
