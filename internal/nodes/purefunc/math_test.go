package purefunc

import (
	"context"
	"math"
	"testing"

	"github.com/yottaapp/yotta/internal/node"
)

// evalMathNode 直接走 framework EvaluatePureData (与 TestEvaluate_22PureFuncs 同范式).
func evalMathNode(t *testing.T, n node.Node, dataWire map[string]any) any {
	t.Helper()
	registry := node.NewRegistry()
	registry.Register(n)
	rn, ok := registry.Get(n.Spec().Kind)
	if !ok {
		t.Fatalf("kind %q not registered", n.Spec().Kind)
	}
	got, err := node.EvaluatePureData(context.Background(), rn, dataWire, nil, node.StubServices())
	if err != nil {
		t.Fatalf("EvaluatePureData err: %v", err)
	}
	return got
}

func wantNum(t *testing.T, got any, want float64) {
	t.Helper()
	f, ok := got.(float64)
	if !ok {
		t.Fatalf("want float64, got %T (%v)", got, got)
	}
	if math.Abs(f-want) > 1e-9 {
		t.Fatalf("want %v, got %v", want, f)
	}
}

func wantNaN(t *testing.T, got any) {
	t.Helper()
	f, ok := got.(float64)
	if !ok || !math.IsNaN(f) {
		t.Fatalf("want NaN, got %T (%v)", got, got)
	}
}

func TestMath_SimpleSeven(t *testing.T) {
	cases := []struct {
		name     string
		n        node.Node
		dataWire map[string]any
		want     float64
	}{
		{"Abs_neg", &Abs{}, map[string]any{"X": -5.5}, 5.5},
		{"Abs_pos", &Abs{}, map[string]any{"X": 3.0}, 3.0},
		{"Min", &Min{}, map[string]any{"A": 3.0, "B": 7.0}, 3.0},
		{"Max", &Max{}, map[string]any{"A": 3.0, "B": 7.0}, 7.0},
		{"Floor_pos", &Floor{}, map[string]any{"X": 3.7}, 3.0},
		{"Floor_neg", &Floor{}, map[string]any{"X": -3.2}, -4.0},
		{"Ceil_pos", &Ceil{}, map[string]any{"X": 3.2}, 4.0},
		{"Ceil_neg", &Ceil{}, map[string]any{"X": -3.7}, -3.0},
		{"Pow", &Pow{}, map[string]any{"Base": 10.0, "Exp": 2.0}, 100.0},
		{"Pow_zero_zero", &Pow{}, map[string]any{"Base": 0.0, "Exp": 0.0}, 1.0},
		{"Sqrt", &Sqrt{}, map[string]any{"X": 9.0}, 3.0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wantNum(t, evalMathNode(t, tc.n, tc.dataWire), tc.want)
		})
	}
}

func TestMath_SpecialValues(t *testing.T) {
	// Sqrt 负数 → NaN 透传 (GIGO, 与 Div 除零返 NaN 一致).
	wantNaN(t, evalMathNode(t, &Sqrt{}, map[string]any{"X": -1.0}))
	// Pow 负底数+分数指数 → NaN.
	wantNaN(t, evalMathNode(t, &Pow{}, map[string]any{"Base": -2.0, "Exp": 0.5}))
	// Pow 0^负 → +Inf.
	if f := evalMathNode(t, &Pow{}, map[string]any{"Base": 0.0, "Exp": -1.0}).(float64); !math.IsInf(f, 1) {
		t.Fatalf("0^-1 want +Inf, got %v", f)
	}
	// Abs(NaN) → NaN 透传.
	wantNaN(t, evalMathNode(t, &Abs{}, map[string]any{"X": math.NaN()}))
}

func TestMath_SpecShape(t *testing.T) {
	for _, n := range []node.Node{&Abs{}, &Min{}, &Max{}, &Floor{}, &Ceil{}, &Pow{}, &Sqrt{}} {
		s := n.Spec()
		if !s.IsPureData || s.Category != "PureFunc" {
			t.Fatalf("%s: must be IsPureData + Category PureFunc, got %+v", s.Kind, s)
		}
		if len(s.Outputs) != 1 || s.Outputs[0].Name != "Result" || s.Outputs[0].Type != "Number" {
			t.Fatalf("%s: want single Result(Number) output, got %+v", s.Kind, s.Outputs)
		}
	}
}

func TestRound_SQLConvention(t *testing.T) {
	cases := []struct {
		name string
		x    float64
		d    int
		want float64
	}{
		{"d0_half_up", 2.5, 0, 3.0},
		{"d0_down", 2.4, 0, 2.0},
		{"d2", 3.14159, 2, 3.14},
		{"neg_d_hundreds", 12345, -2, 12300},
		{"neg_d_tens", 149, -1, 150},
		{"d_overclamp_hi", 1.23456, 99, 1.23456}, // Digits clamp 到 15, 精度内原样
		{"d_overclamp_lo", 12345, -99, 0},        // clamp 到 -15 → 10^15 量级取整 → 0
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := evalMathNode(t, &Round{}, map[string]any{"X": tc.x, "Digits": tc.d})
			wantNum(t, got, tc.want)
		})
	}
}

func TestRound_DefaultDigitsZero(t *testing.T) {
	// 不传 Digits → 默认 0 → 取整.
	wantNum(t, evalMathNode(t, &Round{}, map[string]any{"X": 7.6}), 8.0)
}

func TestClamp_Basic(t *testing.T) {
	cases := []struct {
		name            string
		x, lo, hi, want float64
	}{
		{"below", -5, 0, 10, 0},
		{"above", 15, 0, 10, 10},
		{"inside", 5, 0, 10, 5},
		{"swap_bounds", 5, 10, 0, 5}, // Min>Max 先交换 (与 RandomInt 同惯例)
		{"swap_below", -1, 10, 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := evalMathNode(t, &Clamp{}, map[string]any{"X": tc.x, "Min": tc.lo, "Max": tc.hi})
			wantNum(t, got, tc.want)
		})
	}
}

func TestClamp_NaNPassthrough(t *testing.T) {
	// NaN 任何比较为 false → 不触发任何边界分支 → 原样透传.
	wantNaN(t, evalMathNode(t, &Clamp{}, map[string]any{"X": math.NaN(), "Min": 0.0, "Max": 10.0}))
}

func TestRoundClamp_SpecShape(t *testing.T) {
	for _, n := range []node.Node{&Round{}, &Clamp{}} {
		s := n.Spec()
		if !s.IsPureData || s.Category != "PureFunc" {
			t.Fatalf("%s: bad spec %+v", s.Kind, s)
		}
	}
	// Clamp 的 Min/Max 必须是 Number — 命名分裂守卫要求与 RandomInt/RandomFloat 一致.
	for _, in := range (Clamp{}).Spec().Inputs {
		if (in.Name == "Min" || in.Name == "Max") && in.Type != "Number" {
			t.Fatalf("Clamp.%s must be Number (DetectNameSplits), got %s", in.Name, in.Type)
		}
	}
}
