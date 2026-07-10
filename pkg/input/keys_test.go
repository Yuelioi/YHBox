package input

import "testing"

func TestVK(t *testing.T) {
	tests := map[string]uint32{
		"escape":  0x1B,
		" CTRL ":  0x11,
		"a":       0x41,
		"7":       0x37,
		"F1":      0x70,
		"f12":     0x7B,
		"f13":     0,
		"missing": 0,
	}
	for name, want := range tests {
		if got := VK(name); got != want {
			t.Errorf("VK(%q) = %#x, want %#x", name, got, want)
		}
	}
}
