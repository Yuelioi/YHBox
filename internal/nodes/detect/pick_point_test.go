package detect

import (
	"context"
	"testing"

	"yotta/internal/node"
)

func evalDetectNode(t *testing.T, n node.Node, dataWire map[string]any) any {
	t.Helper()
	node.ResetRegistryForTest()
	node.Register(n)
	rn, ok := node.Get(n.Spec().Kind)
	if !ok {
		t.Fatalf("kind %q not registered", n.Spec().Kind)
	}
	got, err := node.EvaluatePureData(context.Background(), rn, dataWire, nil, node.StubServices())
	if err != nil {
		t.Fatalf("EvaluatePureData err: %v", err)
	}
	return got
}

func assertPoint(t *testing.T, got any, want node.Point) {
	t.Helper()
	pt, ok := got.(node.Point)
	if !ok {
		t.Fatalf("got %T(%v), want node.Point", got, got)
	}
	if !approxEq(pt.X, want.X) || !approxEq(pt.Y, want.Y) {
		t.Fatalf("point = %+v, want %+v", pt, want)
	}
}

func TestPickMatchPoint_FromTemplateMatchSlice(t *testing.T) {
	matches := []node.TemplateMatch{
		{Point: node.Point{X: 0.15, Y: 0.25}, BBox: [4]float64{0.1, 0.2, 0.1, 0.1}, Conf: 0.9},
		{Point: node.Point{X: 0.55, Y: 0.65}, BBox: [4]float64{0.5, 0.6, 0.1, 0.1}, Conf: 0.8},
	}
	got := evalDetectNode(t, &PickMatchPoint{}, map[string]any{
		pmpInMatches: matches,
		pmpInIndex:   1,
		pmpInAnchor:  "botRight",
		pmpInOffsetX: 0.02,
		pmpInOffsetY: -0.01,
	})
	assertPoint(t, got, node.Point{X: 0.62, Y: 0.69})
}

func TestPickMatchPoint_FromJSONMaps(t *testing.T) {
	matches := []any{
		map[string]any{
			"point": map[string]any{"x": 0.25, "y": 0.35},
			"bbox":  []any{0.2, 0.3, 0.1, 0.1},
			"conf":  0.91,
		},
	}
	got := evalDetectNode(t, &PickMatchPoint{}, map[string]any{pmpInMatches: matches})
	assertPoint(t, got, node.Point{X: 0.25, Y: 0.35})
}

func TestPickMatchPoint_OutOfRangeReturnsZeroPoint(t *testing.T) {
	got := evalDetectNode(t, &PickMatchPoint{}, map[string]any{
		pmpInMatches: []node.TemplateMatch{{Point: node.Point{X: 0.1, Y: 0.2}}},
		pmpInIndex:   9,
	})
	assertPoint(t, got, node.Point{})
}

func TestPickBlobPoint_FromBlobSlice(t *testing.T) {
	blobs := []node.BlobEntry{
		{CenterX: 0.2, CenterY: 0.3, X: 0.1, Y: 0.2, W: 0.2, H: 0.2, Area: 10},
		{CenterX: 0.7, CenterY: 0.8, X: 0.6, Y: 0.7, W: 0.2, H: 0.2, Area: 20},
	}
	got := evalDetectNode(t, &PickBlobPoint{}, map[string]any{
		pbpInBlobs:  blobs,
		pbpInIndex:  1,
		pbpInAnchor: "topLeft",
	})
	assertPoint(t, got, node.Point{X: 0.6, Y: 0.7})
}

func TestPickBlobPoint_FromJSONMaps(t *testing.T) {
	blobs := []any{
		map[string]any{"centerX": 0.4, "centerY": 0.5, "x": 0.3, "y": 0.4, "w": 0.2, "h": 0.2},
	}
	got := evalDetectNode(t, &PickBlobPoint{}, map[string]any{pbpInBlobs: blobs})
	assertPoint(t, got, node.Point{X: 0.4, Y: 0.5})
}
