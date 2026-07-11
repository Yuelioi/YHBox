// internal/nodes/detect/roi_color_scan_test.go
package detect

import (
	"context"
	"testing"

	"github.com/yottaapp/yotta/internal/node"
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
	registry := node.NewRegistry()
	registry.Register(&ROIColorScan{})
	rn, _ := registry.Get("ROIColorScan")

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
	registry := node.NewRegistry()
	registry.Register(&ROIColorScan{})
	rn, _ := registry.Get("ROIColorScan")

	vision := &mockVision{clusters: nil}
	cfg := validScanCfg()
	cfg[rcsInMinClusterCount] = 1
	cfg[rcsInTimeoutMs] = 0
	r := node.RunNode(context.Background(), rn, nil, cfg, nil, withVision(vision), false)

	if r.ExitName != rcsOutNotFound {
		t.Errorf("exit = %q, want NotFound", r.ExitName)
	}
}

// 节点责任 = Set Clusters/ClusterCount Data 字段 (framework 路径① 据此写变量)。
func TestROIColorScan_OutputData_Found(t *testing.T) {
	registry := node.NewRegistry()
	registry.Register(&ROIColorScan{})
	rn, _ := registry.Get("ROIColorScan")

	clusters := []node.ClusterEntry{
		{StartPx: 10, EndPx: 20, CenterPx: 15, PxCount: 10},
		{StartPx: 30, EndPx: 35, CenterPx: 32, PxCount: 5},
	}
	vision := &mockVision{clusters: clusters}
	cfg := validScanCfg()
	cfg[rcsInMinClusterCount] = 2
	cfg[rcsInTimeoutMs] = 500
	r := node.RunNode(context.Background(), rn, nil, cfg, nil, withVision(vision), false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != rcsOutFound {
		t.Fatalf("exit = %q, want Found", r.ExitName)
	}
	if got := r.OutputData[rcsDataClusterCount]; got != 2 {
		t.Errorf("OutputData ClusterCount = %v, want 2", got)
	}
	if got, ok := r.OutputData[rcsDataClusters].([]node.ClusterEntry); !ok || len(got) != 2 {
		t.Errorf("OutputData Clusters = %T len mismatch, want []node.ClusterEntry len 2", r.OutputData[rcsDataClusters])
	}
}

func TestROIColorScan_OutputData_NotFound(t *testing.T) {
	registry := node.NewRegistry()
	registry.Register(&ROIColorScan{})
	rn, _ := registry.Get("ROIColorScan")

	vision := &mockVision{clusters: nil}
	cfg := validScanCfg()
	cfg[rcsInMinClusterCount] = 1
	cfg[rcsInTimeoutMs] = 0
	r := node.RunNode(context.Background(), rn, nil, cfg, nil, withVision(vision), false)

	if r.ExitName != rcsOutNotFound {
		t.Fatalf("exit = %q, want NotFound", r.ExitName)
	}
	if got := r.OutputData[rcsDataClusterCount]; got != 0 {
		t.Errorf("OutputData ClusterCount = %v, want 0", got)
	}
	// NotFound exit 不带 Clusters → OutputData 无该字段 (稀疏)。
	if _, ok := r.OutputData[rcsDataClusters]; ok {
		t.Error("OutputData has Clusters on NotFound exit, want absent")
	}
}

func TestROIColorScan_Timeout(t *testing.T) {
	registry := node.NewRegistry()
	registry.Register(&ROIColorScan{})
	rn, _ := registry.Get("ROIColorScan")

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
	registry := node.NewRegistry()
	registry.Register(&ROIColorScan{})
	rn, _ := registry.Get("ROIColorScan")

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
