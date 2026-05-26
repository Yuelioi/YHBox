// Package vision: bar_track.go
//
// AnalyzeDualColorBar — 通用双色条 HSV cluster 追踪算法. 算 inner cluster 在 outer
// cluster 区域里的位置. 适用场景:
//   - 血条 (前景色 + 背景色)
//   - 进度条 (空 + 满双色)
//   - QTE 双色条 / 钓鱼溜鱼 (cursor 高亮色 + target 区域色)
//   - 任何 "小色块在大色带里" 的双色 UI 检测
//
// 算法 (移植自 Fishing v1 实测, 默认参数从那调出来):
//  1. inner cluster: 全帧扫 HSV 命中像素, 按列计数, 取 sum-best 连续 column 段
//     (宽度限 [InnerMinPx, InnerMaxPx]).
//  2. outer cluster: 在 inner 周围 vertical band 内扫 outer HSV 命中, 取 sum-best
//     段 (宽度 >= OuterMinPx).
//  3. confidence: min(0.98, inner_conf*ConfInnerWeight + outer_conf*ConfOuterWeight).
//
// 业务包 (tools/fish) wrap 这个 API + 自己的业务 HSV 默认值 + 字段名映射, vision 包
// 自己不带任何 fishing-specific 知识.

package vision

import (
	"image"
	"math"
)

// HSVRange HSV 阈值区间. H ∈ [0,180] (OpenCV 风格), S/V ∈ [0,255].
// 跟 internal/node.HSVRange 形态一致 (重复定义避免 pkg/vision 反向 import internal/node).
type HSVRange struct {
	HMin, HMax int
	SMin, SMax int
	VMin, VMax int
}

// match returns true if (h,s,v) falls in the range.
func (r HSVRange) match(h, s, v int) bool {
	return h >= r.HMin && h <= r.HMax &&
		s >= r.SMin && s <= r.SMax &&
		v >= r.VMin && v <= r.VMax
}

// DualBarOptions 双色条算法可调参数. 0 值字段走 default (fishing UI 实测出来的默认).
type DualBarOptions struct {
	// InnerMinPx / InnerMaxPx: inner cluster 宽度范围 (px). 默认 [2, max(15, w/40)].
	InnerMinPx int
	InnerMaxPx int // <=0 表 max(15, w/40)
	// OuterMinPx: outer cluster 最小宽度 (px). 默认 w/20.
	OuterMinPx int // <=0 表 w/20
	// BandRatioH: outer band 相对 frame 高度的比例. 默认 0.30.
	BandRatioH float64 // <=0 表 0.30
	// BandRatioInner: outer band 相对 inner cluster 高度的比例. 默认 0.85.
	BandRatioInner float64 // <=0 表 0.85
	// ConfInnerWeight / ConfOuterWeight: 置信度公式权重. 默认 0.42 / 0.58.
	// 两个都 0 才走默认; 设一个就用设的, 另一个补 0.
	ConfInnerWeight float64
	ConfOuterWeight float64
}

func (o DualBarOptions) innerMin() int {
	if o.InnerMinPx > 0 {
		return o.InnerMinPx
	}
	return 2
}
func (o DualBarOptions) innerMax(w int) int {
	if o.InnerMaxPx > 0 {
		return o.InnerMaxPx
	}
	return max(15, w/40)
}
func (o DualBarOptions) outerMin(w int) int {
	if o.OuterMinPx > 0 {
		return o.OuterMinPx
	}
	return w / 20
}
func (o DualBarOptions) bandRatioH() float64 {
	if o.BandRatioH > 0 {
		return o.BandRatioH
	}
	return 0.30
}
func (o DualBarOptions) bandRatioInner() float64 {
	if o.BandRatioInner > 0 {
		return o.BandRatioInner
	}
	return 0.85
}
func (o DualBarOptions) confWeights() (innerW, outerW float64) {
	if o.ConfInnerWeight <= 0 && o.ConfOuterWeight <= 0 {
		return 0.42, 0.58
	}
	return o.ConfInnerWeight, o.ConfOuterWeight
}

// DualColorBarResult 双色条算法单次结果.
//
// Found = true 表 inner + outer 都找到; false 表至少有一个没找到 (Inner/OuterX = -1).
// InnerPx/OuterPx 即使 cluster 没找到也填 HSV 命中像素总数 (调试用).
type DualColorBarResult struct {
	Found      bool
	InnerX     int     // inner cluster 中心 X (frame 坐标). -1 = miss.
	OuterX     int     // outer cluster 中心 X. -1 = miss.
	OuterWidth int     // outer cluster 宽度 (px).
	Confidence float64 // [0, 0.98].
	InnerPx    int     // inner HSV 命中像素总数 (全帧).
	OuterPx    int     // outer HSV 命中像素总数 (band 内).
}

