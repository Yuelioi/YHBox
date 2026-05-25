package system

import "testing"

func TestSubgraphInput_NoInputs(t *testing.T) {
	si := SubgraphInput{}
	if len(si.Spec().Inputs) != 0 {
		t.Errorf("SubgraphInput.Spec.Inputs = %d, want 0 (entry point)", len(si.Spec().Inputs))
	}
}

func TestSubgraphInput_IsGraphMarker(t *testing.T) {
	si := SubgraphInput{}
	if !si.Spec().IsGraphMarker {
		t.Errorf("SubgraphInput.Spec.IsGraphMarker = false, want true")
	}
}
