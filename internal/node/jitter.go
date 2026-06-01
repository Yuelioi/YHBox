package node

import (
	"math"
	"math/rand/v2"
	"time"
)

// jitterSamples — 取 N 个 uniform 样本求均值 → 中心极限近正态
// (值聚在中点, 极端值罕见, 比纯 uniform 拟人).
const jitterSamples = 5

// normalJitterFactor 返回近正态的双边抖动系数, 落在 [-pct/100, +pct/100] (含端点保证: N 个
// [-p,p] uniform 的均值仍在 [-p,p]). pct<=0 → 0 (无抖动).
func normalJitterFactor(pct int) float64 {
	if pct <= 0 {
		return 0
	}
	p := float64(pct) / 100.0
	var sum float64
	for i := 0; i < jitterSamples; i++ {
		sum += (rand.Float64()*2 - 1) * p // uniform [-p, +p]
	}
	return sum / jitterSamples
}

// JitterInt 对 base 施加 ±pct% 近正态抖动后四舍五入取整. pct<=0 → 原值不变.
func JitterInt(base, pct int) int {
	if pct <= 0 {
		return base
	}
	return int(math.Round(float64(base) * (1 + normalJitterFactor(pct))))
}

// JitterDuration 对 d 施加 ±pct% 近正态抖动. pct<=0 → 原值不变.
func JitterDuration(d time.Duration, pct int) time.Duration {
	if pct <= 0 {
		return d
	}
	return time.Duration(float64(d) * (1 + normalJitterFactor(pct)))
}
