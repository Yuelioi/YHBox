package runtime

import (
	"fmt"
	"image"
	"testing"

	"github.com/yottaapp/yotta/internal/automation/controller"
	"github.com/yottaapp/yotta/internal/automation/target"
	automationtrace "github.com/yottaapp/yotta/internal/automation/trace"
	"github.com/yottaapp/yotta/internal/services/container"
	"github.com/yottaapp/yotta/internal/services/execution"
	pkgcapture "github.com/yottaapp/yotta/pkg/capture"
	pkginput "github.com/yottaapp/yotta/pkg/input"
	"github.com/yottaapp/yotta/pkg/winutil"
)

type testWin32ControllerProvider struct {
	input   pkginput.Backend
	capture pkgcapture.IBackend
}

func installTestWin32Input(rt *RuntimeContext, input pkginput.Backend) {
	p := testWin32Provider(rt)
	p.input = input
}

func installTestWin32Capture(rt *RuntimeContext, capture pkgcapture.IBackend) {
	p := testWin32Provider(rt)
	p.capture = capture
}

func installedTestWin32Capture(rt *RuntimeContext) pkgcapture.IBackend {
	p, ok := rt.win32Provider.(*testWin32ControllerProvider)
	if !ok {
		return nil
	}
	return p.capture
}

func testWin32Provider(rt *RuntimeContext) *testWin32ControllerProvider {
	if p, ok := rt.win32Provider.(*testWin32ControllerProvider); ok {
		return p
	}
	p := &testWin32ControllerProvider{}
	rt.win32Provider = p
	return p
}

func (p *testWin32ControllerProvider) NewController(tg target.Target, rec automationtrace.Recorder, need controllerNeed) (controller.Controller, error) {
	deps := controller.Win32Deps{Trace: rec}
	if need.Input {
		if p.input == nil {
			return nil, fmt.Errorf("test input backend not installed")
		}
		deps.Input = testRuntimeWin32Input{backend: p.input}
		deps.Backend = p.input.Name()
	}
	if need.Capture {
		deps.Capture = testRuntimeWin32Capture{backend: p.capture}
		if p.capture != nil {
			deps.Backend = p.capture.Name()
		}
	}
	return controller.NewWin32Controller(tg, deps)
}

func (p *testWin32ControllerProvider) Close() error {
	if p.input != nil {
		_ = p.input.ReleaseAll()
		_ = p.input.Close()
	}
	if p.capture != nil {
		_ = p.capture.Close()
	}
	return nil
}

type testRuntimeWin32Input struct{ backend pkginput.Backend }

func (a testRuntimeWin32Input) Click(hwnd uintptr, x, y float64, button string, durationMs int) error {
	return a.backend.Click(pkginput.Handle(hwnd), x, y, button, durationMs)
}
func (a testRuntimeWin32Input) MouseDown(hwnd uintptr, x, y float64, button string) error {
	return a.backend.MouseDown(pkginput.Handle(hwnd), x, y, button)
}
func (a testRuntimeWin32Input) MouseUp(hwnd uintptr, button string) error {
	return a.backend.MouseUp(pkginput.Handle(hwnd), button)
}
func (a testRuntimeWin32Input) Drag(hwnd uintptr, x1, y1, x2, y2 float64, button string, durationMs int) error {
	return a.backend.Drag(pkginput.Handle(hwnd), x1, y1, x2, y2, button, durationMs)
}
func (a testRuntimeWin32Input) MouseMoveRel(hwnd uintptr, dx, dy, durationMs int) error {
	return a.backend.MouseMoveRel(pkginput.Handle(hwnd), dx, dy, durationMs)
}
func (a testRuntimeWin32Input) KeyDown(hwnd uintptr, key string) error {
	return a.backend.KeyDown(pkginput.Handle(hwnd), key)
}
func (a testRuntimeWin32Input) KeyUp(hwnd uintptr, key string) error {
	return a.backend.KeyUp(pkginput.Handle(hwnd), key)
}
func (a testRuntimeWin32Input) TypeText(hwnd uintptr, value string) error {
	return a.backend.TypeText(pkginput.Handle(hwnd), value)
}
func (a testRuntimeWin32Input) MoveTo(hwnd uintptr, x, y float64) error {
	return a.backend.MoveTo(pkginput.Handle(hwnd), x, y)
}
func (a testRuntimeWin32Input) Scroll(hwnd uintptr, x, y float64, notches int, horizontal bool) error {
	return a.backend.Scroll(pkginput.Handle(hwnd), x, y, notches, horizontal)
}
func (a testRuntimeWin32Input) CursorRatio(hwnd uintptr) (float64, float64, error) {
	return a.backend.CursorRatio(pkginput.Handle(hwnd))
}

type testRuntimeWin32Capture struct{ backend pkgcapture.IBackend }

func (a testRuntimeWin32Capture) Frame(hwnd uintptr) (controller.Frame, error) {
	if a.backend == nil {
		return controller.Frame{Space: target.SpaceWindowClient}, nil
	}
	img, err := a.backend.Frame(pkgcapture.Handle(hwnd))
	if err != nil {
		return controller.Frame{}, err
	}
	size := target.Size{}
	if img != nil {
		size = target.Size{W: img.Bounds().Dx(), H: img.Bounds().Dy()}
	}
	return controller.Frame{Image: img, Space: target.SpaceWindowClient, Size: size}, nil
}

