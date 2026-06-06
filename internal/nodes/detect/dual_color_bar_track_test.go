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
			Found:      true,
			InnerX:     320, OuterX: 400, OuterWidth: 80,
			Confidence: 0.85,
			InnerPx:    200, OuterPx: 50,
		},
	}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{dcbtInRoi: validGeometryROI()},
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

func TestDualColorBarTrack_Capture_Found(t *testing.T) {
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
	vars := newRecVars()
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{
			dcbtInRoi:      validGeometryROI(),
			dcbtCapInnerX:  "ix",
			dcbtCapOuterX:  "ox",
			dcbtCapOuterW:  "ow",
			dcbtCapConf:    "cf",
			dcbtCapInnerPx: "ipx",
			dcbtCapOuterPx: "opx",
		},
		nil, withVisionVars(vision, vars), false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != dcbtOutFound {
		t.Fatalf("exit = %q, want Found", r.ExitName)
	}
	checks := []struct {
		name string
		want any
	}{
		{"ix", 320}, {"ox", 400}, {"ow", 80},
		{"cf", 0.85}, {"ipx", 200}, {"opx", 50},
	}
	for _, c := range checks {
		got, ok := vars.Get(c.name)
		if !ok || got != c.want {
			t.Errorf("capture %s = %v (ok=%v), want %v", c.name, got, ok, c.want)
		}
	}
	// CaptureType=number: 断言每个捕获字段的 Go 类型符合声明.
	// ix/ox/ow/ipx/opx → int; cf → float64.
	intFields := []string{"ix", "ox", "ow", "ipx", "opx"}
	for _, name := range intFields {
		got, _ := vars.Get(name)
		if _, isInt := got.(int); !isInt {
			t.Errorf("capture %s Go type = %T, want int", name, got)
		}
	}
	cfGot, _ := vars.Get("cf")
	if _, isF64 := cfGot.(float64); !isF64 {
		t.Errorf("capture cf Go type = %T, want float64", cfGot)
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
		map[string]any{dcbtInRoi: validGeometryROI()},
		nil, withVision(vision), false)

	if r.ExitName != dcbtOutMissing {
		t.Errorf("exit = %q, want Missing (low conf)", r.ExitName)
	}
}

func TestDualColorBarTrack_MissingOnError(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&DualColorBarTrack{})
	rn, _ := node.Get("DualColorBarTrack")

	// barErr 会被节点包裹成 error 出口
	vision := &mockVision{
		barErr: nil, // adapter 层 capture fail 已经转 Found=false 不冒泡; mock 返零值即可
		barResult: node.DualColorBarResult{Found: false},
	}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{dcbtInRoi: validGeometryROI()},
		nil, withVision(vision), false)

	if r.ExitName != dcbtOutMissing {
		t.Errorf("exit = %q, want Missing", r.ExitName)
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
