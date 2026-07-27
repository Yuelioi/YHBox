package noderuntime

import (
	"context"
	"image"
	"image/color"
	"testing"

	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodeadapter"
	"github.com/yottaapp/yotta/internal/nodes"
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

func TestScanVisionColorProducesMaskAndPixelCentroid(t *testing.T) {
	frame := image.NewRGBA(image.Rect(10, 20, 13, 22))
	frame.SetRGBA(10, 20, color.RGBA{R: 220, G: 20, B: 30, A: 255})
	frame.SetRGBA(12, 21, color.RGBA{R: 210, G: 10, B: 20, A: 255})
	mask := make([]bool, 6)
	rangeRGB := visionColorRange{Space: "rgb", Minimum: [3]int{200, 0, 0}, Maximum: [3]int{255, 40, 40}}
	count, sumX, sumY := scanVisionColor(frame, frame.Bounds(), rangeRGB, mask)
	if count != 2 || sumX != 23 || sumY != 42 || !mask[0] || !mask[5] {
		t.Fatalf("count=%d sum=(%v,%v) mask=%v", count, sumX, sumY, mask)
	}
}

func TestVisionColorRangeInputAndOutputSealingAreStrict(t *testing.T) {
	builtins, err := nodes.Build()
	if err != nil {
		t.Fatal(err)
	}
	seal := func(definition datatype.Definition, raw string) datatype.ValueEnvelope {
		t.Helper()
		value, err := datatype.SealInlineJSON(builtins.Catalog, datatype.RefResolvedType(definition.TypeRef()), []byte(raw))
		if err != nil {
			t.Fatal(err)
		}
		return value
	}
	valid := seal(builtins.ColorRangeType, `{"space":"hsv","minimum":[0,90,90],"maximum":[5,100,100]}`)
	got, err := visionColorRangeInput(nodeadapter.Invocation{Inputs: map[string]datatype.ValueEnvelope{"range": valid}})
	if err != nil || got.Space != "hsv" || got.Maximum != [3]int{5, 100, 100} {
		t.Fatalf("color range = %#v, %v", got, err)
	}
	for _, raw := range []string{
		`{"space":"lab","minimum":[0,0,0],"maximum":[1,1,1]}`,
		`{"space":"rgb","minimum":[10,0,0],"maximum":[5,255,255]}`,
	} {
		invalid := seal(builtins.JSONType, raw)
		if _, err := visionColorRangeInput(nodeadapter.Invocation{Inputs: map[string]datatype.ValueEnvelope{"range": invalid}}); err == nil {
			t.Fatalf("visionColorRangeInput accepted %s", raw)
		}
	}
	if _, err := visionColorRangeInput(nodeadapter.Invocation{}); err == nil {
		t.Fatal("visionColorRangeInput accepted missing input")
	}

	integerType := datatype.RefResolvedType(builtins.IntegerType.TypeRef())
	sealed, err := sealVisionOutputs(builtins, nodeadapter.Invocation{OutputTypes: map[string]datatype.ResolvedType{"count": integerType}}, map[string]any{"count": 3})
	if err != nil || string(sealed.Outputs["count"].InlineJSON()) != "3" {
		t.Fatalf("sealed outputs = %#v, %v", sealed, err)
	}
	if _, err := sealVisionOutputs(builtins, nodeadapter.Invocation{}, map[string]any{"count": 3}); err == nil {
		t.Fatal("sealVisionOutputs accepted an unresolved output")
	}
	if _, err := findColorBlobs(builtins)(context.Background(), nodeadapter.Invocation{}); err == nil {
		t.Fatal("findColorBlobs accepted missing minimum area")
	}
	if _, err := trackDualColorBar(builtins)(context.Background(), nodeadapter.Invocation{}); err == nil {
		t.Fatal("trackDualColorBar accepted missing inputs")
	}
}
