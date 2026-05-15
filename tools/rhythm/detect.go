package rhythm

import "image"

// DarkRatios 算 4 个 ROI 的"暗像素占比"，结果写进 out[i] (0.0~1.0)。
//
// 算法：每像素取 (R+G+B)/3 平均亮度，亮度 < BrightThresh 算"暗"。整 ROI 内暗像素
// 占比 ≥ DarkRatioThreshold 就视为"音符到了"。
//
// 为什么不用 HSV 颜色匹配（旧版方案）：
//   - HSV 阈值边界敏感：音符进入 ROI 边缘时颜色未饱和、anti-aliasing 让 cnt 卡边界
//   - 异环音游背景≈245（亮鼓面），音符≈28（深色音符）—— 亮度落差极大，对 AA 不敏感
//   - ok-nte 同款方案实测 100% 命中
//
// imgs[i] 是 Track i 的 ROI mini-image (capture.Capturer 提供，尺寸 = ROI)。
// 不 alloc，out 由 caller 持有。
func DarkRatios(imgs []*image.RGBA, out *[4]float32) {
	for tr := 0; tr < 4; tr++ {
		img := imgs[tr]
		var dark, total int
		// 直接走 .Pix 字节流；ROI mini-image stride = Dx*4 没 padding
		for j := 0; j < len(img.Pix); j += 4 {
			avg := (int(img.Pix[j]) + int(img.Pix[j+1]) + int(img.Pix[j+2])) / 3
			if avg < BrightThresh {
				dark++
			}
			total++
		}
		if total == 0 {
			out[tr] = 0
			continue
		}
		out[tr] = float32(dark) / float32(total)
	}
}
