package node

import "testing"

func TestResolveScalar(t *testing.T) {
	cases := []struct {
		name   string
		v      float64
		fullPx int
		want   float64
	}{
		{"比例-半", 0.5, 1920, 0.5},
		{"比例-满幅1=100%", 1, 1920, 1},
		{"比例-负", -1, 1080, -1},
		{"像素-正", 12, 1920, 12.0 / 1920.0},
		{"像素-负", -30, 1080, -30.0 / 1080.0},
		{"像素-边界刚过1", 1.5, 100, 0.015},
		{"零", 0, 1920, 0},
		{"fullPx<=0退回原值", 12, 0, 12},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ResolveScalar(c.v, c.fullPx); got != c.want {
				t.Fatalf("ResolveScalar(%v,%d)=%v want %v", c.v, c.fullPx, got, c.want)
			}
		})
	}
}
