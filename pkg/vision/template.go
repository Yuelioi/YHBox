// Package vision 模板匹配（CCOEFF_NORMED，等价 OpenCV TM_CCOEFF_NORMED）。
//
// 实现要点：
//   - 积分图把 patch 的 ∑I/∑I² 降到 O(1)
//   - 行级 goroutine 并行（runtime.NumCPU）
//   - MatchFast：按模板尺寸自适应下采样粗找 + 多候选原图精修
//   - 灰度用 BT.601（与 OpenCV 默认一致）
package vision

import (
	"image"
	"image/draw"
	"image/png"
	"io"
	"runtime"
	"sort"
	"sync"
)

// RGBAToGray 把 RGBA 灰度化成 [0,1] float32 数组（行主序）。
func RGBAToGray(img *image.RGBA) ([]float32, int, int) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	out := make([]float32, w*h)
	for y := 0; y < h; y++ {
		row := img.PixOffset(b.Min.X, b.Min.Y+y)
		dst := y * w
		for x := 0; x < w; x++ {
			r := img.Pix[row+x*4+0]
			g := img.Pix[row+x*4+1]
			bl := img.Pix[row+x*4+2]
			out[dst+x] = (0.299*float32(r) + 0.587*float32(g) + 0.114*float32(bl)) / 255.0
		}
	}
	return out, w, h
}

// Template 模板：预处理过的灰度数据 + 尺寸。
type Template struct {
	Gray []float32
	W, H int
}

// LoadPNG 从 io.Reader 读 PNG，灰度化成 Template。
func LoadPNG(r io.Reader) (*Template, error) {
	src, err := png.Decode(r)
	if err != nil {
		return nil, err
	}
	b := src.Bounds()
	rgba := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(rgba, rgba.Bounds(), src, b.Min, draw.Src)
	gray, w, h := RGBAToGray(rgba)
	return &Template{Gray: gray, W: w, H: h}, nil
}

// CropROI 从 RGBA 帧抠出客户区比例 ROI，返回灰度数组 + ROI 像素坐标 (x, y, w, h)。
func CropROI(frame *image.RGBA, fx, fy, fw, fh float64) ([]float32, int, int, int, int) {
	W, H := frame.Bounds().Dx(), frame.Bounds().Dy()
	x := int(fx * float64(W))
	y := int(fy * float64(H))
	w := int(fw * float64(W))
	h := int(fh * float64(H))
	roi := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(roi, roi.Bounds(), frame, image.Point{X: x, Y: y}, draw.Src)
	gray, _, _ := RGBAToGray(roi)
	return gray, x, y, w, h
}

// CropROIRGBA 从 RGBA 帧抠出客户区比例 ROI，返回 RGBA 子图。
func CropROIRGBA(frame *image.RGBA, fx, fy, fw, fh float64) *image.RGBA {
	W, H := frame.Bounds().Dx(), frame.Bounds().Dy()
	w := int(fw * float64(W))
	h := int(fh * float64(H))
	if w < 1 || h < 1 {
		return nil
	}
	x := int(fx * float64(W))
	y := int(fy * float64(H))
	roi := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(roi, roi.Bounds(), frame, image.Point{X: x, Y: y}, draw.Src)
	return roi
}

// Match CCOEFF_NORMED naive 实现，行并行。返回 ROI 内左上角坐标 + 置信度。
// 找不到（搜索空间为空 / 模板单色）返回 (-1, -1, -1)。
// 与 MatchAll 共享 correlationMap (NCC 全图)；Match = 取全局 argmax。
func Match(img []float32, iw, ih int, tpl *Template, parallel int) (int, int, float32) {
	return matchPrepared(img, iw, ih, prepareCorrelationTemplate(tpl), parallel)
}

