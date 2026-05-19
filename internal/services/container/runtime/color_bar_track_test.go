package runtime

import (
	"context"
	"image"
	"image/color"
	"testing"

	"yhbox/internal/services/container"
)

// 用 mockCaptureBackend 注入合成 bar ROI 图 → 期望 found / missing 路径.

func TestExecColorBarTrack_HappyPath_Found(t *testing.T) {
	rt, r := newTestRunner(t)
	// 合成有黄 cursor + 青 target 的 ROI 图
	img := image.NewRGBA(image.Rect(0, 0, 200, 20))
	paintCursorBar(img, 50)
	paintTargetBar(img, 100, 115)
	rt.Capture.(*mockCaptureBackend).FrameROIResult = img

	node := &container.GraphNode{
		ID:   "cbt1",
		Kind: "ColorBarTrack",
		Config: map[string]any{
			"roi": map[string]any{"x": float64(0), "y": float64(0), "w": float64(200), "h": float64(20)},
		},
	}
	if r.nodesByID == nil {
		r.nodesByID = map[string]*container.GraphNode{}
	}
	r.nodesByID[node.ID] = node
	_, err := r.execColorBarTrack(context.Background(), node, ExecToken{InPin: "in"})
	if err != nil {
		t.Fatalf("happy path 不应报错: %v", err)
	}
	sys := rt.Sys()
	if sys.LastBarTrack.CursorX < 0 {
		t.Errorf("Sys.LastBarTrack.CursorX 应 >= 0, got %d", sys.LastBarTrack.CursorX)
	}
	if sys.LastBarTrack.YellowPx == 0 {
		t.Errorf("YellowPx 应 > 0")
	}
}

func TestExecColorBarTrack_BlackImage_Missing(t *testing.T) {
	rt, r := newTestRunner(t)
	img := image.NewRGBA(image.Rect(0, 0, 200, 20))
	// 全黑 — 默认色 (0,0,0)
	rt.Capture.(*mockCaptureBackend).FrameROIResult = img

	node := &container.GraphNode{
		ID:   "cbt2",
		Kind: "ColorBarTrack",
		Config: map[string]any{
			"roi": map[string]any{"x": float64(0), "y": float64(0), "w": float64(200), "h": float64(20)},
		},
	}
	if r.nodesByID == nil {
		r.nodesByID = map[string]*container.GraphNode{}
	}
	r.nodesByID[node.ID] = node
	_, err := r.execColorBarTrack(context.Background(), node, ExecToken{InPin: "in"})
	if err != nil {
		t.Fatalf("黑图不应错: %v", err)
	}
	// 全黑图: cursor 找不到 → CursorX=-1, 存入 sys 后 CursorX 为 -1
	sys := rt.Sys()
	if sys.LastBarTrack.CursorX >= 0 {
		t.Errorf("黑图 CursorX 应 <0, got %d", sys.LastBarTrack.CursorX)
	}
}

// paintCursorBar / paintTargetBar 跟 pkg/vision/bar_track_test 同款 helper.

func paintCursorBar(img *image.RGBA, x int) {
	cursor := color.RGBA{R: 255, G: 240, B: 60, A: 255} // H=55 S=195 V=255; H[45,70] S>=40 V>=200
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for dx := -1; dx <= 1; dx++ {
			img.Set(x+dx, y, cursor)
		}
	}
}

func paintTargetBar(img *image.RGBA, x0, x1 int) {
	target := color.RGBA{R: 0, G: 230, B: 200, A: 255} // H=172 S=255 V=230; H[160,180] S>=140 V>=100
	bounds := img.Bounds()
	midY := (bounds.Min.Y + bounds.Max.Y) / 2
	for y := midY - 5; y <= midY+5; y++ {
		for x := x0; x <= x1; x++ {
			img.Set(x, y, target)
		}
	}
}
