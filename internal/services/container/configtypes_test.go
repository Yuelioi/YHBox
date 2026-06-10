package container

import (
	"reflect"
	"testing"
)

func TestParseDynamicInputDecls_AnyKind(t *testing.T) {
	n := &GraphNode{Kind: "Script", Config: map[string]any{
		"Inputs": []any{map[string]any{"Name": "hp", "Type": "number"}},
	}}
	decls := ParseDynamicInputDecls(n)
	if len(decls) != 1 || decls[0].Name != "hp" || decls[0].Type != "number" {
		t.Fatalf("got %+v", decls)
	}
}

func TestParseDynamicInputDecls_NilSafe(t *testing.T) {
	if got := ParseDynamicInputDecls(nil); got != nil {
		t.Fatalf("nil node: got %+v", got)
	}
	if got := ParseDynamicInputDecls(&GraphNode{Kind: "Expr"}); got != nil {
		t.Fatalf("nil config: got %+v", got)
	}
	// 空 Name 项跳过; 非 map 项跳过.
	n := &GraphNode{Kind: "Expr", Config: map[string]any{
		"Inputs": []any{map[string]any{"Name": "", "Type": "number"}, "junk", map[string]any{"Name": "i"}},
	}}
	decls := ParseDynamicInputDecls(n)
	if len(decls) != 1 || decls[0].Name != "i" || decls[0].Type != "" {
		t.Fatalf("got %+v", decls)
	}
}

func TestParseSwitchConfig(t *testing.T) {
	cases := []struct {
		name     string
		cfg      map[string]any
		wantCase []string
	}{
		{
			"happy path",
			map[string]any{"cases": []any{"IDLE", "WAITING"}},
			[]string{"IDLE", "WAITING"},
		},
		{"nil config", nil, nil},
		{"missing cases", map[string]any{}, nil},
		{"cases not array", map[string]any{"cases": "nope"}, nil},
		{
			// 原样返 (含空/重复) — 留给 validateSwitchConfig 报错, 不在此过滤.
			"raw passthrough incl empty/dup",
			map[string]any{"cases": []any{"A", "", "A"}},
			[]string{"A", "", "A"},
		},
		{
			"non-string items skipped",
			map[string]any{"cases": []any{"A", 5, "B"}},
			[]string{"A", "B"},
		},
		{
			"CJK + emoji",
			map[string]any{"cases": []any{"钓鱼", "🎣"}},
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
