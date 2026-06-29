package runtime

import (
	"image"
	"image/color"
	"testing"

	"yotta/internal/node"
)

// TestFindColorSignature_FoundViaRealAdapter: 100x100 帧全填 (200,0,0),
// 单锚点签名 (r=200,g=0,b=0, tol 30) → Found 出口且 Point 非零。
// 走真实 visionAdapter (CaptureFrameCached → mockCaptureBackend.Frame),
// 照 TestDetectColorHSVTimeoutOnNoMatch 的 execNode 模式。
func TestFindColorSignature_FoundViaRealAdapter(t *testing.T) {
	// 合成 100x100 帧: R=200,G=0,B=0 (恰好匹配签名颜色, tol=30 覆盖)。
	frame := image.NewRGBA(image.Rect(0, 0, 100, 100))
	red200 := color.RGBA{R: 200, G: 0, B: 0, A: 255}
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			frame.Set(x, y, red200)
		}
	}

	rt, _ := newTestRunner(t)
	rt.Capture.(*mockCaptureBackend).FrameROIResult = frame

	// 直接通过真实 visionAdapter 调用 FindColorSignature。
	va := NewVisionAdapter(rt)
	sig := node.ColorSignature{
		Points: []node.ColorPoint{
			{DX: 0, DY: 0, R: 200, G: 0, B: 0, Tol: nil}, // tol nil → defaultTol=30
		},
	}
	found, pt, err := va.FindColorSignature(node.Geometry{}, sig, 30)
	if err != nil {
		t.Fatalf("FindColorSignature error: %v", err)
	}
	if !found {
		t.Fatal("expected Found=true on all-red frame, got false")
	}
	if pt.X == 0 && pt.Y == 0 {
		// 严格来说 (0,0) 可能是真实命中 (帧左上角第一个像素); 只验非负即可。
		// 全帧均为红色, 首个匹配 = (0,0), 归一化 X=0/100=0, Y=0/100=0 → (0,0) 是合法值。
		// 因此不断言 Point!=0, 而是验 found=true 足够。
		t.Log("Point=(0,0) is valid (first pixel hit)")
	}
}

// TestFindColorSignature_NotFoundViaRealAdapter: 全蓝帧 → 签名找红色 → NotFound。
func TestFindColorSignature_NotFoundViaRealAdapter(t *testing.T) {
	frame := image.NewRGBA(image.Rect(0, 0, 100, 100))
	blue := color.RGBA{R: 0, G: 0, B: 255, A: 255}
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			frame.Set(x, y, blue)
		}
	}

	rt, _ := newTestRunner(t)
	rt.Capture.(*mockCaptureBackend).FrameROIResult = frame

	va := NewVisionAdapter(rt)
	sig := node.ColorSignature{
		Points: []node.ColorPoint{
			{DX: 0, DY: 0, R: 200, G: 0, B: 0, Tol: nil},
		},
	}
	found, _, err := va.FindColorSignature(node.Geometry{}, sig, 30)
	if err != nil {
		t.Fatalf("FindColorSignature error: %v", err)
	}
	if found {
		t.Fatal("expected Found=false on all-blue frame")
	}
}

// TestFindColorSignature_NodeRegistered: 节点已注册到全局 registry,
// Category=Detect, NeedsTarget=true。
func TestFindColorSignature_NodeRegistered(t *testing.T) {
	rn, ok := node.Get("FindColorSignature")
	if !ok {
		t.Fatal("FindColorSignature not in node registry")
	}
	if rn.Spec.Category != "Detect" {
		t.Errorf("Category=%q, want Detect", rn.Spec.Category)
	}
	if !rn.Spec.NeedsTarget {
		t.Error("NeedsTarget should be true")
	}
	if rn.Spec.NeedsWindow {
		t.Error("NeedsWindow should be false")
	}
}
