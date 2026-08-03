package pointermotion

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestBuildInstantEmitsOnlyDestination(t *testing.T) {
	samples, err := Build(Point{X: 0.1, Y: 0.2}, Point{X: 0.8, Y: 0.7}, 0, Instant)
	if err != nil {
		t.Fatal(err)
	}
	if len(samples) != 1 || samples[0].At != 0 || samples[0].Point != (Point{X: 0.8, Y: 0.7}) {
		t.Fatalf("instant samples = %#v", samples)
	}
}

func TestBuildLinearUsesConstantProgress(t *testing.T) {
	samples, err := Build(Point{}, Point{X: 0.9, Y: 0.6}, 48*time.Millisecond, Linear)
	if err != nil {
		t.Fatal(err)
	}
	want := []Sample{
		{At: 16 * time.Millisecond, Point: Point{X: 0.3, Y: 0.2}},
		{At: 32 * time.Millisecond, Point: Point{X: 0.6, Y: 0.4}},
		{At: 48 * time.Millisecond, Point: Point{X: 0.9, Y: 0.6}},
	}
	if len(samples) != len(want) {
		t.Fatalf("linear sample count = %d, want %d", len(samples), len(want))
	}
	for index := range want {
		if samples[index].At != want[index].At || !near(samples[index].Point.X, want[index].Point.X) || !near(samples[index].Point.Y, want[index].Point.Y) {
			t.Fatalf("linear sample %d = %#v, want %#v", index, samples[index], want[index])
		}
	}
}

func TestBuildBezierIsDeterministicCurvedAndEndsExactly(t *testing.T) {
	from := Point{X: 0.1, Y: 0.2}
	to := Point{X: 0.9, Y: 0.8}
	left, err := Build(from, to, 160*time.Millisecond, Bezier)
	if err != nil {
		t.Fatal(err)
	}
	right, err := Build(from, to, 160*time.Millisecond, Bezier)
	if err != nil {
		t.Fatal(err)
	}
	if len(left) < 3 || len(left) != len(right) {
		t.Fatalf("bezier samples = %d and %d", len(left), len(right))
	}
	curved := false
	for index := range left {
		if left[index] != right[index] {
			t.Fatalf("bezier plan is not deterministic at %d: %#v != %#v", index, left[index], right[index])
		}
		progress := float64(index+1) / float64(len(left))
		line := Point{X: from.X + (to.X-from.X)*progress, Y: from.Y + (to.Y-from.Y)*progress}
		if !near(left[index].Point.X, line.X) || !near(left[index].Point.Y, line.Y) {
			curved = true
		}
	}
	if !curved {
		t.Fatal("bezier plan stayed on the straight line")
	}
	last := left[len(left)-1]
	if last.At != 160*time.Millisecond || last.Point != to {
		t.Fatalf("bezier last sample = %#v", last)
	}
}

func TestBuildRejectsInvalidMotion(t *testing.T) {
	for _, test := range []struct {
		name     string
		duration time.Duration
		kind     Kind
	}{
		{name: "instant duration", duration: time.Millisecond, kind: Instant},
		{name: "timed zero", duration: 0, kind: Linear},
		{name: "unknown", duration: time.Millisecond, kind: "arc"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := Build(Point{}, Point{X: 1, Y: 1}, test.duration, test.kind); err == nil {
				t.Fatal("Build accepted invalid motion")
			}
		})
	}
}

func TestPlayHonorsCancellationBeforeEmission(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := Play(ctx, []Sample{{At: time.Second, Point: Point{X: 1, Y: 1}}}, func(Point) error {
		t.Fatal("emitted after cancellation")
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Play error = %v", err)
	}
}

func near(left, right float64) bool {
	delta := left - right
	return delta > -1e-9 && delta < 1e-9
}
