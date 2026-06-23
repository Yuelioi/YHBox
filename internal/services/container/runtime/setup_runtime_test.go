package runtime

import (
	"testing"

	"yotta/internal/services/container"
	"yotta/internal/services/execution"

	_ "yotta/internal/nodes/image" // 注册 Capture (NeedsWindow=true)
)

// TestSetupRuntime_BuildsBackendsWithoutResolvingWindow 验证 setupRuntime 重写后：
// - 含 NeedsWindow 节点时建 Input + Capture 后端
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
				// Capture 有 NeedsWindow=true → containerNeedsWindow 返 true
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
	// 窗口由 WindowTarget.Run 运行时解析，不在 setupRuntime 里解析
	if hwnd := rt.WindowHandle().HWND; hwnd != 0 {
		t.Errorf("setupRuntime 不应解析窗口, 但 HWND = %d", hwnd)
	}
}
