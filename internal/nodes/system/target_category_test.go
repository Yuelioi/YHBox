package system

import "testing"

func TestTargetSelectionNodesUseTargetCategory(t *testing.T) {
	tests := []struct {
		name string
		got  string
	}{
		{name: "WindowTarget", got: (WindowTarget{}).Spec().Category},
		{name: "AndroidTarget", got: (AndroidTarget{}).Spec().Category},
		{name: "BrowserTarget", got: (BrowserTarget{}).Spec().Category},
	}

	for _, tt := range tests {
		if tt.got != "Target" {
			t.Fatalf("%s category = %q, want Target", tt.name, tt.got)
		}
	}
}
