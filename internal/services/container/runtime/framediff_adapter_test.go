package runtime

import (
	"bytes"
	"image"
	"image/color"
	"testing"

	"github.com/yottaapp/yotta/internal/node"
	"github.com/yottaapp/yotta/pkg/vision"
)

// TestVisionAdapter_GridSignature_FullFrame: Geometry 零值 → 全帧路径 → Capture.Frame → vision.Downsample.
func TestVisionAdapter_GridSignature_FullFrame(t *testing.T) {
	frame := image.NewRGBA(image.Rect(0, 0, 8, 8))
	red := color.RGBA{R: 200, G: 10, B: 20, A: 255}
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			frame.SetRGBA(x, y, red)
		}
	}

	rt, _ := newTestRunner(t)
	installedTestWin32Capture(rt).(*mockCaptureBackend).FrameROIResult = frame
	a := NewVisionAdapter(rt)

	// Geometry 零值 → fullFrame=true → cropFrameByGeometry 直接返原帧.
	sig, err := a.GridSignature(node.Geometry{}, 2)
	if err != nil {
		t.Fatal(err)
	}
	want := vision.Downsample(frame, 2)
	if !bytes.Equal(sig, want) {
		t.Errorf("sig = %v, want %v", sig, want)
	}
}

// TestVisionAdapter_GridSignature_PctROI: Pct ROI → cropFrameByGeometry 裁子区 → Downsample.
func TestVisionAdapter_GridSignature_PctROI(t *testing.T) {
	// 8×8 帧: 左半 (x<4) 红, 右半 (x>=4) 蓝.
	frame := image.NewRGBA(image.Rect(0, 0, 8, 8))
	red := color.RGBA{R: 200, G: 10, B: 20, A: 255}
	blue := color.RGBA{R: 10, G: 20, B: 200, A: 255}
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			if x < 4 {
				frame.SetRGBA(x, y, red)
			} else {
				frame.SetRGBA(x, y, blue)
			}
		}
	}

	rt, _ := newTestRunner(t)
	installedTestWin32Capture(rt).(*mockCaptureBackend).FrameROIResult = frame
	a := NewVisionAdapter(rt)

	// ROI = 左半帧 (x=0,y=0,w=0.5,h=1.0) → 4×8 全红子区.
	roi := node.Geometry{Pct: node.Rect{X: 0, Y: 0, W: 0.5, H: 1.0}}
	sig, err := a.GridSignature(roi, 1)
	if err != nil {
		t.Fatal(err)
	}

	// 预期: 左半帧全红 → 1×1 grid → [≈200, ≈10, ≈20].
	if len(sig) != 3 {
		t.Fatalf("sig len = %d, want 3", len(sig))
	}
	// R 明显大于 B.
	if int(sig[0]) < int(sig[2]) {
		t.Errorf("expected red-dominant sig, got R=%d B=%d", sig[0], sig[2])
	}
}
