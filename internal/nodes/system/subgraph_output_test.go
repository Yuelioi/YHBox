package system

import "testing"

func TestSubgraphOutput_NoOutputs(t *testing.T) {
	so := SubgraphOutput{}
	if len(so.Spec().Outputs) != 0 {
		t.Errorf("SubgraphOutput.Spec.Outputs = %d, want 0 (terminal)", len(so.Spec().Outputs))
	}
}

func TestSubgraphOutput_IsGraphMarker(t *testing.T) {
	so := SubgraphOutput{}
	if !so.Spec().IsGraphMarker {
		t.Errorf("SubgraphOutput.Spec.IsGraphMarker = false, want true")
	}
}
