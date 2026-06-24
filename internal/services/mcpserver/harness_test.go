package mcpserver

import (
	"context"
	"image"
	"testing"

	"github.com/lxn/win"

	"yotta/internal/services/container"
	"yotta/internal/services/container/runtime"
	"yotta/internal/services/execution"
	pkgcapture "yotta/pkg/capture"
	pkginput "yotta/pkg/input"
	"yotta/pkg/winutil"

	// 注册节点
	_ "yotta/internal/nodes/collection"
	_ "yotta/internal/nodes/control"
	_ "yotta/internal/nodes/detect"
	_ "yotta/internal/nodes/event"
	_ "yotta/internal/nodes/image"
	_ "yotta/internal/nodes/input"
	_ "yotta/internal/nodes/io"
	_ "yotta/internal/nodes/purefunc"
	_ "yotta/internal/nodes/random"
	_ "yotta/internal/nodes/stopwatch"
	_ "yotta/internal/nodes/system"
	_ "yotta/internal/nodes/variable"
)

// ---------------------------------------------------------------------------
// Compile-time interface assertions
// ---------------------------------------------------------------------------

var _ pkginput.Backend = (*mockInput)(nil)
var _ pkgcapture.IBackend = (*mockCapture)(nil)

// ---------------------------------------------------------------------------
// Mock backends (cross-package copies of runtime package's test mocks)
// ---------------------------------------------------------------------------

// mockCapture implements pkgcapture.IBackend
type mockCapture struct{ frame *image.RGBA }

func (m *mockCapture) Name() string { return "mock-test" }
func (m *mockCapture) Frame(_ win.HWND) (*image.RGBA, error) {
	return m.frame, nil
}
func (m *mockCapture) FrameROI(_ win.HWND, _, _, _, _ int) (*image.RGBA, error) {
	if m.frame == nil {
		return nil, image.ErrFormat
	}
	return m.frame, nil
}
func (m *mockCapture) ClientSize(_ win.HWND) (int, int, error) { return 1920, 1080, nil }
func (m *mockCapture) Close() error                            { return nil }

// mockInput implements pkginput.Backend
type mockInput struct{ clicks int }

func (f *mockInput) Name() string                        { return "fake" }
func (f *mockInput) Capabilities() pkginput.Capabilities { return pkginput.Capabilities{} }
func (f *mockInput) Click(_ win.HWND, _, _ float64, _ string, _ int) error {
	f.clicks++
	return nil
}
func (f *mockInput) KeyDown(win.HWND, string) error                     { return nil }
func (f *mockInput) KeyUp(win.HWND, string) error                       { return nil }
func (f *mockInput) MouseDown(win.HWND, float64, float64, string) error { return nil }
func (f *mockInput) MouseUp(win.HWND, string) error                     { return nil }
func (f *mockInput) MouseMoveRel(win.HWND, int, int, int) error         { return nil }
func (f *mockInput) MoveTo(win.HWND, float64, float64) error            { return nil }
func (f *mockInput) CursorRatio(win.HWND) (float64, float64, error)    { return 0, 0, nil }
func (f *mockInput) Scroll(win.HWND, float64, float64, int) error       { return nil }
func (f *mockInput) Drag(win.HWND, float64, float64, float64, float64, string, int) error {
	return nil
}
func (f *mockInput) ReleaseAll() error                                  { return nil }
func (f *mockInput) Close() error                                       { return nil }

// ---------------------------------------------------------------------------
// Helper: build a RuntimeContext with mock backends injected
// ---------------------------------------------------------------------------

func newMockRT(c *container.Container) *runtime.RuntimeContext {
	rt := runtime.NewRuntimeContext(c, execution.NewInputBus(), runtime.NoopMatcher{}, nil, nil, nil, 0)
	rt.SetActiveWindow(winutil.WindowHandle{HWND: 1})
	rt.Input = &mockInput{}
	rt.Capture = &mockCapture{frame: image.NewRGBA(image.Rect(0, 0, 4, 4))}
	return rt
}

// ---------------------------------------------------------------------------
// buildMicroContainer tests (pure function)
// ---------------------------------------------------------------------------

func TestBuildMicroContainer_WiresStartToExecIn(t *testing.T) {
	c, nodeID, err := buildMicroContainer("ClickAt", map[string]any{"X": 10, "Y": 20})
	if err != nil {
		t.Fatalf("意外 err: %v", err)
	}
	if len(c.Graph.Nodes) != 2 {
		t.Fatalf("应是 Start+目标 两节点, got %d", len(c.Graph.Nodes))
	}
	// 边: start.Done → <nodeID>.<execIn>
	if len(c.Graph.Edges) != 1 || c.Graph.Edges[0].From != "start.Done" {
		t.Fatalf("边接线错: %+v", c.Graph.Edges)
	}
	// 目标节点 config.literal 带上了 params
	var target *container.GraphNode
	for i := range c.Graph.Nodes {
		if c.Graph.Nodes[i].ID == nodeID {
			target = &c.Graph.Nodes[i]
		}
	}
	if target == nil {
		t.Fatalf("找不到节点 %q", nodeID)
	}
	lit, ok := target.Config["literal"].(map[string]any)
	if !ok {
		t.Fatalf("config.literal 不是 map[string]any: %T", target.Config["literal"])
	}
	if lit["X"] != 10 {
		t.Fatalf("params 没进 literal: %+v", lit)
	}

	if _, _, err := buildMicroContainer("Loop", nil); err == nil {
		t.Error("Loop 不可跑, 应返 err")
	}
}

// ---------------------------------------------------------------------------
// runMicroContainer tests (integration: micro-container → run → harvest)
// ---------------------------------------------------------------------------

func TestRunMicroContainer_HarvestsCaptureImage(t *testing.T) {
	c, nodeID, err := buildMicroContainer("Capture", nil)
	if err != nil {
		t.Fatalf("buildMicroContainer: %v", err)
	}
	rt := newMockRT(c)
	res, img := runMicroContainer(context.Background(), rt, c, nodeID)
	if !res.Ok {
		t.Fatalf("应成功, 但得到 error: %+v", res.Error)
	}
	if img == nil {
		t.Fatal("应收割到 Image, 得到 nil")
	}
	if img.Format != "png" {
		t.Errorf("Image.Format = %q, want \"png\"", img.Format)
	}
	if len(img.Data) == 0 {
		t.Error("Image.Data 为空")
	}
	// firedOutput 应识别 Capture 节点的 Done 出口 (含 Image 字段)
	// Image 单独走 img 路径, 不进 data map, 所以 FiredOutput 可能为 ""
	// 核心断言: img != nil 且有合法 PNG 数据
	t.Logf("FiredOutput=%q, data=%v", res.FiredOutput, res.Data)
}
