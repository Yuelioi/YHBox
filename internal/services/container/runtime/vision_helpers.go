package runtime

import (
	"image"

	"yotta/pkg/vision"
)

// Vision helpers shared by node_services.go's VisionAdapter (DetectColorHSV /
// ROIColorScan / DualBarTrack).

// confBarV2 — DualBarTrack 的置信度阈值.
const confBarV2 = 0.50

// clusterEntry — ROIColorScan 结果. SysROIScanResult.Clusters 仍引用此类型.
type clusterEntry struct {
	StartPx  int `json:"startPx"`
	EndPx    int `json:"endPx"`
	CenterPx int `json:"centerPx"`
	PxCount  int `json:"pxCount"`
}

// countHSVInROI 统计 img 中落在 hsvRange 内的像素数和比例.
// 直接访问 Pix 避免 image.At() 接口开销.
func countHSVInROI(img *image.RGBA, h hsvRange) (count int, ratio float64) {
	bounds := img.Bounds()
	w, hh := bounds.Dx(), bounds.Dy()
	if w == 0 || hh == 0 {
		return 0, 0
	}
	for y := 0; y < hh; y++ {
		off := y * img.Stride
		for x := 0; x < w; x++ {
			i := off + x*4
			rv, gv, bv := img.Pix[i], img.Pix[i+1], img.Pix[i+2]
			hv, sv, vv := vision.RGBToHSV(rv, gv, bv)
			if hv >= h.hMin && hv <= h.hMax &&
				sv >= h.sMin && sv <= h.sMax &&
				vv >= h.vMin && vv <= h.vMax {
				count++
			}
		}
	}
	ratio = float64(count) / float64(w*hh)
	return
}

// hsvInRange RGB → HSV 范围 check.
func hsvInRange(rv, gv, bv uint8, h hsvRange) bool {
	hv, sv, vv := vision.RGBToHSV(rv, gv, bv)
	return hv >= h.hMin && hv <= h.hMax &&
		sv >= h.sMin && sv <= h.sMax &&
		vv >= h.vMin && vv <= h.vMax
}

// countColorPixels 在全帧 [x0,x1)×[y0,y1) 矩形内数落在 rng 内的像素, 累加命中坐标 (全帧坐标系).
// mode="rgb": rng=[rMin,rMax,gMin,gMax,bMin,bMax]; 否则 HSV: rng=[hMin,hMax,sMin,sMax,vMin,vMax].
// 给 DetectColor 用 (需 RGB/HSV 双模 + 中心点); HSV-only/比例语义见 countHSVInROI.
func countColorPixels(frame *image.RGBA, x0, y0, x1, y1 int, mode string, rng [6]int) (count, sumX, sumY int) {
	useHSV := mode != "rgb"
	stride := frame.Stride
	for y := y0; y < y1; y++ {
		off := y * stride
		for x := x0; x < x1; x++ {
			i := off + x*4
			r, g, b := frame.Pix[i], frame.Pix[i+1], frame.Pix[i+2]
			var hit bool
			if useHSV {
				hh, ss, vv := vision.RGBToHSV(r, g, b)
				hit = hh >= rng[0] && hh <= rng[1] && ss >= rng[2] && ss <= rng[3] && vv >= rng[4] && vv <= rng[5]
			} else {
				hit = int(r) >= rng[0] && int(r) <= rng[1] && int(g) >= rng[2] && int(g) <= rng[3] && int(b) >= rng[4] && int(b) <= rng[5]
			}
			if hit {
				count++
				sumX += x
				sumY += y
			}
		}
	}
	return
}

// blobPx — findColorBlobs 的内部结果，全帧像素坐标。
type blobPx struct {
	CenterX, CenterY int // 像素质心（全帧）
	X, Y, W, H       int // 外接 bbox（全帧像素）
	Area             int // 命中像素数
}

// colorHit 单像素命中判定。useHSV=false 时按 RGB。与 countColorPixels 同款语义。
func colorHit(r, g, b uint8, useHSV bool, rng [6]int) bool {
	if useHSV {
		hh, ss, vv := vision.RGBToHSV(r, g, b)
		return hh >= rng[0] && hh <= rng[1] && ss >= rng[2] && ss <= rng[3] && vv >= rng[4] && vv <= rng[5]
	}
	return int(r) >= rng[0] && int(r) <= rng[1] &&
		int(g) >= rng[2] && int(g) <= rng[3] &&
		int(b) >= rng[4] && int(b) <= rng[5]
}

