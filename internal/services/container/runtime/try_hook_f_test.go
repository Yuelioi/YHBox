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

	"yhbox/internal/services/container"
	"yhbox/internal/services/execution"
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
					"subgraphId": "try_hook_F",
					"literal":    map[string]any{"pollIntervalMs": pollIntervalMs},
				}},
				{ID: "stop", Kind: "Stop"},
			},
			Edges: []container.GraphEdge{
				{From: "start.out", To: "call.in"},
				{From: "call.done", To: "stop.in"},
				{From: "call.failed", To: "stop.in"},
			},
		},
	}
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, NoopColorDetector{}, nil, nil, nil, nil, 0)
	stubRuntimeWindowAndInput(rt)
	spy := &spyInputBackend{}
	rt.Input = spy
	mock := &mockCaptureBackend{FrameROIResult: frame}
	rt.Capture = mock
	r := NewContainerRunner(rt)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := r.Run(ctx)
	return spy, c, rt, err
}

func TestTryHookF_FoundFast(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 200, 20))
	paintCursorBar(img, 50)
	paintTargetBar(img, 100, 115)
	spy, _, rt, err := runTryHookF(t, 1.0, img)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	want := []string{"press:f"}
	if !equalStrings(spy.keyEvents, want) {
		t.Fatalf("FoundFast: want keyEvents %v, got %v", want, spy.keyEvents)
	}
	sys := rt.Sys()
	if sys.LastBarTrack.CursorX <= 0 {
		t.Errorf("FoundFast: expected sys.LastBarTrack.CursorX > 0, got %d", sys.LastBarTrack.CursorX)
	}
}

func TestTryHookF_Exhausted(t *testing.T) {
	spy, _, rt, err := runTryHookF(t, 1.0, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(spy.keyEvents) != 30 {
		t.Fatalf("Exhausted: want 30 KeyPress events, got %d: %v", len(spy.keyEvents), spy.keyEvents)
	}
	for i, ev := range spy.keyEvents {
		if ev != "press:f" {
			t.Fatalf("Exhausted: event %d want press:f, got %q", i, ev)
		}
	}
	found, _ := rt.Vars()["_hookFFound"].(bool)
	if found {
		t.Errorf("Exhausted: _hookFFound expected false, got true")
	}
}
