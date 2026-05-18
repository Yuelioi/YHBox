package runtime

import (
	"testing"

	"yhbox/internal/services/container"
	"yhbox/internal/services/expr"
)

// makePureFuncNode builds a single-node ContainerRunner ready for evalPureFunc.
// Caller passes its *testing.T so newTestRunner can register Helper().
func makePureFuncNode(t *testing.T, kind string, literals map[string]any) (*ContainerRunner, *container.GraphNode) {
	t.Helper()
	_, runner := newTestRunner(t)
	n := &container.GraphNode{
		ID:     "p1",
		Kind:   kind,
		Config: map[string]any{"literal": literals},
	}
	runner.nodesByID = map[string]*container.GraphNode{"p1": n}
	return runner, n
}

func TestPureFunc_Add(t *testing.T) {
	r, n := makePureFuncNode(t, "Add", map[string]any{"a": 3.0, "b": 4.0})
	v, _ := r.evalPureFunc(n)
	if got, _ := expr.AsNumber(v); got != 7.0 {
		t.Fatalf("Add 3+4: want 7, got %v", v)
	}
}

func TestPureFunc_Sub(t *testing.T) {
	r, n := makePureFuncNode(t, "Sub", map[string]any{"a": 10.0, "b": 3.0})
	v, _ := r.evalPureFunc(n)
	if got, _ := expr.AsNumber(v); got != 7.0 {
		t.Fatalf("Sub 10-3: want 7, got %v", v)
	}
}

func TestPureFunc_Mul_Div_Mod_Neg(t *testing.T) {
	cases := []struct {
		kind string
		a, b float64
		want float64
	}{
		{"Mul", 6, 7, 42},
		{"Div", 20, 4, 5},
		{"Mod", 10, 3, 1},
	}
	for _, c := range cases {
		t.Run(c.kind, func(t *testing.T) {
			r, n := makePureFuncNode(t, c.kind, map[string]any{"a": c.a, "b": c.b})
			v, _ := r.evalPureFunc(n)
			got, _ := expr.AsNumber(v)
			if got != c.want {
				t.Fatalf("%s %v,%v: want %v, got %v", c.kind, c.a, c.b, c.want, got)
			}
		})
	}
	// Neg has only 'x'
	r, n := makePureFuncNode(t, "Neg", map[string]any{"x": 5.0})
	v, _ := r.evalPureFunc(n)
	got, _ := expr.AsNumber(v)
	if got != -5.0 {
		t.Fatalf("Neg 5: want -5, got %v", v)
	}
}

func TestPureFunc_Comparison(t *testing.T) {
	cases := []struct {
		kind string
		a, b float64
		want bool
	}{
		{"Lt", 3, 5, true}, {"Lt", 5, 5, false},
		{"LtEq", 5, 5, true}, {"LtEq", 6, 5, false},
		{"Gt", 5, 3, true}, {"Gt", 3, 5, false},
		{"GtEq", 5, 5, true}, {"GtEq", 4, 5, false},
	}
	for _, c := range cases {
		t.Run(c.kind, func(t *testing.T) {
			r, n := makePureFuncNode(t, c.kind, map[string]any{"a": c.a, "b": c.b})
			v, _ := r.evalPureFunc(n)
			if v != c.want {
				t.Fatalf("%s %v,%v: want %v, got %v", c.kind, c.a, c.b, c.want, v)
			}
		})
	}
}

func TestPureFunc_Eq_NotEq_SameType(t *testing.T) {
	// Number == number
	r, n := makePureFuncNode(t, "Eq", map[string]any{"a": 5.0, "b": 5.0})
	v, _ := r.evalPureFunc(n)
	if v != true {
		t.Fatalf("Eq 5==5: want true, got %v", v)
	}
	// String == string
	r2, n2 := makePureFuncNode(t, "Eq", map[string]any{"a": "FISHING", "b": "FISHING"})
	v2, _ := r2.evalPureFunc(n2)
	if v2 != true {
		t.Fatalf("Eq FISHING==FISHING: want true, got %v", v2)
	}
	// NotEq same → false
	r3, n3 := makePureFuncNode(t, "NotEq", map[string]any{"a": "X", "b": "X"})
	v3, _ := r3.evalPureFunc(n3)
	if v3 != false {
		t.Fatalf("NotEq X==X: want false, got %v", v3)
	}
}

