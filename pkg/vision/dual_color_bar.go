package vision

import (
	"image"
	"math"
)

// ColorRange describes inclusive RGB or HSV channel bounds.
type ColorRange struct {
	Space   string
	Minimum [3]int
	Maximum [3]int
}

// DualColorBarOptions controls the column-cluster detector. Zero-valued size
// fields select resolution-relative defaults.
type DualColorBarOptions struct {
	InnerMinimumWidth     int
	InnerMaximumWidth     int
	OuterMinimumWidth     int
	BandHeightRatio       float64
	BandInnerHeightRatio  float64
	InnerConfidenceWeight float64
	OuterConfidenceWeight float64
}

// DualColorBarResult reports the inner cursor and outer target positions in
// source-image pixel coordinates.
type DualColorBarResult struct {
	Found       bool
	InnerX      int
	OuterX      int
	OuterWidth  int
	Confidence  float64
	InnerPixels int
	OuterPixels int
}

// AnalyzeDualColorBar finds a narrow inner color cluster and a wider outer
// color cluster inside region. The algorithm intentionally groups matching
// columns instead of connected components so anti-aliasing gaps do not split a
// thin UI bar into unrelated blobs.
func AnalyzeDualColorBar(img *image.RGBA, region image.Rectangle, inner, outer ColorRange, options DualColorBarOptions) DualColorBarResult {
	region = region.Intersect(img.Bounds())
	width, height := region.Dx(), region.Dy()
	if width < 10 || height < 4 {
		return DualColorBarResult{InnerX: -1, OuterX: -1}
	}

	innerX, innerHeight, innerConfidence, innerPixels := findInnerColorCluster(img, region, inner, options)
	if innerX < 0 {
		return DualColorBarResult{InnerX: -1, OuterX: -1, InnerPixels: innerPixels}
	}

	bandHeightRatio := options.BandHeightRatio
	if bandHeightRatio <= 0 {
		bandHeightRatio = 0.30
	}
	bandInnerHeightRatio := options.BandInnerHeightRatio
	if bandInnerHeightRatio <= 0 {
		bandInnerHeightRatio = 0.85
	}
	bandHalf := max(int(float64(height)*bandHeightRatio), int(float64(innerHeight)*bandInnerHeightRatio))
	centerY := region.Min.Y + height/2
	band := image.Rect(region.Min.X, max(region.Min.Y, centerY-bandHalf), region.Max.X, min(region.Max.Y, centerY+bandHalf+1))

	outerX, outerWidth, outerConfidence, outerPixels := findOuterColorCluster(img, band, outer)
	outerMinimumWidth := options.OuterMinimumWidth
	if outerMinimumWidth <= 0 {
		outerMinimumWidth = width / 20
	}
	if outerX < 0 || outerWidth < outerMinimumWidth {
		return DualColorBarResult{
			InnerX: innerX, OuterX: -1, InnerPixels: innerPixels, OuterPixels: outerPixels,
		}
	}

	innerWeight, outerWeight := options.InnerConfidenceWeight, options.OuterConfidenceWeight
	if innerWeight <= 0 && outerWeight <= 0 {
		innerWeight, outerWeight = 0.42, 0.58
	}
	return DualColorBarResult{
		Found: true, InnerX: innerX, OuterX: outerX, OuterWidth: outerWidth,
		Confidence:  math.Min(0.98, innerConfidence*innerWeight+outerConfidence*outerWeight),
		InnerPixels: innerPixels, OuterPixels: outerPixels,
	}
}

func findInnerColorCluster(img *image.RGBA, region image.Rectangle, color ColorRange, options DualColorBarOptions) (centerX, clusterHeight int, confidence float64, totalPixels int) {
	width, height := region.Dx(), region.Dy()
	columns := make([]int, width)
	for y := region.Min.Y; y < region.Max.Y; y++ {
		for x := region.Min.X; x < region.Max.X; x++ {
			offset := img.PixOffset(x, y)
			if colorRangeMatches(img.Pix[offset], img.Pix[offset+1], img.Pix[offset+2], color) {
				columns[x-region.Min.X]++
				totalPixels++
			}
		}
	}

	minimumWidth := options.InnerMinimumWidth
	if minimumWidth <= 0 {
		minimumWidth = 2
	}
	maximumWidth := options.InnerMaximumWidth
	if maximumWidth <= 0 {
		maximumWidth = max(15, width/40)
	}
	bestStart, bestEnd, bestSum := -1, -1, 0
	for index := 0; index < width; {
		if columns[index] == 0 {
			index++
			continue
		}
		start, sum := index, 0
		for index < width && columns[index] > 0 {
			sum += columns[index]
			index++
		}
		runWidth := index - start
		if runWidth >= minimumWidth && runWidth <= maximumWidth && sum > bestSum {
			bestStart, bestEnd, bestSum = start, index, sum
		}
	}
	if bestSum < 2 {
		return -1, 0, 0, totalPixels
	}

	clusterWidth := bestEnd - bestStart
	centerX = region.Min.X + (bestStart+bestEnd)/2
	clusterHeight = bestSum / max(1, clusterWidth)
	aspect := float64(clusterHeight) / float64(max(1, clusterWidth))
	fillRatio := float64(bestSum) / float64(max(1, clusterWidth*height))
	confidence = math.Min(1, aspect*0.3+fillRatio*2)
	if aspect < 1 {
		confidence *= 0.5
	}
	return centerX, clusterHeight, math.Min(1, confidence), totalPixels
}

func findOuterColorCluster(img *image.RGBA, band image.Rectangle, color ColorRange) (centerX, width int, confidence float64, totalPixels int) {
	columns := make([]int, band.Dx())
	for y := band.Min.Y; y < band.Max.Y; y++ {
		for x := band.Min.X; x < band.Max.X; x++ {
			offset := img.PixOffset(x, y)
			if colorRangeMatches(img.Pix[offset], img.Pix[offset+1], img.Pix[offset+2], color) {
				columns[x-band.Min.X]++
				totalPixels++
			}
		}
	}
	if totalPixels < 3 {
		return -1, 0, 0, totalPixels
	}

	bestStart, bestEnd, bestSum := -1, -1, 0
	for index := 0; index < len(columns); {
		if columns[index] == 0 {
			index++
			continue
		}
		start, sum := index, 0
		for index < len(columns) && columns[index] > 0 {
			sum += columns[index]
			index++
		}
		if sum > bestSum {
			bestStart, bestEnd, bestSum = start, index, sum
		}
	}
	if bestStart < 0 {
		return -1, 0, 0, totalPixels
	}
	width = bestEnd - bestStart
	centerX = band.Min.X + (bestStart+bestEnd)/2
	fillRatio := float64(bestSum) / float64(max(1, width*band.Dy()))
	return centerX, width, math.Min(1, fillRatio*2.5), totalPixels
}

func colorRangeMatches(red, green, blue uint8, color ColorRange) bool {
	values := [3]int{int(red), int(green), int(blue)}
	if color.Space == "hsv" {
		values[0], values[1], values[2] = RGBToHSV(red, green, blue)
	}
	for index := range values {
		if values[index] < color.Minimum[index] || values[index] > color.Maximum[index] {
			return false
		}
	}
	return color.Space == "rgb" || color.Space == "hsv"
}
