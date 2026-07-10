package runtime

import (
	"context"
	"errors"
	"testing"

	nodepkg "github.com/yottaapp/yotta/internal/node"
	"github.com/yottaapp/yotta/internal/services/container"
)

// windowOverrideRecorder 记录 Run 期间 ctx.Window().HWND() 看到的窗口, 验证派发期覆盖生效。
var c3RecordedHWND uintptr

type windowOverrideRecorder struct{}

func (windowOverrideRecorder) Spec() nodepkg.Spec {
	return nodepkg.Spec{
		Kind: "TestWindowOverrideRecorder", Category: "System", NeedsWindow: true,
		Inputs:  append([]nodepkg.InputSpec{{Name: "In", Type: "Exec"}}, nodepkg.WindowInputSpec()),
		Outputs: []nodepkg.OutputSpec{{Name: "Done", Type: "Exec"}},
	}
}

func (windowOverrideRecorder) Run(ctx nodepkg.Ctx, _ nodepkg.Inputs) (nodepkg.Outputs, error) {
	c3RecordedHWND = ctx.Window().HWND()
	return ctx.Out("Done").Fire(), nil
}

func registerRecorderOnce() {
	if _, ok := nodepkg.Get("TestWindowOverrideRecorder"); !ok {
		nodepkg.Register(&windowOverrideRecorder{})
	}
}

func TestExecNodeViaFramework_WindowOverride_AppliesAndPops(t *testing.T) {
	registerRecorderOnce()
	rt, r := newTestRunner(t) // 粘性窗口 HWND=1

	gw := &container.GraphNode{ID: "gw", Kind: "GetWindow"}
	rec := &container.GraphNode{ID: "rec", Kind: "TestWindowOverrideRecorder"}
	r.nodesByID = map[string]*container.GraphNode{"gw": gw, "rec": rec}
	r.dataEdges = buildDataEdgeIndex(container.Graph{
		Nodes: []container.GraphNode{*gw, *rec},
		Edges: []container.GraphEdge{{From: "gw.Window", To: "rec.Window"}},
	})
	r.execOutputs["gw.Window"] = nodepkg.Window{HWND: 2, Title: "子窗口"}

	orig := isWindowFn
	defer func() { isWindowFn = orig }()
	isWindowFn = func(h uintptr) bool { return h != 0 }

	c3RecordedHWND = 0
	if _, err := r.execNodeViaFramework(context.Background(), rec, ExecToken{}); err != nil {
		t.Fatalf("execNodeViaFramework: %v", err)
	}
	if c3RecordedHWND != 2 {
		t.Fatalf("Run 期间应看到覆盖窗口 2, got %d", c3RecordedHWND)
	}
	if got := rt.WindowHandle().HWND; got != 1 {
		t.Fatalf("跑完应回粘性窗口 1 (override 已 pop), got %d", got)
	}
}

func TestExecNodeViaFramework_WindowInvalid_ReturnsCode(t *testing.T) {
	registerRecorderOnce()
	_, r := newTestRunner(t)

	gw := &container.GraphNode{ID: "gw", Kind: "GetWindow"}
	rec := &container.GraphNode{ID: "rec", Kind: "TestWindowOverrideRecorder"}
	r.nodesByID = map[string]*container.GraphNode{"gw": gw, "rec": rec}
	r.dataEdges = buildDataEdgeIndex(container.Graph{
		Nodes: []container.GraphNode{*gw, *rec},
		Edges: []container.GraphEdge{{From: "gw.Window", To: "rec.Window"}},
	})
	r.execOutputs["gw.Window"] = nodepkg.Window{HWND: 99}

	orig := isWindowFn
	defer func() { isWindowFn = orig }()
	isWindowFn = func(uintptr) bool { return false } // 句柄失效

	_, err := r.execNodeViaFramework(context.Background(), rec, ExecToken{})
	if err == nil {
		t.Fatal("失效 Window 应报错")
	}
	var coded nodepkg.Coded
	if !errors.As(err, &coded) || coded.ErrCode() != nodepkg.CodeWindowInvalid {
		t.Fatalf("应是 CodeWindowInvalid, got %v", err)
	}
}
