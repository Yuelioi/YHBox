package purefunc

import (
	"context"
	"math"
	"testing"

	"yotta/internal/node"
)

// evalMathNode 直接走 framework EvaluatePureData (与 TestEvaluate_22PureFuncs 同范式).
func evalMathNode(t *testing.T, n node.Node, dataWire map[string]any) any {
	t.Helper()
	node.ResetRegistryForTest()
	node.Register(n)
	rn, ok := node.Get(n.Spec().Kind)
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
