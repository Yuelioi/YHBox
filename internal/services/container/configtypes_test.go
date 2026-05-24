package container

import (
	"reflect"
	"testing"
)

func TestParseSwitchConfig(t *testing.T) {
	cases := []struct {
		name     string
		cfg      map[string]any
		wantVal  string
		wantCase []string
	}{
		{
			"happy path",
			map[string]any{"Value": "$vars.state", "Case1Value": "IDLE", "Case2Value": "WAITING"},
			"$vars.state", []string{"IDLE", "WAITING"},
		},
		{"nil config", nil, "", nil},
		{"missing fields", map[string]any{}, "", nil},
		{
			"CaseN gaps silently skipped",
			map[string]any{"Value": "x", "Case1Value": "A", "Case3Value": "B"},
			"x", []string{"A", "B"},
		},
		{
			"CJK + emoji",
			map[string]any{"Value": "x", "Case1Value": "钓鱼", "Case2Value": "🎣"},
			"x", []string{"钓鱼", "🎣"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			n := &GraphNode{Kind: "Switch", Config: c.cfg}
			cfg, err := ParseSwitchConfig(n)
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			if cfg.Value != c.wantVal {
				t.Errorf("Value = %q, want %q", cfg.Value, c.wantVal)
			}
			if !reflect.DeepEqual(cfg.Cases, c.wantCase) {
				t.Errorf("Cases = %v, want %v", cfg.Cases, c.wantCase)
			}
		})
	}
}
