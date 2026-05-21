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

func loadStateFISHING(t *testing.T) container.Subgraph {
	t.Helper()
	_, thisFile, _, _ := gort.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..")
	jsonPath := filepath.Join(root, "bin", "data", "library", "subgraphs", "fishing-v2", "state_FISHING.json")
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("read state_FISHING.json: %v", err)
	}
	var sg container.Subgraph
	if err := json.Unmarshal(data, &sg); err != nil {
		t.Fatalf("unmarshal state_FISHING.json: %v", err)
	}
	return sg
}

func runStateFISHING(t *testing.T, preFishingStartMsAgo float64, hits map[string]bool, frame *image.RGBA) (*spyInputBackend, *RuntimeContext, error) {
	t.Helper()
	stateFISHING := loadStateFISHING(t)
	pressEsc := loadFishingV2Helper(t, "press_esc_until_clear.json")
	c := &container.Container{
		SchemaVersion: 1,
		ID:            "state_fishing_test",
		Name:          "state_fishing_test",
		Vars: []container.VarDecl{
			{Name: "state", Type: "string", Default: "FISHING"},
			{Name: "fishingStart", Type: "number", Default: 0.0},
			{Name: "controlDir", Type: "number", Default: 0.0},
			{Name: "_pressEscCleared", Type: "bool", Default: false},
		},
		Subgraphs: []container.Subgraph{stateFISHING, pressEsc},
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "call", Kind: "Subgraph", Config: map[string]any{"subgraphId": "state_FISHING"}},
				{ID: "stop", Kind: "Stop"},
			},
			Edges: []container.GraphEdge{
				{From: "start.out", To: "call.in"},
				{From: "call.done", To: "stop.in"},
			},
		},
	}
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, NoopColorDetector{}, nil, nil, nil, nil, 0)
	stubRuntimeWindowAndInput(rt)
	spy := &spyInputBackend{}
	rt.Input = spy
	rt.Matcher = &mockMatcher{HitTemplates: hits}
	rt.Capture = &mockCaptureBackend{FrameROIResult: frame}
	now := float64(time.Now().UnixMilli())
	rt.SetVar("fishingStart", now-preFishingStartMsAgo)
	r := NewContainerRunner(rt)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	err := r.Run(ctx)
	return spy, rt, err
}

func TestStateFISHING_Timeout(t *testing.T) {
	_, rt, err := runStateFISHING(t, 50000, map[string]bool{}, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, _ := rt.Vars()["state"].(string)
	if got != "RECOVERING" {
		t.Fatalf("Timeout: want state=RECOVERING, got %q", got)
	}
}

func TestStateFISHING_BarVisible_DirRight(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 200, 20))
	paintCursorBar(img, 50)
	paintTargetBar(img, 100, 115)
	spy, _, err := runStateFISHING(t, 10000, map[string]bool{}, img)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	foundDownD := false
	for _, e := range spy.keyEvents {
		if e == "down:d" {
			foundDownD = true
			break
		}
	}
	if !foundDownD {
		t.Fatalf("BarVisible_DirRight: expected down:d in keyEvents, got %v", spy.keyEvents)
	}
}

func TestStateFISHING_BarMissing_FishEscape(t *testing.T) {
	_, rt, err := runStateFISHING(t, 5000, map[string]bool{"fishing.fish_escape": true}, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, _ := rt.Vars()["state"].(string)
	if got != "IDLE" {
		t.Fatalf("BarMissing_FishEscape: want state=IDLE, got %q", got)
	}
}

func TestStateFISHING_BarMissing_Result(t *testing.T) {
	_, rt, err := runStateFISHING(t, 5000, map[string]bool{"fishing.result": true}, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, _ := rt.Vars()["state"].(string)
	if got != "IDLE" {
		t.Fatalf("BarMissing_Result: want state=IDLE (via press_esc), got %q", got)
	}
}

func TestStateFISHING_BarMissing_TooEarly(t *testing.T) {
	_, rt, err := runStateFISHING(t, 1000, map[string]bool{"fishing.result": true, "fishing.fish_escape": true}, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, _ := rt.Vars()["state"].(string)
	if got != "FISHING" {
		t.Fatalf("BarMissing_TooEarly: want state=FISHING (animation grace), got %q", got)
	}
}