// findColorBlobs 在全帧 [x0,x1)×[y0,y1) 内做 8-邻域 flood-fill 连通域标记。
// 返回每块的全帧像素 bbox/质心/面积（已加 (x0,y0) offset），过滤 Area<minArea。
// w/h<=0 → 返回 nil。
func findColorBlobs(frame *image.RGBA, x0, y0, x1, y1 int, mode string, rng [6]int, minArea int) []blobPx {
	w, h := x1-x0, y1-y0
	if w <= 0 || h <= 0 {
		return nil
	}
	useHSV := mode != "rgb"
	stride := frame.Stride

	mask := make([]bool, w*h)
	for yy := 0; yy < h; yy++ {
		off := (y0 + yy) * stride
		row := yy * w
		for xx := 0; xx < w; xx++ {
			i := off + (x0+xx)*4
			if colorHit(frame.Pix[i], frame.Pix[i+1], frame.Pix[i+2], useHSV, rng) {
				mask[row+xx] = true
			}
		}
	}

	visited := make([]bool, w*h)
	stack := make([]int, 0, 64)
	var blobs []blobPx
	for start := 0; start < w*h; start++ {
		if !mask[start] || visited[start] {
			continue
		}
		visited[start] = true
		stack = stack[:0]
		stack = append(stack, start)
		var area, sumX, sumY int
		minX, minY, maxX, maxY := w, h, -1, -1
		for len(stack) > 0 {
			p := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			px, py := p%w, p/w
			area++
			sumX += px
			sumY += py
			if px < minX {
				minX = px
			}
			if px > maxX {
				maxX = px
			}
			if py < minY {
				minY = py
			}
			if py > maxY {
				maxY = py
			}
			for dy := -1; dy <= 1; dy++ {
				ny := py + dy
				if ny < 0 || ny >= h {
					continue
				}
				for dx := -1; dx <= 1; dx++ {
					if dx == 0 && dy == 0 {
						continue
					}
					nx := px + dx
					if nx < 0 || nx >= w {
						continue
					}
					q := ny*w + nx
					if mask[q] && !visited[q] {
						visited[q] = true
						stack = append(stack, q)
					}
				}
			}
		}
		if area < minArea {
			continue
		}
		blobs = append(blobs, blobPx{
			CenterX: x0 + sumX/area,
			CenterY: y0 + sumY/area,
			X:       x0 + minX,
			Y:       y0 + minY,
			W:       maxX - minX + 1,
			H:       maxY - minY + 1,
			Area:    area,
		})
	}
	return blobs
}

// scanClusters 沿 axis (x/y) 扫描连通 cluster (满足 HSV 范围的连续行/列).
// 输出按 axis 顺序 start→end. minPx/maxPx 区间外的 cluster 丢弃.
func scanClusters(img *image.RGBA, h hsvRange, axis string, minPx, maxPx int) []clusterEntry {
	bounds := img.Bounds()
	w, hh := bounds.Dx(), bounds.Dy()

	var lineCount []int
	if axis == "x" {
		lineCount = make([]int, w)
		for y := 0; y < hh; y++ {
			off := y * img.Stride
			for x := 0; x < w; x++ {
				i := off + x*4
				if hsvInRange(img.Pix[i], img.Pix[i+1], img.Pix[i+2], h) {
					lineCount[x]++
				}
			}
		}
	} else {
		lineCount = make([]int, hh)
		for y := 0; y < hh; y++ {
			off := y * img.Stride
			for x := 0; x < w; x++ {
				i := off + x*4
				if hsvInRange(img.Pix[i], img.Pix[i+1], img.Pix[i+2], h) {
					lineCount[y]++
				}
			}
		}
	}

	var clusters []clusterEntry
	i := 0
	for i < len(lineCount) {
		if lineCount[i] == 0 {
			i++
			continue
		}
		start := i
		sum := 0
		for i < len(lineCount) && lineCount[i] > 0 {
			sum += lineCount[i]
			i++
		}
		runLen := i - start
		if runLen < minPx || runLen > maxPx {
			continue
		}
		clusters = append(clusters, clusterEntry{
			StartPx:  start,
			EndPx:    i,
			CenterPx: (start + i) / 2,
			PxCount:  sum,
		})
	}
	return clusters
}
