package expr

import (
	"sort"
	"strings"
	"testing"
)

// TestBuiltins_SetAndArity — 函数表是 validator/前端补全的单一来源, 集合+arity 锁死防漂移.
func TestBuiltins_SetAndArity(t *testing.T) {
	want := map[string][2]int{
		"abs":     {1, 1},
		"min":     {2, 2},
		"max":     {2, 2},
		"now":     {0, 0},
		"floor":   {1, 1},
		"ceil":    {1, 1},
		"sqrt":    {1, 1},
		"round":   {1, 2},
		"pow":     {2, 2},
		"clamp":   {3, 3},
		"rand":    {0, 0},
		"randint": {2, 2},
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

// TestFunctions_DTO — RPC 喂前端的元数据: 按名排序, 与 Builtins 同集合, Sig 以 name( 开头.
// 这张表是补全/校验的唯一权威 (FE 手写表已删), 别让 Sig 缺项.
func TestFunctions_DTO(t *testing.T) {
	fns := Functions()
	if len(fns) != len(builtins) {
		t.Fatalf("Functions() = %d entries, builtins has %d", len(fns), len(builtins))
	}
	if !sort.SliceIsSorted(fns, func(i, j int) bool { return fns[i].Name < fns[j].Name }) {
		t.Error("Functions() not sorted by name")
	}
	for _, f := range fns {
		b, ok := builtins[f.Name]
		if !ok {
			t.Errorf("Functions() has %q not in builtins", f.Name)
			continue
		}
		if f.MinArgs != b.MinArgs || f.MaxArgs != b.MaxArgs {
			t.Errorf("%s arity DTO [%d,%d] != builtin [%d,%d]", f.Name, f.MinArgs, f.MaxArgs, b.MinArgs, b.MaxArgs)
		}
		if !strings.HasPrefix(f.Sig, f.Name+"(") {
			t.Errorf("%s Sig = %q, want prefix %q", f.Name, f.Sig, f.Name+"(")
		}
	}
}

// TestRandBuiltins — rand ∈ [0,1); randint 闭区间 (多次采样须命中两端点); min>max 交换.
func TestRandBuiltins(t *testing.T) {
	randAst, err := Parse("rand()")
	if err != nil {
		t.Fatalf("Parse rand(): %v", err)
	}
	for range 100 {
		v, err := Eval(randAst, nil)
		if err != nil {
			t.Fatalf("rand(): %v", err)
		}
		x, ok := AsNumber(v)
		if !ok || x < 0 || x >= 1 {
			t.Fatalf("rand() = %v, want [0,1)", v)
		}
	}

	intAst, err := Parse("randint(1, 3)")
	if err != nil {
		t.Fatalf("Parse randint: %v", err)
	}
	hit := map[float64]bool{}
	for range 500 {
		v, err := Eval(intAst, nil)
		if err != nil {
			t.Fatalf("randint: %v", err)
		}
		x, ok := AsNumber(v)
		if !ok || x < 1 || x > 3 || x != float64(int(x)) {
			t.Fatalf("randint(1,3) = %v, want integer in [1,3]", v)
		}
		hit[x] = true
	}
	if !hit[1] || !hit[3] {
		t.Errorf("randint(1,3) 500 samples never hit an endpoint: %v (闭区间语义破了?)", hit)
	}

	// min > max 交换 (与 RandomInt 节点一致)
	swapAst, _ := Parse("randint(5, 3)")
	for range 50 {
		v, _ := Eval(swapAst, nil)
		x, _ := AsNumber(v)
		if x < 3 || x > 5 {
			t.Fatalf("randint(5,3) = %v, want [3,5] (swap)", v)
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
