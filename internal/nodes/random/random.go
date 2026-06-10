// Package random — 非确定随机数节点 (RandomInt / RandomFloat / RandomBool).
// 全 IsPureData=true + IsNonDeterministic=true: 框架在 per-dispatch eval cache 里记忆化成功
// 结果, 同一求值内多路径引用拿同值 (见 specs/2026-06-10-random-nodes.md A 节). 底层 math/rand/v2.
package random

import (
	"encoding/json"
	"math/rand/v2"

	"yotta/internal/node"
)

func init() {
	for _, n := range []node.Node{&RandomInt{}, &RandomFloat{}, &RandomBool{}} {
		node.Register(n)
	}
}

// randomSamples — centered 分布的 Bates 样本数. 与 internal/node/jitter.go::jitterSamples
// 同为 5、同 CLT 思路 (独立维护, 不为一个 int 跨包耦合).
const randomSamples = 5

// bellUnit 返回 [0,1) 内向 0.5 聚集的样本 (Bates: randomSamples 个 uniform 求均值).
func bellUnit() float64 {
	var sum float64
	for i := 0; i < randomSamples; i++ {
		sum += rand.Float64()
	}
	return sum / randomSamples
}

const (
	distUniform  = "uniform"
	distCentered = "centered"
)

// distributionInput — uniform/centered 下拉 (Advanced).
func distributionInput() node.InputSpec {
	return node.InputSpec{
		Name: "Distribution", Type: "String", Default: distUniform, Advanced: true,
		Widget: node.WidgetSpec{Kind: "dropdown",
			Props: node.MarshalProps(node.DropdownProps{
				Options: []node.EnumOption{{Value: distUniform}, {Value: distCentered}}})},
	}
}

// ===== RandomInt =====

type RandomInt struct{}

func (RandomInt) Spec() node.Spec {
	return node.Spec{
		Kind: "RandomInt", Category: "Random",
		Inputs: []node.InputSpec{
			{Name: "Min", Type: "Number", Default: json.Number("0"), Widget: node.WidgetSpec{Kind: "number"}},
			{Name: "Max", Type: "Number", Default: json.Number("100"), Widget: node.WidgetSpec{Kind: "number"}},
			distributionInput(),
		},
		Outputs:            []node.OutputSpec{{Name: "Result", Type: "Integer"}},
		IsPureData:         true,
		IsNonDeterministic: true,
	}
}

func (RandomInt) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	lo, hi := int64(in.Int("Min")), int64(in.Int("Max"))
	if lo > hi {
		lo, hi = hi, lo
	}
	if lo == hi {
		return int(lo), nil
	}
	span := hi - lo + 1 // int64 防普通 int32 满区间溢出
	if in.String("Distribution") == distCentered {
		n := lo + int64(bellUnit()*float64(span))
		if n < lo { // 纯防御: Float64()∈[0,1) → 均值<1 → 理论不触发; 勿当冗余删
			n = lo
		}
		if n > hi {
			n = hi
		}
		return int(n), nil
	}
	return int(lo + rand.Int64N(span)), nil
}

// ===== RandomFloat =====

type RandomFloat struct{}

func (RandomFloat) Spec() node.Spec {
	return node.Spec{
		Kind: "RandomFloat", Category: "Random",
		Inputs: []node.InputSpec{
			{Name: "Min", Type: "Number", Default: json.Number("0"), Widget: node.WidgetSpec{Kind: "number"}},
			{Name: "Max", Type: "Number", Default: json.Number("1"), Widget: node.WidgetSpec{Kind: "number"}},
			distributionInput(),
		},
		Outputs:            []node.OutputSpec{{Name: "Result", Type: "Number"}},
		IsPureData:         true,
		IsNonDeterministic: true,
	}
}

func (RandomFloat) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	lo, hi := in.Float64("Min"), in.Float64("Max")
	if lo > hi {
		lo, hi = hi, lo
	}
	if lo == hi {
		return lo, nil
	}
	u := rand.Float64()
	if in.String("Distribution") == distCentered {
		u = bellUnit()
	}
	return lo + u*(hi-lo), nil
}

// ===== RandomBool =====

type RandomBool struct{}

func (RandomBool) Spec() node.Spec {
	return node.Spec{
		Kind: "RandomBool", Category: "Random",
		Inputs: []node.InputSpec{
			{Name: "Prob", Type: "Number", Default: json.Number("0.5"), Widget: node.WidgetSpec{Kind: "number"}},
		},
		Outputs:            []node.OutputSpec{{Name: "Result", Type: "Bool"}},
		IsPureData:         true,
		IsNonDeterministic: true,
	}
}

// Evaluate — Prob 越界自然夹紧: Float64()∈[0,1) → Prob<=0 恒 false, Prob>=1 恒 true.
func (RandomBool) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	return rand.Float64() < in.Float64("Prob"), nil
}
