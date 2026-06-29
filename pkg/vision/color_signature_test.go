// pkg/vision/color_signature_test.go
package vision

import (
	"image"
	"testing"
)

// 造 20x20 RGBA: 在 (12,8) 放红 (200,30,30), 其右 (14,8) 放白 (255,255,255)。
func sigTestFrame() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 20, 20))
	for i := range img.Pix {
		img.Pix[i] = 0
	}
	set := func(x, y, r, g, b int) {
		o := img.PixOffset(x, y)
		img.Pix[o], img.Pix[o+1], img.Pix[o+2], img.Pix[o+3] = uint8(r), uint8(g), uint8(b), 255
	}
	set(12, 8, 200, 30, 30)
	set(14, 8, 255, 255, 255)
	return img
}

func TestFindColorSignature_HitWithOffset(t *testing.T) {
	f := sigTestFrame()
	sig := []ColorSigPoint{
		{DX: 0, DY: 0, R: 200, G: 30, B: 30, Tol: 8},
		{DX: 2, DY: 0, R: 255, G: 255, B: 255, Tol: 8},
	}
	found, ax, ay := FindColorSignature(f, 0, 0, 20, 20, sig)
	if !found || ax != 12 || ay != 8 {
		t.Fatalf("want (true,12,8), got (%v,%d,%d)", found, ax, ay)
	}
}

func TestFindColorSignature_OffsetOutOfFrameMiss(t *testing.T) {
	f := sigTestFrame()
	// 偏移点指向帧外 (dx=+100) → miss。
	sig := []ColorSigPoint{
		{DX: 0, DY: 0, R: 200, G: 30, B: 30, Tol: 8},
		{DX: 100, DY: 0, R: 255, G: 255, B: 255, Tol: 8},
	}
	if found, _, _ := FindColorSignature(f, 0, 0, 20, 20, sig); found {
		t.Fatal("offset out of frame should miss")
	}
}

func TestFindColorSignature_ROINonZeroOrigin(t *testing.T) {
	f := sigTestFrame()
	sig := []ColorSigPoint{{DX: 0, DY: 0, R: 200, G: 30, B: 30, Tol: 8}}
	// ROI 从 (10,5) 起, 仍应在 (12,8) 命中锚点。
	found, ax, ay := FindColorSignature(f, 10, 5, 8, 10, sig)
	if !found || ax != 12 || ay != 8 {
		t.Fatalf("want (true,12,8), got (%v,%d,%d)", found, ax, ay)
	}
}