func matchPrepared(img []float32, iw, ih int, tpl *correlationTemplate, parallel int) (int, int, float32) {
	confMap, sw, sh := correlationMapPrepared(img, iw, ih, tpl, parallel)
	if sw <= 0 || sh <= 0 {
		return -1, -1, -1
	}
	bestX, bestY := -1, -1
	best := corrSkip
	for sy := 0; sy < sh; sy++ {
		for sx := 0; sx < sw; sx++ {
			if c := confMap[sy*sw+sx]; c > best {
				best, bestX, bestY = c, sx, sy
			}
		}
	}
	if bestX < 0 || best <= corrSkip {
		return -1, -1, -1
	}
	return bestX, bestY, best
}

func matchCandidatesPrepared(img []float32, iw, ih int, tpl *correlationTemplate, parallel, limit int) []MatchHit {
	confMap, searchWidth, searchHeight := correlationMapPrepared(img, iw, ih, tpl, parallel)
	if searchWidth <= 0 || searchHeight <= 0 || limit <= 0 {
		return nil
	}
	candidates := make([]MatchHit, 0, limit)
	for y := 0; y < searchHeight; y++ {
		for x := 0; x < searchWidth; x++ {
			confidence := confMap[y*searchWidth+x]
			if confidence <= corrSkip || !isLocalMax(confMap, searchWidth, searchHeight, x, y, confidence) {
				continue
			}
			candidates = append(candidates, MatchHit{X: x, Y: y, Conf: confidence})
		}
	}
	sort.Slice(candidates, func(left, right int) bool {
		if candidates[left].Conf != candidates[right].Conf {
			return candidates[left].Conf > candidates[right].Conf
		}
		if candidates[left].Y != candidates[right].Y {
			return candidates[left].Y < candidates[right].Y
		}
		return candidates[left].X < candidates[right].X
	})
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	return candidates
}

func downsample(img []float32, w, h, factor int) ([]float32, int, int) {
	if factor <= 1 {
		out := append([]float32(nil), img...)
		return out, w, h
	}
	nw, nh := w/factor, h/factor
	out := make([]float32, nw*nh)
	for y := 0; y < nh; y++ {
		sy := y * factor
		for x := 0; x < nw; x++ {
			sx := x * factor
			var sum float32
			for offsetY := 0; offsetY < factor; offsetY++ {
				row := (sy + offsetY) * w
				for offsetX := 0; offsetX < factor; offsetX++ {
					sum += img[row+sx+offsetX]
				}
			}
			out[y*nw+x] = sum / float32(factor*factor)
		}
	}
	return out, nw, nh
}

// FastTemplate caches immutable template preprocessing shared by every fresh
// frame observation in one automation-node invocation.
type FastTemplate struct {
	full    *correlationTemplate
	coarse  *correlationTemplate
	factor  int
	scratch sync.Pool
}

const fastRefinementCandidateLimit = 4

// PrepareFastTemplate builds the reusable normalized and coarse template
// representations used by MatchFastPrepared.
func PrepareFastTemplate(tpl *Template) *FastTemplate {
	full := prepareCorrelationTemplate(tpl)
	if full == nil {
		return nil
	}
	factor := 1
	if tpl.W >= 64 && tpl.H >= 24 {
		factor = 4
	} else if tpl.W >= 16 && tpl.H >= 16 {
		factor = 2
	}
	if factor == 1 {
		return &FastTemplate{full: full, factor: 1}
	}
	gray, width, height := downsample(tpl.Gray, tpl.W, tpl.H, factor)
	coarse := prepareCorrelationTemplate(&Template{Gray: gray, W: width, H: height})
	if coarse == nil || width < 6 || height < 6 {
		return &FastTemplate{full: full, factor: 1}
	}
	return &FastTemplate{full: full, coarse: coarse, factor: factor}
}

// MatchFast uses an adaptive coarse pass followed by a bounded
// full-resolution refinement. Small search spaces fall back to Match.
func MatchFast(img []float32, iw, ih int, tpl *Template, parallel int) (int, int, float32) {
	return MatchFastPrepared(img, iw, ih, PrepareFastTemplate(tpl), parallel)
}