// AnalyzeDualColorBar 在 ROI 帧内检测 inner + outer 双色 HSV cluster.
// 返回 nil 当 ROI 尺寸太小 (w<10 || h<4).
func AnalyzeDualColorBar(img *image.RGBA, inner, outer HSVRange, opts DualBarOptions) *DualColorBarResult {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w < 10 || h < 4 {
		return nil
	}

	innerX, innerH, innerConf, innerPx := findInnerCluster(img, inner, opts)

	if innerX < 0 {
		return &DualColorBarResult{
			Found:   false,
			InnerX:  -1,
			OuterX:  -1,
			InnerPx: innerPx,
		}
	}

	bandHalf := max(int(float64(h)*opts.bandRatioH()), int(float64(innerH)*opts.bandRatioInner()))
	innerY := h / 2
	bandY1 := max(0, innerY-bandHalf)
	bandY2 := min(h, innerY+bandHalf+1)

	outerX, outerW, outerConf, outerPx := findOuterCluster(img, outer, bandY1, bandY2, opts)
	if outerX < 0 || outerW < opts.outerMin(w) {
		return &DualColorBarResult{
			Found:   false,
			InnerX:  innerX,
			OuterX:  -1,
			InnerPx: innerPx,
			OuterPx: outerPx,
		}
	}

	iw, ow := opts.confWeights()
	conf := math.Min(0.98, innerConf*iw+outerConf*ow)
	return &DualColorBarResult{
		Found:      true,
		InnerX:     innerX,
		OuterX:     outerX,
		OuterWidth: outerW,
		Confidence: conf,
		InnerPx:    innerPx,
		OuterPx:    outerPx,
	}
}

// findInnerCluster 全帧按列扫 inner HSV 命中, 取 sum-best 连续段 (宽度 ∈ [innerMin, innerMax(w)]).
func findInnerCluster(img *image.RGBA, hsv HSVRange, opts DualBarOptions) (cx, height int, conf float64, totalPx int) {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	colCount := make([]int, w)
	for y := range h {
		off := y * img.Stride
		for x := range w {
			i := off + x*4
			r, g, b := img.Pix[i], img.Pix[i+1], img.Pix[i+2]
			hh, ss, vv := RGBToHSV(r, g, b)
			if hsv.match(hh, ss, vv) {
				colCount[x]++
				totalPx++
			}
		}
	}

	minW := opts.innerMin()
	maxW := opts.innerMax(w)
	bestStart, bestEnd, bestSum := -1, -1, 0
	i := 0
	for i < w {
		if colCount[i] == 0 {
			i++
			continue
		}
		start := i
		sum := 0
		for i < w && colCount[i] > 0 {
			sum += colCount[i]
			i++
		}
		runW := i - start
		if runW >= minW && runW <= maxW && sum > bestSum {
			bestSum = sum
			bestStart = start
			bestEnd = i
		}
	}

	if bestSum < 2 {
		return -1, 0, 0, totalPx
	}

	clusterW := bestEnd - bestStart
	cx = (bestStart + bestEnd) / 2
	height = bestSum / max(1, clusterW)

	aspect := float64(height) / float64(max(1, clusterW))
	fillRatio := float64(bestSum) / float64(max(1, clusterW*h))
	conf = math.Min(1.0, aspect*0.3+fillRatio*2.0)
	if aspect < 1.0 {
		conf *= 0.5
	}
	return cx, height, math.Min(1.0, conf), totalPx
}

// findOuterCluster 在 [bandY1, bandY2) 行内按列扫 outer HSV 命中, 取 sum-best 段.
func findOuterCluster(img *image.RGBA, hsv HSVRange, bandY1, bandY2 int, _ DualBarOptions) (cx, width int, conf float64, totalPx int) {
	bounds := img.Bounds()
	w := bounds.Dx()

	colCount := make([]int, w)
	for y := bandY1; y < bandY2; y++ {
		off := y * img.Stride
		for x := range w {
			i := off + x*4
			r, g, b := img.Pix[i], img.Pix[i+1], img.Pix[i+2]
			hh, ss, vv := RGBToHSV(r, g, b)
			if hsv.match(hh, ss, vv) {
				colCount[x]++
				totalPx++
			}
		}
	}

	if totalPx < 3 {
		return -1, 0, 0, totalPx
	}

	bestStart, bestEnd, bestSum := -1, -1, 0
	i := 0
	for i < w {
		if colCount[i] == 0 {
			i++
			continue
		}
		start := i
		sum := 0
		for i < w && colCount[i] > 0 {
			sum += colCount[i]
			i++
		}
		if sum > bestSum {
			bestSum = sum
			bestStart = start
			bestEnd = i
		}
	}

	if bestStart < 0 {
		return -1, 0, 0, totalPx
	}

	width = bestEnd - bestStart
	cx = (bestStart + bestEnd) / 2
	bandH := bandY2 - bandY1
	fillRatio := float64(bestSum) / float64(max(1, width*bandH))
	conf = math.Min(1.0, fillRatio*2.5)
	return cx, width, conf, totalPx
}
