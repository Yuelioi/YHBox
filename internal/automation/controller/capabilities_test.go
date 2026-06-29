package controller

import "testing"

func TestCapabilitySetHas(t *testing.T) {
	caps := CapabilitySet{
		Screenshot:   true,
		Click:        true,
		Drag:         true,
		MoveRelative: false,
		KeyChord:     false,
	}
	if !caps.Has(CapabilityScreenshot) {
		t.Fatalf("expected screenshot capability")
	}
	if !caps.Has(CapabilityDrag) {
		t.Fatalf("expected drag capability")
	}
	if caps.Has(CapabilityMoveRelative) {
		t.Fatalf("did not expect relative move capability")
	}
	if caps.Has(CapabilityKeyChord) {
		t.Fatalf("did not expect key chord capability")
	}
}
