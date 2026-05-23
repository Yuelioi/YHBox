package runtime

import (
	"context"
	"image"
	"image/color"
	"testing"

	"yhbox/internal/services/container"
)

// 用 mockCaptureBackend 注入合成 bar ROI 图 → 期望 found / missing 路径.
// mockCaptureBackend.ClientSize 固定返 1920x1080, 所以 rois entry 用 [1920,1080] 命中.

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
			"rois": []any{
				map[string]any{
					"resolution": []any{float64(1920), float64(1080)},
					"x":          float64(0),
					"y":          float64(0),
					"w":          float64(200),
					"h":          float64(20),
				},
			},
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
			"rois": []any{
				map[string]any{
					"resolution": []any{float64(1920), float64(1080)},
					"x":          float64(0),
					"y":          float64(0),
					"w":          float64(200),
					"h":          float64(20),
				},
			},
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

// pickROIByResolution 单元测试 — 不依赖 runner.

func TestPickROIByResolution_ExactMatch(t *testing.T) {
	rois := []any{
		map[string]any{
			"resolution": []any{float64(1920), float64(1080)},
			"x":          float64(612),
			"y":          float64(69),
			"w":          float64(704),
			"h":          float64(12),
		},
		map[string]any{
			"resolution": []any{float64(1280), float64(720)},
			"x":          float64(410),
			"y":          float64(46),
			"w":          float64(468),
			"h":          float64(8),
		},
	}
	r, ok := pickROIByResolution(rois, 1280, 720)
	if !ok || r.X != 410 || r.Y != 46 || r.W != 468 || r.H != 8 {
		t.Errorf("PickROI(1280,720) = %+v, want {X:410 Y:46 W:468 H:8}", r)
	}
	r, ok = pickROIByResolution(rois, 1920, 1080)
	if !ok || r.X != 612 || r.Y != 69 || r.W != 704 || r.H != 12 {
		t.Errorf("PickROI(1920,1080) = %+v", r)
	}
}

func TestPickROIByResolution_NoMatch(t *testing.T) {
	rois := []any{
		map[string]any{
			"resolution": []any{float64(1920), float64(1080)},
			"x":          float64(612),
			"y":          float64(69),
			"w":          float64(704),
			"h":          float64(12),
		},
	}
	if _, ok := pickROIByResolution(rois, 1366, 768); ok {
		t.Error("PickROI(1366,768) should miss")
	}
}

func TestPickROIByResolution_EmptyOrInvalid(t *testing.T) {
	if _, ok := pickROIByResolution(nil, 1920, 1080); ok {
		t.Error("nil rois should miss")
	}
	if _, ok := pickROIByResolution([]any{}, 1920, 1080); ok {
		t.Error("empty rois should miss")
	}
	if _, ok := pickROIByResolution([]any{"not an object"}, 1920, 1080); ok {
		t.Error("non-object entry should miss")
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
