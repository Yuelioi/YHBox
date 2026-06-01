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

func loadStateBUYBAIT(t *testing.T) container.Subgraph {
	t.Helper()
	_, thisFile, _, _ := gort.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..")
	jsonPath := filepath.Join(root, "bin", "data", "containers", "fishing-v2", "subgraphs", "state_BUYBAIT.json")
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("read state_BUYBAIT.json: %v", err)
	}
	var sg container.Subgraph
	if err := json.Unmarshal(data, &sg); err != nil {
		t.Fatalf("unmarshal state_BUYBAIT.json: %v", err)
	}
	return sg
}

func runStateBUYBAIT(t *testing.T, hits map[string]bool) (*spyInputBackend, *RuntimeContext, error) {
	t.Helper()
	sg := loadStateBUYBAIT(t)
	c := &container.Container{
		SchemaVersion: 1,
		ID:            "state_buybait_test",
		Name:          "state_buybait_test",
		Vars: []container.VarDecl{
			{Name: "state", Type: "string", Default: "BUYBAIT"},
		},
		Subgraphs: []container.Subgraph{sg},
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "call", Kind: "Subgraph", Config: map[string]any{"SubgraphID": "state_BUYBAIT"}},
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
	rt.Capture = &mockCaptureBackend{}
	rt.SetVar("state", "BUYBAIT")
	r := NewContainerRunner(rt)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err := r.Run(ctx)
	return spy, rt, err
}

func TestStateBUYBAIT_HappyPath(t *testing.T) {
	spy, rt, err := runStateBUYBAIT(t, map[string]bool{
		"fishing.bait_product": true,
		"fishing.buy_max":      true,
		"fishing.buy_button":   true,
		"fishing.buy_confirm":  true,
		"fishing.buy_success":  true,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, _ := rt.Vars()["state"].(string)
	if got != "CHANGEBAIT" {
		t.Fatalf("HappyPath: want CHANGEBAIT, got %q", got)
	}
	if spy.clicks < 4 {
		t.Errorf("HappyPath: want spy.clicks >= 4, got %d", spy.clicks)
	}
}

func TestStateBUYBAIT_BaitProductTimeout(t *testing.T) {
	_, rt, err := runStateBUYBAIT(t, map[string]bool{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, _ := rt.Vars()["state"].(string)
	if got != "RECOVERING" {
		t.Fatalf("BaitProductTimeout: want RECOVERING, got %q", got)
	}
}

func TestStateBUYBAIT_ConfirmSkip(t *testing.T) {
	_, rt, err := runStateBUYBAIT(t, map[string]bool{
		"fishing.bait_product": true,
		"fishing.buy_max":      true,
		"fishing.buy_button":   true,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, _ := rt.Vars()["state"].(string)
	if got != "CHANGEBAIT" {
		t.Fatalf("ConfirmSkip: want CHANGEBAIT (confirm missSkip path), got %q", got)
	}
}
