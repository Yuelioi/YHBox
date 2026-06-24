package runtime

import (
	"image"
	"testing"

	"github.com/lxn/win"

	"yotta/internal/services/container"
	"yotta/internal/services/execution"
	pkgcapture "yotta/pkg/capture"
	pkginput "yotta/pkg/input"
	"yotta/pkg/winutil"
)

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
	// stub Window + Input 让 setupRuntime 幂等跳过
	rt.SetActiveWindow(winutil.WindowHandle{HWND: 1})
	rt.Input = &fakeInputBackend{}
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
	rt.Input = &fakeInputBackend{}
	mock := &mockCaptureBackend{}
	rt.Capture = mock
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
func (m *mockCaptureBackend) Frame(_ win.HWND) (*image.RGBA, error) {
	return m.FrameROIResult, nil
}
func (m *mockCaptureBackend) FrameROI(_ win.HWND, _, _, _, _ int) (*image.RGBA, error) {
	if m.FrameROIResult == nil {
		return nil, image.ErrFormat
	}
	return m.FrameROIResult, nil
}
func (m *mockCaptureBackend) ClientSize(_ win.HWND) (int, int, error) { return 1920, 1080, nil }
func (m *mockCaptureBackend) Close() error                            { return nil }

// fakeInputBackend 测试用 pkginput.Backend 实现. 计数 Click 调用次数, 其他方法返 nil.
// 共享给所有 _test.go (runner_test / playclip_test / safe_backend_test 等).
type fakeInputBackend struct {
	clicks int
}

func (f *fakeInputBackend) Name() string                        { return "fake" }
func (f *fakeInputBackend) Capabilities() pkginput.Capabilities { return pkginput.Capabilities{} }
func (f *fakeInputBackend) Click(_ win.HWND, _, _ float64, _ string, _ int) error {
	f.clicks++
	return nil
}
func (f *fakeInputBackend) KeyDown(win.HWND, string) error                     { return nil }
func (f *fakeInputBackend) KeyUp(win.HWND, string) error                       { return nil }
func (f *fakeInputBackend) MouseDown(win.HWND, float64, float64, string) error { return nil }
func (f *fakeInputBackend) MouseUp(win.HWND, string) error                     { return nil }
func (f *fakeInputBackend) MouseMoveRel(win.HWND, int, int, int) error         { return nil }
func (f *fakeInputBackend) Drag(win.HWND, float64, float64, float64, float64, string, int) error {
	return nil
}
func (f *fakeInputBackend) MoveTo(win.HWND, float64, float64) error        { return nil }
func (f *fakeInputBackend) CursorRatio(win.HWND) (float64, float64, error) { return 0, 0, nil }
func (f *fakeInputBackend) Scroll(win.HWND, float64, float64, int) error   { return nil }
func (f *fakeInputBackend) ReleaseAll() error                              { return nil }
func (f *fakeInputBackend) Close() error                                   { return nil }
