package imageutil

import (
	"image"
	"image/color"
	"testing"
)

func TestScaleRGBA_2x(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			src.Set(x, y, color.RGBA{R: 255, A: 255})
		}
	}
	dst := ScaleRGBA(src, 20, 20)
	if dst.Bounds().Dx() != 20 || dst.Bounds().Dy() != 20 {
		t.Errorf("size = %v, want 20x20", dst.Bounds())
	}
	c := dst.RGBAAt(10, 10)
	if c.R < 200 {
		t.Errorf("center pixel R = %d, expect > 200", c.R)
	}
}
