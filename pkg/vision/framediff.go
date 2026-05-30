// pkg/vision/framediff.go — 帧间像素差分原语 (通用感知 building block).
//
// Downsample 把一帧 box-average 降采样成 gridSize×gridSize 的 RGB 签名 ([]uint8,
// 行主序, 每格 3 字节). 签名抗噪 (格内平均掉压缩抖动) 且极小 (32² = 3KB), 可复用于
// 帧差 (WaitStable/WaitChange) / 后续 pHash / 场景分类 / 多帧投票.
//
// 两个度量都归一到 [0,1], 越大越不同, 故节点用同一 Threshold 字段跨度量通用.
// 通道顺序无关: 仅按位置比对, 不区分 RGB/BGR.
package vision

import "image"

// Downsample box-average 整帧 → gridSize×gridSize, 返 flat RGB (len = gridSize*gridSize*3).
// gridSize<1 视作 1. 取像素前三通道 (跳过 alpha).
func Downsample(frame *image.RGBA, gridSize int) []uint8 {
	if gridSize < 1 {
		gridSize = 1
	}
	b := frame.Bounds()
	w, h := b.Dx(), b.Dy()
	sig := make([]uint8, gridSize*gridSize*3)
	if w <= 0 || h <= 0 {
		return sig
	}
	for gy := 0; gy < gridSize; gy++ {
		y0 := b.Min.Y + gy*h/gridSize
		y1 := b.Min.Y + (gy+1)*h/gridSize
		if y1 <= y0 {
			y1 = y0 + 1
		}
		for gx := 0; gx < gridSize; gx++ {
			x0 := b.Min.X + gx*w/gridSize
			x1 := b.Min.X + (gx+1)*w/gridSize
			if x1 <= x0 {
				x1 = x0 + 1
			}
			var sumR, sumG, sumB, n uint64
			for y := y0; y < y1; y++ {
				off := frame.PixOffset(x0, y)
				for x := x0; x < x1; x++ {
					sumR += uint64(frame.Pix[off])
					sumG += uint64(frame.Pix[off+1])
					sumB += uint64(frame.Pix[off+2])
					off += 4
					n++
				}
			}
			ci := (gy*gridSize + gx) * 3
			if n > 0 {
				sig[ci] = uint8(sumR / n)
				sig[ci+1] = uint8(sumG / n)
				sig[ci+2] = uint8(sumB / n)
			}
		}
	}
	return sig
}

// GridChangedRatio 逐格 max(|ΔR|,|ΔG|,|ΔB|) > cellDelta 计"变了", 返 变格/总格 ∈ [0,1].
// 长度不等或为空 → 1.0 (视作完全不同, 安全侧).
func GridChangedRatio(a, b []uint8, cellDelta int) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 1.0
	}
	cells := len(a) / 3
	if cells == 0 {
		return 1.0
	}
	changed := 0
	for c := 0; c < cells; c++ {
		i := c * 3
		dr := absDiff(a[i], b[i])
		dg := absDiff(a[i+1], b[i+1])
		db := absDiff(a[i+2], b[i+2])
		m := dr
		if dg > m {
			m = dg
		}
		if db > m {
			m = db
		}
		if m > cellDelta {
			changed++
		}
	}
	return float64(changed) / float64(cells)
}

// GridMeanDiff 全通道 |Δ| 均值 /255 ∈ [0,1]. 长度不等或为空 → 1.0.
func GridMeanDiff(a, b []uint8) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 1.0
	}
	var sum uint64
	for i := range a {
		sum += uint64(absDiff(a[i], b[i]))
	}
	return float64(sum) / (float64(len(a)) * 255.0)
}

func absDiff(x, y uint8) int {
	if x > y {
		return int(x - y)
	}
	return int(y - x)
}
