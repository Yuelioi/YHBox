package expr

import "testing"

func TestAsString_Cases(t *testing.T) {
	cases := []struct {
		name string
		in   Value
		want string
	}{
		{"string passthrough", "IDLE", "IDLE"},
		{"int", 42, "42"},
		{"int64", int64(42), "42"},
		{"float", 3.14, "3.14"},
		{"float int-like (no trailing .0)", 5.0, "5"},
		{"bool true", true, "true"},
		{"bool false", false, "false"},
		{"nil", nil, ""},
		{"empty string", "", ""},
		{"CJK", "钓鱼", "钓鱼"},
		{"emoji", "🎣", "🎣"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := AsString(c.in); got != c.want {
				t.Errorf("AsString(%v) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}
