package vision

import "math"

// RGBToHSV 把 8-bit RGB 转 HSV。返回 H ∈ [0,360]，S/V ∈ [0,255]。
//
// 之前在 action_adapters.go / tools/fish/bar.go / internal/services/tools/pixel.go
// 各有一份实现；同时存在容易漂——比如有版本算 H 时漏了 60 倍精度，曾导致
// DetectColor 节点的 hsv 阈值对不上钓鱼脚本调出的阈值。统一在这里。
func RGBToHSV(r, g, b uint8) (h int, s int, v int) {
	rf := float64(r) / 255.0
	gf := float64(g) / 255.0
	bf := float64(b) / 255.0
	maxC := math.Max(rf, math.Max(gf, bf))
	minC := math.Min(rf, math.Min(gf, bf))
	delta := maxC - minC

	v = int(maxC * 255)
	if maxC == 0 {
		return 0, 0, v
	}
	s = int((delta / maxC) * 255)

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
