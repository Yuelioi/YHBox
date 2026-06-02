package runtime

import (
	"yotta/internal/node"
)

// ResolveGeometry 把 Geometry (pct + overrides) 按当前帧尺寸还原成像素 ROI.
//
// 优先级: override 精确匹配当前分辨率 > pct×帧尺寸 > 全帧 (w==0||h==0, 忽略 x/y).
// fullFrame=true 时返回 (0,0,frameW,frameH), 调用方可跳过裁剪直接用全帧.
func ResolveGeometry(g node.Geometry, frameW, frameH int) (x, y, w, h int, fullFrame bool) {
	for _, ov := range g.Overrides {
		if ov.Resolution.W == frameW && ov.Resolution.H == frameH {
			x, y, w, h = clampRect(ov.Px.X, ov.Px.Y, ov.Px.W, ov.Px.H, frameW, frameH)
			return x, y, w, h, false
		}
	}
	if g.Pct.W == 0 || g.Pct.H == 0 {
		return 0, 0, frameW, frameH, true
	}
	x = int(g.Pct.X * float64(frameW))
	y = int(g.Pct.Y * float64(frameH))
	w = int(g.Pct.W * float64(frameW))
	h = int(g.Pct.H * float64(frameH))
	x, y, w, h = clampRect(x, y, w, h, frameW, frameH)
	return x, y, w, h, false
}

// clampRect 确保矩形不超出帧边界; 不打日志 (越界是正常配置输入, 上层负责校验).
func clampRect(x, y, w, h, fw, fh int) (int, int, int, int) {
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	if x+w > fw {
		w = fw - x
	}
	if y+h > fh {
		h = fh - y
	}
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	return x, y, w, h
}
