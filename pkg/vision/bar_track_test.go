package vision

import (
	"image"
	"image/color"
	"testing"
)

// TestAnalyzeBar_EmptyImage: 全黑帧 → 不返 cursor/target.
func TestAnalyzeBar_EmptyImage(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 200, 20))
	// 全黑默认
	result := AnalyzeBar(img)
	if result == nil {
		return // OK, w<10 || h<4 时返 nil
	}
	if result.CursorX >= 0 && result.TargetX >= 0 {
		t.Errorf("空帧不应找到 cursor+target, got: %+v", result)
	}
}

// TestAnalyzeBar_SyntheticCursorAndTarget: 合成有黄 cursor + 青 target 的帧,
// 期望 conf >= 0.3 且 cursorX/targetX 在合理范围.
func TestAnalyzeBar_SyntheticCursorAndTarget(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 200, 20))
	// 画黄 cursor (HSV H~55) 在 x=50, w=3, full height
	cursor := color.RGBA{R: 255, G: 240, B: 60, A: 255} // H≈55 S≈76 V≈100
	for y := range 20 {
		for x := 49; x <= 51; x++ {
			img.Set(x, y, cursor)
		}
	}
	// 画青 target (HSV H~172) 在 x=100, w=15, 中段
	target := color.RGBA{R: 0, G: 230, B: 200, A: 255} // H=172 S=255 V=230; 在 H[160,180] S>=140 V>=100 范围内
	for y := 5; y < 15; y++ {
		for x := 100; x <= 114; x++ {
			img.Set(x, y, target)
		}
	}
	result := AnalyzeBar(img)
	if result == nil {
		t.Fatal("合成图应找到 bar")
	}
	if result.CursorX < 49 || result.CursorX > 51 {
		t.Errorf("cursor X 偏离, got %d, want ~50", result.CursorX)
	}
	if result.TargetX < 100 || result.TargetX > 114 {
		t.Errorf("target X 偏离, got %d, want ~107", result.TargetX)
	}
	if result.Confidence < 0.3 {
		t.Errorf("合成图 conf 应 >0.3, got %.2f", result.Confidence)
	}
}

// TestAnalyzeBar_CursorOnly: 只有黄无青 → CursorX 找到, TargetX = -1.
func TestAnalyzeBar_CursorOnly(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 200, 20))
	cursor := color.RGBA{R: 255, G: 240, B: 60, A: 255}
	for y := range 20 {
		for x := 49; x <= 51; x++ {
			img.Set(x, y, cursor)
		}
	}
	result := AnalyzeBar(img)
	if result == nil {
		t.Fatal("应返非 nil (cursor 存在)")
	}
	if result.CursorX < 0 {
		t.Errorf("cursor 应找到, got CursorX=%d", result.CursorX)
	}
	if result.TargetX >= 0 {
		t.Errorf("target 应找不到, got TargetX=%d", result.TargetX)
	}
	if result.YellowPx == 0 {
		t.Errorf("YellowPx 应 > 0")
	}
}
