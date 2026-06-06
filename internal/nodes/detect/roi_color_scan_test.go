// internal/nodes/detect/roi_color_scan_test.go
package detect

import (
	"context"
	"testing"

	"yotta/internal/node"
)

func validScanCfg() map[string]any {
	return map[string]any{
		// ROI 零值 Geometry → 全帧 (adapter 内 ResolveGeometry 处理).
		rcsInROI: node.Geometry{},
		rcsInHSV: map[string]any{"hMin": 0.0, "hMax": 360.0,
			"sMin": 0.0, "sMax": 100.0, "vMin": 0.0, "vMax": 100.0},
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
	r := node.RunNode(context.Background(), rn, nil, cfg, nil, withVision(vision), false)

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
	r := node.RunNode(context.Background(), rn, nil, cfg, nil, withVision(vision), false)

	if r.ExitName != rcsOutNotFound {
		t.Errorf("exit = %q, want NotFound", r.ExitName)
	}
}

func TestROIColorScan_Capture_Found(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ROIColorScan{})
	rn, _ := node.Get("ROIColorScan")

	clusters := []node.ClusterEntry{
		{StartPx: 10, EndPx: 20, CenterPx: 15, PxCount: 10},
		{StartPx: 30, EndPx: 35, CenterPx: 32, PxCount: 5},
	}
	vision := &mockVision{clusters: clusters}
	cfg := validScanCfg()
	cfg[rcsInMinClusterCount] = 2
	cfg[rcsInTimeoutMs] = 500
	cfg[rcsCapCount] = "n"
	cfg[rcsCapClusters] = "cl"
	vars := newRecVars()
	r := node.RunNode(context.Background(), rn, nil, cfg, nil, withVisionVars(vision, vars), false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != rcsOutFound {
		t.Fatalf("exit = %q, want Found", r.ExitName)
	}
	if got, ok := vars.Get("n"); !ok || got != 2 {
		t.Errorf("capture n = %v (ok=%v), want 2", got, ok)
	} else if _, isInt := got.(int); !isInt {
		// CaptureType=number: 断言写入值的 Go 类型是 int.
		t.Errorf("capture n Go type = %T, want int", got)
	}
	gc, ok := vars.Get("cl")
	if !ok {
		t.Fatal("capture cl not written")
	}
	// CaptureType=any: 只断言写入, 不断 Go 类型. 实际是 []node.ClusterEntry.
	if got, ok := gc.([]node.ClusterEntry); !ok || len(got) != 2 {
		t.Errorf("capture cl = %T len mismatch, want []node.ClusterEntry len 2", gc)
	}
}

func TestROIColorScan_Capture_NotFound(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ROIColorScan{})
	rn, _ := node.Get("ROIColorScan")

	vision := &mockVision{clusters: nil}
	cfg := validScanCfg()
	cfg[rcsInMinClusterCount] = 1
	cfg[rcsInTimeoutMs] = 0
	cfg[rcsCapCount] = "n"
	cfg[rcsCapClusters] = "cl"
	vars := newRecVars()
	r := node.RunNode(context.Background(), rn, nil, cfg, nil, withVisionVars(vision, vars), false)

	if r.ExitName != rcsOutNotFound {
		t.Fatalf("exit = %q, want NotFound", r.ExitName)
	}
	if got, ok := vars.Get("n"); !ok || got != 0 {
		t.Errorf("capture n = %v (ok=%v), want 0", got, ok)
	}
	// NotFound exit 不带 Clusters → 不写.
	if _, ok := vars.Get("cl"); ok {
		t.Error("capture cl written on NotFound exit, want unwritten")
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
	r := node.RunNode(context.Background(), rn, nil, cfg, nil, withVision(vision), false)

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
	r := node.RunNode(context.Background(), rn, nil, cfg, nil, withVision(&mockVision{}), false)

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
