// internal/nodes/detect/detect_color_test.go
package detect

import (
	"context"
	"errors"
	"testing"

	"yotta/internal/node"
)

func TestDetectColor_Hit(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&DetectColor{})
	rn, _ := node.Get("DetectColor")

	vision := &mockVision{colorCount: 42, colorCX: 0.5, colorCY: 0.6}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{
			dcInRegion:    node.Geometry{Pct: node.Rect{X: 0.4, Y: 0.5, W: 0.2, H: 0.05}},
			dcInMode:      "hsv",
			dcInRange:     []any{50.0, 60.0, 26.0, 50.0, 99.0, 100.0},
			dcInMinPixels: 5,
		},
		nil, withVision(vision), false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != dcOutYes {
		t.Errorf("exit = %q, want Yes", r.ExitName)
	}
}

func TestDetectColor_Miss(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&DetectColor{})
	rn, _ := node.Get("DetectColor")

	vision := &mockVision{colorCount: 2}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{
			dcInMode:      "hsv",
			dcInRange:     []any{0.0, 360.0, 0.0, 100.0, 0.0, 100.0},
			dcInMinPixels: 5,
		},
		nil, withVision(vision), false)

	if r.ExitName != dcOutNo {
		t.Errorf("exit = %q, want No", r.ExitName)
	}
}

func TestDetectColor_BackendError(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&DetectColor{})
	rn, _ := node.Get("DetectColor")

	vision := &mockVision{colorErr: errors.New("capture failed")}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{dcInMode: "rgb", dcInRange: []any{0.0, 255.0, 0.0, 255.0, 0.0, 255.0}},
		nil, withVision(vision), false)

	if r.Error == nil {
		t.Error("expected backend error propagation")
	}
}

func TestDetectColor_InvalidMode_ValidationError(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&DetectColor{})
	rn, _ := node.Get("DetectColor")

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{dcInMode: "yuv"},
		nil, withVision(&mockVision{}), false)

	if len(r.Validation) != 1 || r.Validation[0].Code != "INVALID_COLOR_MODE" {
		t.Errorf("validation = %v, want INVALID_COLOR_MODE", r.Validation)
	}
}
