// math.go — 数学补全节点 (9 个): Abs/Min/Max/Floor/Ceil/Round/Clamp/Pow/Sqrt.
// 见 specs/2026-06-10-math-nodes.md. 注册在 purefunc.go::init() "数学" 组.
// NaN/Inf 一律透传 (GIGO, 与 Div 除零返 NaN 同惯例), 不特判.
package purefunc

import (
	"encoding/json"
	"math"

	"yotta/internal/node"
)

// numXIn 单 X Number 输入 (Abs/Floor/Ceil/Sqrt 共用形态, 同 Neg).
func numXIn() node.InputSpec {
	return node.InputSpec{Name: "X", Type: "Number", Default: json.Number("0"), Widget: node.WidgetSpec{Kind: "number"}}
}

// ===== 单输入族 =====

type Abs struct{}

func (Abs) Spec() node.Spec {
	return specBuilder("Abs", []node.InputSpec{numXIn()}, "Number")
}
func (Abs) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	return math.Abs(in.Float64("X")), nil
}

type Floor struct{}

func (Floor) Spec() node.Spec {
	return specBuilder("Floor", []node.InputSpec{numXIn()}, "Number")
}
func (Floor) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	return math.Floor(in.Float64("X")), nil
}

type Ceil struct{}

func (Ceil) Spec() node.Spec {
	return specBuilder("Ceil", []node.InputSpec{numXIn()}, "Number")
}
func (Ceil) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	return math.Ceil(in.Float64("X")), nil
}

type Sqrt struct{}

func (Sqrt) Spec() node.Spec {
	return specBuilder("Sqrt", []node.InputSpec{numXIn()}, "Number")
}

// Evaluate — X<0 → NaN 透传不特判 (i18n 已提示).
func (Sqrt) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	return math.Sqrt(in.Float64("X")), nil
}

// ===== 双输入族 =====

type Min struct{}

func (Min) Spec() node.Spec {
	return specBuilder("Min", numIn(), "Number")
}
func (Min) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	return math.Min(in.Float64("A"), in.Float64("B")), nil
}

type Max struct{}

func (Max) Spec() node.Spec {
	return specBuilder("Max", numIn(), "Number")
}
func (Max) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	return math.Max(in.Float64("A"), in.Float64("B")), nil
}

type Pow struct{}

func (Pow) Spec() node.Spec {
	return specBuilder("Pow", []node.InputSpec{
		{Name: "Base", Type: "Number", Default: json.Number("0"), Widget: node.WidgetSpec{Kind: "number"}},
		{Name: "Exp", Type: "Number", Default: json.Number("1"), Widget: node.WidgetSpec{Kind: "number"}},
	}, "Number")
}

// Evaluate — 特殊值走 Go math.Pow 语义透传: 负底数+分数指数→NaN, 0^负→Inf, 0^0→1.
func (Pow) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	return math.Pow(in.Float64("Base"), in.Float64("Exp")), nil
}

// ===== Round / Clamp =====

type Round struct{}

func (Round) Spec() node.Spec {
	return specBuilder("Round", []node.InputSpec{
		numXIn(),
		{Name: "Digits", Type: "Integer", Default: json.Number("0"), Widget: node.WidgetSpec{Kind: "number"}},
	}, "Number")
}

// Evaluate — SQL ROUND 约定: Digits=0→最近整数, 2→两位小数, -2→取整到百位.
// Digits clamp 到 [-15,15]: 防 10^Digits 上溢 Inf/下溢 0 出垃圾 (float64 有效位本就 ~15-17).
func (Round) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	x := in.Float64("X")
	d := in.Int("Digits")
	if d > 15 {
		d = 15
	}
	if d < -15 {
		d = -15
	}
	factor := math.Pow(10, float64(d))
	return math.Round(x*factor) / factor, nil
}

type Clamp struct{}

func (Clamp) Spec() node.Spec {
	return specBuilder("Clamp", []node.InputSpec{
		numXIn(),
		{Name: "Min", Type: "Number", Default: json.Number("0"), Widget: node.WidgetSpec{Kind: "number"}},
		{Name: "Max", Type: "Number", Default: json.Number("100"), Widget: node.WidgetSpec{Kind: "number"}},
	}, "Number")
}

// Evaluate — Min>Max 先交换 (与 RandomInt 同惯例). NaN 比较恒 false → 原样透传.
func (Clamp) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	x, lo, hi := in.Float64("X"), in.Float64("Min"), in.Float64("Max")
	if lo > hi {
		lo, hi = hi, lo
	}
	switch {
	case x < lo:
		return lo, nil
	case x > hi:
		return hi, nil
	}
	return x, nil
}
