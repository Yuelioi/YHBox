// internal/nodes/detect/detect_color_hsv_test.go
package detect

import (
	"context"
	"testing"

	"yhbox/internal/node"
)

func validHSVCfg() map[string]any {
	return map[string]any{
		dchInROI: map[string]any{"x": 0.0, "y": 0.0, "w": 100.0, "h": 50.0},
		dchInHSV: map[string]any{"hMin": 0.0, "hMax": 180.0,
			"sMin": 0.0, "sMax": 255.0, "vMin": 0.0, "vMax": 255.0},
	}
}

func TestDetectColorHSV_HitFirstPoll(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&DetectColorHSV{})
	rn, _ := node.Get("DetectColorHSV")

	vision := &mockVision{hsvCount: 100, hsvRatio: 0.2}
	cfg := validHSVCfg()
	cfg[dchInMinPixelRatio] = 0.1
	cfg[dchInTimeoutMs] = 1000
	r := node.RunNode(context.Background(), rn, nil, cfg, nil, withVision(vision))

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != dchOutYes {
		t.Errorf("exit = %q, want Yes", r.ExitName)
	}
}

func TestDetectColorHSV_NoOnSingleScan(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&DetectColorHSV{})
	rn, _ := node.Get("DetectColorHSV")

	// timeoutMs<=0 + 首次未命中 → No (而非 Timeout 永远轮询).
	vision := &mockVision{hsvCount: 2, hsvRatio: 0.01}
	cfg := validHSVCfg()
	cfg[dchInMinPixelRatio] = 0.5
	cfg[dchInTimeoutMs] = 0
	r := node.RunNode(context.Background(), rn, nil, cfg, nil, withVision(vision))

	if r.ExitName != dchOutNo {
		t.Errorf("exit = %q, want No", r.ExitName)
	}
}

func TestDetectColorHSV_Timeout(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&DetectColorHSV{})
	rn, _ := node.Get("DetectColorHSV")

	vision := &mockVision{hsvCount: 0, hsvRatio: 0.0}
	cfg := validHSVCfg()
	cfg[dchInMinPixelRatio] = 0.99
	cfg[dchInTimeoutMs] = 30
	cfg[dchInPollIntervalMs] = 10
	r := node.RunNode(context.Background(), rn, nil, cfg, nil, withVision(vision))

	if r.ExitName != dchOutTimeout {
		t.Errorf("exit = %q, want Timeout", r.ExitName)
	}
}

func TestDetectColorHSV_InvalidROI(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&DetectColorHSV{})
	rn, _ := node.Get("DetectColorHSV")

	cfg := map[string]any{
		dchInROI: map[string]any{"x": 0.0, "y": 0.0, "w": 0.0, "h": 0.0},
		dchInHSV: map[string]any{"hMin": 0.0, "hMax": 180.0,
			"sMin": 0.0, "sMax": 255.0, "vMin": 0.0, "vMax": 255.0},
		dchInTimeoutMs: 100,
	}
	r := node.RunNode(context.Background(), rn, nil, cfg, nil, withVision(&mockVision{}))

	if r.Error == nil {
		t.Error("expected error on invalid ROI w/h=0")
	}
}
