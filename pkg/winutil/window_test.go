package winutil

import "testing"

func TestIsEmptyMatch(t *testing.T) {
	cases := []struct {
		name string
		spec MatchSpec
		want bool
	}{
		{"all empty", MatchSpec{}, true},
		{"title only", MatchSpec{Title: "异环"}, false},
		{"class only", MatchSpec{Class: "Unreal"}, false},
		{"process only", MatchSpec{ProcessName: "game.exe"}, false},
		{"regex dot star alone", MatchSpec{Title: ".*", TitleMatch: "regex"}, true},
		{"regex dot plus alone", MatchSpec{Title: ".+", TitleMatch: "regex"}, true},
		{"regex anchored dot star alone", MatchSpec{Title: "^.*$", TitleMatch: "regex"}, true},
		{"regex .* with class", MatchSpec{Title: ".*", TitleMatch: "regex", Class: "Unreal"}, false},
		{"regex specific", MatchSpec{Title: "异环", TitleMatch: "regex"}, false},
		{"exact .*", MatchSpec{Title: ".*", TitleMatch: "exact"}, false}, // exact 模式下 .* 字面意义不是万能
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := IsEmptyMatch(c.spec); got != c.want {
				t.Errorf("IsEmptyMatch(%+v) = %v, want %v", c.spec, got, c.want)
			}
		})
	}
}

func TestCompileTitle(t *testing.T) {
	if _, err := CompileTitle(MatchSpec{Title: "异环", TitleMatch: "exact"}); err != nil {
		t.Errorf("exact mode should not compile: %v", err)
	}
	if r, err := CompileTitle(MatchSpec{Title: "异环.*", TitleMatch: "regex"}); err != nil || r == nil {
		t.Errorf("valid regex should compile, got err=%v r=%v", err, r)
	}
	if _, err := CompileTitle(MatchSpec{Title: "[invalid", TitleMatch: "regex"}); err == nil {
		t.Errorf("invalid regex should fail")
	}
}
