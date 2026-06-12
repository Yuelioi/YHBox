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

func loadStateSETUP(t *testing.T) container.Subgraph {
	t.Helper()
	_, thisFile, _, _ := gort.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..")
	jsonPath := filepath.Join(root, "internal", "services", "container", "runtime", "testdata", "fishing-v2", "subgraphs", "state_SETUP.json")
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("read state_SETUP.json: %v", err)
	}
	var sg container.Subgraph
	if err := json.Unmarshal(data, &sg); err != nil {
		t.Fatalf("unmarshal state_SETUP.json: %v", err)
	}
	return sg
}

func runStateSETUP(t *testing.T, hits map[string]bool, postClickDelayMs float64) (*spyInputBackend, *RuntimeContext, error) {
	t.Helper()
	sg := loadStateSETUP(t)
	c := &container.Container{
		SchemaVersion: 1,
		ID:            "state_setup_test",
		Name:          "state_setup_test",
		Vars: []container.VarDecl{
			{Name: "state", Type: "string", Default: "SETUP"},
		},
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "call", Kind: "Subgraph", Config: map[string]any{
					"SubgraphID": "state_SETUP",
					"literal":    map[string]any{"postClickDelayMs": postClickDelayMs},
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
	rt.Subgraphs = []container.Subgraph{sg}
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

func TestStateSETUP_ClickAndIdle(t *testing.T) {
	spy, rt, err := runStateSETUP(t, map[string]bool{"fishing.start_fish": true}, 1)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, _ := rt.Vars()["state"].(string)
	if got != "IDLE" {
		t.Fatalf("ClickAndIdle: want state=IDLE, got %q", got)
	}
	if spy.clicks != 1 {
		t.Errorf("ClickAndIdle: want 1 click, got %d", spy.clicks)
	}
}

func TestStateSETUP_ClickThenBuyBait(t *testing.T) {
	spy, rt, err := runStateSETUP(t, map[string]bool{"fishing.start_fish": true, "fishing.need_bait": true}, 1)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, _ := rt.Vars()["state"].(string)
	if got != "BUYBAIT" {
		t.Fatalf("ClickThenBuyBait: want state=BUYBAIT, got %q", got)
	}
	if spy.clicks != 1 {
		t.Errorf("ClickThenBuyBait: want 1 click, got %d", spy.clicks)
	}
}

func TestStateSETUP_NotFoundIdle(t *testing.T) {
	spy, rt, err := runStateSETUP(t, map[string]bool{}, 1)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, _ := rt.Vars()["state"].(string)
	if got != "IDLE" {
		t.Fatalf("NotFoundIdle: want state=IDLE (ClickTemplate timeout), got %q", got)
	}
	if spy.clicks != 0 {
		t.Errorf("NotFoundIdle: want 0 click, got %d", spy.clicks)
	}
}
