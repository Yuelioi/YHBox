package vision

import "math"

// RGBToHSV 把 8-bit RGB 转 HSV。返回 H ∈ [0,360]，S/V ∈ [0,100]。
//
// 跟主流取色器 (Photoshop/Windows/Figma) 对齐：H 满程角度、S/V 百分比，用户截图取色
// 读到的值能直接填进 DetectColor 的 hsv 阈值。统一一份实现避免漂移：H 算法若漏 60 倍
// 精度，会让阈值对不上取色工具调出的值。
func RGBToHSV(r, g, b uint8) (h int, s int, v int) {
	rf := float64(r) / 255.0
	gf := float64(g) / 255.0
	bf := float64(b) / 255.0
	maxC := math.Max(rf, math.Max(gf, bf))
	minC := math.Min(rf, math.Min(gf, bf))
	delta := maxC - minC

	v = int(maxC * 100)
	if maxC == 0 {
		return 0, 0, v
	}
	s = int((delta / maxC) * 100)

	var hf float64
	switch {
	case delta == 0:
		hf = 0
	case maxC == rf:
		hf = 60 * math.Mod((gf-bf)/delta, 6)
	case maxC == gf:
		hf = 60 * ((bf-rf)/delta + 2)
	default:
		hf = 60 * ((rf-gf)/delta + 4)
	}
	if hf < 0 {
		hf += 360
	}
	h = int(hf)
	return
}
