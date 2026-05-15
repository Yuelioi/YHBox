package hotkey

import "testing"

func TestParseHotkey_ValidCombinations(t *testing.T) {
	cases := []struct {
		in       string
		wantMods uint32
		wantVK   uint32
	}{
		{"Ctrl+Shift+F", hkModCtrl | hkModShift, 'F'},
		{"ctrl+shift+f", hkModCtrl | hkModShift, 'F'},
		{"Ctrl+1", hkModCtrl, '1'},
		{"F12", 0, 0x7B},
		{"F1", 0, 0x70},
		{"Ctrl+Alt+Delete", hkModCtrl | hkModAlt, 0x2E},
		{"Shift+Space", hkModShift, 0x20},
		{"Alt+Up", hkModAlt, 0x26},
		{"Ctrl+Shift+Alt+A", hkModCtrl | hkModShift | hkModAlt, 'A'},
	}
	for _, c := range cases {
		mods, vk, err := parseHotkey(c.in)
		if err != nil {
			t.Errorf("parseHotkey(%q) err: %v", c.in, err)
			continue
		}
		if mods != c.wantMods || vk != c.wantVK {
			t.Errorf("parseHotkey(%q): got mods=0x%x vk=0x%x, want mods=0x%x vk=0x%x",
				c.in, mods, vk, c.wantMods, c.wantVK)
		}
	}
}

func TestParseHotkey_Invalid(t *testing.T) {
	cases := []string{
		"",
		"   ",
		"X+Ctrl",
		"Ctrl+",
		"Ctrl",
		"Ctrl+Bad",
		"+Ctrl+F",
		"Ctrl+F+G",
	}
	for _, in := range cases {
		if _, _, err := parseHotkey(in); err == nil {
			t.Errorf("parseHotkey(%q) 应返回 error", in)
		}
	}
}

func TestIsReservedHotkey_SystemReserved(t *testing.T) {
	if hit, _ := IsReservedHotkey("Ctrl+Shift+R"); !hit {
		t.Error("Ctrl+Shift+R 应 reserved")
	}
	if hit, _ := IsReservedHotkey("ctrl+c"); !hit {
		t.Error("Ctrl+C 应 reserved")
	}
	if hit, _ := IsReservedHotkey("Alt+F4"); !hit {
		t.Error("Alt+F4 应 reserved")
	}
}

func TestIsReservedHotkey_BareSingleChar(t *testing.T) {
	if hit, _ := IsReservedHotkey("A"); !hit {
		t.Error("纯字母键应 reserved")
	}
	if hit, _ := IsReservedHotkey("5"); !hit {
		t.Error("纯数字键应 reserved")
	}
}

func TestIsReservedHotkey_AllowedKeys(t *testing.T) {
	cases := []string{"F1", "F12", "Ctrl+F1", "Shift+Alt+R", "Ctrl+Shift+5"}
	for _, c := range cases {
		if hit, reason := IsReservedHotkey(c); hit {
			t.Errorf("%q 不该 reserved，得到 reason=%q", c, reason)
		}
	}
}
