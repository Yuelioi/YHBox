package main

import "testing"

func TestLongEdgeScale(t *testing.T) {
	cases := []struct {
		fw, fh, vw, vh int
		want           float64
	}{
		{1920, 1080, 1920, 1080, 1.0},
		{2560, 1440, 1280, 720, 2.0},
		{1280, 720, 2560, 1440, 0.5},
		{1366, 768, 1280, 720, 1366.0 / 1280},
		{1920, 1080, 0, 0, 0},
	}
	for _, c := range cases {
		if got := longEdgeScale(c.fw, c.fh, c.vw, c.vh); got != c.want {
			t.Errorf("longEdgeScale(%d,%d,%d,%d) = %v, want %v", c.fw, c.fh, c.vw, c.vh, got, c.want)
		}
	}
}

func TestWithinScaleTolerance(t *testing.T) {
	cases := []struct {
		scale, k float64
		want     bool
	}{
		{1.0, 2.0, true},
		{2.0, 2.0, true},
		{0.5, 2.0, true},
		{2.5, 2.0, false},
		{0.4, 2.0, false},
		{1.2, 1.0, false},
		{1.0, 1.0, true},
		{0, 2.0, false},
		{1.0, 0.5, true},  // k<1 归一到 1 → 仅精确放行
		{1.2, 0.5, false}, // k<1 归一到 1 → 非精确判否
	}
	for _, c := range cases {
		if got := withinScaleTolerance(c.scale, c.k); got != c.want {
			t.Errorf("withinScaleTolerance(%v,%v) = %v, want %v", c.scale, c.k, got, c.want)
		}
	}
}