// MatchFastPrepared matches a fresh image against a reusable prepared
// template. Large templates use a 4x coarse pass followed by a bounded
// full-resolution refinement; smaller templates keep the existing 2x path.
func MatchFastPrepared(img []float32, iw, ih int, prepared *FastTemplate, parallel int) (int, int, float32) {
	if prepared == nil || prepared.full == nil {
		return -1, -1, -1
	}
	tpl := prepared.full.template
	tw, th := tpl.W, tpl.H
	factor := prepared.factor
	if factor <= 1 || iw-tw < factor*2 || ih-th < factor*2 {
		return matchPrepared(img, iw, ih, prepared.full, parallel)
	}
	coarseImage, coarseWidth, coarseHeight := downsample(img, iw, ih, factor)
	coarseCandidates := matchCandidatesPrepared(coarseImage, coarseWidth, coarseHeight, prepared.coarse, parallel, fastRefinementCandidateLimit)
	if len(coarseCandidates) == 0 {
		return -1, -1, -1
	}
	bestX, bestY, bestConfidence := -1, -1, float32(-1)
	workspace, _ := prepared.scratch.Get().(*correlationWorkspace)
	if workspace == nil {
		workspace = &correlationWorkspace{}
	}
	defer prepared.scratch.Put(workspace)
	for _, candidate := range coarseCandidates {
		x, y, confidence := refineFastCandidate(img, iw, ih, prepared.full, candidate.X*factor, candidate.Y*factor, factor*2, parallel, workspace)
		if confidence > bestConfidence || (confidence == bestConfidence && (y < bestY || (y == bestY && x < bestX))) {
			bestX, bestY, bestConfidence = x, y, confidence
		}
	}
	return bestX, bestY, bestConfidence
}

func refineFastCandidate(img []float32, iw, ih int, template *correlationTemplate, fx, fy, pad, parallel int, workspace *correlationWorkspace) (int, int, float32) {
	tw, th := template.template.W, template.template.H
	x0 := fx - pad
	if x0 < 0 {
		x0 = 0
	}
	y0 := fy - pad
	if y0 < 0 {
		y0 = 0
	}
	x1 := fx + pad
	if x1+tw > iw {
		x1 = iw - tw
	}
	y1 := fy + pad
	if y1+th > ih {
		y1 = ih - th
	}
	rw := x1 + tw - x0
	rh := y1 + th - y0
	if rw <= 0 || rh <= 0 || rw < tw || rh < th {
		return fx, fy, -1
	}

	workspace.region = resizeFloat32(workspace.region, rw*rh)
	region := workspace.region
	for ry := 0; ry < rh; ry++ {
		copy(region[ry*rw:(ry+1)*rw], img[(y0+ry)*iw+x0:(y0+ry)*iw+x0+rw])
	}
	bx, by, bc := matchPreparedWithWorkspace(region, rw, rh, template, parallel, workspace)
	if bx < 0 {
		return -1, -1, -1
	}
	return x0 + bx, y0 + by, bc
}

// ResizeGray 双线性把灰度数组从 (sw,sh) 缩放到 (dw,dh)。
// 用于多尺度匹配：模板缩放后再做 NCC。
func ResizeGray(src []float32, sw, sh, dw, dh int) []float32 {
	if dw == sw && dh == sh {
		out := make([]float32, len(src))
		copy(out, src)
		return out
	}
	out := make([]float32, dw*dh)
	if sw < 2 || sh < 2 {
		// 退化场景：源太小，最近邻填充
		for y := 0; y < dh; y++ {
			sy := y * sh / dh
			for x := 0; x < dw; x++ {
				sx := x * sw / dw
				out[y*dw+x] = src[sy*sw+sx]
			}
		}
		return out
	}
	fx := float32(sw) / float32(dw)
	fy := float32(sh) / float32(dh)
	for y := 0; y < dh; y++ {
		syf := (float32(y)+0.5)*fy - 0.5
		y0 := int(syf)
		wy := syf - float32(y0)
		if y0 < 0 {
			y0 = 0
			wy = 0
		} else if y0 > sh-2 {
			y0 = sh - 2
			wy = 1
		}
		row0 := y0 * sw
		row1 := (y0 + 1) * sw
		for x := 0; x < dw; x++ {
			sxf := (float32(x)+0.5)*fx - 0.5
			x0 := int(sxf)
			wx := sxf - float32(x0)
			if x0 < 0 {
				x0 = 0
				wx = 0
			} else if x0 > sw-2 {
				x0 = sw - 2
				wx = 1
			}
			v00 := src[row0+x0]
			v01 := src[row0+x0+1]
			v10 := src[row1+x0]
			v11 := src[row1+x0+1]
			top := v00*(1-wx) + v01*wx
			bot := v10*(1-wx) + v11*wx
			out[y*dw+x] = top*(1-wy) + bot*wy
		}
	}
	return out
}

