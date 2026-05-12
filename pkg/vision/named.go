// Package vision: 命名模板加载 + ROI 匹配。
//
// 模板元数据来自 assets/templates.toml（embed 进 exe），见 spec.go。
// 文件名只是引用 key，不再编码语义。
package vision

import (
	"bytes"
	"image"
	"image/draw"
	"image/png"
	"io"
)

// LoadPNGFromReader 把 PNG 解码成 RGBA。
func LoadPNGFromReader(r io.Reader) (*image.RGBA, error) {
	src, err := png.Decode(r)
	if err != nil {
		return nil, err
	}
	b := src.Bounds()
	rgba := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(rgba, rgba.Bounds(), src, b.Min, draw.Src)
	return rgba, nil
}

// NamedTemplate 把模板像素 + 元数据（来自 templates.toml 的 TemplateSpec）打包。
type NamedTemplate struct {
	Spec  TemplateSpec // 来自 templates.toml 的元数据
	RGBA  *image.RGBA  // 原始模板像素
	Tpl   *Template    // 灰度化好的，scale=1（基准 WxH）
	BaseW int          // 基准宽度（= Spec.Resolution[0]）
	BaseH int          // 基准高度（= Spec.Resolution[1]）
	BBox  [4]int       // 复制 Spec.BBox 方便访问
}

// HasBBox returns true（按设计 spec 必须有 bbox，永远 true，留给 MatchTextROI 兼容用）。
func (nt *NamedTemplate) HasBBox() bool { return true }

// BuildNamedTemplate 从 spec + PNG 字节构造 NamedTemplate。
func BuildNamedTemplate(spec TemplateSpec, pngData []byte) (*NamedTemplate, error) {
	rgba, err := LoadPNGFromReader(bytes.NewReader(pngData))
	if err != nil {
		return nil, err
	}
	gray, w, h := RGBAToGray(rgba)
	return &NamedTemplate{
		Spec:  spec,
		RGBA:  rgba,
		Tpl:   &Template{Gray: gray, W: w, H: h},
		BaseW: spec.Resolution[0],
		BaseH: spec.Resolution[1],
		BBox:  spec.BBox,
	}, nil
}

// PickBest 返回与 (frameW, frameH) 精确匹配 Resolution 的模板。
// 无精确匹配返回 nil — 早期阶段强制资源齐全，不再做"最近邻 + ScaleTemplate"兜底。
// 缺新分辨率请提交模板给作者添加。
func PickBest(tpls []*NamedTemplate, frameW, frameH int) *NamedTemplate {
	if len(tpls) == 0 {
		return nil
	}
	for _, t := range tpls {
		if t.BaseW == frameW && t.BaseH == frameH {
			return t
		}
	}
	return nil
}

// PickBestByH 兼容老 API（vision 内部其他地方可能还用）。
// 16:9 推断 frameW 再走 PickBest，依然要求精确匹配。
func PickBestByH(tpls []*NamedTemplate, frameH int) *NamedTemplate {
	return PickBest(tpls, frameH*16/9, frameH)
}

// MatchTextROI 在原始全帧上按命名模板的 bbox（按当前分辨率自动缩放）+ 一定冗余抠 ROI，
// 跑 NCC 匹配，返回 (conf, 命中位置在客户区像素坐标)。
//
//   - frame:  当前游戏全帧
//   - nt:     已加载的命名模板（必须 BaseH>0）
//   - padPx:  ROI 边界相对模板上下左右的扩边像素数（按当前分辨率）
//   - parallel: NCC 并行度
//
// 使用场景：上钩文字检测——模板 bbox 给定"应该出现的位置"，
// ROI 在那个位置周围搜，避免全帧扫描的误匹配 + 算力浪费。
func MatchTextROI(frame *image.RGBA, nt *NamedTemplate, padPx, parallel int) (conf float32, clientX, clientY int) {
	if nt == nil || nt.BaseH <= 0 {
		return -1, -1, -1
	}
	fb := frame.Bounds()
	frameH := fb.Dy()
	// PickBest 已保证 BaseH/BaseW 精确匹配，无需缩放。
	if nt.BaseH != frameH {
		return -1, -1, -1 // 防御性：理论上不会发生
	}
	tplScaled := nt.Tpl
	bx1 := nt.BBox[0]
	by1 := nt.BBox[1]

	rx := bx1 - padPx
	ry := by1 - padPx
	rw := tplScaled.W + 2*padPx
	rh := tplScaled.H + 2*padPx
	if rx < 0 {
		rw += rx
		rx = 0
	}
	if ry < 0 {
		rh += ry
		ry = 0
	}
	if rx+rw > fb.Dx() {
		rw = fb.Dx() - rx
	}
	if ry+rh > fb.Dy() {
		rh = fb.Dy() - ry
	}
	if rw < tplScaled.W || rh < tplScaled.H {
		return -1, -1, -1
	}

	roiRGBA := image.NewRGBA(image.Rect(0, 0, rw, rh))
	draw.Draw(roiRGBA, roiRGBA.Bounds(), frame, image.Point{X: rx, Y: ry}, draw.Src)
	roiGray, riw, rih := RGBAToGray(roiRGBA)

	x, y, c := MatchFast(roiGray, riw, rih, tplScaled, parallel)
	if x < 0 {
		return -1, -1, -1
	}
	return c, x + rx + tplScaled.W/2, y + ry + tplScaled.H/2
}
