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

func loadStateSHOPSELL(t *testing.T) container.Subgraph {
	t.Helper()
	_, thisFile, _, _ := gort.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..")
	jsonPath := filepath.Join(root, "internal", "services", "container", "runtime", "testdata", "fishing-v2", "subgraphs", "state_SHOPSELL.json")
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("read state_SHOPSELL.json: %v", err)
	}
	var sg container.Subgraph
	if err := json.Unmarshal(data, &sg); err != nil {
		t.Fatalf("unmarshal state_SHOPSELL.json: %v", err)
	}
	speedUpStateFixtureTimingForTest(&sg)
	return sg
}

func runStateSHOPSELL(t *testing.T, hits map[string]bool) (*spyInputBackend, *RuntimeContext, error) {
	t.Helper()
	sg := loadStateSHOPSELL(t)
	c := &container.Container{
		SchemaVersion: 1,
		ID:            "state_shopsell_test",
		Name:          "state_shopsell_test",
		Vars: []container.VarDecl{
			{Name: "state", Type: "string", Default: "SHOPSELL"},
		},
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "call", Kind: "Subgraph", Config: map[string]any{"SubgraphID": "state_SHOPSELL"}},
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
	rt.Capture = &mockCaptureBackend{}
	rt.SetVar("state", "SHOPSELL")
	r := NewContainerRunner(rt)
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	err := r.Run(ctx)
	return spy, rt, err
}

func TestStateSHOPSELL_HappyPath(t *testing.T) {
	spy, rt, err := runStateSHOPSELL(t, map[string]bool{
		"fishing.shop_bag_tab":      true,
		"fishing.shop_sell_all":     true,
		"fishing.shop_confirm_sell": true,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, _ := rt.Vars()["state"].(string)
	if got != "IDLE" {
		t.Fatalf("HappyPath: want state=IDLE, got %q", got)
	}
	if spy.clicks != 3 {
		t.Errorf("HappyPath: want spy.clicks=3 (3 ClickTemplates), got %d", spy.clicks)
	}
}

func TestStateSHOPSELL_BagTabTimeout(t *testing.T) {
	_, rt, err := runStateSHOPSELL(t, map[string]bool{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, _ := rt.Vars()["state"].(string)
	if got != "RECOVERING" {
		t.Fatalf("BagTabTimeout: want state=RECOVERING, got %q", got)
	}
}

func TestStateSHOPSELL_ConfirmTimeout(t *testing.T) {
	_, rt, err := runStateSHOPSELL(t, map[string]bool{
		"fishing.shop_bag_tab":  true,
		"fishing.shop_sell_all": true,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, _ := rt.Vars()["state"].(string)
	if got != "RECOVERING" {
		t.Fatalf("ConfirmTimeout: want state=RECOVERING (confirm step timeout), got %q", got)
	}
}
