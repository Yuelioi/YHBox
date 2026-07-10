package runtime

import (
	"context"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/services/container"
	"github.com/yottaapp/yotta/internal/services/execution"
	"github.com/yottaapp/yotta/pkg/winutil"
)

// TestWin32WindowTarget_RunIsCalledInDispatch: 验证 Start→Win32WindowTarget→Stop 执行时,
// Win32WindowTarget.Run 真的被调——即 resolveWindowFn 被触发, rt.SetActiveWindow 改为 stub 值.
//
// 关键: setupRuntime 因测试 controller provider 已注入而跳过 (stubRuntimeWindowAndInput).
// Win32WindowTarget 在 execNode 路径中正常执行, 通过 windowAdapter.SetActive → resolveWindowFn
// 把 rt 的 WindowHandle 更新为 stub 返回的 HWND=999.
func TestWin32WindowTarget_RunIsCalledInDispatch(t *testing.T) {
	// stub resolveWindowFn: 返 HWND=999
	orig := resolveWindowFn
	defer func() { resolveWindowFn = orig }()
	resolveWindowFn = func(_ context.Context, _ winutil.MatchSpec, _, _ time.Duration) (winutil.WindowHandle, error) {
		return winutil.WindowHandle{HWND: 999, ClientW: 1920, ClientH: 1080}, nil
	}

	c := &container.Container{
		SchemaVersion: 1,
		ID:            "test-wt-dispatch",
		Name:          "test-wt-dispatch",
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "wt", Kind: "Win32WindowTarget", Config: map[string]any{
					"Title":      "GameWindow",
					"TitleMatch": "exact",
				}},
				{ID: "stop", Kind: "Stop"},
			},
			Edges: []container.GraphEdge{
				{From: "start.Done", To: "wt.In"},
				{From: "wt.Done", To: "stop.In"},
			},
		},
	}
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, nil, nil, nil, 0)
	stubRuntimeWindowAndInput(rt) // HWND=1, Input!=nil → setupRuntime 跳过
	r := NewContainerRunner(rt)

	if err := r.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	// Win32WindowTarget.Run 被执行后 rt 的活动窗口 HWND 应被更新为 stub 的 999
	if got := rt.WindowHandle().HWND; got != 999 {
		t.Errorf("Win32WindowTarget.Run 未被执行: HWND = %d, want 999", got)
	}
}
