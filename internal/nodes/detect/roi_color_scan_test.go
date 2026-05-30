// internal/nodes/detect/roi_color_scan_test.go
package detect

import (
	"context"
	"testing"

	"yhbox/internal/node"
)

func validScanCfg() map[string]any {
	return map[string]any{
		// ROI 零值 Geometry → 全帧 (adapter 内 ResolveGeometry 处理).
		rcsInROI: node.Geometry{},
		rcsInHSV: map[string]any{"hMin": 0.0, "hMax": 180.0,
			"sMin": 0.0, "sMax": 255.0, "vMin": 0.0, "vMax": 255.0},
		rcsInAxis: "x",
	}
}

func TestROIColorScan_Found(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ROIColorScan{})
	rn, _ := node.Get("ROIColorScan")

	vision := &mockVision{
		clusters: []node.ClusterEntry{
			{StartPx: 10, EndPx: 20, CenterPx: 15, PxCount: 10},
			{StartPx: 30, EndPx: 35, CenterPx: 32, PxCount: 5},
		},
	}
	cfg := validScanCfg()
	cfg[rcsInMinClusterCount] = 2
	cfg[rcsInTimeoutMs] = 500
	r := node.RunNode(context.Background(), rn, nil, cfg, nil, withVision(vision))

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != rcsOutFound {
		t.Errorf("exit = %q, want Found", r.ExitName)
	}
}

func TestROIColorScan_NotFoundSingleScan(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ROIColorScan{})
	rn, _ := node.Get("ROIColorScan")

	vision := &mockVision{clusters: nil}
	cfg := validScanCfg()
	cfg[rcsInMinClusterCount] = 1
	cfg[rcsInTimeoutMs] = 0
	r := node.RunNode(context.Background(), rn, nil, cfg, nil, withVision(vision))

	if r.ExitName != rcsOutNotFound {
		t.Errorf("exit = %q, want NotFound", r.ExitName)
	}
}

func TestROIColorScan_Timeout(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ROIColorScan{})
	rn, _ := node.Get("ROIColorScan")

	vision := &mockVision{clusters: nil}
	cfg := validScanCfg()
	cfg[rcsInMinClusterCount] = 1
	cfg[rcsInTimeoutMs] = 30
	cfg[rcsInPollIntervalMs] = 10
	r := node.RunNode(context.Background(), rn, nil, cfg, nil, withVision(vision))

	if r.ExitName != rcsOutTimeout {
		t.Errorf("exit = %q, want Timeout", r.ExitName)
	}
}

func TestROIColorScan_InvalidAxis_ValidationError(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ROIColorScan{})
	rn, _ := node.Get("ROIColorScan")

	cfg := validScanCfg()
	cfg[rcsInAxis] = "z"
	r := node.RunNode(context.Background(), rn, nil, cfg, nil, withVision(&mockVision{}))

	if len(r.Validation) == 0 {
		t.Fatal("expected validation error")
	}
	found := false
	for _, e := range r.Validation {
		if e.Code == "INVALID_SCAN_AXIS" {
			found = true
		}
	}
	if !found {
		t.Errorf("want INVALID_SCAN_AXIS, got %v", r.Validation)
	}
}
