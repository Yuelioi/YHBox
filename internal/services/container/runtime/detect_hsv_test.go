package runtime

import (
	"context"
	"image"
	"image/color"
	"testing"
	"time"

	"yhbox/internal/services/container"
)

// TestDetectColorHSVMatchesYellowFrame: 100x100 全黄帧, HSV 范围 [50,70][200,255][200,255],
// minPixelRatio=0.5 → 应命中 yes 出口, pixelRatio ~1.0。
func TestDetectColorHSVMatchesYellowFrame(t *testing.T) {
	frame := image.NewRGBA(image.Rect(0, 0, 100, 100))
	yellow := color.RGBA{R: 255, G: 255, B: 0, A: 255}
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			frame.Set(x, y, yellow)
		}
	}

	rt, r := newTestRunner(t)
	rt.Capture.(*mockCaptureBackend).FrameROIResult = frame

	node := &container.GraphNode{
		ID:   "hsv1",
		Kind: "DetectColorHSV",
		Config: map[string]any{
			"roi": map[string]any{
				"x": float64(0), "y": float64(0),
				"w": float64(100), "h": float64(100),
			},
			"hsv": map[string]any{
				"hMin": float64(50), "hMax": float64(70),
				"sMin": float64(200), "sMax": float64(255),
				"vMin": float64(200), "vMax": float64(255),
			},
			"minPixelRatio":  float64(0.5),
			"pollIntervalMs": float64(50),
			"timeoutMs":      float64(500),
		},
	}
	_, err := r.execNode(context.Background(), node, ExecToken{InPin: "in"})
	if err != nil {
		t.Fatalf("execNode err: %v", err)
	}
	sys := r.rt.Sys()
	if sys.LastDetect.PixelRatio < 0.9 {
		t.Fatalf("expected pixelRatio ~1.0, got %v", sys.LastDetect.PixelRatio)
	}
	if sys.LastDetect.PixelCount != 10000 {
		t.Fatalf("expected pixelCount=10000, got %d", sys.LastDetect.PixelCount)
	}
}

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
			"roi": map[string]any{
				"x": float64(0), "y": float64(0),
				"w": float64(100), "h": float64(100),
			},
			"hsv": map[string]any{
				"hMin": float64(50), "hMax": float64(70),
				"sMin": float64(200), "sMax": float64(255),
				"vMin": float64(200), "vMax": float64(255),
			},
			"minPixelRatio":  float64(0.5),
			"pollIntervalMs": float64(20),
			"timeoutMs":      float64(100),
		},
	}
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
