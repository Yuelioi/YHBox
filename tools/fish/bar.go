package fish

import (
	"image"
	"math"

	"yhbox/pkg/vision"
)

type BarResult struct {
	TargetX    int
	CursorX    int
	TargetW    int
	Confidence float64
	YellowPx   int
	GreenPx    int
}

func analyzeBar(roi *image.RGBA) *BarResult {
	bounds := roi.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w < 10 || h < 4 {
		return nil
	}

	cursorX, cursorH, cursorConf, yellowPx := findCursor(roi)

	if cursorX < 0 {
		return &BarResult{TargetX: -1, CursorX: -1, YellowPx: yellowPx}
	}

	bandHalf := max(int(float64(h)*0.30), int(float64(cursorH)*0.85))
	cursorY := h / 2
	bandY1 := max(0, cursorY-bandHalf)
	bandY2 := min(h, cursorY+bandHalf+1)

	targetX, targetW, targetConf, greenPx := findTarget(roi, bandY1, bandY2)
	if targetX < 0 || targetW < w/20 {
		return &BarResult{TargetX: -1, CursorX: cursorX, YellowPx: yellowPx, GreenPx: greenPx}
	}

	conf := math.Min(0.98, cursorConf*0.42+targetConf*0.58)
	return &BarResult{
		TargetX:    targetX,
		CursorX:    cursorX,
		TargetW:    targetW,
		Confidence: conf,
		YellowPx:   yellowPx,
		GreenPx:    greenPx,
	}
}

func findCursor(img *image.RGBA) (cx int, height int, conf float64, totalYellow int) {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	colCount := make([]int, w)
	for y := range h {
		off := y * img.Stride
		for x := range w {
			i := off + x*4
			r, g, b := img.Pix[i], img.Pix[i+1], img.Pix[i+2]
			// 光标=浅黄高亮: H∈[45,70] S≥40 V≥200。RGB 阈值会漏边缘抗锯齿像素，
			// HSV 用色相主筛能稳住光标聚簇。debug/溜鱼/1080 实测：H 50-60 S 67-127 V 253-255。
			hh, ss, vv := rgbToHSV(r, g, b)
			if hh >= 45 && hh <= 70 && ss >= 40 && vv >= 200 {
				colCount[x]++
				totalYellow++
			}
		}
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
		runW := i - start
		if runW >= 2 && runW <= max(15, w/40) && sum > bestSum {
			bestSum = sum
			bestStart = start
			bestEnd = i
		}
	}

	if bestSum < 2 {
		return -1, 0, 0, totalYellow
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
	return cx, height, math.Min(1.0, conf), totalYellow
}

func findTarget(img *image.RGBA, bandY1, bandY2 int) (cx int, width int, conf float64, totalGreen int) {
	bounds := img.Bounds()
	w := bounds.Dx()

	colCount := make([]int, w)
	for y := bandY1; y < bandY2; y++ {
		off := y * img.Stride
		for x := range w {
			i := off + x*4
			r, g, b := img.Pix[i], img.Pix[i+1], img.Pix[i+2]
			// 目标条带=高饱和青色: H∈[160,180] S≥140 V≥100。原 g>=200 阈值漏掉一半边缘
			// 渐变像素（fillRatio 偏低 conf 偏低）；HSV 色相筛同时挡掉森林绿(H~100)和蓝色
			// 轨道(H~210)。debug/溜鱼/1080 实测：H 164-172 S 190-209 V 175-250。
			hh, ss, vv := rgbToHSV(r, g, b)
			if hh >= 160 && hh <= 180 && ss >= 140 && vv >= 100 {
				colCount[x]++
				totalGreen++
			}
		}
	}

	if totalGreen < 3 {
		return -1, 0, 0, totalGreen
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
		return -1, 0, 0, totalGreen
	}

	width = bestEnd - bestStart
	cx = (bestStart + bestEnd) / 2
	bandH := bandY2 - bandY1
	fillRatio := float64(bestSum) / float64(max(1, width*bandH))
	conf = math.Min(1.0, fillRatio*2.5)
	return cx, width, conf, totalGreen
}

// rgbToHSV 转发 pkg/vision.RGBToHSV，本包内 caller 保持短名。
func rgbToHSV(r, g, b uint8) (h int, s int, v int) {
	return vision.RGBToHSV(r, g, b)
}
