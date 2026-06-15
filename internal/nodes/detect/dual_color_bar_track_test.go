// internal/nodes/detect/dual_color_bar_track_test.go
package detect

import (
	"context"
	"testing"

	"yotta/internal/node"
)

// stubWindow 用于注 ClientSize 报 1920x1080. 跟 stubCapture 同款本地一份.
type stubWindow struct{ w, h int }

func (s stubWindow) BringForeground() error        { return nil }
func (s stubWindow) HWND() uintptr                 { return 0 }
func (s stubWindow) ClientSize() (int, int, error) { return s.w, s.h, nil }
func (s stubWindow) SetActive(ctx context.Context, title, class, processName, titleMatch string) error {
	return nil
}

func withVisionAndWindow(v node.VisionService, w node.WindowService) node.ServiceBundle {
	b := node.StubServices()
	b.Vision = v
	b.Window = w
	return b
}

// validGeometryROI 返一个 pct-based Geometry, 模拟 50% 宽高居中 ROI.
func validGeometryROI() node.Geometry {
	return node.Geometry{
		Pct: node.Rect{X: 0.3, Y: 0.55, W: 0.4, H: 0.05},
	}
}

func TestDualColorBarTrack_Found(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&DualColorBarTrack{})
	rn, _ := node.Get("DualColorBarTrack")

	vision := &mockVision{
		barResult: node.DualColorBarResult{
			Found:  true,
			InnerX: 320, OuterX: 400, OuterWidth: 80,
			Confidence: 0.85,
			InnerPx:    200, OuterPx: 50,
		},
	}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{dcbtInROI: validGeometryROI()},
		nil, withVision(vision), false)

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

// 节点责任 = Set 6 个 Data 字段 (framework 路径① 据此写变量)。
func TestDualColorBarTrack_OutputData_Found(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&DualColorBarTrack{})
	rn, _ := node.Get("DualColorBarTrack")

	vision := &mockVision{
		barResult: node.DualColorBarResult{
			Found:  true,
			InnerX: 320, OuterX: 400, OuterWidth: 80,
			Confidence: 0.85,
			InnerPx:    200, OuterPx: 50,
		},
	}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{dcbtInROI: validGeometryROI()},
		nil, withVision(vision), false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != dcbtOutFound {
		t.Fatalf("exit = %q, want Found", r.ExitName)
	}
	checks := []struct {
		field string
		want  any
	}{
		{dcbtDataInnerX, 320}, {dcbtDataOuterX, 400}, {dcbtDataOuterW, 80},
		{dcbtDataConf, 0.85}, {dcbtDataInnerPx, 200}, {dcbtDataOuterPx, 50},
	}
	for _, c := range checks {
		if got := r.OutputData[c.field]; got != c.want {
			t.Errorf("OutputData %s = %v, want %v", c.field, got, c.want)
		}
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
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{dcbtInROI: validGeometryROI()},
		nil, withVision(vision), false)

	if r.ExitName != dcbtOutNotFound {
		t.Errorf("exit = %q, want NotFound (low conf)", r.ExitName)
	}
}

func TestDualColorBarTrack_MissingOnError(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&DualColorBarTrack{})
	rn, _ := node.Get("DualColorBarTrack")

	// barErr 会被节点包裹成 error 出口
	vision := &mockVision{
		barErr:    nil, // adapter 层 capture fail 已经转 Found=false 不冒泡; mock 返零值即可
		barResult: node.DualColorBarResult{Found: false},
	}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{dcbtInROI: validGeometryROI()},
		nil, withVision(vision), false)

	if r.ExitName != dcbtOutNotFound {
		t.Errorf("exit = %q, want NotFound", r.ExitName)
	}
}

func TestDualColorBarTrack_RequiredMissing(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&DualColorBarTrack{})
	rn, _ := node.Get("DualColorBarTrack")

	// 没有 Roi 输入 → framework Required 检查报 REQUIRED_FIELD_MISSING
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{},
		nil, withVision(&mockVision{}), false)
	if len(r.Validation) == 0 {
		t.Error("expected validation error on missing Roi")
	}
}
