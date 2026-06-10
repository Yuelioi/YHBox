package expr

import "testing"

// TestBuiltins_SetAndArity — 函数表是 validator/前端补全的单一来源, 集合+arity 锁死防漂移.
func TestBuiltins_SetAndArity(t *testing.T) {
	want := map[string][2]int{
		"abs":   {1, 1},
		"min":   {2, 2},
		"max":   {2, 2},
		"now":   {0, 0},
		"floor": {1, 1},
		"ceil":  {1, 1},
		"sqrt":  {1, 1},
		"round": {1, 2},
		"pow":   {2, 2},
		"clamp": {3, 3},
	}
	got := Builtins()
	if len(got) != len(want) {
		t.Fatalf("Builtins() has %d entries, want %d", len(got), len(want))
	}
	for name, ar := range want {
		b, ok := got[name]
		if !ok {
			t.Errorf("Builtins() missing %q", name)
			continue
		}
		if b.MinArgs != ar[0] || b.MaxArgs != ar[1] {
			t.Errorf("%s arity = [%d,%d], want [%d,%d]", name, b.MinArgs, b.MaxArgs, ar[0], ar[1])
		}
	}
}

// TestEvalCall_ArityErrors — 集中 arity gate 后错误仍带函数名 + col.
func TestEvalCall_ArityErrors(t *testing.T) {
	cases := []string{"abs()", "abs(1, 2)", "now(1)", "round()", "round(1, 2, 3)", "clamp(1)"}
	for _, src := range cases {
		ast, err := Parse(src)
		if err != nil {
			t.Fatalf("Parse(%q): %v", src, err)
		}
		if _, err := Eval(ast, nil); err == nil {
			t.Errorf("Eval(%q): want arity error, got nil", src)
		}
	}
}

func TestCallRefs(t *testing.T) {
	ast, err := Parse(`clamp(abs(x), 0, max(1, 2)) + foo("min(a)")`)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	refs := CallRefs(ast)
	got := map[string]int{}
	for _, r := range refs {
		got[r.Name] = r.ArgN
	}
	want := map[string]int{"clamp": 3, "abs": 1, "max": 2, "foo": 1}
	if len(got) != len(want) {
		t.Fatalf("CallRefs = %v, want names %v", got, want)
	}
	for name, argN := range want {
		if got[name] != argN {
			t.Errorf("CallRefs[%s].ArgN = %d, want %d", name, got[name], argN)
		}
	}
	// 字符串字面量里的 "min(a)" 是 foo 的参数字面量, 不该被当成 call
	if _, ok := got["min"]; ok {
		t.Error(`CallRefs picked up "min" from inside a string literal`)
	}
}