// newTestRunnerWithSubgraph 构造一个最小容器 + 注入子图 + stub backend,
// 返回 (RuntimeContext, ContainerRunner) — region (Subgraph/CollapsedNode) 测试用.
// sgID: 子图 ID; sgNodes/sgEdges: 子图内节点+边.
func newTestRunnerWithSubgraph(t *testing.T, sgID string, sgNodes []*container.GraphNode, sgEdges []container.GraphEdge) (*RuntimeContext, *ContainerRunner) {
	t.Helper()
	// 展开 []*GraphNode → []GraphNode (Container 持久化层用值类型)
	nodes := make([]container.GraphNode, len(sgNodes))
	for i, n := range sgNodes {
		nodes[i] = *n
	}
	sg := container.Subgraph{
		ID:    sgID,
		Label: sgID,
		Graph: container.Graph{Nodes: nodes, Edges: sgEdges},
	}
	// 主图只需 Start 节点 (region 测试直接调 execNode, 不走 Run)
	c := &container.Container{
		SchemaVersion: 1,
		ID:            "test-try",
		Name:          "test-try",
		Graph: container.Graph{
			Nodes: []container.GraphNode{{ID: "start", Kind: "Start"}},
		},
	}
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, nil, nil, nil, 0)
	rt.Subgraphs = []container.Subgraph{sg}
	// stub Window + controller provider 让 setupRuntime 幂等跳过
	rt.SetActiveWindow(winutil.WindowHandle{HWND: 1})
	installTestWin32Input(rt, &fakeInputBackend{})
	r := NewContainerRunner(rt)
	return rt, r
}

// newTestRunner 构造一个最小容器 + mockCaptureBackend, 用于 DetectColorHSV 等 Capture 测试.
// 返回 (RuntimeContext, ContainerRunner).
func newTestRunner(t *testing.T) (*RuntimeContext, *ContainerRunner) {
	t.Helper()
	c := &container.Container{
		SchemaVersion: 1,
		ID:            "test-capture",
		Name:          "test-capture",
		Graph: container.Graph{
			Nodes: []container.GraphNode{{ID: "start", Kind: "Start"}},
		},
	}
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, nil, nil, nil, 0)
	rt.SetActiveWindow(winutil.WindowHandle{HWND: 1})
	installTestWin32Input(rt, &fakeInputBackend{})
	mock := &mockCaptureBackend{}
	installTestWin32Capture(rt, mock)
	r := NewContainerRunner(rt)
	return rt, r
}

// mockCaptureBackend 测试用 pkgcapture.IBackend 实现.
// FrameROIResult: 下一次 FrameROI 返回的帧 (nil = 返 error).
type mockCaptureBackend struct {
	FrameROIResult *image.RGBA
}

var _ pkgcapture.IBackend = (*mockCaptureBackend)(nil)

func (m *mockCaptureBackend) Name() string { return "mock-test" }
func (m *mockCaptureBackend) Frame(_ pkgcapture.Handle) (*image.RGBA, error) {
	return m.FrameROIResult, nil
}
func (m *mockCaptureBackend) FrameROI(_ pkgcapture.Handle, _, _, _, _ int) (*image.RGBA, error) {
	if m.FrameROIResult == nil {
		return nil, image.ErrFormat
	}
	return m.FrameROIResult, nil
}
func (m *mockCaptureBackend) ClientSize(_ pkgcapture.Handle) (int, int, error) {
	return 1920, 1080, nil
}
func (m *mockCaptureBackend) Close() error { return nil }

// fakeInputBackend 测试用 pkginput.Backend 实现. 计数 Click 调用次数, 其他方法返 nil.
// 共享给所有 _test.go (runner_test / playclip_test / safe_backend_test 等).
type fakeInputBackend struct {
	clicks int
}

func (f *fakeInputBackend) Name() string                        { return "fake" }
func (f *fakeInputBackend) Capabilities() pkginput.Capabilities { return pkginput.Capabilities{} }
func (f *fakeInputBackend) Click(_ pkginput.Handle, _, _ float64, _ string, _ int) error {
	f.clicks++
	return nil
}
func (f *fakeInputBackend) KeyDown(pkginput.Handle, string) error                     { return nil }
func (f *fakeInputBackend) KeyUp(pkginput.Handle, string) error                       { return nil }
func (f *fakeInputBackend) MouseDown(pkginput.Handle, float64, float64, string) error { return nil }
func (f *fakeInputBackend) MouseUp(pkginput.Handle, string) error                     { return nil }
func (f *fakeInputBackend) MouseMoveRel(pkginput.Handle, int, int, int) error         { return nil }
func (f *fakeInputBackend) Drag(pkginput.Handle, float64, float64, float64, float64, string, int) error {
	return nil
}
func (f *fakeInputBackend) MoveTo(pkginput.Handle, float64, float64) error            { return nil }
func (f *fakeInputBackend) CursorRatio(pkginput.Handle) (float64, float64, error)     { return 0, 0, nil }
func (f *fakeInputBackend) Scroll(pkginput.Handle, float64, float64, int, bool) error { return nil }
func (f *fakeInputBackend) TypeText(_ pkginput.Handle, _ string) error                { return nil }
func (f *fakeInputBackend) ReleaseAll() error                                         { return nil }
func (f *fakeInputBackend) Close() error                                              { return nil }
