package runtime

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	gort "runtime"
	"testing"
	"time"

	"yhbox/internal/services/container"
	"yhbox/internal/services/execution"
)

func loadStateRESULT(t *testing.T) container.Subgraph {
	t.Helper()
	_, thisFile, _, _ := gort.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..")
	jsonPath := filepath.Join(root, "bin", "data", "containers", "fishing-v2", "subgraphs", "state_RESULT.json")
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("read state_RESULT.json: %v", err)
	}
	var sg container.Subgraph
	if err := json.Unmarshal(data, &sg); err != nil {
		t.Fatalf("unmarshal state_RESULT.json: %v", err)
	}
	return sg
}

func runStateRESULT(t *testing.T, preResultEnteredAtMsAgo float64, hits map[string]bool) (*spyInputBackend, *RuntimeContext, error) {
	t.Helper()
	stateRESULT := loadStateRESULT(t)
	pressEsc := loadFishingV2Helper(t, "press_esc_until_clear.json")
	c := &container.Container{
		SchemaVersion: 1,
		ID:            "state_result_test",
		Name:          "state_result_test",
		Vars: []container.VarDecl{
			{Name: "state", Type: "string", Default: "RESULT"},
			{Name: "resultEnteredAt", Type: "number", Default: 0.0},
			{Name: "_pressEscCleared", Type: "bool", Default: false},
		},
		Subgraphs: []container.Subgraph{stateRESULT, pressEsc},
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "call", Kind: "Subgraph", Config: map[string]any{"SubgraphID": "state_RESULT"}},
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
	rt.Capture = &mockCaptureBackend{}
	if preResultEnteredAtMsAgo > 0 {
		now := float64(time.Now().UnixMilli())
		rt.SetVar("resultEnteredAt", now-preResultEnteredAtMsAgo)
	}
	r := NewContainerRunner(rt)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	err := r.Run(ctx)
	return spy, rt, err
}

func TestStateRESULT_FirstEntry_NoMatch(t *testing.T) {
	_, rt, err := runStateRESULT(t, 0, map[string]bool{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, _ := rt.Vars()["state"].(string)
	if got != "RESULT" {
		t.Fatalf("FirstEntry_NoMatch: want state=RESULT, got %q", got)
	}
	r, _ := rt.Vars()["resultEnteredAt"].(float64)
	if r == 0 {
		t.Errorf("FirstEntry_NoMatch: want resultEnteredAt > 0 (first entry sets timestamp), got 0")
	}
}

func TestStateRESULT_ResultHit(t *testing.T) {
	_, rt, err := runStateRESULT(t, 2000, map[string]bool{"fishing.result": true})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, _ := rt.Vars()["state"].(string)
	if got != "IDLE" {
		t.Fatalf("ResultHit: want state=IDLE, got %q", got)
	}
	r, _ := rt.Vars()["resultEnteredAt"].(float64)
	if r != 0 {
		t.Errorf("ResultHit: want resultEnteredAt=0, got %v", r)
	}
}

func TestStateRESULT_FishEscape(t *testing.T) {
	_, rt, err := runStateRESULT(t, 2000, map[string]bool{"fishing.fish_escape": true})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, _ := rt.Vars()["state"].(string)
	if got != "IDLE" {
		t.Fatalf("FishEscape: want state=IDLE, got %q", got)
	}
	r, _ := rt.Vars()["resultEnteredAt"].(float64)
	if r != 0 {
		t.Errorf("FishEscape: want resultEnteredAt=0, got %v", r)
	}
}

func TestStateRESULT_Timeout(t *testing.T) {
	_, rt, err := runStateRESULT(t, 10000, map[string]bool{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, _ := rt.Vars()["state"].(string)
	if got != "RECOVERING" {
		t.Fatalf("Timeout: want state=RECOVERING, got %q", got)
	}
	r, _ := rt.Vars()["resultEnteredAt"].(float64)
	if r != 0 {
		t.Errorf("Timeout: want resultEnteredAt=0, got %v", r)
	}
}
