package runtime

import (
	"image"
	"testing"
)

// fillRect 在 RGBA 上把 [x0,x1)×[y0,y1) 涂成 (r,g,b)。
func fillRect(img *image.RGBA, x0, y0, x1, y1 int, r, g, b uint8) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			i := y*img.Stride + x*4
			img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = r, g, b, 255
		}
	}
}

// 纯红 RGB 范围 [200..255, 0..60, 0..60]
var redRGB = [6]int{200, 255, 0, 60, 0, 60}

func TestFindColorBlobs_TwoSeparateBlobs(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	fillRect(img, 10, 10, 20, 20, 255, 0, 0) // 10x10 块
	fillRect(img, 60, 60, 64, 64, 255, 0, 0) // 4x4 块
	blobs := findColorBlobs(img, 0, 0, 100, 100, "rgb", redRGB, 1)
	if len(blobs) != 2 {
		t.Fatalf("got %d blobs, want 2", len(blobs))
	}
	var big blobPx
	for _, b := range blobs {
		if b.Area > big.Area {
			big = b
		}
	}
	if big.Area != 100 || big.X != 10 || big.Y != 10 || big.W != 10 || big.H != 10 {
		t.Errorf("big blob = %+v, want Area100 bbox(10,10,10,10)", big)
	}
	if big.CenterX != 14 || big.CenterY != 14 {
		t.Errorf("big centroid = (%d,%d), want (14,14)", big.CenterX, big.CenterY)
	}
}

func TestFindColorBlobs_8ConnDiagonalMerges(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 20, 20))
	fillRect(img, 5, 5, 6, 6, 255, 0, 0)
	fillRect(img, 6, 6, 7, 7, 255, 0, 0)
	blobs := findColorBlobs(img, 0, 0, 20, 20, "rgb", redRGB, 1)
	if len(blobs) != 1 {
		t.Fatalf("got %d blobs, want 1 (8-邻域对角合并)", len(blobs))
	}
}

func TestFindColorBlobs_MinAreaFilter(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 50, 50))
	fillRect(img, 10, 10, 20, 20, 255, 0, 0)
	fillRect(img, 30, 30, 32, 31, 255, 0, 0)
	blobs := findColorBlobs(img, 0, 0, 50, 50, "rgb", redRGB, 10)
	if len(blobs) != 1 {
		t.Fatalf("got %d blobs, want 1 (minArea=10 滤掉 2px 噪点)", len(blobs))
	}
}

func TestFindColorBlobs_ROIOffsetFullFrameCoords(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	fillRect(img, 70, 70, 80, 80, 255, 0, 0)
	blobs := findColorBlobs(img, 60, 60, 100, 100, "rgb", redRGB, 1)
	if len(blobs) != 1 {
		t.Fatalf("got %d blobs, want 1", len(blobs))
	}
	if blobs[0].X != 70 || blobs[0].Y != 70 {
		t.Errorf("bbox origin = (%d,%d), want 全帧 (70,70) 非 ROI 局部", blobs[0].X, blobs[0].Y)
	}
}

func TestFindColorBlobs_EmptyROI(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 50, 50))
	if got := findColorBlobs(img, 10, 10, 10, 5, "rgb", redRGB, 1); got != nil {
		t.Errorf("w/h<=0 应返回 nil, got %v", got)
	}
}
