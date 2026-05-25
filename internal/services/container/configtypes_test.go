package container

import (
	"reflect"
	"testing"
)

func TestParseSwitchConfig(t *testing.T) {
	cases := []struct {
		name     string
		cfg      map[string]any
		wantCase []string
	}{
		{
			"happy path",
			map[string]any{"Case1Value": "IDLE", "Case2Value": "WAITING"},
			[]string{"IDLE", "WAITING"},
		},
		{"nil config", nil, nil},
		{"missing fields", map[string]any{}, nil},
		{
			"CaseN gaps silently skipped",
			map[string]any{"Case1Value": "A", "Case3Value": "B"},
			[]string{"A", "B"},
		},
		{
			"CJK + emoji",
			map[string]any{"Case1Value": "钓鱼", "Case2Value": "🎣"},
			[]string{"钓鱼", "🎣"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			n := &GraphNode{Kind: "Switch", Config: c.cfg}
			cfg, err := ParseSwitchConfig(n)
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			if !reflect.DeepEqual(cfg.Cases, c.wantCase) {
				t.Errorf("Cases = %v, want %v", cfg.Cases, c.wantCase)
			}
		})
	}
}
