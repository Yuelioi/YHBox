package rhythm

import (
	"image"
	"image/color"
	"testing"
)

// makeROIImage 构造一张 *image.RGBA 模拟 capture.Capturer 给出的 mini-image：
// Rect=ROI absolute coords，Pix=ROI 像素数 × 4。
func makeROIImage(roi image.Rectangle, fill color.RGBA) *image.RGBA {
	img := &image.RGBA{
		Pix:    make([]byte, roi.Dx()*roi.Dy()*4),
		Stride: roi.Dx() * 4,
		Rect:   roi,
	}
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i+0] = fill.R
		img.Pix[i+1] = fill.G
		img.Pix[i+2] = fill.B
		img.Pix[i+3] = fill.A
	}
	return img
}

// 4 个 ROI：Track 0 填深色音符颜色，其它填亮色背景。
// 期望：ratios[0] ≥ DarkRatioThreshold，其它 < 阈值。
func TestDarkRatios_OneTrackDark(t *testing.T) {
	cfg := Configs[ResolutionKey{1920, 1080}]
	dark := color.RGBA{R: 28, G: 28, B: 28, A: 255}    // 音符暗色
	bright := color.RGBA{R: 245, G: 245, B: 245, A: 255} // 背景亮色

	imgs := []*image.RGBA{
		makeROIImage(cfg.ROIs[0], dark),
		makeROIImage(cfg.ROIs[1], bright),
		makeROIImage(cfg.ROIs[2], bright),
		makeROIImage(cfg.ROIs[3], bright),
	}
	var ratios [4]float32
	DarkRatios(imgs, &ratios)
	if ratios[0] < DarkRatioThreshold {
		t.Errorf("Track 0 全黑应 ratio≈1，得 %.3f", ratios[0])
	}
	for i := 1; i < 4; i++ {
		if ratios[i] >= DarkRatioThreshold {
			t.Errorf("Track %d 全亮应 ratio≈0，得 %.3f", i, ratios[i])
		}
	}
}

// 中间亮度 (avg=128) 高于阈值 100，不算暗。
func TestDarkRatios_MidBrightnessNotDark(t *testing.T) {
	cfg := Configs[ResolutionKey{1920, 1080}]
	mid := color.RGBA{R: 128, G: 128, B: 128, A: 255}

	imgs := []*image.RGBA{
		makeROIImage(cfg.ROIs[0], mid),
		makeROIImage(cfg.ROIs[1], mid),
		makeROIImage(cfg.ROIs[2], mid),
		makeROIImage(cfg.ROIs[3], mid),
	}
	var ratios [4]float32
	DarkRatios(imgs, &ratios)
	for i := 0; i < 4; i++ {
		if ratios[i] != 0 {
			t.Errorf("Track %d 全 128 灰应 ratio=0 (avg=128 > BrightThresh=100)，得 %.3f", i, ratios[i])
		}
	}
}

// 一半暗一半亮：ratio ≈ 0.5
func TestDarkRatios_HalfDark(t *testing.T) {
	cfg := Configs[ResolutionKey{1920, 1080}]
	roi := cfg.ROIs[0]
	dark := color.RGBA{R: 28, G: 28, B: 28, A: 255}
	bright := color.RGBA{R: 245, G: 245, B: 245, A: 255}

	img := &image.RGBA{
		Pix:    make([]byte, roi.Dx()*roi.Dy()*4),
		Stride: roi.Dx() * 4,
		Rect:   roi,
	}
	// 上半暗下半亮
	half := roi.Dy() / 2
	for j := 0; j < roi.Dy(); j++ {
		c := bright
		if j < half {
			c = dark
		}
		for i := 0; i < roi.Dx(); i++ {
			off := j*img.Stride + i*4
			img.Pix[off+0] = c.R
			img.Pix[off+1] = c.G
			img.Pix[off+2] = c.B
			img.Pix[off+3] = 255
		}
	}
	emptyROI := func(i int) *image.RGBA {
		r := cfg.ROIs[i]
		return &image.RGBA{Pix: make([]byte, r.Dx()*r.Dy()*4), Stride: r.Dx() * 4, Rect: r}
	}
	imgs := []*image.RGBA{img, emptyROI(1), emptyROI(2), emptyROI(3)}
	var ratios [4]float32
	DarkRatios(imgs, &ratios)
	expected := float32(half) / float32(roi.Dy())
	delta := ratios[0] - expected
	if delta < -0.05 || delta > 0.05 {
		t.Errorf("Track 0 半暗，期望 ratio≈%.3f, 得 %.3f", expected, ratios[0])
	}
}
