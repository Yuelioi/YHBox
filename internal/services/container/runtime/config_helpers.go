package runtime

import (
	"fmt"

	"yotta/internal/services/container"
)

// roiRect ROI 像素矩形（客户区绝对坐标）。
type roiRect struct{ X, Y, W, H int }

// hsvRange HSV 阈值区间。H ∈ [0,360]，S/V ∈ [0,100]。
type hsvRange struct {
	hMin, hMax int
	sMin, sMax int
	vMin, vMax int
}

// configROI 从节点 Config[key]（map[string]any）解析 roiRect。
// 要求 w >= 1 且 h >= 1，否则返 error。
func configROI(n *container.GraphNode, key string) (roiRect, error) {
	raw, ok := n.Config[key].(map[string]any)
	if !ok {
		return roiRect{}, fmt.Errorf("missing %s", key)
	}
	f := func(k string) int {
		v, _ := raw[k].(float64)
		return int(v)
	}
	r := roiRect{X: f("x"), Y: f("y"), W: f("w"), H: f("h")}
	if r.W < 1 || r.H < 1 {
		return r, fmt.Errorf("ROI %s w/h must be >=1, got %dx%d", key, r.W, r.H)
	}
	return r, nil
}

// configHSV 从节点 Config[key]（map[string]any）解析 hsvRange。
// 要求各区间 min <= max，否则返 error。
func configHSV(n *container.GraphNode, key string) (hsvRange, error) {
	raw, ok := n.Config[key].(map[string]any)
	if !ok {
		return hsvRange{}, fmt.Errorf("missing %s", key)
	}
	f := func(k string) int {
		v, _ := raw[k].(float64)
		return int(v)
	}
	h := hsvRange{
		hMin: f("hMin"), hMax: f("hMax"),
		sMin: f("sMin"), sMax: f("sMax"),
		vMin: f("vMin"), vMax: f("vMax"),
	}
	if h.hMin > h.hMax || h.sMin > h.sMax || h.vMin > h.vMax {
		return h, fmt.Errorf("HSV range inverted: H=[%d,%d] S=[%d,%d] V=[%d,%d]",
			h.hMin, h.hMax, h.sMin, h.sMax, h.vMin, h.vMax)
	}
	return h, nil
}
