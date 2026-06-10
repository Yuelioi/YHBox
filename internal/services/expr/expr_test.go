package expr

import (
	"math"
	"strings"
	"testing"
)

func eval(t *testing.T, src string, env Env) Value {
	t.Helper()
	n, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse(%q): %v", src, err)
	}
	v, err := Eval(n, env)
	if err != nil {
		t.Fatalf("Eval(%q): %v", src, err)
	}
	return v
}

func TestLiterals(t *testing.T) {
	env := MapEnv{}
	if got := eval(t, "42", env); got != 42.0 {
		t.Errorf("42 → %v want 42", got)
	}
	if got := eval(t, "3.14", env); got != 3.14 {
		t.Errorf("3.14 → %v want 3.14", got)
	}
	if got := eval(t, `"hi"`, env); got != "hi" {
		t.Errorf("string → %v", got)
	}
	if got := eval(t, "true", env); got != true {
		t.Errorf("true → %v", got)
	}
	if got := eval(t, "null", env); got != nil {
		t.Errorf("null → %v", got)
	}
}

// $名字 变量引用 (2026-06-11 恢复): MapEnv 的 $-键通道现成。
func TestVarRef(t *testing.T) {
	env := MapEnv{"$hp": 37.0, "$max": 50.0, "$名字": "鱼"}
	if got := eval(t, "$hp / $max * 100", env); got != 74.0 {
		t.Errorf("$hp/$max*100 → %v want 74", got)
	}
	if got := eval(t, `$名字 + "!"`, env); got != "鱼!" {
		t.Errorf("unicode var → %v", got)
	}
	// 未知变量 → env miss → null (编辑期 validator 报 EXPR_UNKNOWN_VAR)
	if got := eval(t, "$ghost", env); got != nil {
		t.Errorf("$ghost → %v want nil", got)
	}
	// bare 与 $ 命名空间不撞: 同名 input 与变量互不干扰
	env2 := struct{ MapEnv }{MapEnv{"hp": 1.0, "$hp": 2.0}}
	if got := eval(t, "hp + $hp", env2); got != 3.0 {
		t.Errorf("namespace split → %v want 3", got)
	}
	// v3 点路径不回归: $vars.hp 在 '.' 处报错
	if _, err := Parse("$vars.hp"); err == nil {
		t.Error("$vars.hp should be a parse error")
	}
	// $ 后非标识符 → 报错
	if _, err := Parse("$ + 1"); err == nil {
		t.Error("bare $ should be a lex error")
	}
	// VarRefs 收集
	n, _ := Parse("$a + b + $c")
	refs := VarRefs(n)
	if len(refs) != 2 || refs[0] != "a" || refs[1] != "c" {
		t.Errorf("VarRefs → %v", refs)
	}
}

func TestArith(t *testing.T) {
	env := MapEnv{}
	cases := map[string]float64{
		"1 + 2":         3,
		"5 - 3":         2,
		"4 * 6":         24,
		"10 / 4":        2.5,
		"10 % 3":        1,
		"-5":            -5,
		"1 + 2 * 3":     7,
		"(1 + 2) * 3":   9,
		"2 + 3 * 4 - 1": 13,
	}
	for src, want := range cases {
		got, _ := AsNumber(eval(t, src, env))
		if got != want {
			t.Errorf("%s → %v want %v", src, got, want)
		}
	}
}

func TestStringConcat(t *testing.T) {
	env := MapEnv{}
	if got := eval(t, `"a" + "b"`, env); got != "ab" {
		t.Errorf("concat: %v", got)
	}
	if got := eval(t, `"v=" + 42`, env); got != "v=42" {
		t.Errorf("mixed concat: %v", got)
	}
}

func TestCompareEq(t *testing.T) {
	env := MapEnv{}
	cases := map[string]bool{
		"1 < 2":      true,
		"2 < 2":      false,
		"2 <= 2":     true,
		"3 > 2":      true,
		"3 >= 3":     true,
		"1 == 1":     true,
		`"x" == "x"`: true,
		`"x" != "y"`: true,
		"null == null": true,
		"null == 0":  false,
	}
	for src, want := range cases {
		if got := eval(t, src, env); got != want {
			t.Errorf("%s → %v want %v", src, got, want)
		}
	}
}

func TestBoolShortCircuit(t *testing.T) {
	env := MapEnv{}
	if got := eval(t, "true && false", env); got != false {
		t.Errorf("and: %v", got)
	}
	if got := eval(t, "true || false", env); got != true {
		t.Errorf("or: %v", got)
	}
	if got := eval(t, "!true", env); got != false {
		t.Errorf("not: %v", got)
	}
	// short-circuit：右侧不会触发 1/0 错误
	if got := eval(t, "false && (1/0)", env); got != false {
		t.Errorf("short-circuit and failed: %v", got)
	}
	if got := eval(t, "true || (1/0)", env); got != true {
		t.Errorf("short-circuit or failed: %v", got)
	}
}

