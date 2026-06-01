package capture

import (
	"image"
	"testing"

	"github.com/lxn/win"
)

// TestCapturer_ConstructFailsOnZeroHWND HWND=0 → 拿不到 DC → 返回 err，不 panic
func TestCapturer_ConstructFailsOnZeroHWND(t *testing.T) {
	rois := []image.Rectangle{image.Rect(0, 0, 10, 10)}
	c, err := newGDICapturer(win.HWND(0), rois)
	if err == nil {
		_ = c.Close()
		t.Fatalf("newGDICapturer with HWND=0 should fail")
	}
}

// TestCapturer_FrameSizeMatchesROIs 4 ROI 各自小图维度正确，绝对坐标 Rect 一致
//
// 需要有效游戏窗口；找不到 t.Skip。
func TestCapturer_FrameSizeMatchesROIs(t *testing.T) {
	hwnd := findCapturerTestWindow()
	if hwnd == 0 {
		t.Skip("no foreground window; need any visible window for live capture test")
	}
	// 这些坐标可能在测试窗口里属于"白板"区域，但只验证 Capturer 几何性质，不验证像素内容
	rois := []image.Rectangle{
		image.Rect(10, 10, 60, 50),    // 50×40
		image.Rect(100, 10, 169, 95),  // 69×85
	}
	c, err := newGDICapturer(hwnd, rois)
	if err != nil {
		t.Fatalf("newGDICapturer: %v", err)
	}
	defer c.Close()

	imgs, err := c.Frame()
	if err != nil {
		t.Fatalf("Frame: %v", err)
	}
	if len(imgs) != len(rois) {
		t.Fatalf("Frame returned %d imgs, want %d", len(imgs), len(rois))
	}
	for i, want := range rois {
		got := imgs[i].Bounds()
		if got != want {
			t.Errorf("imgs[%d].Bounds = %v, want %v", i, got, want)
		}
		// Pix 长度应正好等于 ROI 像素数 × 4
		wantLen := want.Dx() * want.Dy() * 4
		if len(imgs[i].Pix) != wantLen {
			t.Errorf("imgs[%d].Pix len = %d, want %d (ROI %dx%d × 4)",
				i, len(imgs[i].Pix), wantLen, want.Dx(), want.Dy())
		}
		// Stride = ROI 宽 × 4
		if imgs[i].Stride != want.Dx()*4 {
			t.Errorf("imgs[%d].Stride = %d, want %d", i, imgs[i].Stride, want.Dx()*4)
		}
	}
}

// TestCapturer_BuffersAreReused 抓 2 次，返回的每个 *image.RGBA 的 Pix slice 第一字节
// 地址跟第一次相同（同一 buffer）。这是 0-alloc 的最直接验证。
func TestCapturer_BuffersAreReused(t *testing.T) {
	hwnd := findCapturerTestWindow()
	if hwnd == 0 {
		t.Skip("no foreground window")
	}
	rois := []image.Rectangle{
		image.Rect(10, 10, 30, 30),
		image.Rect(50, 10, 70, 30),
	}
	c, err := newGDICapturer(hwnd, rois)
	if err != nil {
		t.Fatalf("newGDICapturer: %v", err)
	}
	defer c.Close()

	imgs1, err := c.Frame()
	if err != nil {
		t.Fatalf("Frame 1: %v", err)
	}
	addr1 := []*byte{&imgs1[0].Pix[0], &imgs1[1].Pix[0]}

	imgs2, err := c.Frame()
	if err != nil {
		t.Fatalf("Frame 2: %v", err)
	}
	for i := range rois {
		if &imgs2[i].Pix[0] != addr1[i] {
			t.Errorf("Buffer %d not reused: addr changed", i)
		}
	}
}

// TestCapturer_CloseIdempotent 重复 Close 不 panic
func TestCapturer_CloseIdempotent(t *testing.T) {
	hwnd := findCapturerTestWindow()
	if hwnd == 0 {
		t.Skip("no foreground window")
	}
	c, err := newGDICapturer(hwnd, []image.Rectangle{image.Rect(0, 0, 10, 10)})
	if err != nil {
		t.Fatalf("newGDICapturer: %v", err)
	}
	if err := c.Close(); err != nil {
		t.Errorf("first Close: %v", err)
	}
	if err := c.Close(); err != nil {
		t.Errorf("second Close: %v", err)
	}
}

func findCapturerTestWindow() win.HWND {
	h := win.GetForegroundWindow()
	if h == 0 || !win.IsWindowVisible(h) {
		return 0
	}
	return h
}
