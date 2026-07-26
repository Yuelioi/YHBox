package hotkey

import "testing"

func TestNormalizeHotkey_ModifierOrder(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"Shift+Ctrl+R", "ctrl+shift+r"},
		{"Alt+Ctrl+F1", "ctrl+alt+f1"},
		{"Alt+Shift+Ctrl+A", "ctrl+shift+alt+a"},
		{"Ctrl+Shift+R", "ctrl+shift+r"},
		{"CTRL+SHIFT+R", "ctrl+shift+r"},
		{"Control+Shift+R", "ctrl+shift+r"},
		{"F12", "f12"},
		{"Escape", "esc"},
		{"return", "enter"},
	}
	for _, c := range cases {
		got, err := NormalizeHotkey(c.in)
		if err != nil {
			t.Errorf("NormalizeHotkey(%q) err: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("NormalizeHotkey(%q): got %q want %q", c.in, got, c.want)
		}
	}
}

func TestNormalizeHotkey_Invalid(t *testing.T) {
	cases := []string{"", "  ", "Ctrl+", "Ctrl", "+R", "Bad"}
	for _, c := range cases {
		if _, err := NormalizeHotkey(c); err == nil {
			t.Errorf("NormalizeHotkey(%q) 应返 error", c)
		}
	}
}
