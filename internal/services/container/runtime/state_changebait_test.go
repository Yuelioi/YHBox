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

func loadStateCHANGEBAIT(t *testing.T) container.Subgraph {
	t.Helper()
	_, thisFile, _, _ := gort.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..")
	jsonPath := filepath.Join(root, "bin", "data", "containers", "fishing-v2", "subgraphs", "state_CHANGEBAIT.json")
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("read state_CHANGEBAIT.json: %v", err)
	}
	var sg container.Subgraph
	if err := json.Unmarshal(data, &sg); err != nil {
		t.Fatalf("unmarshal state_CHANGEBAIT.json: %v", err)
	}
	return sg
}

func runStateCHANGEBAIT(t *testing.T, hits map[string]bool) (*spyInputBackend, *RuntimeContext, error) {
	t.Helper()
	sg := loadStateCHANGEBAIT(t)
	c := &container.Container{
		SchemaVersion: 1,
		ID:            "state_changebait_test",
		Name:          "state_changebait_test",
		Vars: []container.VarDecl{
			{Name: "state", Type: "string", Default: "CHANGEBAIT"},
		},
		Subgraphs: []container.Subgraph{sg},
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "call", Kind: "Subgraph", Config: map[string]any{"subgraphId": "state_CHANGEBAIT"}},
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
	rt.Capture = &mockCaptureBackend{}
	rt.SetVar("state", "CHANGEBAIT")
	r := NewContainerRunner(rt)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err := r.Run(ctx)
	return spy, rt, err
}

func TestStateCHANGEBAIT_HappyPath(t *testing.T) {
	// hits: change_bait_confirm hit (mockMatcher static → loop 3 iter all click, then loop.complete)
	// → go_fishing hit → checkGoFishing.yes → Break wait loop first iter → state=IDLE.
	spy, rt, err := runStateCHANGEBAIT(t, map[string]bool{
		"fishing.change_bait_confirm": true,
		"fishing.go_fishing":          true,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, _ := rt.Vars()["state"].(string)
	if got != "IDLE" {
		t.Fatalf("HappyPath: want IDLE, got %q", got)
	}
	if spy.clicks < 1 {
		t.Errorf("HappyPath: want spy.clicks >= 1, got %d", spy.clicks)
	}
}

func TestStateCHANGEBAIT_ConfirmDisappears(t *testing.T) {
	// change_bait_confirm not hit → ClickTemplate timeout 2s → breakConfirm → loop complete.
	// go_fishing hit → wait loop first iter break → state=IDLE.
	_, rt, err := runStateCHANGEBAIT(t, map[string]bool{"fishing.go_fishing": true})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, _ := rt.Vars()["state"].(string)
	if got != "IDLE" {
		t.Fatalf("ConfirmDisappears: want IDLE, got %q", got)
	}
}

func TestStateCHANGEBAIT_GoFishingTimeout(t *testing.T) {
	// change_bait_confirm hit (3 iter all click), go_fishing miss → wait loop 20 iter exhaust → state=IDLE.
	// changeBaitFlow has no missFailPause — even after wait timeout, OnDone is still IDLE.
	_, rt, err := runStateCHANGEBAIT(t, map[string]bool{"fishing.change_bait_confirm": true})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, _ := rt.Vars()["state"].(string)
	if got != "IDLE" {
		t.Fatalf("GoFishingTimeout: want IDLE (loop count exhausted, still OnDone IDLE), got %q", got)
	}
}
