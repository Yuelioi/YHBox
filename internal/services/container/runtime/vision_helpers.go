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
