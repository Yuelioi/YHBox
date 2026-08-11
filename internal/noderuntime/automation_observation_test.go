package noderuntime

import (
	"context"
	"math"
	"testing"
	"time"
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

func TestPollFrameObservationUsesOneDeadlineIncludingCaptureTime(t *testing.T) {
	now := time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC)
	waits := 0
	captures := 0
	difference, output, gotCaptures, captureBytes, err := pollFrameObservationWithClock(
		context.Background(),
		func(context.Context, time.Duration) error {
			waits++
			return nil
		},
		func() time.Time { return now },
		10*time.Millisecond,
		10*time.Millisecond,
		0,
		0.5,
		12,
		false,
		func(context.Context) (frameSignature, int64, error) {
			captures++
			now = now.Add(20 * time.Millisecond)
			return frameSignature{grid: []uint8{0, 0, 0}}, 64, nil
		},
	)
	if err != nil || output != "timeout" || gotCaptures != 1 || captures != 1 || captureBytes != 64 || waits != 0 ||
		difference != (frameDifference{}) {
		t.Fatalf("difference=%#v output=%q captures=%d/%d bytes=%d waits=%d err=%v", difference, output, gotCaptures, captures, captureBytes, waits, err)
	}
}

func TestPollFrameObservationCountsCaptureTimeTowardStableDuration(t *testing.T) {
	now := time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC)
	difference, output, captures, _, err := pollFrameObservationWithClock(
		context.Background(),
		func(_ context.Context, delay time.Duration) error {
			now = now.Add(delay)
			return nil
		},
		func() time.Time { return now },
		time.Second,
		10*time.Millisecond,
		60*time.Millisecond,
		0.02,
		12,
		true,
		func(context.Context) (frameSignature, int64, error) {
			now = now.Add(50 * time.Millisecond)
			return frameSignature{grid: []uint8{10, 10, 10}}, 64, nil
		},
	)
	if err != nil || output != "stable" || captures != 2 || difference.changedRatio != 0 {
		t.Fatalf("difference=%#v output=%q captures=%d err=%v", difference, output, captures, err)
	}
}
