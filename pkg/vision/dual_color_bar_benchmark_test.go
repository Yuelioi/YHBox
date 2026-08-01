package vision

import (
	"image"
	"image/color"
	"testing"
)

func fishingBiteROIMatchFixture() ([]float32, int, int, *FastTemplate, int, int) {
	const width, height = 636, 197
	template := patternedTemplate(396, 37)
	prepared := PrepareFastTemplate(template)
	imageGray := make([]float32, width*height)
	for index := range imageGray {
		imageGray[index] = float32((index*13+index/width*7)%251) / 250
	}
	const matchX, matchY = 120, 80
	for y := 0; y < template.H; y++ {
		copy(imageGray[(matchY+y)*width+matchX:(matchY+y)*width+matchX+template.W], template.Gray[y*template.W:(y+1)*template.W])
	}
	return imageGray, width, height, prepared, matchX, matchY
}

func BenchmarkAnalyzeDualColorBarFishingROI(b *testing.B) {
	frame := image.NewRGBA(image.Rect(0, 0, 705, 12))
	for y := range 12 {
		for x := range 705 {
			frame.SetRGBA(x, y, color.RGBA{R: uint8(x), G: uint8(y * 17), B: uint8(x + y), A: 255})
		}
	}
	inner := ColorRange{Space: "hsv", Minimum: [3]int{45, 16, 78}, Maximum: [3]int{70, 100, 100}}
	outer := ColorRange{Space: "hsv", Minimum: [3]int{160, 55, 39}, Maximum: [3]int{180, 100, 100}}
	b.ReportAllocs()
	for range b.N {
		_ = AnalyzeDualColorBar(frame, frame.Bounds(), inner, outer, DualColorBarOptions{
			InnerMinimumWidth: 2, BandHeightRatio: 0.3, BandInnerHeightRatio: 0.85,
			InnerConfidenceWeight: 0.42, OuterConfidenceWeight: 0.58,
		})
	}
}

func BenchmarkMatchFastFishingBiteROI(b *testing.B) {
	imageGray, width, height, prepared, matchX, matchY := fishingBiteROIMatchFixture()
	b.ReportAllocs()
	for range b.N {
		x, y, score := MatchFastPrepared(imageGray, width, height, prepared, DefaultParallel())
		if x != matchX || y != matchY || score < 0.99 {
			b.Fatalf("match = (%d,%d %.4f)", x, y, score)
		}
	}
}

func TestMatchFastFishingBiteROIAllocationBudget(t *testing.T) {
	imageGray, width, height, prepared, matchX, matchY := fishingBiteROIMatchFixture()
	allocations := testing.AllocsPerRun(3, func() {
		x, y, score := MatchFastPrepared(imageGray, width, height, prepared, DefaultParallel())
		if x != matchX || y != matchY || score < 0.99 {
			t.Fatalf("match = (%d,%d %.4f)", x, y, score)
		}
	})
	if allocations >= 200 {
		t.Fatalf("fishing bite ROI match allocations = %.0f, want < 200", allocations)
	}
}

func TestMatchFastPreparedScratchDoesNotLeakAcrossFrames(t *testing.T) {
	first, width, height, prepared, firstX, firstY := fishingBiteROIMatchFixture()
	second := make([]float32, width*height)
	for index := range second {
		second[index] = float32((index*17+index/width*19)%241) / 240
	}
	const secondX, secondY = 24, 32
	template := prepared.full.template
	for y := 0; y < template.H; y++ {
		copy(second[(secondY+y)*width+secondX:(secondY+y)*width+secondX+template.W], template.Gray[y*template.W:(y+1)*template.W])
	}
	for _, testCase := range []struct {
		name         string
		frame        []float32
		wantX, wantY int
	}{
		{name: "first", frame: first, wantX: firstX, wantY: firstY},
		{name: "second", frame: second, wantX: secondX, wantY: secondY},
		{name: "first-again", frame: first, wantX: firstX, wantY: firstY},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			x, y, score := MatchFastPrepared(testCase.frame, width, height, prepared, DefaultParallel())
			if x != testCase.wantX || y != testCase.wantY || score < 0.99 {
				t.Fatalf("match = (%d,%d %.4f), want (%d,%d)", x, y, score, testCase.wantX, testCase.wantY)
			}
		})
	}
}
