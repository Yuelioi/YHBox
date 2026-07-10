package random

import (
	"testing"

	"github.com/yottaapp/yotta/internal/node"
)

// evalInt 用 config.literal 构造 Inputs 跑 Evaluate, 返回 int 结果.
func evalInt(t *testing.T, n node.Evaluator, cfg map[string]any) int {
	t.Helper()
	in := node.NewInputsFromConfig(map[string]any{"literal": cfg})
	v, err := n.Evaluate(nil, in)
	if err != nil {
		t.Fatalf("Evaluate err: %v", err)
	}
	got, ok := v.(int)
	if !ok {
		t.Fatalf("want int, got %T (%v)", v, v)
	}
	return got
}

func TestRandomInt_Spec_Flags(t *testing.T) {
	s := RandomInt{}.Spec()
	if !s.IsPureData || !s.IsNonDeterministic {
		t.Fatalf("RandomInt must be IsPureData+IsNonDeterministic, got %+v", s)
	}
	if s.Category != "Random" {
		t.Fatalf("Category = %q, want Random", s.Category)
	}
}

func TestRandomInt_Uniform_InClosedRange(t *testing.T) {
	for i := 0; i < 2000; i++ {
		got := evalInt(t, RandomInt{}, map[string]any{"Min": 3, "Max": 7})
		if got < 3 || got > 7 {
			t.Fatalf("uniform out of [3,7]: %d", got)
		}
	}
}

func TestRandomInt_Centered_InClosedRange(t *testing.T) {
	for i := 0; i < 2000; i++ {
		got := evalInt(t, RandomInt{}, map[string]any{"Min": 0, "Max": 10, "Distribution": "centered"})
		if got < 0 || got > 10 {
			t.Fatalf("centered out of [0,10]: %d", got)
		}
	}
}

func TestRandomInt_MinGtMax_Swaps(t *testing.T) {
	for i := 0; i < 500; i++ {
		got := evalInt(t, RandomInt{}, map[string]any{"Min": 9, "Max": 2})
		if got < 2 || got > 9 {
			t.Fatalf("swap out of [2,9]: %d", got)
		}
	}
}

func TestRandomInt_MinEqMax_ReturnsMin(t *testing.T) {
	got := evalInt(t, RandomInt{}, map[string]any{"Min": 5, "Max": 5})
	if got != 5 {
		t.Fatalf("Min==Max: want 5, got %d", got)
	}
}

func TestRandomInt_LargeRange_NoOverflow(t *testing.T) {
	// 接近 int32 边界的大区间, 证明 int64 内部运算常规安全.
	got := evalInt(t, RandomInt{}, map[string]any{"Min": -2000000000, "Max": 2000000000})
	if got < -2000000000 || got > 2000000000 {
		t.Fatalf("large range out of bounds: %d", got)
	}
}

func evalFloat(t *testing.T, n node.Evaluator, cfg map[string]any) float64 {
	t.Helper()
	in := node.NewInputsFromConfig(map[string]any{"literal": cfg})
	v, err := n.Evaluate(nil, in)
	if err != nil {
		t.Fatalf("Evaluate err: %v", err)
	}
	got, ok := v.(float64)
	if !ok {
		t.Fatalf("want float64, got %T (%v)", v, v)
	}
	return got
}

func TestRandomFloat_Spec_Flags(t *testing.T) {
	s := RandomFloat{}.Spec()
	if !s.IsPureData || !s.IsNonDeterministic || s.Category != "Random" {
		t.Fatalf("bad spec: %+v", s)
	}
}

func TestRandomFloat_Uniform_InHalfOpenRange(t *testing.T) {
	for i := 0; i < 2000; i++ {
		got := evalFloat(t, RandomFloat{}, map[string]any{"Min": 2.0, "Max": 5.0})
		if got < 2.0 || got >= 5.0 {
			t.Fatalf("out of [2,5): %v", got)
		}
	}
}

func TestRandomFloat_Centered_InRange(t *testing.T) {
	for i := 0; i < 2000; i++ {
		got := evalFloat(t, RandomFloat{}, map[string]any{"Min": 0.0, "Max": 1.0, "Distribution": "centered"})
		if got < 0.0 || got >= 1.0 {
			t.Fatalf("centered out of [0,1): %v", got)
		}
	}
}

