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

func loadWatchdogCheck(t *testing.T) container.Subgraph {
	t.Helper()
	_, thisFile, _, _ := gort.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..")
	jsonPath := filepath.Join(root, "bin", "data", "containers", "fishing-v2", "subgraphs", "watchdog_check.json")
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("read watchdog_check.json: %v", err)
	}
	var sg container.Subgraph
	if err := json.Unmarshal(data, &sg); err != nil {
		t.Fatalf("unmarshal watchdog_check.json: %v", err)
	}
	return sg
}

func runWatchdogCheck(t *testing.T, thresholdMs float64, initialState string) *RuntimeContext {
	t.Helper()
	sg := loadWatchdogCheck(t)
	c := &container.Container{
		SchemaVersion: 1,
		ID:            "watchdog_check_test",
		Name:          "watchdog_check_test",
		Vars: []container.VarDecl{
			{Name: "state", Type: "string", Default: "IDLE"},
		},
		Subgraphs: []container.Subgraph{sg},
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "call", Kind: "Subgraph", Config: map[string]any{
					"SubgraphID": "watchdog_check",
					"literal":    map[string]any{"thresholdMs": thresholdMs},
				}},
				{ID: "stop", Kind: "Stop"},
			},
			Edges: []container.GraphEdge{
				{From: "start.Done", To: "call.in"},
				{From: "call.Done", To: "stop.in"},
			},
		},
	}
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, NoopColorDetector{}, nil, nil, nil, 0)
	stubRuntimeWindowAndInput(rt)
	rt.SetVar("state", initialState)
	r := NewContainerRunner(rt)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := r.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}
	return rt
}

func TestWatchdog_StuckAndTriggered(t *testing.T) {
	rt := runWatchdogCheck(t, 0, "IDLE")
	got, _ := rt.Vars()["state"].(string)
	if got != "RECOVERING" {
		t.Fatalf("StuckAndTriggered: want state=RECOVERING (threshold=0 永触发 + IDLE 不在白名单), got %q", got)
	}
}

func TestWatchdog_NotStuck(t *testing.T) {
	rt := runWatchdogCheck(t, 120000, "IDLE")
	got, _ := rt.Vars()["state"].(string)
	if got != "IDLE" {
		t.Fatalf("NotStuck: want state=IDLE (threshold 120s 没过), got %q", got)
	}
}

func TestWatchdog_StuckButFishing(t *testing.T) {
	rt := runWatchdogCheck(t, 0, "FISHING")
	got, _ := rt.Vars()["state"].(string)
	if got != "FISHING" {
		t.Fatalf("StuckButFishing: want state=FISHING (白名单豁免), got %q", got)
	}
}

func TestWatchdog_StuckButRecovering(t *testing.T) {
	rt := runWatchdogCheck(t, 0, "RECOVERING")
	got, _ := rt.Vars()["state"].(string)
	if got != "RECOVERING" {
		t.Fatalf("StuckButRecovering: want state=RECOVERING (白名单豁免), got %q", got)
	}
}
