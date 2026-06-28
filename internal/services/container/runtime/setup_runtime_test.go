package runtime

import (
	"testing"

	"yotta/internal/services/container"
	"yotta/internal/services/execution"

	_ "yotta/internal/nodes/image" // 注册 Capture (NeedsTarget=true)
	_ "yotta/internal/nodes/system"
)

// TestSetupRuntime_BuildsBackendsWithoutResolvingWindow 验证 setupRuntime 重写后：
// - 含 NeedsTarget 节点且没有显式非 Win32 target 时建 Win32 Input + Capture 后端
// - 不解析窗口 (HWND 仍 == 0)
func TestSetupRuntime_BuildsBackendsWithoutResolvingWindow(t *testing.T) {
	c := &container.Container{
		SchemaVersion:  1,
		ID:             "test-setup-rt",
		Name:           "test-setup-rt",
		InputBackend:   "postmessage",
		CaptureBackend: "auto",
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "start", Kind: "Start"},
				// Capture 有 NeedsTarget=true 且无显式 target → 走 Windows 默认 backend
				{ID: "ss", Kind: "Capture"},
			},
		},
	}

	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, nil, nil, nil, 0)
	// 不预设 rt.Input / Window，让 setupRuntime 真正执行
	r := NewContainerRunner(rt)

	if err := r.setupRuntime(); err != nil {
		t.Fatalf("setupRuntime: %v", err)
	}

	if rt.Input == nil {
		t.Error("rt.Input 应已建立, 实际 nil")
	}
	if rt.Capture == nil {
		t.Error("rt.Capture 应已建立, 实际 nil")
	}
	// 窗口由 Win32WindowTarget.Run 运行时解析，不在 setupRuntime 里解析
	if hwnd := rt.WindowHandle().HWND; hwnd != 0 {
		t.Errorf("setupRuntime 不应解析窗口, 但 HWND = %d", hwnd)
	}
}

func TestSetupRuntime_AndroidTargetDoesNotBuildWin32Backends(t *testing.T) {
	c := &container.Container{
		SchemaVersion:  1,
		ID:             "test-android-target-rt",
		Name:           "test-android-target-rt",
		InputBackend:   "postmessage",
		CaptureBackend: "auto",
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "at", Kind: "AndroidTarget", Config: map[string]any{
					"literal": map[string]any{"Serial": "emulator-5554", "Width": 1080, "Height": 1920},
				}},
				{ID: "ss", Kind: "Capture"},
			},
			Edges: []container.GraphEdge{
				{From: "start.Done", To: "at.In"},
				{From: "at.Done", To: "ss.In"},
			},
		},
	}

	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, nil, nil, nil, 0)
	r := NewContainerRunner(rt)

	if err := r.setupRuntime(); err != nil {
		t.Fatalf("setupRuntime: %v", err)
	}
	if rt.Input != nil {
		t.Fatalf("Android target graph should not initialise Win32 input backend, got %T", rt.Input)
	}
	if rt.Capture != nil {
		t.Fatalf("Android target graph should not initialise Win32 capture backend, got %T", rt.Capture)
	}
}
