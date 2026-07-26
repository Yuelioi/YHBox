package vision

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
	"testing"
)

func TestTemplateImagePreparation(t *testing.T) {
	frame := image.NewRGBA(image.Rect(5, 7, 9, 11))
	for y := frame.Bounds().Min.Y; y < frame.Bounds().Max.Y; y++ {
		for x := frame.Bounds().Min.X; x < frame.Bounds().Max.X; x++ {
			frame.SetRGBA(x, y, color.RGBA{R: uint8(x * 10), G: uint8(y * 10), B: 30, A: 255})
		}
	}
	gray, width, height := RGBAToGray(frame)
	if width != 4 || height != 4 || len(gray) != 16 || gray[0] <= 0 {
		t.Fatalf("gray image = %dx%d %#v", width, height, gray)
	}

	var encoded bytes.Buffer
	if err := png.Encode(&encoded, frame); err != nil {
		t.Fatal(err)
	}
	template, err := LoadPNG(&encoded)
	if err != nil {
		t.Fatal(err)
	}
	if template.W != 4 || template.H != 4 || len(template.Gray) != 16 {
		t.Fatalf("template = %#v", template)
	}
	if _, err := LoadPNG(bytes.NewBufferString("not a png")); err == nil {
		t.Fatal("LoadPNG accepted invalid input")
	}

	roi, x, y, roiWidth, roiHeight := CropROI(frame, 0.25, 0.25, 0.5, 0.5)
	if x != 1 || y != 1 || roiWidth != 2 || roiHeight != 2 || len(roi) != 4 {
		t.Fatalf("ROI = (%d,%d %dx%d) %#v", x, y, roiWidth, roiHeight, roi)
	}
	if rgba := CropROIRGBA(frame, 0.25, 0.25, 0.5, 0.5); rgba == nil || rgba.Bounds().Dx() != 2 || rgba.Bounds().Dy() != 2 {
		t.Fatalf("RGBA ROI = %#v", rgba)
	}
	if rgba := CropROIRGBA(frame, 0, 0, 0, 0.5); rgba != nil {
		t.Fatalf("zero-width ROI = %#v", rgba)
	}
}

func TestTemplateScalingAndPyramidBoundaries(t *testing.T) {
	source := []float32{0, 1, 1, 0}
	copyAtSameSize := ResizeGray(source, 2, 2, 2, 2)
	copyAtSameSize[0] = 9
	if source[0] != 0 {
		t.Fatal("same-size resize aliased its input")
	}

	nearest := ResizeGray([]float32{0.25}, 1, 1, 3, 2)
	for _, value := range nearest {
		if value != 0.25 {
			t.Fatalf("nearest resize = %#v", nearest)
		}
	}
	bilinear := ResizeGray(source, 2, 2, 3, 3)
	if len(bilinear) != 9 || math.Abs(float64(bilinear[4]-0.5)) > 0.0001 {
		t.Fatalf("bilinear resize = %#v", bilinear)
	}

	template := patternedTemplate(16, 16)
	if scaled := ScaleTemplate(template, 0.4); scaled != nil {
		t.Fatalf("undersized template = %#v", scaled)
	}
	if scaled := ScaleTemplate(template, 0.5); scaled == nil || scaled.W != 8 || scaled.H != 8 {
		t.Fatalf("scaled template = %#v", scaled)
	}
	if got := BuildScalePyramid(template, 0.5, 1, 0); len(got) != 1 || got[0] != template {
		t.Fatalf("zero-step pyramid = %#v", got)
	}
	if got := BuildScalePyramid(template, 0.5, 1, 1); len(got) != 1 || got[0].W != 12 {
		t.Fatalf("one-step pyramid = %#v", got)
	}
	if got := BuildScalePyramid(template, 0.25, 1, 4); len(got) != 3 || got[0].W != 8 || got[2].W != 16 {
		t.Fatalf("filtered pyramid = %#v", got)
	}
}

func TestMatchFastAndCandidateSelection(t *testing.T) {
	template := patternedTemplate(20, 20)
	imageGray := make([]float32, 40*36)
	for y := 0; y < template.H; y++ {
		copy(imageGray[(8+y)*40+12:(8+y)*40+12+template.W], template.Gray[y*template.W:(y+1)*template.W])
	}
	x, y, confidence := MatchFast(imageGray, 40, 36, template, 2)
	if x != 12 || y != 8 || confidence < 0.99 {
		t.Fatalf("fast match = (%d,%d %.4f)", x, y, confidence)
	}

	uniform := &Template{Gray: make([]float32, 20*20), W: 20, H: 20}
	if x, y, confidence := MatchFast(imageGray, 40, 36, uniform, 1); x != -1 || y != -1 || confidence != -1 {
		t.Fatalf("uniform match = (%d,%d %.4f)", x, y, confidence)
	}
	if x, y, confidence := MatchFast([]float32{0, 1, 1, 0}, 2, 2, &Template{Gray: []float32{0, 1, 1, 0}, W: 2, H: 2}, 1); x != 0 || y != 0 || confidence < 0.99 {
		t.Fatalf("fallback match = (%d,%d %.4f)", x, y, confidence)
	}

	bestX, bestY, bestConfidence, bestIndex := MatchBest(imageGray, 40, 36, []*Template{uniform, template}, 2)
	if bestX != 12 || bestY != 8 || bestConfidence < 0.99 || bestIndex != 1 {
		t.Fatalf("best match = (%d,%d %.4f,%d)", bestX, bestY, bestConfidence, bestIndex)
	}
	bestX, bestY, bestConfidence, bestIndex = MatchBestEarlyExit(imageGray, 40, 36, []*Template{template, uniform}, 2, 0.9)
	if bestX != 12 || bestY != 8 || bestConfidence < 0.99 || bestIndex != 0 {
		t.Fatalf("early match = (%d,%d %.4f,%d)", bestX, bestY, bestConfidence, bestIndex)
	}
	if x, y, confidence, index := MatchBest(nil, 0, 0, nil, 1); x != -1 || y != -1 || confidence != -1 || index != -1 {
		t.Fatalf("empty best match = (%d,%d %.4f,%d)", x, y, confidence, index)
	}
	if x, y, confidence, index := MatchBestEarlyExit(nil, 0, 0, nil, 1, 0.9); x != -1 || y != -1 || confidence != -1 || index != -1 {
		t.Fatalf("empty early match = (%d,%d %.4f,%d)", x, y, confidence, index)
	}
	if parallel := DefaultParallel(); parallel < 1 || parallel > 4 {
		t.Fatalf("default parallelism = %d", parallel)
	}
}

func patternedTemplate(width, height int) *Template {
	gray := make([]float32, width*height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			gray[y*width+x] = float32((x*7+y*11+x*y)%29) / 28
		}
	}
	return &Template{Gray: gray, W: width, H: height}
}
