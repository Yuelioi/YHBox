package runtime

import (
	"testing"

	"yhbox/internal/node"
)

func TestResolveGeometry_ResolutionSwitch(t *testing.T) {
	g := node.Geometry{
		Pct: node.Rect{X: 0.1, Y: 0.1, W: 0.5, H: 0.2},
		Overrides: []node.GeoOverride{{
			Resolution: node.Resolution{W: 1920, H: 1080},
			Px:         node.PixelRect{X: 10, Y: 20, W: 100, H: 50},
		}},
	}
	x, _, w, _, full := ResolveGeometry(g, 1920, 1080)
	if full || x != 10 || w != 100 {
		t.Fatalf("1920x1080 应走 override, got x=%d w=%d full=%v", x, w, full)
	}
	x, _, w, _, full = ResolveGeometry(g, 2560, 1440)
	if full || x != int(0.1*2560) || w != int(0.5*2560) {
		t.Fatalf("2560x1440 应走 pct, got x=%d w=%d full=%v", x, w, full)
	}
}

func TestResolveGeometry_FullFrame(t *testing.T) {
	g := node.Geometry{Pct: node.Rect{X: 0.3, Y: 0.3, W: 0, H: 0}} // x/y 非 0 但 w/h=0
	_, _, w, h, full := ResolveGeometry(g, 1920, 1080)
	if !full || w != 1920 || h != 1080 {
		t.Fatalf("w/h==0 应全帧(忽略 x/y), got w=%d h=%d full=%v", w, h, full)
	}
}
