// internal/nodes/detect/detect_color_test.go
package detect

import (
	"context"
	"errors"
	"testing"

	"github.com/yottaapp/yotta/internal/node"
)

func TestDetectColor_Hit(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&DetectColor{})
	rn, _ := node.Get("DetectColor")

	vision := &mockVision{colorCount: 42, colorCX: 0.5, colorCY: 0.6}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{
			dcInROI:       node.Geometry{Pct: node.Rect{X: 0.4, Y: 0.5, W: 0.2, H: 0.05}},
			dcInMode:      "hsv",
			dcInRange:     []any{50.0, 60.0, 26.0, 50.0, 99.0, 100.0},
			dcInMinPixels: 5,
		},
		nil, withVision(vision), false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != dcOutFound {
		t.Errorf("exit = %q, want Found", r.ExitName)
	}
}

func TestDetectColor_Miss(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&DetectColor{})
	rn, _ := node.Get("DetectColor")

	vision := &mockVision{colorCount: 2}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{
			dcInMode:      "hsv",
			dcInRange:     []any{0.0, 360.0, 0.0, 100.0, 0.0, 100.0},
			dcInMinPixels: 5,
		},
		nil, withVision(vision), false)

	if r.ExitName != dcOutNotFound {
		t.Errorf("exit = %q, want NotFound", r.ExitName)
	}
}

func TestDetectColor_BackendError(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&DetectColor{})
	rn, _ := node.Get("DetectColor")

	vision := &mockVision{colorErr: errors.New("capture failed")}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{dcInMode: "rgb", dcInRange: []any{0.0, 255.0, 0.0, 255.0, 0.0, 255.0}},
		nil, withVision(vision), false)

	if r.Error == nil {
		t.Error("expected backend error propagation")
	}
}

// 节点责任 = Set 正确 Data 字段 (framework 路径① 据此写变量, 见 runtime dispatch 测试)。
func TestDetectColor_OutputData_Hit(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&DetectColor{})
	rn, _ := node.Get("DetectColor")

	vision := &mockVision{colorCount: 10, colorCX: 0.5, colorCY: 0.6}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{
			dcInROI:       node.Geometry{Pct: node.Rect{X: 0.4, Y: 0.5, W: 0.2, H: 0.05}},
			dcInMode:      "hsv",
			dcInRange:     []any{50.0, 60.0, 26.0, 50.0, 99.0, 100.0},
			dcInMinPixels: 5,
		},
		nil, withVision(vision), false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != dcOutFound {
		t.Fatalf("exit = %q, want Found", r.ExitName)
	}
	if got := r.OutputData[dcDataCount]; got != 10 {
		t.Errorf("OutputData Count = %v, want 10", got)
	}
	if _, ok := r.OutputData[dcDataCenter].(node.Point); !ok {
		t.Errorf("OutputData Center = %T, want node.Point", r.OutputData[dcDataCenter])
	}
}

func TestDetectColor_OutputData_Miss(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&DetectColor{})
	rn, _ := node.Get("DetectColor")

	// count(1) < minPx(5) → NotFound 出口只带 Count, 不带 Center.
	vision := &mockVision{colorCount: 1, colorCX: 0.5, colorCY: 0.6}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{
			dcInMode:      "hsv",
			dcInRange:     []any{0.0, 360.0, 0.0, 100.0, 0.0, 100.0},
			dcInMinPixels: 5,
		},
		nil, withVision(vision), false)

	if r.ExitName != dcOutNotFound {
		t.Fatalf("exit = %q, want NotFound", r.ExitName)
	}
	if got := r.OutputData[dcDataCount]; got != 1 {
		t.Errorf("OutputData Count = %v, want 1", got)
	}
	// NotFound exit 不带 Center → OutputData 无该字段 (稀疏)。
	if _, ok := r.OutputData[dcDataCenter]; ok {
		t.Error("OutputData has Center on NotFound exit, want absent")
	}
}

func TestDetectColor_InvalidMode_ValidationError(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&DetectColor{})
	rn, _ := node.Get("DetectColor")

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{dcInMode: "yuv"},
		nil, withVision(&mockVision{}), false)

	if len(r.Validation) != 1 || r.Validation[0].Code != "INVALID_COLOR_MODE" {
		t.Errorf("validation = %v, want INVALID_COLOR_MODE", r.Validation)
	}
}
