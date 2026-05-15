package capture

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// writeTestPNG 在 dir 下写一张 w×h 纯色 PNG，给 mock 加载用。
func writeTestPNG(t *testing.T, dir, name string, w, h int, c color.RGBA) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i+0] = c.R
		img.Pix[i+1] = c.G
		img.Pix[i+2] = c.B
		img.Pix[i+3] = 255
	}
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

// resetMockState 让每个 test case 独立加载 dir。mock.go 用 sync.Once 一次性
// 加载，测试里要重置全局状态绕过 Once。
func resetMockState() {
	mockFrames = nil
	mockLoadErr = nil
	mockCursorPkg.Store(0)
	// 重置 sync.Once: 没法直接 reset Once，用替换 Once 实例的办法
	mockOnce = newOnce()
}

// newOnce 是测试 helper 帮 reset sync.Once。
func newOnce() sync.Once { return sync.Once{} }

func TestMockLoadsAndCycles(t *testing.T) {
	dir := t.TempDir()
	writeTestPNG(t, dir, "01.png", 100, 50, color.RGBA{255, 0, 0, 255}) // 红
	writeTestPNG(t, dir, "02.png", 100, 50, color.RGBA{0, 255, 0, 255}) // 绿

	t.Setenv("YHBOX_MOCK_DIR", dir)
	resetMockState()

	if err := initMock(); err != nil {
		t.Fatalf("initMock: %v", err)
	}
	if len(mockFrames) != 2 {
		t.Fatalf("expect 2 frames, got %d", len(mockFrames))
	}

	// 第一次 Frame 应该返回 01.png（红）
	img1, _ := mockFrame(0)
	r, g, b, _ := img1.At(0, 0).RGBA()
	if r>>8 != 255 || g>>8 != 0 || b>>8 != 0 {
		t.Errorf("frame 1 应红, got R=%d G=%d B=%d", r>>8, g>>8, b>>8)
	}

	// 第二次 Frame 应该返回 02.png（绿）
	img2, _ := mockFrame(0)
	r, g, b, _ = img2.At(0, 0).RGBA()
	if r>>8 != 0 || g>>8 != 255 || b>>8 != 0 {
		t.Errorf("frame 2 应绿, got R=%d G=%d B=%d", r>>8, g>>8, b>>8)
	}

	// 第三次回到红 (cursor wrap)
	img3, _ := mockFrame(0)
	r, _, _, _ = img3.At(0, 0).RGBA()
	if r>>8 != 255 {
		t.Errorf("frame 3 (wrap) 应红, got R=%d", r>>8)
	}
}

func TestMockEmptyDirFails(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("YHBOX_MOCK_DIR", dir)
	resetMockState()
	if err := initMock(); err == nil {
		t.Fatal("空目录应该返错")
	}
}

func TestMockCapturer(t *testing.T) {
	dir := t.TempDir()
	writeTestPNG(t, dir, "01.png", 100, 50, color.RGBA{255, 0, 0, 255})
	t.Setenv("YHBOX_MOCK_DIR", dir)
	resetMockState()

	rois := []image.Rectangle{
		image.Rect(10, 10, 30, 30), // 20×20
	}
	c, err := newMockCapturer(0, rois)
	if err != nil {
		t.Fatalf("newMockCapturer: %v", err)
	}
	defer c.Close()
	imgs, err := c.Frame()
	if err != nil {
		t.Fatalf("Frame: %v", err)
	}
	if len(imgs) != 1 {
		t.Fatalf("expect 1 ROI image, got %d", len(imgs))
	}
	if imgs[0].Bounds().Dx() != 20 || imgs[0].Bounds().Dy() != 20 {
		t.Errorf("ROI size wrong: %v", imgs[0].Bounds())
	}
	// ROI 内全红 — imgs[0].Rect 是 ROI 全局坐标 (10,10,30,30)，At(0,0) 不在范围
	// 内会返 zero（capture.Capturer 的现有约定，caller 用 .Pix 直接访问）。
	pix := imgs[0].Pix
	if pix[0] != 255 || pix[1] != 0 || pix[2] != 0 {
		t.Errorf("ROI[0,0] 应红 (R=255 G=0 B=0), got R=%d G=%d B=%d", pix[0], pix[1], pix[2])
	}
}
