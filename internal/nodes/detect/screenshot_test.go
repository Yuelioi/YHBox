// internal/nodes/detect/screenshot_test.go
package detect

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"yotta/internal/node"
)

// stubCapture 实现 CaptureService — 跟 input 包 recordingInput 同款思路, 本地一份避免跨包.
type stubCapture struct {
	pngFull []byte
	pngROI  []byte
	err     error
	calls   []string
}

func (s *stubCapture) Capture() ([]byte, error) {
	s.calls = append(s.calls, "Capture")
	return s.pngFull, s.err
}
func (s *stubCapture) CaptureROI(roi node.Geometry) ([]byte, error) {
	s.calls = append(s.calls, "CaptureROI")
	return s.pngROI, s.err
}

func withCapture(c node.CaptureService) node.ServiceBundle {
	b := node.StubServices()
	b.Capture = c
	return b
}

func TestScreenshot_FullFrame(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Screenshot{})
	rn, _ := node.Get("Screenshot")

	// 用 t.TempDir 隔离, YOTTA_DATA_DIR 覆盖默认 bin/data/screenshots.
	tmp := t.TempDir()
	t.Setenv("YOTTA_DATA_DIR", tmp)

	// 全帧路径: ROI pin 不传 → Geometry 零值 → adapter 视为全帧. 节点始终走 CaptureROI.
	cap := &stubCapture{pngROI: []byte{0x89, 0x50, 0x4e, 0x47}} // PNG signature 4 bytes
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{ssInPathTemplate: "test-{ts}.png"},
		nil, withCapture(cap), false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != ssOutDone {
		t.Errorf("exit = %q, want Done", r.ExitName)
	}
	abs := r.OutputData[ssDataPath].(string)
	data, readErr := os.ReadFile(abs)
	if readErr != nil {
		t.Fatalf("expected file written: %v", readErr)
	}
	if len(data) != 4 {
		t.Errorf("file size = %d, want 4 (PNG sig)", len(data))
	}
	if len(cap.calls) != 1 || cap.calls[0] != "CaptureROI" {
		t.Errorf("calls = %v, want [CaptureROI]", cap.calls)
	}
	// abs 必须在 tmp 下
	if !filepath.HasPrefix(abs, tmp) {
		t.Errorf("written path %q not under tmp %q", abs, tmp)
	}
}

func TestScreenshot_ROI(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Screenshot{})
	rn, _ := node.Get("Screenshot")

	t.Setenv("YOTTA_DATA_DIR", t.TempDir())

	// ROI Geometry → CaptureROI; adapter 内 cropFrameByGeometry 按比例裁.
	roi := node.Geometry{Pct: node.Rect{X: 0.1, Y: 0.2, W: 0.5, H: 0.3}}
	cap := &stubCapture{pngROI: []byte{0x01}}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{
			ssInPathTemplate: "roi-{ts}.png",
			ssInROI:          roi,
		},
		nil, withCapture(cap), false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if len(cap.calls) != 1 || cap.calls[0] != "CaptureROI" {
		t.Errorf("calls = %v, want [CaptureROI]", cap.calls)
	}
}

func TestScreenshot_Capture_Path(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Screenshot{})
	rn, _ := node.Get("Screenshot")

	t.Setenv("YOTTA_DATA_DIR", t.TempDir())

	cap := &stubCapture{pngROI: []byte{0x89, 0x50, 0x4e, 0x47}}
	vars := newRecVars()
	b := withCapture(cap)
	b.Vars = vars
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{ssInPathTemplate: "cap-{ts}.png", ssCapPath: "p"},
		nil, b, false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != ssOutDone {
		t.Fatalf("exit = %q, want Done", r.ExitName)
	}
	got, ok := vars.Get("p")
	if !ok {
		t.Fatal("capture p not written")
	}
	// CaptureType=string: 断言写入值的 Go 类型是 string.
	s, isStr := got.(string)
	if !isStr || s == "" {
		t.Errorf("capture p = %v (%T), want non-empty string", got, got)
	}
	if s != r.OutputData[ssDataPath].(string) {
		t.Errorf("capture p = %q, want = Path out %q", s, r.OutputData[ssDataPath])
	}
}

func TestScreenshot_BackendError(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Screenshot{})
	rn, _ := node.Get("Screenshot")

	t.Setenv("YOTTA_DATA_DIR", t.TempDir())

	cap := &stubCapture{err: errors.New("hwnd not found")}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{ssInPathTemplate: "x.png"},
		nil, withCapture(cap), false)

	if r.Error == nil {
		t.Error("expected capture error propagation")
	}
}

func TestScreenshot_UnsafePath_ValidationError(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Screenshot{})
	rn, _ := node.Get("Screenshot")

	for _, bad := range []string{
		"/abs/x.png", `\abs\x.png`, "C:/x.png", "../escape.png", "ok/../bad.png",
	} {
		r := node.RunNode(context.Background(), rn, nil,
			map[string]any{ssInPathTemplate: bad},
			nil, withCapture(&stubCapture{}), false)
		if len(r.Validation) == 0 {
			t.Errorf("path %q: expected UNSAFE_SCREENSHOT_PATH validation", bad)
		}
	}
}
