package detect

import (
	"context"
	"testing"

	"github.com/yottaapp/yotta/internal/node"
)

func runBlobs(t *testing.T, vision *mockVision, cfg map[string]any) node.RunResult {
	t.Helper()
	registry := node.NewRegistry()
	registry.Register(&DetectColorBlobs{})
	rn, _ := registry.Get("DetectColorBlobs")
	return node.RunNode(context.Background(), rn, nil, cfg, nil, withVision(vision), false)
}

func TestDetectColorBlobs_NotFound(t *testing.T) {
	r := runBlobs(t, &mockVision{blobs: nil}, map[string]any{dcbInTimeoutMs: 0})
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != dcbOutNotFound {
		t.Fatalf("exit = %q, want NotFound", r.ExitName)
	}
	if r.OutputData[dcbDataBlobCount] != 0 {
		t.Errorf("BlobCount = %v, want 0", r.OutputData[dcbDataBlobCount])
	}
}

func TestDetectColorBlobs_AreaDescPrimary(t *testing.T) {
	vision := &mockVision{blobs: []node.BlobEntry{
		{CenterX: 0.1, CenterY: 0.1, Area: 50},
		{CenterX: 0.9, CenterY: 0.9, Area: 200},
		{CenterX: 0.5, CenterY: 0.5, Area: 100},
	}}
	r := runBlobs(t, vision, map[string]any{dcbInTimeoutMs: 0, dcbInSort: "area_desc"})
	if r.ExitName != dcbOutFound {
		t.Fatalf("exit = %q, want Found", r.ExitName)
	}
	if r.OutputData[dcbDataPrimaryArea] != 200 {
		t.Errorf("PrimaryArea = %v, want 200 (最大块)", r.OutputData[dcbDataPrimaryArea])
	}
	if r.OutputData[dcbDataBlobCount] != 3 {
		t.Errorf("BlobCount = %v, want 3", r.OutputData[dcbDataBlobCount])
	}
}

func TestDetectColorBlobs_DistScreenCenterPrimary(t *testing.T) {
	vision := &mockVision{blobs: []node.BlobEntry{
		{CenterX: 0.1, CenterY: 0.1, Area: 200},
		{CenterX: 0.5, CenterY: 0.5, Area: 30},
	}}
	r := runBlobs(t, vision, map[string]any{dcbInTimeoutMs: 0, dcbInSort: "dist_screen_center"})
	if r.OutputData[dcbDataPrimaryArea] != 30 {
		t.Errorf("PrimaryArea = %v, want 30 (距中心最近，Primary 非最大块)", r.OutputData[dcbDataPrimaryArea])
	}
}

func TestDetectColorBlobs_MaxBlobsTruncateButBlobCountFull(t *testing.T) {
	blobs := make([]node.BlobEntry, 37)
	for i := range blobs {
		blobs[i] = node.BlobEntry{Area: i + 1}
	}
	r := runBlobs(t, &mockVision{blobs: blobs}, map[string]any{
		dcbInTimeoutMs: 0, dcbInSort: "area_desc", dcbInMaxBlobs: 5,
	})
	got := r.OutputData[dcbDataBlobs].([]node.BlobEntry)
	if len(got) != 5 {
		t.Errorf("len(Blobs) = %d, want 5 (MaxBlobs 截断)", len(got))
	}
	if r.OutputData[dcbDataBlobCount] != 37 {
		t.Errorf("BlobCount = %v, want 37 (不受 MaxBlobs 影响)", r.OutputData[dcbDataBlobCount])
	}
}

// dist_point 未设 RefPoint → 默认 (0,0)，按距原点排序，不报错（替代已废的 MISSING_REF_POINT 校验）。
func TestDetectColorBlobs_DistPointDefaultsToOrigin(t *testing.T) {
	vision := &mockVision{blobs: []node.BlobEntry{
		{CenterX: 0.8, CenterY: 0.8, Area: 200}, // 远离原点
		{CenterX: 0.1, CenterY: 0.1, Area: 30},  // 近原点
	}}
	r := runBlobs(t, vision, map[string]any{dcbInTimeoutMs: 0, dcbInSort: "dist_point"})
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != dcbOutFound {
		t.Fatalf("exit = %q, want Found", r.ExitName)
	}
	if r.OutputData[dcbDataPrimaryArea] != 30 {
		t.Errorf("PrimaryArea = %v, want 30 (距原点(0,0)最近)", r.OutputData[dcbDataPrimaryArea])
	}
}
