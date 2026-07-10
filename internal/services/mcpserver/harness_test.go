package mcpserver

import (
	"context"
	"image"
	"testing"

	"github.com/yottaapp/yotta/internal/automation/controller"
	"github.com/yottaapp/yotta/internal/automation/target"
	automationtrace "github.com/yottaapp/yotta/internal/automation/trace"
	"github.com/yottaapp/yotta/internal/services/container"
	"github.com/yottaapp/yotta/internal/services/container/runtime"
	"github.com/yottaapp/yotta/internal/services/execution"
	"github.com/yottaapp/yotta/pkg/winutil"

	_ "github.com/yottaapp/yotta/internal/nodes/all"
)

type mockControllerFactory struct {
	frame *image.RGBA
}

func (f mockControllerFactory) NewController(tg target.Target, rec automationtrace.Recorder) (controller.Controller, error) {
	return controller.NewWin32Controller(tg, controller.Win32Deps{
		Input:   mockControllerInput{},
		Capture: mockControllerCapture(f),
		Trace:   rec,
		Backend: "mock-test",
	})
}

type mockControllerInput struct{}

func (a mockControllerInput) Click(hwnd uintptr, x, y float64, button string, durationMs int) error {
	return nil
}
func (a mockControllerInput) MouseDown(hwnd uintptr, x, y float64, button string) error {
	return nil
}
func (a mockControllerInput) MouseUp(hwnd uintptr, button string) error {
	return nil
}
func (a mockControllerInput) Drag(hwnd uintptr, x1, y1, x2, y2 float64, button string, durationMs int) error {
	return nil
}
func (a mockControllerInput) MouseMoveRel(hwnd uintptr, dx, dy, durationMs int) error {
	return nil
}
func (a mockControllerInput) KeyDown(hwnd uintptr, key string) error {
	return nil
}
func (a mockControllerInput) KeyUp(hwnd uintptr, key string) error {
	return nil
}
func (a mockControllerInput) TypeText(hwnd uintptr, value string) error {
	return nil
}
func (a mockControllerInput) MoveTo(hwnd uintptr, x, y float64) error {
	return nil
}
func (a mockControllerInput) Scroll(hwnd uintptr, x, y float64, notches int, horizontal bool) error {
	return nil
}
func (a mockControllerInput) CursorRatio(hwnd uintptr) (float64, float64, error) {
	return 0, 0, nil
}

type mockControllerCapture struct{ frame *image.RGBA }

func (a mockControllerCapture) Frame(hwnd uintptr) (controller.Frame, error) {
	size := target.Size{}
	if a.frame != nil {
		size = target.Size{W: a.frame.Bounds().Dx(), H: a.frame.Bounds().Dy()}
	}
	return controller.Frame{Image: a.frame, Space: target.SpaceWindowClient, Size: size}, nil
}

// ---------------------------------------------------------------------------
// Helper: build a RuntimeContext with mock backends injected
// ---------------------------------------------------------------------------

func newMockRT(c *container.Container) *runtime.RuntimeContext {
	rt := runtime.NewRuntimeContext(c, execution.NewInputBus(), runtime.NoopMatcher{}, nil, nil, nil, 0)
	rt.SetActiveWindow(winutil.WindowHandle{HWND: 1})
	rt.SetWin32ControllerFactory(mockControllerFactory{frame: image.NewRGBA(image.Rect(0, 0, 4, 4))})
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
