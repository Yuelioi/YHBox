package runtime

import (
	"context"
	"testing"
	"time"

	"yhbox/internal/services/container"
	"yhbox/internal/services/execution"
	"yhbox/internal/services/expr"
)

// TestV4_FishingV2Proto_StateMachine validates v4 expressiveness for the fish bot
// state-machine pattern (the v2 north star). Real Fishing v2 has 9 states wired through
// subgraphs + image/HSV detection — that's Phase F work. This proto exercises just the
// state-machine dispatch core: SetVar state → Switch on state → branch to per-state SetVar.
//
// Topology:
//
//	Start → SetVar(state="IDLE",global)
//	      → Switch(value=GetVar(state)) cases ["IDLE", "FISHING", "RESULT"]
//	          .IDLE     → SetVar(stepLog="went_IDLE",global) → SetVar(state="FISHING") → re-enter Switch (one-shot here, no loop for proto)
//	          .FISHING  → SetVar(stepLog="went_FISHING")
//	          .RESULT   → SetVar(stepLog="went_RESULT")
//	          .default  → SetVar(stepLog="unknown")
//	      → Stop
//
// Proves: state-machine pattern works end-to-end on v4 (GetVar global + Switch value via
// data edge + multiple exec branches all writing global vars). Loop+re-dispatch deferred.
func TestV4_FishingV2Proto_StateMachine(t *testing.T) {
	c := &container.Container{
		SchemaVersion: 1,
		ID:            "v4-fishing-proto",
		Vars: []container.VarDecl{
			{Name: "state", Type: "string", Default: ""},
			{Name: "stepLog", Type: "string", Default: ""},
		},
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "setInitState", Kind: "SetVar", Config: map[string]any{
					"varName": "state", "scope": "global",
					"literal": map[string]any{"value": "IDLE"},
				}},
				{ID: "getState", Kind: "GetVar", Config: map[string]any{
					"varName": "state", "scope": "global",
				}},
				{ID: "switch1", Kind: "Switch", Config: map[string]any{
					"cases": []any{"IDLE", "FISHING", "RESULT"},
				}},
				{ID: "logIDLE", Kind: "SetVar", Config: map[string]any{
					"varName": "stepLog", "scope": "global",
					"literal": map[string]any{"value": "went_IDLE"},
				}},
				{ID: "logFISHING", Kind: "SetVar", Config: map[string]any{
					"varName": "stepLog", "scope": "global",
					"literal": map[string]any{"value": "went_FISHING"},
				}},
				{ID: "logRESULT", Kind: "SetVar", Config: map[string]any{
					"varName": "stepLog", "scope": "global",
					"literal": map[string]any{"value": "went_RESULT"},
				}},
				{ID: "logDefault", Kind: "SetVar", Config: map[string]any{
					"varName": "stepLog", "scope": "global",
					"literal": map[string]any{"value": "unknown"},
				}},
				{ID: "stop", Kind: "Stop"},
			},
			Edges: []container.GraphEdge{
				// Exec
				{From: "start.out", To: "setInitState.in"},
				{From: "setInitState.out", To: "switch1.in"},
				{From: "switch1.IDLE", To: "logIDLE.in"},
				{From: "switch1.FISHING", To: "logFISHING.in"},
				{From: "switch1.RESULT", To: "logRESULT.in"},
				{From: "switch1.default", To: "logDefault.in"},
				{From: "logIDLE.out", To: "stop.in"},
				{From: "logFISHING.out", To: "stop.in"},
				{From: "logRESULT.out", To: "stop.in"},
				{From: "logDefault.out", To: "stop.in"},
				// Data: GetVar(state) → Switch.value
				{From: "getState.value", To: "switch1.value", Kind: "data"},
			},
		},
	}
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, NoopColorDetector{}, nil, nil, nil, nil, 0)
	stubRuntimeWindowAndInput(rt)
	r := NewContainerRunner(rt)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := r.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := expr.AsString(rt.Vars()["stepLog"]); got != "went_IDLE" {
		t.Fatalf("state-machine dispatch: state=IDLE should route to IDLE branch, got log=%v", got)
	}
}
