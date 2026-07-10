package platform

import (
	"errors"
	"testing"
)

func TestUnsupportedErrorSupportsClassification(t *testing.T) {
	err := NewUnsupportedError("window capture")
	if !errors.Is(err, ErrUnsupported) {
		t.Fatal("unsupported error must match ErrUnsupported")
	}
	var unsupported *UnsupportedError
	if !errors.As(err, &unsupported) {
		t.Fatal("unsupported error must preserve typed details")
	}
	if unsupported.Capability != "window capture" {
		t.Fatalf("capability = %q, want window capture", unsupported.Capability)
	}
	if unsupported.OS == "" {
		t.Fatal("OS must be populated")
	}
}
