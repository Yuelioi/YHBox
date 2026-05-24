// internal/nodes/detect/color_bar_track_test.go
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

func TestColorBarTrack_Found(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ColorBarTrack{})
	rn, _ := node.Get("ColorBarTrack")

	vision := &mockVision{
		barResult: node.BarTrackResult{
			Found:      true,
			CursorX:    320, TargetX: 400, TargetW: 80,
			Confidence: 0.85,
			YellowPx:   200, GreenPx: 50,
		},
	}
	win := stubWindow{w: 1920, h: 1080}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{cbtInRois: validRois1080p()},
		nil, withVisionAndWindow(vision, win))

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != cbtOutFound {
		t.Errorf("exit = %q, want Found", r.ExitName)
	}
	if r.OutputData[cbtDataCursorX].(int) != 320 {
		t.Errorf("cursorX = %v, want 320", r.OutputData[cbtDataCursorX])
	}
}

func TestColorBarTrack_MissingNoResolutionMatch(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ColorBarTrack{})
	rn, _ := node.Get("ColorBarTrack")

	// rois 只 1080p, client 720p → Missing.
	win := stubWindow{w: 1280, h: 720}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{cbtInRois: validRois1080p()},
		nil, withVisionAndWindow(&mockVision{}, win))

	if r.ExitName != cbtOutMissing {
		t.Errorf("exit = %q, want Missing", r.ExitName)
	}
}

func TestColorBarTrack_MissingLowConfidence(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ColorBarTrack{})
	rn, _ := node.Get("ColorBarTrack")

	vision := &mockVision{
		barResult: node.BarTrackResult{
			Found: true, CursorX: 10, TargetX: 50, Confidence: 0.3, // < confBarThreshold 0.50
		},
	}
	win := stubWindow{w: 1920, h: 1080}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{cbtInRois: validRois1080p()},
		nil, withVisionAndWindow(vision, win))

	if r.ExitName != cbtOutMissing {
		t.Errorf("exit = %q, want Missing (low conf)", r.ExitName)
	}
}

func TestColorBarTrack_InvalidROIs_ValidationError(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ColorBarTrack{})
	rn, _ := node.Get("ColorBarTrack")

	// 空数组 → INVALID
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{cbtInRois: []any{}},
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
		map[string]any{cbtInRois: bad},
		nil, withVisionAndWindow(&mockVision{}, stubWindow{}))
	if len(r2.Validation) == 0 {
		t.Error("expected validation error on out-of-bounds ROI")
	}
}
