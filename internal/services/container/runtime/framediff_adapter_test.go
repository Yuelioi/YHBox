package runtime

import (
	"bytes"
	"image"
	"image/color"
	"testing"

	"yhbox/internal/node"
	"yhbox/pkg/vision"
)

// TestVisionAdapter_GridSignature: 全帧路径 (roi 零) → Capture.Frame → vision.Downsample.
func TestVisionAdapter_GridSignature(t *testing.T) {
	frame := image.NewRGBA(image.Rect(0, 0, 8, 8))
	red := color.RGBA{R: 200, G: 10, B: 20, A: 255}
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			frame.SetRGBA(x, y, red)
		}
	}

	rt, _ := newTestRunner(t)
	rt.Capture.(*mockCaptureBackend).FrameROIResult = frame
	a := NewVisionAdapter(rt)

	sig, err := a.GridSignature(node.Rect{}, 2)
	if err != nil {
		t.Fatal(err)
	}
	want := vision.Downsample(frame, 2)
	if !bytes.Equal(sig, want) {
		t.Errorf("sig = %v, want %v", sig, want)
	}
}
