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
			map[string]any{"value": "$var.state", "cases": []any{"IDLE", "WAITING"}},
			"$var.state", []string{"IDLE", "WAITING"},
		},
		{"nil config", nil, "", nil},
		{"missing fields", map[string]any{}, "", nil},
		{
			"cases 含非字符串元素 (silently skip)",
			map[string]any{"value": "x", "cases": []any{"A", 42, "B"}},
			"x", []string{"A", "B"},
		},
		{
			"CJK + emoji",
			map[string]any{"value": "x", "cases": []any{"钓鱼", "🎣"}},
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

func TestParseParallelConfig(t *testing.T) {
	cases := []struct {
		name string
		cfg  map[string]any
		want int
	}{
		{"explicit n", map[string]any{"n": float64(5)}, 5},
		{"default n=2", map[string]any{}, 2},
		{"nil config default", nil, 2},
		{"n=0 forced to default", map[string]any{"n": float64(0)}, 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			n := &GraphNode{Kind: "Parallel", Config: c.cfg}
			cfg, _ := ParseParallelConfig(n)
			if cfg.N != c.want {
				t.Errorf("N = %d, want %d", cfg.N, c.want)
			}
		})
	}
}
