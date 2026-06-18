// pkg/vision/color_signature.go
// FindColorSignature — 在 ROI 矩形内 (锚点搜索区) 行主序找首个完整签名命中。
// 锚点色命中的像素上验全部偏移点 (偏移点采样整帧, 越界=miss=整签名失败, 继续搜)。
// 纯基元类型, 不 import internal/node。spec §节点1。
package vision

import "image"

// ColorSigPoint 一个签名点, Tol 已被 caller 解析成具体值 (不再 nullable)。
type ColorSigPoint struct {
	DX, DY  int
	R, G, B int
	Tol     int
}

// chMatch 逐通道绝对差 ≤ tol (只比 RGB)。
func chMatch(r, g, b uint8, p ColorSigPoint) bool {
	return absI(int(r)-p.R) <= p.Tol && absI(int(g)-p.G) <= p.Tol && absI(int(b)-p.B) <= p.Tol
}

func absI(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// FindColorSignature: rx,ry,rw,rh = 锚点搜索区像素矩形 (半开 [rx,rx+rw)×[ry,ry+rh))。
// sig[0] = 锚点 (DX=DY=0)。返回锚点像素坐标 (全帧)。
func FindColorSignature(frame *image.RGBA, rx, ry, rw, rh int, sig []ColorSigPoint) (bool, int, int) {
	if len(sig) == 0 {
		return false, 0, 0
	}
	b := frame.Bounds()
	x0, y0 := maxI(rx, b.Min.X), maxI(ry, b.Min.Y)
	x1, y1 := minI(rx+rw, b.Max.X), minI(ry+rh, b.Max.Y)
	anchor := sig[0]
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			o := frame.PixOffset(x, y)
			if !chMatch(frame.Pix[o], frame.Pix[o+1], frame.Pix[o+2], anchor) {
				continue
			}
			if verifyOffsets(frame, x, y, sig) {
				return true, x, y
			}
		}
	}
	return false, 0, 0
}

// verifyOffsets 验 sig[1:] 各偏移点 (采样整帧; 越界即 false)。
func verifyOffsets(frame *image.RGBA, ax, ay int, sig []ColorSigPoint) bool {
	b := frame.Bounds()
	for _, p := range sig[1:] {
		px, py := ax+p.DX, ay+p.DY
		if px < b.Min.X || px >= b.Max.X || py < b.Min.Y || py >= b.Max.Y {
			return false
		}
		o := frame.PixOffset(px, py)
		if !chMatch(frame.Pix[o], frame.Pix[o+1], frame.Pix[o+2], p) {
			return false
		}
	}
	return true
}

func maxI(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minI(a, b int) int {
	if a < b {
		return a
	}
	return b
}
