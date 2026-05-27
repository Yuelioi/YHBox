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

func loadStateWAITING(t *testing.T) container.Subgraph {
	t.Helper()
	_, thisFile, _, _ := gort.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..")
	jsonPath := filepath.Join(root, "bin", "data", "containers", "fishing-v2", "subgraphs", "state_WAITING.json")
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("read state_WAITING.json: %v", err)
	}
	var sg container.Subgraph
	if err := json.Unmarshal(data, &sg); err != nil {
		t.Fatalf("unmarshal state_WAITING.json: %v", err)
	}
	return sg
}

func loadFishingV2Helper(t *testing.T, name string) container.Subgraph {
	t.Helper()
	_, thisFile, _, _ := gort.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..")
	jsonPath := filepath.Join(root, "bin", "data", "containers", "fishing-v2", "subgraphs", name)
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	var sg container.Subgraph
	if err := json.Unmarshal(data, &sg); err != nil {
		t.Fatalf("unmarshal %s: %v", name, err)
	}
	return sg
}

func runStateWAITING(t *testing.T, preWaitingStartMsAgo, preHookStreak float64, hits map[string]bool, frame *image.RGBA, pollIntervalMs float64) (*spyInputBackend, *RuntimeContext, error) {
	t.Helper()
	stateWAITING := loadStateWAITING(t)
	tryHookF := loadFishingV2Helper(t, "try_hook_F.json")
	inspectPhase := loadFishingV2Helper(t, "inspect_phase.json")
	c := &container.Container{
		SchemaVersion: 1,
		ID:            "state_waiting_test",
		Name:          "state_waiting_test",
		Vars: []container.VarDecl{
			{Name: "state", Type: "string", Default: "WAITING"},
			{Name: "waitingStart", Type: "number", Default: 0.0},
			{Name: "hookStreak", Type: "number", Default: 0.0},
			{Name: "fishingStart", Type: "number", Default: 0.0},
			{Name: "emptyCastStreak", Type: "number", Default: 0.0},
			{Name: "_hookFFound", Type: "bool", Default: false},
			{Name: "_inspectPhaseResult", Type: "string", Default: "unknown"},
		},
		Subgraphs: []container.Subgraph{stateWAITING, tryHookF, inspectPhase},
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "call", Kind: "Subgraph", Config: map[string]any{
					"SubgraphID": "state_WAITING",
					"literal":    map[string]any{"pollIntervalMs": pollIntervalMs},
				}},
				{ID: "stop", Kind: "Stop"},
			},
			Edges: []container.GraphEdge{
				{From: "start.out", To: "call.in"},
				{From: "call.Done", To: "stop.in"},
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
	rt.SetVar("waitingStart", now-preWaitingStartMsAgo)
	rt.SetVar("hookStreak", preHookStreak)
	r := NewContainerRunner(rt)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := r.Run(ctx)
	return spy, rt, err
}

func TestStateWAITING_TooEarly(t *testing.T) {
	_, rt, err := runStateWAITING(t, 100, 0, map[string]bool{"fishing.hook_text": true}, nil, 1)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, _ := rt.Vars()["state"].(string)
	if got != "WAITING" {
		t.Fatalf("TooEarly: want state=WAITING, got %q", got)
	}
}

func TestStateWAITING_HookStreakIncrement(t *testing.T) {
	_, rt, err := runStateWAITING(t, 1000, 0, map[string]bool{"fishing.hook_text": true}, nil, 1)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, _ := rt.Vars()["state"].(string)
	if got != "WAITING" {
		t.Fatalf("HookStreakIncrement: want state=WAITING (streak=1), got %q", got)
	}
	streak, _ := rt.Vars()["hookStreak"].(float64)
	if streak != 1.0 {
		t.Errorf("HookStreakIncrement: want hookStreak=1, got %v", streak)
	}
}

func TestStateWAITING_HookStreakReady_TryHookFSucceeds(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 200, 20))
	paintCursorBar(img, 50)
	paintTargetBar(img, 100, 115)
	spy, rt, err := runStateWAITING(t, 1000, 1, map[string]bool{"fishing.hook_text": true}, img, 1)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, _ := rt.Vars()["state"].(string)
	if got != "FISHING" {
		t.Fatalf("StreakReady_Success: want state=FISHING, got %q", got)
	}
	fishingStart, _ := rt.Vars()["fishingStart"].(float64)
	if fishingStart == 0 {
		t.Errorf("StreakReady_Success: want fishingStart > 0, got %v", fishingStart)
	}
	if len(spy.keyEvents) == 0 {
		t.Errorf("StreakReady_Success: want try_hook_F KeyPress events, got none")
	}
}

func TestStateWAITING_TextMissResetsStreak(t *testing.T) {
	_, rt, err := runStateWAITING(t, 1000, 2, map[string]bool{}, nil, 1)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, _ := rt.Vars()["state"].(string)
	if got != "WAITING" {
		t.Fatalf("TextMissResetsStreak: want state=WAITING, got %q", got)
	}
	streak, _ := rt.Vars()["hookStreak"].(float64)
	if streak != 0.0 {
		t.Errorf("TextMissResetsStreak: want hookStreak=0, got %v", streak)
	}
}

func TestStateWAITING_TimeoutRoutesToSetup(t *testing.T) {
	_, rt, err := runStateWAITING(t, 80000, 0, map[string]bool{"fishing.start_fish": true}, nil, 1)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, _ := rt.Vars()["state"].(string)
	if got != "SETUP" {
		t.Fatalf("Timeout: want state=SETUP (via inspect_phase routing), got %q (_inspectPhaseResult=%v)",
			got, rt.Vars()["_inspectPhaseResult"])
	}
}
