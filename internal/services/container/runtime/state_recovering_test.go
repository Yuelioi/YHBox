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

func loadStateRECOVERING(t *testing.T) container.Subgraph {
	t.Helper()
	_, thisFile, _, _ := gort.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..")
	jsonPath := filepath.Join(root, "bin", "data", "containers", "fishing-v2", "subgraphs", "state_RECOVERING.json")
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("read state_RECOVERING.json: %v", err)
	}
	var sg container.Subgraph
	if err := json.Unmarshal(data, &sg); err != nil {
		t.Fatalf("unmarshal state_RECOVERING.json: %v", err)
	}
	return sg
}

func runStateRECOVERING(t *testing.T, hits map[string]bool, preEscDone bool) (*spyInputBackend, *RuntimeContext, error) {
	t.Helper()
	stateRECOVERING := loadStateRECOVERING(t)
	inspectPhase := loadFishingV2Helper(t, "inspect_phase.json")
	c := &container.Container{
		SchemaVersion: 1,
		ID:            "state_recovering_test",
		Name:          "state_recovering_test",
		Vars: []container.VarDecl{
			{Name: "state", Type: "string", Default: "RECOVERING"},
			{Name: "_inspectPhaseResult", Type: "string", Default: "unknown"},
			{Name: "recoveryEscDone", Type: "bool", Default: false},
		},
		Subgraphs: []container.Subgraph{stateRECOVERING, inspectPhase},
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "call", Kind: "Subgraph", Config: map[string]any{"SubgraphID": "state_RECOVERING"}},
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
	rt.SetVar("recoveryEscDone", preEscDone)
	r := NewContainerRunner(rt)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := r.Run(ctx)
	return spy, rt, err
}

func containsKeyEvent(events []string, target string) bool {
	for _, e := range events {
		if e == target {
			return true
		}
	}
	return false
}

func TestStateRECOVERING_FirstESC_NoMatch(t *testing.T) {
	spy, rt, err := runStateRECOVERING(t, map[string]bool{}, false)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, _ := rt.Vars()["state"].(string)
	if got != "RECOVERING" {
		t.Fatalf("FirstESC_NoMatch: want state=RECOVERING (unchanged), got %q", got)
	}
	escDone, _ := rt.Vars()["recoveryEscDone"].(bool)
	if !escDone {
		t.Errorf("FirstESC_NoMatch: want recoveryEscDone=true after first kick, got false")
	}
	if !containsKeyEvent(spy.keyEvents, "down:esc") {
		t.Errorf("FirstESC_NoMatch: want spy.keyEvents to include down:esc, got %v", spy.keyEvents)
	}
}

func TestStateRECOVERING_SecondTick_AlreadyEscDone(t *testing.T) {
	spy, rt, err := runStateRECOVERING(t, map[string]bool{}, true)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, _ := rt.Vars()["state"].(string)
	if got != "RECOVERING" {
		t.Fatalf("SecondTick: want state=RECOVERING (unchanged), got %q", got)
	}
	escDone, _ := rt.Vars()["recoveryEscDone"].(bool)
	if !escDone {
		t.Errorf("SecondTick: want recoveryEscDone=true (unchanged), got false")
	}
	if containsKeyEvent(spy.keyEvents, "down:esc") {
		t.Errorf("SecondTick: want no ESC press (already done), got %v", spy.keyEvents)
	}
}

func TestStateRECOVERING_InspectRoutes_StartFish(t *testing.T) {
	_, rt, err := runStateRECOVERING(t, map[string]bool{"fishing.start_fish": true}, false)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, _ := rt.Vars()["state"].(string)
	if got != "SETUP" {
		t.Fatalf("InspectRoutes_StartFish: want state=SETUP (via inspect_phase setup), got %q (_inspectPhaseResult=%v)",
			got, rt.Vars()["_inspectPhaseResult"])
	}
}

func TestStateRECOVERING_InspectRoutes_HookIcon(t *testing.T) {
	_, rt, err := runStateRECOVERING(t, map[string]bool{"fishing.hook_icon": true}, false)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, _ := rt.Vars()["state"].(string)
	if got != "IDLE" {
		t.Fatalf("InspectRoutes_HookIcon: want state=IDLE (via inspect_phase ready), got %q (_inspectPhaseResult=%v)",
			got, rt.Vars()["_inspectPhaseResult"])
	}
}

func TestStateRECOVERING_InspectRoutes_NeedBait(t *testing.T) {
	_, rt, err := runStateRECOVERING(t, map[string]bool{"fishing.need_bait": true}, false)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, _ := rt.Vars()["state"].(string)
	if got != "BUYBAIT" {
		t.Fatalf("InspectRoutes_NeedBait: want state=BUYBAIT, got %q (_inspectPhaseResult=%v)",
			got, rt.Vars()["_inspectPhaseResult"])
	}
}
