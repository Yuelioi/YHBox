package runtime

import (
	"context"
	"encoding/json"
	"image"
	"os"
	"path/filepath"
	gort "runtime"
	"testing"
	"time"

	"yotta/internal/services/container"
	"yotta/internal/services/execution"
)

func loadTryHookF(t *testing.T) container.Subgraph {
	t.Helper()
	_, thisFile, _, _ := gort.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..")
	jsonPath := filepath.Join(root, "bin", "data", "containers", "fishing-v2", "subgraphs", "try_hook_F.json")
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("read try_hook_F.json: %v", err)
	}
	var sg container.Subgraph
	if err := json.Unmarshal(data, &sg); err != nil {
		t.Fatalf("unmarshal try_hook_F.json: %v", err)
	}
	return sg
}

func runTryHookF(t *testing.T, pollIntervalMs float64, frame *image.RGBA) (*spyInputBackend, *container.Container, *RuntimeContext, error) {
	t.Helper()
	sg := loadTryHookF(t)
	c := &container.Container{
		SchemaVersion: 1,
		ID:            "try_hook_f_test",
		Name:          "try_hook_f_test",
		Vars: []container.VarDecl{
			{Name: "_hookFFound", Type: "bool", Default: false},
		},
		Subgraphs: []container.Subgraph{sg},
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "call", Kind: "Subgraph", Config: map[string]any{
					"SubgraphID": "try_hook_F",
					"literal":    map[string]any{"pollIntervalMs": pollIntervalMs},
				}},
				{ID: "stop", Kind: "Stop"},
			},
			Edges: []container.GraphEdge{
				{From: "start.Done", To: "call.in"},
				{From: "call.Done", To: "stop.in"},
				{From: "call.failed", To: "stop.in"},
			},
		},
	}
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, nil, nil, nil, 0)
	stubRuntimeWindowAndInput(rt)
	spy := &spyInputBackend{}
	rt.Input = spy
	mock := &mockCaptureBackend{FrameROIResult: frame}
	rt.Capture = mock
	r := NewContainerRunner(rt)
	// 15s 而非 5s: Exhausted 路径真跑 30 casts × 每次 bar-track 超时 ≈ 5s, 贴着 5s deadline
	// 在并行满载跑 `go test ./...` 时会偶发 deadline exceeded (假阳)。断言是「60 事件 + 没找到」,
	// 不是「正好 5s 内跑完」, 故放宽 deadline 当安全网, 不削弱断言。
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	err := r.Run(ctx)
	return spy, c, rt, err
}

func TestTryHookF_FoundFast(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 200, 20))
	paintCursorBar(img, 50)
	paintTargetBar(img, 100, 115)
	spy, _, _, err := runTryHookF(t, 1.0, img)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	// 按出 F (down/up) 即证明 DualBar 找到了内/外条 → 走 hook 命中分支.
	want := []string{"down:f", "up:f"} // KeyPress 节点 #4 后拆 down/up
	if !equalStrings(spy.keyEvents, want) {
		t.Fatalf("FoundFast: want keyEvents %v, got %v", want, spy.keyEvents)
	}
}

func TestTryHookF_Exhausted(t *testing.T) {
	spy, _, rt, err := runTryHookF(t, 1.0, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	// 30 次 cast，每次 KeyPress 节点拆成 down:f + up:f (#4 可取消长按) → 60 事件，交替成对
	if len(spy.keyEvents) != 60 {
		t.Fatalf("Exhausted: want 60 events (30 casts × down/up), got %d: %v", len(spy.keyEvents), spy.keyEvents)
	}
	for i, ev := range spy.keyEvents {
		want := "down:f"
		if i%2 == 1 {
			want = "up:f"
		}
		if ev != want {
			t.Fatalf("Exhausted: event %d want %q, got %q", i, want, ev)
		}
	}
	found, _ := rt.Vars()["_hookFFound"].(bool)
	if found {
		t.Errorf("Exhausted: _hookFFound expected false, got true")
	}
}