func TestPureFunc_EqCrossType_ViaToString(t *testing.T) {
	// 5 == "5" via ToString: both stringify to "5" → true
	r, n := makePureFuncNode(t, "Eq", map[string]any{"a": 5.0, "b": "5"})
	v, _ := r.evalPureFunc(n)
	if v != true {
		t.Fatalf("5 == \"5\" via ToString: want true, got %v", v)
	}
}

func TestPureFunc_Logic(t *testing.T) {
	r, n := makePureFuncNode(t, "And", map[string]any{"a": true, "b": true})
	v, _ := r.evalPureFunc(n)
	if v != true {
		t.Fatalf("And true,true: want true, got %v", v)
	}
	r, n = makePureFuncNode(t, "And", map[string]any{"a": false, "b": true})
	v, _ = r.evalPureFunc(n)
	if v != false {
		t.Fatalf("And false,_: want false (short-circuit), got %v", v)
	}
	r, n = makePureFuncNode(t, "Or", map[string]any{"a": true, "b": false})
	v, _ = r.evalPureFunc(n)
	if v != true {
		t.Fatalf("Or true,_: want true (short-circuit), got %v", v)
	}
	r, n = makePureFuncNode(t, "Not", map[string]any{"x": true})
	v, _ = r.evalPureFunc(n)
	if v != false {
		t.Fatalf("Not true: want false, got %v", v)
	}
}

func TestPureFunc_String(t *testing.T) {
	r, n := makePureFuncNode(t, "Concat", map[string]any{"a": "hello ", "b": "world"})
	v, _ := r.evalPureFunc(n)
	if v != "hello world" {
		t.Fatalf("Concat: want 'hello world', got %v", v)
	}
	r, n = makePureFuncNode(t, "Contains", map[string]any{"haystack": "fishing rod", "needle": "rod"})
	v, _ = r.evalPureFunc(n)
	if v != true {
		t.Fatalf("Contains rod: want true, got %v", v)
	}
	r, n = makePureFuncNode(t, "Length", map[string]any{"s": "abcde"})
	v, _ = r.evalPureFunc(n)
	if got, _ := expr.AsNumber(v); got != 5.0 {
		t.Fatalf("Length abcde: want 5, got %v", v)
	}
}

func TestPureFunc_Conversion(t *testing.T) {
	r, n := makePureFuncNode(t, "ToString", map[string]any{"x": 42.0})
	v, _ := r.evalPureFunc(n)
	if v != "42" {
		t.Fatalf("ToString 42: want '42', got %v", v)
	}
	r, n = makePureFuncNode(t, "ToNumber", map[string]any{"x": "3.14"})
	v, _ = r.evalPureFunc(n)
	if got, _ := expr.AsNumber(v); got != 3.14 {
		t.Fatalf("ToNumber 3.14: want 3.14, got %v", v)
	}
	r, n = makePureFuncNode(t, "ToBool", map[string]any{"x": 1.0})
	v, _ = r.evalPureFunc(n)
	if v != true {
		t.Fatalf("ToBool 1: want true, got %v", v)
	}
}

func TestPureFunc_Select(t *testing.T) {
	r, n := makePureFuncNode(t, "Select", map[string]any{"cond": true, "a": "yes", "b": "no"})
	v, _ := r.evalPureFunc(n)
	if v != "yes" {
		t.Fatalf("Select true: want 'yes', got %v", v)
	}
	r, n = makePureFuncNode(t, "Select", map[string]any{"cond": false, "a": "yes", "b": "no"})
	v, _ = r.evalPureFunc(n)
	if v != "no" {
		t.Fatalf("Select false: want 'no', got %v", v)
	}
}
