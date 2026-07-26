package locale

import "testing"

func TestValid(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"zh", true},
		{"en", true},
		{"ja", false},
		{"", false},
		{"ZH", false}, // 大小写敏感
	}
	for _, c := range cases {
		if got := Valid(c.in); got != c.want {
			t.Errorf("Valid(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestAll(t *testing.T) {
	got := All()
	if len(got) != 2 {
		t.Fatalf("All() len = %d, want 2", len(got))
	}
	if got[0] != Zh || got[1] != En {
		t.Errorf("All() = %v, want [zh, en]", got)
	}
}
