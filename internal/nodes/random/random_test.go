package random

import (
	"testing"

	"yotta/internal/node"
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
