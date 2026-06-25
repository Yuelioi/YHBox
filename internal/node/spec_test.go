package node

import "testing"

func TestWindowInputSpec(t *testing.T) {
	got := WindowInputSpec()
	if got.Name != "Window" || got.Type != "Window" {
		t.Fatalf("WindowInputSpec = %+v", got)
	}
}
