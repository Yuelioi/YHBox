package runtime

import (
	"context"
	"image"
	"image/color"
	"testing"

	"yhbox/internal/services/container"
)

// 制造一帧含 3 个垂直黄色 cluster (x=10-15, x=50-55, x=80-83), 其余蓝色.
// scanAxis=x 应输出 3 个 cluster.
func TestROIColorScanFindsThreeClusters(t *testing.T) {
	frame := image.NewRGBA(image.Rect(0, 0, 100, 50))
	for y := 0; y < 50; y++ {
		for x := 0; x < 100; x++ {
			c := color.RGBA{R: 0, G: 0, B: 255, A: 255} // blue bg
			if (x >= 10 && x <= 15) || (x >= 50 && x <= 55) || (x >= 80 && x <= 83) {
				c = color.RGBA{R: 255, G: 255, B: 0, A: 255} // yellow cluster
			}
			frame.Set(x, y, c)
		}
	}
	rt, r := newTestRunner(t)
	rt.Capture.(*mockCaptureBackend).FrameROIResult = frame

	node := &container.GraphNode{
		ID:   "scan1",
		Kind: "ROIColorScan",
		Config: map[string]any{
			"roi":             map[string]any{"x": float64(0), "y": float64(0), "w": float64(100), "h": float64(50)},
			"hsv":             map[string]any{"hMin": float64(50), "hMax": float64(70), "sMin": float64(200), "sMax": float64(255), "vMin": float64(200), "vMax": float64(255)},
			"scanAxis":        "x",
			"minClusterPx":    float64(2),
			"maxClusterPx":    float64(20),
			"minClusterCount": float64(1),
			"pollIntervalMs":  float64(50),
			"timeoutMs":       float64(500),
		},
	}
	_, err := r.execNode(context.Background(), node, ExecToken{InPin: "in"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	sys := r.rt.Sys()
	if sys.LastROIScan.ClusterCount != 3 {
		t.Fatalf("expected 3 clusters, got %d", sys.LastROIScan.ClusterCount)
	}
}
