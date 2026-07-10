package runtime

import (
	"context"
	"image"
	"image/color"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/services/container"
)

// DetectColorHSV Outputs 行为由 internal/nodes/detect/detect_color_hsv_test.go 覆盖.
// runtime 集成靠 TestDetectColorHSVTimeoutOnNoMatch (execNode + 测 elapsed).

// TestDetectColorHSVTimeoutOnNoMatch: 100x100 全蓝帧, HSV 黄色范围 → 无命中 → timeout 出口。
// timeoutMs=100, pollIntervalMs=20 → 应在 ~100ms 后返回。
func TestDetectColorHSVTimeoutOnNoMatch(t *testing.T) {
	frame := image.NewRGBA(image.Rect(0, 0, 100, 100))
	blue := color.RGBA{R: 0, G: 0, B: 255, A: 255}
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			frame.Set(x, y, blue)
		}
	}

	rt, r := newTestRunner(t)
	rt.Capture.(*mockCaptureBackend).FrameROIResult = frame

	node := &container.GraphNode{
		ID:   "hsv2",
		Kind: "DetectColorHSV",
		Config: map[string]any{
			"ROI": map[string]any{
				"x": float64(0), "y": float64(0),
				"w": float64(100), "h": float64(100),
			},
			"HSV": map[string]any{
				"hMin": float64(50), "hMax": float64(70),
				"sMin": float64(78), "sMax": float64(100),
				"vMin": float64(78), "vMax": float64(100),
			},
			// numeric thresholds via inline literal pin.
			"literal": map[string]any{
				"MinPixelRatio":  float64(0.5),
				"PollIntervalMs": float64(20),
				"TimeoutMs":      float64(100),
			},
		},
	}
	if r.nodesByID == nil {
		r.nodesByID = map[string]*container.GraphNode{}
	}
	r.nodesByID[node.ID] = node // pullDataPin lookup
	start := time.Now()
	_, err := r.execNode(context.Background(), node, ExecToken{InPin: "in"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	elapsed := time.Since(start)
	if elapsed < 80*time.Millisecond || elapsed > 300*time.Millisecond {
		t.Fatalf("expected ~100ms timeout, got %v", elapsed)
	}
}
