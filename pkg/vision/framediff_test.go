package vision

import (
	"image"
	"image/color"
	"testing"
)

// solidRGBA 构造 w×h 单色帧.
func solidRGBA(w, h int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

func TestDownsample_SolidToSingleCell(t *testing.T) {
	img := solidRGBA(8, 8, color.RGBA{R: 10, G: 20, B: 30, A: 255})
	sig := Downsample(img, 1)
	want := []uint8{10, 20, 30}
	if len(sig) != 3 {
		t.Fatalf("len = %d, want 3", len(sig))
	}
	for i := range want {
		if sig[i] != want[i] {
			t.Errorf("sig[%d] = %d, want %d", i, sig[i], want[i])
		}
	}
}

func TestDownsample_QuadrantsToGrid(t *testing.T) {
	// 4×4: 左上红, 右上绿, 左下蓝, 右下白. gridSize 2 → 每格 = 该象限纯色.
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	fill := func(x0, y0 int, c color.RGBA) {
		for y := y0; y < y0+2; y++ {
			for x := x0; x < x0+2; x++ {
				img.SetRGBA(x, y, c)
			}
		}
	}
	fill(0, 0, color.RGBA{R: 255, A: 255})
	fill(2, 0, color.RGBA{G: 255, A: 255})
	fill(0, 2, color.RGBA{B: 255, A: 255})
	fill(2, 2, color.RGBA{R: 255, G: 255, B: 255, A: 255})
	sig := Downsample(img, 2)
	// cell 顺序: 行主序 (左上, 右上, 左下, 右下).
	want := []uint8{255, 0, 0, 0, 255, 0, 0, 0, 255, 255, 255, 255}
	if len(sig) != len(want) {
		t.Fatalf("len = %d, want %d", len(sig), len(want))
	}
	for i := range want {
		if sig[i] != want[i] {
			t.Errorf("sig[%d] = %d, want %d", i, sig[i], want[i])
		}
	}
}

func TestGridChangedRatio_Identical(t *testing.T) {
	a := []uint8{10, 20, 30, 40, 50, 60}
	if r := GridChangedRatio(a, a, 12); r != 0 {
		t.Errorf("ratio = %v, want 0", r)
	}
}

func TestGridChangedRatio_AllChanged(t *testing.T) {
	a := []uint8{0, 0, 0, 0, 0, 0}   // 2 格
	b := []uint8{255, 255, 255, 255, 255, 255}
	if r := GridChangedRatio(a, b, 12); r != 1.0 {
		t.Errorf("ratio = %v, want 1.0", r)
	}
}

func TestGridChangedRatio_DeltaBelowThreshold(t *testing.T) {
	a := []uint8{100, 100, 100}      // 1 格
	b := []uint8{105, 105, 105}      // 每通道差 5 < cellDelta 12 → 没变
	if r := GridChangedRatio(a, b, 12); r != 0 {
		t.Errorf("ratio = %v, want 0", r)
	}
}

func TestGridChangedRatio_LengthMismatch(t *testing.T) {
	if r := GridChangedRatio([]uint8{1, 2, 3}, []uint8{1, 2, 3, 4, 5, 6}, 12); r != 1.0 {
		t.Errorf("ratio = %v, want 1.0", r)
	}
}

func TestGridMeanDiff_Identical(t *testing.T) {
	a := []uint8{10, 20, 30, 40}
	if d := GridMeanDiff(a, a); d != 0 {
		t.Errorf("diff = %v, want 0", d)
	}
}

func TestGridMeanDiff_FullSwing(t *testing.T) {
	a := []uint8{0, 0, 0}
	b := []uint8{255, 255, 255}
	if d := GridMeanDiff(a, b); d != 1.0 {
		t.Errorf("diff = %v, want 1.0", d)
	}
}

func TestGridMeanDiff_LengthMismatch(t *testing.T) {
	if d := GridMeanDiff([]uint8{1}, []uint8{1, 2}); d != 1.0 {
		t.Errorf("diff = %v, want 1.0", d)
	}
}