func TestRandomFloat_MinEqMax_ReturnsMin(t *testing.T) {
	got := evalFloat(t, RandomFloat{}, map[string]any{"Min": 3.5, "Max": 3.5})
	if got != 3.5 {
		t.Fatalf("Min==Max: want 3.5, got %v", got)
	}
}

func evalBool(t *testing.T, n node.Evaluator, cfg map[string]any) bool {
	t.Helper()
	in := node.NewInputsFromConfig(map[string]any{"literal": cfg})
	v, err := n.Evaluate(nil, in)
	if err != nil {
		t.Fatalf("Evaluate err: %v", err)
	}
	got, ok := v.(bool)
	if !ok {
		t.Fatalf("want bool, got %T (%v)", v, v)
	}
	return got
}

func TestRandomBool_Spec_Flags(t *testing.T) {
	s := RandomBool{}.Spec()
	if !s.IsPureData || !s.IsNonDeterministic || s.Category != "Random" {
		t.Fatalf("bad spec: %+v", s)
	}
}

func TestRandomBool_ProbZero_AlwaysFalse(t *testing.T) {
	for i := 0; i < 500; i++ {
		if evalBool(t, RandomBool{}, map[string]any{"Prob": 0.0}) {
			t.Fatal("Prob=0 should always be false")
		}
	}
}

func TestRandomBool_ProbOne_AlwaysTrue(t *testing.T) {
	for i := 0; i < 500; i++ {
		if !evalBool(t, RandomBool{}, map[string]any{"Prob": 1.0}) {
			t.Fatal("Prob=1 should always be true")
		}
	}
}

func TestRandomBool_ProbHalf_BothSeen(t *testing.T) {
	var sawT, sawF bool
	for i := 0; i < 2000 && !(sawT && sawF); i++ {
		if evalBool(t, RandomBool{}, map[string]any{"Prob": 0.5}) {
			sawT = true
		} else {
			sawF = true
		}
	}
	if !sawT || !sawF {
		t.Fatalf("Prob=0.5 should yield both; sawT=%v sawF=%v", sawT, sawF)
	}
}

func TestRandomChoice_Spec_Flags(t *testing.T) {
	s := RandomChoice{}.Spec()
	if !s.IsPureData || !s.IsNonDeterministic || s.Category != "Random" {
		t.Fatalf("bad spec: %+v", s)
	}
	if s.Outputs[0].Type != "*" {
		t.Fatalf("output type = %q, want *", s.Outputs[0].Type)
	}
}

func TestRandomChoice_PicksFromList(t *testing.T) {
	in := node.NewInputsFromConfig(map[string]any{"literal": map[string]any{"List": []any{"a", "b", "c"}}})
	seen := map[any]bool{}
	for i := 0; i < 500; i++ {
		v, err := (RandomChoice{}).Evaluate(nil, in)
		if err != nil {
			t.Fatal(err)
		}
		if v != "a" && v != "b" && v != "c" {
			t.Fatalf("picked %v, not in list", v)
		}
		seen[v] = true
	}
	if len(seen) < 2 {
		t.Fatalf("500 picks saw only %v — not random", seen)
	}
}

func TestRandomChoice_EmptyOrNonList_Nil(t *testing.T) {
	for _, lv := range []any{[]any{}, nil, "not a list"} {
		in := node.NewInputsFromConfig(map[string]any{"literal": map[string]any{"List": lv}})
		v, err := (RandomChoice{}).Evaluate(nil, in)
		if err != nil || v != nil {
			t.Fatalf("List=%v: got (%v, %v), want (nil, nil)", lv, v, err)
		}
	}
}

func TestRandomChoice_NilElementPickable(t *testing.T) {
	in := node.NewInputsFromConfig(map[string]any{"literal": map[string]any{"List": []any{nil}}})
	v, err := (RandomChoice{}).Evaluate(nil, in)
	if err != nil || v != nil {
		t.Fatalf("got (%v, %v), want (nil, nil) — 元素本身是 nil", v, err)
	}
}
