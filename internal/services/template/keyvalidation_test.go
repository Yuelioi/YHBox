package template

import "testing"

func TestValidateKey(t *testing.T) {
	cases := []struct {
		key   string
		valid bool
	}{
		{"fishing.hook_icon", true},
		{"fishing.ui.close", true},
		{"vendor.fishing.hook_icon", true},
		{"hook_icon", false},  // no dot
		{"fishing.", false},   // empty base
		{".hook_icon", false}, // empty ns
		{"Fishing.Hook", false}, // uppercase
		{"fish-ing.hook", false}, // dash
		{"", false},
		{"a.b.c.d.e.f.g.h.i.j.k.l.m.n.o.p.q.r.s.t.u.v.w.x.y.z.aa.bb.cc.dd.ee.ff.gg.hh.ii.jj", false}, // > 64 char
	}
	for _, c := range cases {
		got := ValidateKey(c.key) == nil
		if got != c.valid {
			t.Errorf("ValidateKey(%q) = %v, want %v", c.key, got, c.valid)
		}
	}
}
