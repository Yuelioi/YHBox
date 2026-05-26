// internal/nodes/detect/dual_color_bar_track_test.go
package detect

import (
	"context"
	"testing"

	"yhbox/internal/node"
)

// stubWindow 用于注 ClientSize 报 1920x1080. 跟 stubCapture 同款本地一份.
type stubWindow struct{ w, h int }

func (s stubWindow) BringForeground() error        { return nil }
func (s stubWindow) HWND() uintptr                 { return 0 }
func (s stubWindow) ClientSize() (int, int, error) { return s.w, s.h, nil }

func withVisionAndWindow(v node.VisionService, w node.WindowService) node.ServiceBundle {
	b := node.StubServices()
	b.Vision = v
	b.Window = w
	return b
}

func validRois1080p() []any {
	return []any{
		map[string]any{
			"resolution": []any{1920.0, 1080.0},
			"x":          576.0, "y": 594.0, "w": 768.0, "h": 54.0,
		},
	}
}

func TestDualColorBarTrack_Found(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&DualColorBarTrack{})
	rn, _ := node.Get("DualColorBarTrack")

	vision := &mockVision{
		barResult: node.DualColorBarResult{
			Found:      true,
			InnerX:     320, OuterX: 400, OuterWidth: 80,
			Confidence: 0.85,
			InnerPx:    200, OuterPx: 50,
		},
	}
	win := stubWindow{w: 1920, h: 1080}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{dcbtInRois: validRois1080p()},
		nil, withVisionAndWindow(vision, win))

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != dcbtOutFound {
		t.Errorf("exit = %q, want Found", r.ExitName)
	}
	if r.OutputData[dcbtDataInnerX].(int) != 320 {
		t.Errorf("innerX = %v, want 320", r.OutputData[dcbtDataInnerX])
	}
}

func TestDualColorBarTrack_MissingNoResolutionMatch(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&DualColorBarTrack{})
	rn, _ := node.Get("DualColorBarTrack")

	// rois 只 1080p, client 720p → Missing.
	win := stubWindow{w: 1280, h: 720}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{dcbtInRois: validRois1080p()},
		nil, withVisionAndWindow(&mockVision{}, win))

	if r.ExitName != dcbtOutMissing {
		t.Errorf("exit = %q, want Missing", r.ExitName)
	}
}

func TestDualColorBarTrack_MissingNotFound(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&DualColorBarTrack{})
	rn, _ := node.Get("DualColorBarTrack")

	// vision adapter 已经按 confBarV2 阈值 (0.50) 设 Found, mock 直接给 Found=false 模拟低 conf.
	vision := &mockVision{
		barResult: node.DualColorBarResult{
			Found: false, InnerX: 10, OuterX: 50, Confidence: 0.3,
		},
	}
	win := stubWindow{w: 1920, h: 1080}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{dcbtInRois: validRois1080p()},
		nil, withVisionAndWindow(vision, win))

	if r.ExitName != dcbtOutMissing {
		t.Errorf("exit = %q, want Missing (low conf)", r.ExitName)
	}
}

func TestDualColorBarTrack_InvalidROIs_ValidationError(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&DualColorBarTrack{})
	rn, _ := node.Get("DualColorBarTrack")

	// 空数组 → INVALID
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{dcbtInRois: []any{}},
		nil, withVisionAndWindow(&mockVision{}, stubWindow{}))
	if len(r.Validation) == 0 {
		t.Error("expected validation error on empty rois")
	}

	// ROI 越界
	bad := []any{
		map[string]any{
			"resolution": []any{1000.0, 500.0},
			"x":          500.0, "y": 0.0, "w": 600.0, "h": 100.0, // 500+600 > 1000
		},
	}
	r2 := node.RunNode(context.Background(), rn, nil,
		map[string]any{dcbtInRois: bad},
		nil, withVisionAndWindow(&mockVision{}, stubWindow{}))
	if len(r2.Validation) == 0 {
		t.Error("expected validation error on out-of-bounds ROI")
	}
}
