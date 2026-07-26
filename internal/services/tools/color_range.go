package tools

import (
	"fmt"
	"math"
	"sort"

	"github.com/yottaapp/yotta/pkg/vision"
)

// RGB 一个采样像素 (前端降采样后发来). wails 绑定生成 tools.RGB {R,G,B:number}.
type RGB struct {
	R, G, B uint8
}

// ColorRangeResult 吸管提取结果. 槽序为命名契约 (与 dcRangeSchema/hsvObjSchema 槽位对齐):
// hsv=[hMin,hMax,sMin,sMax,vMin,vMax], rgb=[rMin,rMax,gMin,gMax,bMin,bMax].
type ColorRangeResult struct {
	Range   [6]int `json:"range"`
	HueWrap bool   `json:"hueWrap"`
}

const (
	huePad = 5
	svPad  = 10
	rgbPad = 15
	sGate  = 15
)

// extractColorRange 纯函数: 降采样 RGB 样本 → 颜色范围. 复用 vision.RGBToHSV (单一算法源).
func extractColorRange(samples []RGB, colorSpace string) (ColorRangeResult, error) {
	if len(samples) == 0 {
		return ColorRangeResult{}, fmt.Errorf("extractColorRange: 空样本")
	}

	if colorSpace == "rgb" {
		rs := make([]int, len(samples))
		gs := make([]int, len(samples))
		bs := make([]int, len(samples))
		for i, p := range samples {
			rs[i], gs[i], bs[i] = int(p.R), int(p.G), int(p.B)
		}
		sort.Ints(rs)
		sort.Ints(gs)
		sort.Ints(bs)
		rMin, rMax := channelRange(rs, rgbPad, 0, 255)
		gMin, gMax := channelRange(gs, rgbPad, 0, 255)
		bMin, bMax := channelRange(bs, rgbPad, 0, 255)
		return ColorRangeResult{Range: [6]int{rMin, rMax, gMin, gMax, bMin, bMax}}, nil
	}

	ss := make([]int, len(samples))
	vs := make([]int, len(samples))
	hs := make([]int, 0, len(samples))
	for i, p := range samples {
		h, s, v := vision.RGBToHSV(p.R, p.G, p.B)
		ss[i], vs[i] = s, v
		if s >= sGate {
			hs = append(hs, h)
		}
	}
	sort.Ints(ss)
	sort.Ints(vs)
	sMin, sMax := channelRange(ss, svPad, 0, 100)
	vMin, vMax := channelRange(vs, svPad, 0, 100)

	if len(hs) == 0 {
		return ColorRangeResult{Range: [6]int{0, 360, sMin, sMax, vMin, vMax}}, nil
	}
	sort.Ints(hs)
	if hueWraps(hs) {
		return ColorRangeResult{Range: [6]int{0, 360, sMin, sMax, vMin, vMax}, HueWrap: true}, nil
	}
	hMin := clampInt(percentile(hs, 2)-huePad, 0, 360)
	hMax := clampInt(percentile(hs, 98)+huePad, 0, 360)
	return ColorRangeResult{Range: [6]int{hMin, hMax, sMin, sMax, vMin, vMax}}, nil
}

// channelRange 单通道: P2/P98 → 先加 padding 后 clamp.
func channelRange(sorted []int, pad, lo, hi int) (mn, mx int) {
	mn = clampInt(percentile(sorted, 2)-pad, lo, hi)
	mx = clampInt(percentile(sorted, 98)+pad, lo, hi)
	return
}

// percentile sorted 切片在 p (0-100) 处的线性插值 (sorted 已排序非空).
func percentile(sorted []int, p float64) int {
	n := len(sorted)
	if n == 1 {
		return sorted[0]
	}
	rank := p / 100 * float64(n-1)
	lo := int(math.Floor(rank))
	hi := int(math.Ceil(rank))
	if lo == hi {
		return sorted[lo]
	}
	frac := rank - float64(lo)
	return int(math.Round(float64(sorted[lo]) + frac*float64(sorted[hi]-sorted[lo])))
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// hueWraps 排序后色相点集的最小覆盖弧是否跨 0/360.
// 最大相邻间隙(含末→首过零的环回间隙)的补 = 覆盖弧. 若最大间隙=环回间隙 → 覆盖弧不跨零;
// 否则最大空隙在内部 → 覆盖弧跨零 → 环绕.
func hueWraps(sorted []int) bool {
	n := len(sorted)
	if n < 2 {
		return false
	}
	maxGap, wrapIsMax := -1, false
	for i := 0; i < n-1; i++ {
		if g := sorted[i+1] - sorted[i]; g > maxGap {
			maxGap, wrapIsMax = g, false
		}
	}
	if wrapGap := (360 - sorted[n-1]) + sorted[0]; wrapGap > maxGap {
		wrapIsMax = true
	}
	return !wrapIsMax
}

// ExtractColorRange wails RPC: 前端降采样像素 → 颜色范围. 点一次吸管调一次(非每帧).
func (s *Service) ExtractColorRange(samples []RGB, colorSpace string) (ColorRangeResult, error) {
	return extractColorRange(samples, colorSpace)
}
