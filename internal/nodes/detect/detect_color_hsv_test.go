// internal/nodes/detect/detect_color_hsv_test.go
package detect

import (
	"context"
	"testing"

	"yotta/internal/node"
)

func validHSVCfg() map[string]any {
	return map[string]any{
		// ROI 零值 Geometry → 全帧 (adapter 内 ResolveGeometry 处理).
		dchInROI: node.Geometry{},
		dchInHSV: map[string]any{"hMin": 0.0, "hMax": 360.0,
			"sMin": 0.0, "sMax": 100.0, "vMin": 0.0, "vMax": 100.0},
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
	r := node.RunNode(context.Background(), rn, nil, cfg, nil, withVision(vision), false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != dchOutFound {
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
	r := node.RunNode(context.Background(), rn, nil, cfg, nil, withVision(vision), false)

	if r.ExitName != dchOutNotFound {
		t.Errorf("exit = %q, want No", r.ExitName)
	}
}

func TestDetectColorHSV_Capture_Hit(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&DetectColorHSV{})
	rn, _ := node.Get("DetectColorHSV")

	vision := &mockVision{hsvCount: 100, hsvRatio: 0.2}
	cfg := validHSVCfg()
	cfg[dchInMinPixelRatio] = 0.1
	cfg[dchInTimeoutMs] = 1000
	cfg[dchCapCount] = "n"
	cfg[dchCapRatio] = "r"
	vars := newRecVars()
	res := node.RunNode(context.Background(), rn, nil, cfg, nil, withVisionVars(vision, vars), false)

	if res.Error != nil {
		t.Fatal(res.Error)
	}
	if res.ExitName != dchOutFound {
		t.Fatalf("exit = %q, want Yes", res.ExitName)
	}
	if got, ok := vars.Get("n"); !ok || got != 100 {
		t.Errorf("capture n = %v (ok=%v), want 100", got, ok)
	} else if _, isInt := got.(int); !isInt {
		// CaptureType=number: 断言写入值的 Go 类型是 int.
		t.Errorf("capture n Go type = %T, want int", got)
	}
	if got, ok := vars.Get("r"); !ok || got != 0.2 {
		t.Errorf("capture r = %v (ok=%v), want 0.2", got, ok)
	} else if _, isF64 := got.(float64); !isF64 {
		// CaptureType=number: 断言写入值的 Go 类型是 float64.
		t.Errorf("capture r Go type = %T, want float64", got)
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
	r := node.RunNode(context.Background(), rn, nil, cfg, nil, withVision(vision), false)

	if r.ExitName != dchOutTimeout {
		t.Errorf("exit = %q, want Timeout", r.ExitName)
	}
}
