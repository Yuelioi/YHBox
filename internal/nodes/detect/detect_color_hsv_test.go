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

// 节点责任 = Set 正确 Data 字段 (framework 路径① 据此写变量)。
func TestDetectColorHSV_OutputData_Found(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&DetectColorHSV{})
	rn, _ := node.Get("DetectColorHSV")

	vision := &mockVision{hsvCount: 100, hsvRatio: 0.2}
	cfg := validHSVCfg()
	cfg[dchInMinPixelRatio] = 0.1
	cfg[dchInTimeoutMs] = 1000
	res := node.RunNode(context.Background(), rn, nil, cfg, nil, withVision(vision), false)

	if res.Error != nil {
		t.Fatal(res.Error)
	}
	if res.ExitName != dchOutFound {
		t.Fatalf("exit = %q, want Found", res.ExitName)
	}
	if got := res.OutputData[dchDataCount]; got != 100 {
		t.Errorf("OutputData PixelCount = %v, want 100", got)
	}
	if got := res.OutputData[dchDataRatio]; got != 0.2 {
		t.Errorf("OutputData PixelRatio = %v, want 0.2", got)
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