// ScaleTemplate 用 ResizeGray 把模板按 scale 缩放，返回新 Template。
// scale=1.0 直接复制；过小（<8px）返回 nil。
func ScaleTemplate(tpl *Template, scale float32) *Template {
	tw := int(float32(tpl.W)*scale + 0.5)
	th := int(float32(tpl.H)*scale + 0.5)
	if tw < 8 || th < 8 {
		return nil
	}
	g := ResizeGray(tpl.Gray, tpl.W, tpl.H, tw, th)
	return &Template{Gray: g, W: tw, H: th}
}

// BuildScalePyramid 按线性步长在 [low, high] 区间生成 steps 个尺度模板。
// 用于一次性预生成多尺度候选，避免每帧重复 resize。
// 过小的尺度（<8px）会被跳过。
func BuildScalePyramid(tpl *Template, low, high float32, steps int) []*Template {
	if steps < 1 {
		return []*Template{tpl}
	}
	out := make([]*Template, 0, steps)
	for i := 0; i < steps; i++ {
		var s float32
		if steps == 1 {
			s = (low + high) / 2
		} else {
			s = low + (high-low)*float32(i)/float32(steps-1)
		}
		t := ScaleTemplate(tpl, s)
		if t != nil {
			out = append(out, t)
		}
	}
	return out
}

// MatchBest 在多张候选模板里取置信度最高的命中（如 hover/未 hover 两态）。
// 返回 (best ROI 内 x, y, conf, 命中的模板索引)。所有模板都未命中返回 (-1,-1,-1,-1)。
func MatchBest(img []float32, iw, ih int, tpls []*Template, parallel int) (int, int, float32, int) {
	bestX, bestY, bestConf, bestIdx := -1, -1, float32(-1), -1
	for i, t := range tpls {
		x, y, c := MatchFast(img, iw, ih, t, parallel)
		if c > bestConf {
			bestX, bestY, bestConf, bestIdx = x, y, c, i
		}
	}
	if bestIdx < 0 || bestConf < 0 {
		return -1, -1, -1, -1
	}
	return bestX, bestY, bestConf, bestIdx
}

// MatchBestEarlyExit 与 MatchBest 相同，但 conf >= earlyAccept 时立刻返回。
func MatchBestEarlyExit(img []float32, iw, ih int, tpls []*Template, parallel int, earlyAccept float32) (int, int, float32, int) {
	bestX, bestY, bestConf, bestIdx := -1, -1, float32(-1), -1
	for i, t := range tpls {
		x, y, c := MatchFast(img, iw, ih, t, parallel)
		if c > bestConf {
			bestX, bestY, bestConf, bestIdx = x, y, c, i
			if c >= earlyAccept {
				return bestX, bestY, bestConf, bestIdx
			}
		}
	}
	if bestIdx < 0 || bestConf < 0 {
		return -1, -1, -1, -1
	}
	return bestX, bestY, bestConf, bestIdx
}

// DefaultParallel 返回匹配并行度，限制到最多 4 线程以降低 CPU 占用。
func DefaultParallel() int {
	n := runtime.NumCPU() / 2
	if n > 4 {
		n = 4
	}
	if n < 1 {
		n = 1
	}
	return n
}
