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
