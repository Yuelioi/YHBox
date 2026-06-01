package runtime

import (
	"image"
	"testing"
)

func TestCountColorPixels_RGB(t *testing.T) {
	// 4x1 帧: 2 红 2 黑. RGB 区间只命中红.
	img := image.NewRGBA(image.Rect(0, 0, 4, 1))
	red := []byte{255, 0, 0, 255}
	copy(img.Pix[0:], red)
	copy(img.Pix[4:], red)
	// x=2,3 留黑 (0,0,0).
	count, sumX, sumY := countColorPixels(img, 0, 0, 4, 1, "rgb", [6]int{200, 255, 0, 50, 0, 50})
	if count != 2 {
		t.Fatalf("count = %d, want 2", count)
	}
	if sumX != 0+1 || sumY != 0 {
		t.Fatalf("sumX,sumY = %d,%d, want 1,0", sumX, sumY)
	}
}

func TestCountColorPixels_Rect(t *testing.T) {
	// 全红 4x4, 只数 [1,1)-(3,3) 子矩形 = 4 像素.
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+3] = 255, 255
	}
	count, _, _ := countColorPixels(img, 1, 1, 3, 3, "rgb", [6]int{200, 255, 0, 50, 0, 50})
	if count != 4 {
		t.Fatalf("count = %d, want 4", count)
	}
}