func TestTernary(t *testing.T) {
	env := MapEnv{}
	if got := eval(t, "1 < 2 ? 10 : 20", env); got != 10.0 {
		t.Errorf("ternary then: %v", got)
	}
	if got := eval(t, "1 > 2 ? 10 : 20", env); got != 20.0 {
		t.Errorf("ternary else: %v", got)
	}
}

// v4: TestVars (v3 $vars/$sys/$params resolution) removed — no $-namespace in v4.
// Bare-identifier + InputEnv coverage lives in input_env_test.go.

func TestFunctions(t *testing.T) {
	env := MapEnv{}
	if got, _ := AsNumber(eval(t, "abs(-5)", env)); got != 5 {
		t.Errorf("abs: %v", got)
	}
	if got, _ := AsNumber(eval(t, "min(3, 7)", env)); got != 3 {
		t.Errorf("min: %v", got)
	}
	if got, _ := AsNumber(eval(t, "max(3, 7)", env)); got != 7 {
		t.Errorf("max: %v", got)
	}
	got, _ := AsNumber(eval(t, "now()", env))
	if got <= 0 || got > 1e15 || math.IsNaN(got) {
		t.Errorf("now: %v", got)
	}
}

func TestErrors(t *testing.T) {
	env := MapEnv{}
	bad := []string{
		"1 +",
		"(1 + 2",
		"1 == ",
		"& 1",
		`"unterm`,
		"unknown_func(1)",
		"1 / 0",
		"1 < \"abc\"",
	}
	for _, src := range bad {
		n, perr := Parse(src)
		if perr == nil {
			if _, err := Eval(n, env); err == nil {
				t.Errorf("expected error for %q", src)
			}
		}
	}
}

func TestErrorPositionInfo(t *testing.T) {
	_, err := Parse("1 + ")
	if err == nil || !strings.Contains(err.Error(), "col") {
		t.Errorf("expected col in error, got %v", err)
	}
}

// evalErr parses and evaluates src, returning the error (nil if none).
// Used for arg-count / type error assertions.
func evalErr(t *testing.T, src string, env Env) (Value, error) {
	t.Helper()
	n, perr := Parse(src)
	if perr != nil {
		return nil, perr
	}
	return Eval(n, env)
}

func TestEvalCall_MathFunctions(t *testing.T) {
	env := MapEnv{}

	// floor / ceil / sqrt — 1 arg
	if got, _ := AsNumber(eval(t, "floor(3.7)", env)); got != 3 {
		t.Errorf("floor(3.7) = %v, want 3", got)
	}
	if got, _ := AsNumber(eval(t, "ceil(3.2)", env)); got != 4 {
		t.Errorf("ceil(3.2) = %v, want 4", got)
	}
	if got, _ := AsNumber(eval(t, "sqrt(9)", env)); got != 3 {
		t.Errorf("sqrt(9) = %v, want 3", got)
	}

	// round — 1 or 2 args (aligned with Round node)
	if got, _ := AsNumber(eval(t, "round(2.5)", env)); got != 3 {
		t.Errorf("round(2.5) = %v, want 3", got)
	}
	if got, _ := AsNumber(eval(t, "round(3.14159, 2)", env)); math.Abs(got-3.14) > 1e-9 {
		t.Errorf("round(3.14159, 2) = %v, want 3.14", got)
	}
	if got, _ := AsNumber(eval(t, "round(12345, -2)", env)); got != 12300 {
		t.Errorf("round(12345, -2) = %v, want 12300", got)
	}

	// pow — 2 args
	if got, _ := AsNumber(eval(t, "pow(10, 2)", env)); math.Abs(got-100) > 1e-9 {
		t.Errorf("pow(10,2) = %v, want 100", got)
	}

	// clamp — 3 args, lo>hi swap
	if got, _ := AsNumber(eval(t, "clamp(15, 0, 10)", env)); got != 10 {
		t.Errorf("clamp(15,0,10) = %v, want 10", got)
	}
	if got, _ := AsNumber(eval(t, "clamp(5, 10, 0)", env)); got != 5 {
		t.Errorf("clamp(5,10,0) = %v, want 5 (swap)", got)
	}

	// special value: sqrt negative → NaN
	if got, _ := AsNumber(eval(t, "sqrt(-1)", env)); !math.IsNaN(got) {
		t.Errorf("sqrt(-1) = %v, want NaN", got)
	}
}

func TestEvalCall_MathFunctions_ArgErrors(t *testing.T) {
	env := MapEnv{}
	for _, expr := range []string{"floor()", "ceil(1, 2)", "sqrt()", "round()", "round(1, 2, 3)", "pow(1)", "clamp(1, 2)"} {
		if _, err := evalErr(t, expr, env); err == nil {
			t.Errorf("%s: want arg-count error, got nil", expr)
		}
	}
}
