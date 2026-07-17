package controller

import "testing"

func TestValidKeyNameIsPortableAndExact(t *testing.T) {
	for _, value := range []string{"A", "0", "Ctrl", "F12", ","} {
		if !ValidKeyName(value) {
			t.Errorf("ValidKeyName(%q) = false", value)
		}
	}
	for _, value := range []string{"", " Ctrl", "F13", "mouse-left", "你"} {
		if ValidKeyName(value) {
			t.Errorf("ValidKeyName(%q) = true", value)
		}
	}
}
