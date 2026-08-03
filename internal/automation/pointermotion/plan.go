package pointermotion

import (
	"context"
	"errors"
	"math"
	"time"
)

const (
	Instant Kind = "instant"
	Linear  Kind = "linear"
	Bezier  Kind = "bezier"

	frameDuration = 16 * time.Millisecond
	MaxDuration   = 60 * time.Second
)

type Kind string

func (kind Kind) Valid() bool {
	return kind == Instant || kind == Linear || kind == Bezier
}

type Point struct {
	X float64
	Y float64
}

type Sample struct {
	At    time.Duration
	Point Point
}

// Build converts one semantic pointer movement into deterministic delivery
// samples. Adapters only deliver these points; they do not own path semantics.
func Build(from, to Point, duration time.Duration, kind Kind) ([]Sample, error) {
	if !finitePoint(from) || !finitePoint(to) || !kind.Valid() {
		return nil, errors.New("pointer motion is invalid")
	}
	if kind == Instant {
		if duration != 0 {
			return nil, errors.New("instant pointer motion cannot have a duration")
		}
		return []Sample{{Point: to}}, nil
	}
	if duration <= 0 || duration > MaxDuration {
		return nil, errors.New("timed pointer motion duration is invalid")
	}

	steps := max(1, int(math.Ceil(float64(duration)/float64(frameDuration))))
	samples := make([]Sample, 0, steps)
	control1, control2 := bezierControls(from, to)
	for step := 1; step <= steps; step++ {
		progress := float64(step) / float64(steps)
		point := linearPoint(from, to, progress)
		if kind == Bezier {
			point = cubicBezier(from, control1, control2, to, smoothstep(progress))
		}
		at := time.Duration((int64(duration) * int64(step)) / int64(steps))
		if step == steps {
			at = duration
			point = to
		}
		samples = append(samples, Sample{At: at, Point: point})
	}
	return samples, nil
}

// Play emits samples on their planned timeline and stops promptly when the Run
// is cancelled. The final sample is emitted exactly at the requested duration.
func Play(ctx context.Context, samples []Sample, emit func(Point) error) error {
	if emit == nil {
		return errors.New("pointer motion emitter is missing")
	}
	started := time.Now()
	for _, sample := range samples {
		if wait := sample.At - time.Since(started); wait > 0 {
			timer := time.NewTimer(wait)
			select {
			case <-ctx.Done():
				if !timer.Stop() {
					<-timer.C
				}
				return ctx.Err()
			case <-timer.C:
			}
		} else if err := ctx.Err(); err != nil {
			return err
		}
		if err := emit(sample.Point); err != nil {
			return err
		}
	}
	return nil
}

func finitePoint(point Point) bool {
	return !math.IsNaN(point.X) && !math.IsNaN(point.Y) && !math.IsInf(point.X, 0) && !math.IsInf(point.Y, 0)
}

func linearPoint(from, to Point, progress float64) Point {
	return Point{X: from.X + (to.X-from.X)*progress, Y: from.Y + (to.Y-from.Y)*progress}
}

func bezierControls(from, to Point) (Point, Point) {
	dx, dy := to.X-from.X, to.Y-from.Y
	distance := math.Hypot(dx, dy)
	if distance == 0 {
		return from, to
	}
	offset := distance * 0.12
	normalX, normalY := -dy/distance, dx/distance
	return Point{
			X: from.X + dx/3 + normalX*offset,
			Y: from.Y + dy/3 + normalY*offset,
		}, Point{
			X: from.X + dx*2/3 + normalX*offset,
			Y: from.Y + dy*2/3 + normalY*offset,
		}
}

func cubicBezier(start, control1, control2, end Point, progress float64) Point {
	inverse := 1 - progress
	return Point{
		X: inverse*inverse*inverse*start.X + 3*inverse*inverse*progress*control1.X + 3*inverse*progress*progress*control2.X + progress*progress*progress*end.X,
		Y: inverse*inverse*inverse*start.Y + 3*inverse*inverse*progress*control1.Y + 3*inverse*progress*progress*control2.Y + progress*progress*progress*end.Y,
	}
}

func smoothstep(progress float64) float64 {
	return progress * progress * (3 - 2*progress)
}
