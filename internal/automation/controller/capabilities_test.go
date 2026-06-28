package controller

import "testing"

func TestCapabilitySetHas(t *testing.T) {
	caps := CapabilitySet{
		Screenshot: true,
		Click:      true,
		KeyChord:   false,
	}
	if !caps.Has(CapabilityScreenshot) {
		t.Fatalf("expected screenshot capability")
	}
	if caps.Has(CapabilityKeyChord) {
		t.Fatalf("did not expect key chord capability")
	}
}
