package nodes31runtime

import (
	"image"
	"testing"
)

func TestConnectedVisionColorBlobsAreDeterministicAndOffsetToFrame(t *testing.T) {
	region := image.Rect(10, 20, 16, 25)
	mask := make([]bool, region.Dx()*region.Dy())
	set := func(x, y int) { mask[y*region.Dx()+x] = true }
	for _, point := range [][2]int{{0, 0}, {1, 0}, {0, 1}, {4, 1}, {5, 1}, {4, 2}, {5, 2}, {4, 3}, {2, 4}} {
		set(point[0], point[1])
	}
	blobs, err := connectedVisionColorBlobs(mask, region, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(blobs) != 2 {
		t.Fatalf("blobs = %#v", blobs)
	}
	if blobs[0].Area != 5 || blobs[0].Bounds.X != 14 || blobs[0].Bounds.Y != 21 || blobs[0].Bounds.Width != 2 || blobs[0].Bounds.Height != 3 {
		t.Fatalf("largest blob = %#v", blobs[0])
	}
	if blobs[1].Area != 3 || blobs[1].Bounds.X != 10 || blobs[1].Bounds.Y != 20 || blobs[1].Center.X != 10.833333333333334 || blobs[1].Center.Y != 20.833333333333332 {
		t.Fatalf("second blob = %#v", blobs[1])
	}
}

func TestVisionColorRangesAreInclusiveAndSpaceExplicit(t *testing.T) {
	rgb := visionColorRange{Space: "rgb", Minimum: [3]int{200, 10, 20}, Maximum: [3]int{220, 30, 40}}
	if !matchesVisionColor(200, 30, 40, rgb) || matchesVisionColor(199, 30, 40, rgb) {
		t.Fatal("RGB inclusive range mismatch")
	}
	hsv := visionColorRange{Space: "hsv", Minimum: [3]int{0, 90, 90}, Maximum: [3]int{5, 100, 100}}
	if !matchesVisionColor(255, 0, 0, hsv) || matchesVisionColor(0, 255, 0, hsv) {
		t.Fatal("HSV range mismatch")
	}
}
