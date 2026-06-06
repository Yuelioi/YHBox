package vision

import (
	"image"
	"image/color"
	"testing"
)

// 示例 HSV 阈值 — 给 test 用.
var (
	fishingCursorHSV = HSVRange{HMin: 45, HMax: 70, SMin: 16, SMax: 100, VMin: 78, VMax: 100}
	fishingTargetHSV = HSVRange{HMin: 160, HMax: 180, SMin: 55, SMax: 100, VMin: 39, VMax: 100}
)

// TestAnalyzeDualColorBar_EmptyImage: 全黑帧 → 不返 inner/outer.
func TestAnalyzeDualColorBar_EmptyImage(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 200, 20))
	// 全黑默认
	result := AnalyzeDualColorBar(img, fishingCursorHSV, fishingTargetHSV, DualBarOptions{})
	if result == nil {
		return // OK, w<10 || h<4 时返 nil
	}
	if result.InnerX >= 0 && result.OuterX >= 0 {
		t.Errorf("空帧不应找到 inner+outer, got: %+v", result)
	}
}

// TestAnalyzeDualColorBar_SyntheticInnerAndOuter: 合成有黄 inner + 青 outer 的帧
// (fishing 默认阈值), 期望 conf >= 0.3 且 InnerX/OuterX 在合理范围.
func TestAnalyzeDualColorBar_SyntheticInnerAndOuter(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 200, 20))
	// 画黄 inner (HSV H~55) 在 x=50, w=3, full height
	inner := color.RGBA{R: 255, G: 240, B: 60, A: 255} // H≈55 S≈76 V≈100
	for y := range 20 {
		for x := 49; x <= 51; x++ {
			img.Set(x, y, inner)
		}
	}
	// 画青 outer (HSV H~172) 在 x=100, w=15, 中段
	outer := color.RGBA{R: 0, G: 230, B: 200, A: 255} // H=172 S=100 V=90; 在 H[160,180] S>=55 V>=39 范围内
	for y := 5; y < 15; y++ {
		for x := 100; x <= 114; x++ {
			img.Set(x, y, outer)
		}
	}
	result := AnalyzeDualColorBar(img, fishingCursorHSV, fishingTargetHSV, DualBarOptions{})
	if result == nil {
		t.Fatal("合成图应找到 bar")
	}
	if result.InnerX < 49 || result.InnerX > 51 {
		t.Errorf("inner X 偏离, got %d, want ~50", result.InnerX)
	}
	if result.OuterX < 100 || result.OuterX > 114 {
		t.Errorf("outer X 偏离, got %d, want ~107", result.OuterX)
	}
	if result.Confidence < 0.3 {
		t.Errorf("合成图 conf 应 >0.3, got %.2f", result.Confidence)
	}
}

// TestAnalyzeDualColorBar_InnerOnly: 只有 inner 无 outer → InnerX 找到, OuterX = -1.
func TestAnalyzeDualColorBar_InnerOnly(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 200, 20))
	inner := color.RGBA{R: 255, G: 240, B: 60, A: 255}
	for y := range 20 {
		for x := 49; x <= 51; x++ {
			img.Set(x, y, inner)
		}
	}
	result := AnalyzeDualColorBar(img, fishingCursorHSV, fishingTargetHSV, DualBarOptions{})
	if result == nil {
		t.Fatal("应返非 nil (inner 存在)")
	}
	if result.InnerX < 0 {
		t.Errorf("inner 应找到, got InnerX=%d", result.InnerX)
	}
	if result.OuterX >= 0 {
		t.Errorf("outer 应找不到, got OuterX=%d", result.OuterX)
	}
	if result.InnerPx == 0 {
		t.Errorf("InnerPx 应 > 0")
	}
}

// TestAnalyzeDualColorBar_CustomOptions: 自定义 InnerMaxPx 限制让宽 inner 不命中.
func TestAnalyzeDualColorBar_CustomOptions(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 200, 20))
	inner := color.RGBA{R: 255, G: 240, B: 60, A: 255}
	// 画 8 px 宽 inner
	for y := range 20 {
		for x := 46; x <= 53; x++ {
			img.Set(x, y, inner)
		}
	}
	// 默认 Options 应该能命中 (InnerMaxPx default = max(15, w/40) = 15, 8 <= 15)
	r1 := AnalyzeDualColorBar(img, fishingCursorHSV, fishingTargetHSV, DualBarOptions{})
	if r1 == nil || r1.InnerX < 0 {
		t.Errorf("default Options 应命中 8px inner, got %+v", r1)
	}
	// 设 InnerMaxPx=5, 8 > 5 不命中
	r2 := AnalyzeDualColorBar(img, fishingCursorHSV, fishingTargetHSV, DualBarOptions{InnerMaxPx: 5})
	if r2 != nil && r2.InnerX >= 0 {
		t.Errorf("InnerMaxPx=5 不应命中 8px inner, got %+v", r2)
	}
}
