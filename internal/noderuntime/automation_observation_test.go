package noderuntime

import (
	"math"
	"testing"
)

func TestFrameObservationUsesBoundedGridDifference(t *testing.T) {
	baseline := frameSignature{grid: []uint8{0, 0, 0, 10, 10, 10}}
	stable := frameSignature{grid: []uint8{1, 1, 1, 11, 11, 11}}
	changed := frameSignature{grid: []uint8{255, 255, 255, 10, 10, 10}}
	if got := compareFrameSignatures(baseline, stable, 2); got.changedRatio != 0 || got.meanDifference <= 0 {
		t.Fatalf("stable difference = %#v", got)
	}
	got := compareFrameSignatures(baseline, changed, 2)
	if math.Abs(got.changedRatio-0.5) > 0.000001 || got.meanDifference <= 0 {
		t.Fatalf("changed difference = %#v", got)
	}
}
