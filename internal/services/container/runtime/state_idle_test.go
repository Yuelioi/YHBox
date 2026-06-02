package runtime

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	gort "runtime"
	"testing"
	"time"

	"yotta/internal/services/container"
	"yotta/internal/services/execution"
)

func loadStateIDLE(t *testing.T) container.Subgraph {
	t.Helper()
	_, thisFile, _, _ := gort.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..")
	jsonPath := filepath.Join(root, "bin", "data", "containers", "fishing-v2", "subgraphs", "state_IDLE.json")
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("read state_IDLE.json: %v", err)
	}
	var sg container.Subgraph
	if err := json.Unmarshal(data, &sg); err != nil {
		t.Fatalf("unmarshal state_IDLE.json: %v", err)
	}
	return sg
}

func loadPressEsc(t *testing.T) container.Subgraph {
	t.Helper()
	_, thisFile, _, _ := gort.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..")
	jsonPath := filepath.Join(root, "bin", "data", "containers", "fishing-v2", "subgraphs", "press_esc_until_clear.json")
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("read press_esc_until_clear.json: %v", err)
	}
	var sg container.Subgraph
	if err := json.Unmarshal(data, &sg); err != nil {
		t.Fatalf("unmarshal press_esc_until_clear.json: %v", err)
	}
	return sg
}

func runStateIDLE(t *testing.T, hits map[string]bool, baitProbeMs, castRemainingMs float64) (*spyInputBackend, *RuntimeContext, error) {
	t.Helper()
	stateIDLE := loadStateIDLE(t)
	pressEsc := loadPressEsc(t)
	c := &container.Container{
		SchemaVersion: 1,
		ID:            "state_idle_test",
		Name:          "state_idle_test",
		Vars: []container.VarDecl{
			{Name: "state", Type: "string", Default: "IDLE"},
			{Name: "waitingStart", Type: "number", Default: 0.0},
			{Name: "hookStreak", Type: "number", Default: 0.0},
			{Name: "_pressEscCleared", Type: "bool", Default: false},
		},
		Subgraphs: []container.Subgraph{stateIDLE, pressEsc},
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "call", Kind: "Subgraph", Config: map[string]any{
					"SubgraphID": "state_IDLE",
					"literal": map[string]any{
						"baitProbeDelayMs":     baitProbeMs,
						"castRemainingDelayMs": castRemainingMs,
					},
				}},
				{ID: "stop", Kind: "Stop"},
			},
			Edges: []container.GraphEdge{
				{From: "start.Done", To: "call.in"},
				{From: "call.Done", To: "stop.in"},
			},
		},
	}
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, nil, nil, nil, 0)
	stubRuntimeWindowAndInput(rt)
	spy := &spyInputBackend{}
	rt.Input = spy
	rt.Matcher = &mockMatcher{HitTemplates: hits}
	r := NewContainerRunner(rt)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := r.Run(ctx)
	return spy, rt, err
}

func TestStateIDLE_DetectStartFish(t *testing.T) {
	spy, rt, err := runStateIDLE(t, map[string]bool{"fishing.start_fish": true}, 1, 1)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, _ := rt.Vars()["state"].(string)
	if got != "SETUP" {
		t.Fatalf("DetectStartFish: want state=SETUP, got %q", got)
	}
	if len(spy.keyEvents) != 0 {
		t.Errorf("DetectStartFish: want no KeyPress, got %v", spy.keyEvents)
	}
}

func TestStateIDLE_CastToWaiting(t *testing.T) {
	spy, rt, err := runStateIDLE(t, map[string]bool{"fishing.hook_icon": true}, 1, 1)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, _ := rt.Vars()["state"].(string)
	if got != "WAITING" {
		t.Fatalf("CastToWaiting: want state=WAITING, got %q", got)
	}
	want := []string{"down:f", "up:f"} // KeyPress 节点 #4 后拆 down/up (可取消长按)
	if !equalStrings(spy.keyEvents, want) {
		t.Fatalf("CastToWaiting: want spy.keyEvents %v, got %v", want, spy.keyEvents)
	}
}

func TestStateIDLE_CastBlockedNeedBait(t *testing.T) {
	spy, rt, err := runStateIDLE(t, map[string]bool{"fishing.hook_icon": true, "fishing.need_bait": true}, 1, 1)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, _ := rt.Vars()["state"].(string)
	if got != "BUYBAIT" {
		t.Fatalf("CastBlockedNeedBait: want state=BUYBAIT, got %q", got)
	}
	want := []string{"down:f", "up:f"} // KeyPress 节点 #4 后拆 down/up (可取消长按)
	if !equalStrings(spy.keyEvents, want) {
		t.Fatalf("CastBlockedNeedBait: want spy.keyEvents %v, got %v", want, spy.keyEvents)
	}
}

func TestStateIDLE_NoMatchStaysIDLE(t *testing.T) {
	spy, rt, err := runStateIDLE(t, map[string]bool{}, 1, 1)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, _ := rt.Vars()["state"].(string)
	if got != "IDLE" {
		t.Fatalf("NoMatchStaysIDLE: want state=IDLE, got %q", got)
	}
	if len(spy.keyEvents) != 0 {
		t.Errorf("NoMatchStaysIDLE: want no KeyPress, got %v", spy.keyEvents)
	}
}
