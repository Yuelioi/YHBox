package system

import "testing"

func TestTargetSelectionNodesUseTargetCategory(t *testing.T) {
	tests := []struct {
		name string
		got  string
	}{
		{name: "Win32WindowTarget", got: (Win32WindowTarget{}).Spec().Category},
		{name: "AndroidTarget", got: (AndroidTarget{}).Spec().Category},
	}

	for _, tt := range tests {
		if tt.got != "Target" {
			t.Fatalf("%s category = %q, want Target", tt.name, tt.got)
		}
	}
}
