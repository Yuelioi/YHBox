package vision

import (
	"image"
	"image/color"
	"testing"
)

func TestAnalyzeDualColorBarFindsColumnClustersInSourceCoordinates(t *testing.T) {
	frame := image.NewRGBA(image.Rect(0, 0, 160, 40))
	for y := 12; y < 15; y++ {
		for x := 50; x < 91; x++ {
			frame.SetRGBA(x, y, color.RGBA{B: 255, A: 255})
		}
	}
	for y := 16; y < 22; y++ {
		for x := 62; x < 65; x++ {
			frame.SetRGBA(x, y, color.RGBA{R: 255, A: 255})
		}
	}
	result := AnalyzeDualColorBar(
		frame,
		image.Rect(20, 10, 120, 24),
		ColorRange{Space: "rgb", Minimum: [3]int{250, 0, 0}, Maximum: [3]int{255, 5, 5}},
		ColorRange{Space: "rgb", Minimum: [3]int{0, 0, 250}, Maximum: [3]int{5, 5, 255}},
		DualColorBarOptions{},
	)
	if !result.Found || result.InnerX != 63 || result.OuterX != 70 || result.OuterWidth != 41 {
		t.Fatalf("result = %#v", result)
	}
	if result.InnerPixels != 18 || result.OuterPixels != 123 || result.Confidence <= 0 || result.Confidence > 0.98 {
		t.Fatalf("metrics = %#v", result)
	}
}

func TestAnalyzeDualColorBarReportsMissingOuterCluster(t *testing.T) {
	frame := image.NewRGBA(image.Rect(0, 0, 80, 20))
	for y := 7; y < 12; y++ {
		for x := 30; x < 33; x++ {
			frame.SetRGBA(x, y, color.RGBA{R: 255, A: 255})
		}
	}
	result := AnalyzeDualColorBar(
		frame,
		frame.Bounds(),
		ColorRange{Space: "rgb", Minimum: [3]int{250, 0, 0}, Maximum: [3]int{255, 5, 5}},
		ColorRange{Space: "rgb", Minimum: [3]int{0, 0, 250}, Maximum: [3]int{5, 5, 255}},
		DualColorBarOptions{},
	)
	if result.Found || result.InnerX < 0 || result.OuterX != -1 {
		t.Fatalf("result = %#v", result)
	}
}
