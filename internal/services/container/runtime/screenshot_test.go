package runtime

import (
	"context"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"yhbox/internal/services/container"
)

// makeRedFrame 创建 10x10 纯红色 RGBA 图像，供截图测试使用。
func makeRedFrame(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}
	return img
}

// makeScreenshotNode 构造一个 Screenshot 节点。
func makeScreenshotNode(id string, cfg map[string]any) *container.GraphNode {
	return &container.GraphNode{ID: id, Kind: "Screenshot", Config: cfg}
}

// TestScreenshotWritesFile 验证截图节点写出 PNG 文件，
// 并将绝对路径存入 $sys.lastScreenshot.path。
func TestScreenshotWritesFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("YHBOX_DATA_DIR", tmpDir)

	rt, r := newTestRunner(t)
	rt.Capture.(*mockCaptureBackend).FrameROIResult = makeRedFrame(10, 10)

	n := makeScreenshotNode("ss1", map[string]any{
		"pathTemplate": "test_{nodeId}.png",
	})

	ctx := context.Background()
	_, err := r.execScreenshot(ctx, n, ExecToken{})
	if err != nil {
		t.Fatalf("execScreenshot returned error: %v", err)
	}

	// 验证 SysState 已写入路径
	gotPath := rt.Sys().LastScreenshot.Path
	if gotPath == "" {
		t.Fatal("LastScreenshot.Path is empty")
	}
	if !strings.Contains(gotPath, "test_ss1.png") {
		t.Errorf("LastScreenshot.Path %q does not contain 'test_ss1.png'", gotPath)
	}

	// 验证文件确实存在于磁盘
	if _, statErr := os.Stat(gotPath); statErr != nil {
		t.Errorf("screenshot file not found on disk: %v", statErr)
	}

	// 验证路径在 tmpDir 下
	if !strings.HasPrefix(filepath.ToSlash(gotPath), filepath.ToSlash(tmpDir)) {
		t.Errorf("screenshot path %q not under tmpDir %q", gotPath, tmpDir)
	}
}

// TestScreenshotRejectsAbsolutePath 验证以盘符开头的绝对路径被拒绝。
func TestScreenshotRejectsAbsolutePath(t *testing.T) {
	_, r := newTestRunner(t)

	n := makeScreenshotNode("ss2", map[string]any{
		"pathTemplate": "C:/evil.png",
	})

	ctx := context.Background()
	_, err := r.execScreenshot(ctx, n, ExecToken{})
	if err == nil {
		t.Fatal("expected error for absolute path, got nil")
	}
	if !strings.Contains(err.Error(), "unsafe") {
		t.Errorf("error %q does not contain 'unsafe'", err.Error())
	}
}

// TestScreenshotRejectsDotDot 验证含 ".." 段的路径被拒绝。
func TestScreenshotRejectsDotDot(t *testing.T) {
	_, r := newTestRunner(t)

	n := makeScreenshotNode("ss3", map[string]any{
		"pathTemplate": "../escape.png",
	})

	ctx := context.Background()
	_, err := r.execScreenshot(ctx, n, ExecToken{})
	if err == nil {
		t.Fatal("expected error for '..' path, got nil")
	}
	if !strings.Contains(err.Error(), "unsafe") {
		t.Errorf("error %q does not contain 'unsafe'", err.Error())
	}
}
