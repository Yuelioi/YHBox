package runtime

import (
	"image"

	"yhbox/pkg/vision"
)

// Vision helpers shared by node_services.go's VisionAdapter (DetectColorHSV /
// ROIColorScan / BarTrack). 从老 detect_hsv.go / roi_scan.go / color_bar_track.go
// 抽出来 — atomic #5 拆老 execX 时把这些 pure helper 留下.

// confBarV2 — vision.AnalyzeBar BarTrack 的置信度阈值. 历史: 复刻 fish bot config.
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
