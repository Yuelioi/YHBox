// 灰度预处理候选算法 — 半透明 UI 上 NCC 跨场景漂的对照工具。
//
// 背景：CCOEFF_NORMED 的分母是 patch 方差，背景像素全部进分母。
// 半透明文字的背景跨场景方差差太大 → 同模板分母不同 → conf 漂。
// 本文件提供 V/Otsu/Sobel 几种预处理，在测试里量化对比效果。
package vision

import (
	"image"
	"math"
)

// VChannel 用 max(R,G,B) 提取亮度通道，返回 [0,1] float32。
// 白字在 V 通道里恒为 1，背景仍含信息但比 BT.601 更稳。
func VChannel(img *image.RGBA) ([]float32, int, int) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	out := make([]float32, w*h)
	for y := 0; y < h; y++ {
		row := img.PixOffset(b.Min.X, b.Min.Y+y)
		dst := y * w
		for x := 0; x < w; x++ {
			r := img.Pix[row+x*4+0]
			g := img.Pix[row+x*4+1]
			bl := img.Pix[row+x*4+2]
			m := r
			if g > m {
				m = g
			}
			if bl > m {
				m = bl
			}
			out[dst+x] = float32(m) / 255.0
		}
	}
	return out, w, h
}

// OtsuThreshold 算 Otsu 阈值（最大化类间方差），输入灰度 [0,1]，返回阈值 [0,1]。
// 数据全空或全单色返回 0.5。
func OtsuThreshold(gray []float32) float32 {
	if len(gray) == 0 {
		return 0.5
	}
	const bins = 256
	var hist [bins]int
	for _, v := range gray {
		i := int(v * 255)
		if i < 0 {
			i = 0
		} else if i >= bins {
			i = bins - 1
		}
		hist[i]++
	}
	total := float64(len(gray))
	var sumAll float64
	for i, c := range hist {
		sumAll += float64(i) * float64(c)
	}
	var (
		wB, sumB float64
		bestVar  float64 = -1
		bestThr  int
	)
	for t := 0; t < bins; t++ {
		wB += float64(hist[t])
		if wB == 0 {
			continue
		}
		wF := total - wB
		if wF == 0 {
			break
		}
		sumB += float64(t) * float64(hist[t])
		mB := sumB / wB
		mF := (sumAll - sumB) / wF
		v := wB * wF * (mB - mF) * (mB - mF)
		if v > bestVar {
			bestVar = v
			bestThr = t
		}
	}
	if bestVar < 0 {
		return 0.5
	}
	return float32(bestThr) / 255.0
}

// Binarize 按阈值二值化为 0/1 float32 数组（>= 阈值 = 1）。
func Binarize(gray []float32, threshold float32) []float32 {
	out := make([]float32, len(gray))
	for i, v := range gray {
		if v >= threshold {
			out[i] = 1
		}
	}
	return out
}

// SobelMag 算 3x3 Sobel 梯度幅值，边界用 clamp。返回 [0,1] 归一化的梯度图。
// 平坦背景梯度 = 0；文字边缘强。归一化按 max 缩到 1。
func SobelMag(gray []float32, w, h int) []float32 {
	out := make([]float32, w*h)
	if w < 3 || h < 3 {
		return out
	}
	get := func(x, y int) float32 {
		if x < 0 {
			x = 0
		} else if x >= w {
			x = w - 1
		}
		if y < 0 {
			y = 0
		} else if y >= h {
			y = h - 1
		}
		return gray[y*w+x]
	}
	var maxMag float32
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			gx := -get(x-1, y-1) + get(x+1, y-1) +
				-2*get(x-1, y) + 2*get(x+1, y) +
				-get(x-1, y+1) + get(x+1, y+1)
			gy := -get(x-1, y-1) - 2*get(x, y-1) - get(x+1, y-1) +
				get(x-1, y+1) + 2*get(x, y+1) + get(x+1, y+1)
			m := float32(math.Sqrt(float64(gx*gx + gy*gy)))
			out[y*w+x] = m
			if m > maxMag {
				maxMag = m
			}
		}
	}
	if maxMag > 1e-6 {
		inv := 1 / maxMag
		for i := range out {
			out[i] *= inv
		}
	}
	return out
}
