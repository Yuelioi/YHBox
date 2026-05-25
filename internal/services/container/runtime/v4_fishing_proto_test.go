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
					"VarName": "state", "Scope": "global",
					"literal": map[string]any{"Value": "IDLE"},
				}},
				{ID: "getState", Kind: "GetVar", Config: map[string]any{
					"VarName": "state", "Scope": "global",
				}},
				{ID: "switch1", Kind: "Switch", Config: map[string]any{
					"Case1Value": "IDLE", "Case2Value": "FISHING", "Case3Value": "RESULT",
				}},
				{ID: "logIDLE", Kind: "SetVar", Config: map[string]any{
					"VarName": "stepLog", "Scope": "global",
					"literal": map[string]any{"Value": "went_IDLE"},
				}},
				{ID: "logFISHING", Kind: "SetVar", Config: map[string]any{
					"VarName": "stepLog", "Scope": "global",
					"literal": map[string]any{"Value": "went_FISHING"},
				}},
				{ID: "logRESULT", Kind: "SetVar", Config: map[string]any{
					"VarName": "stepLog", "Scope": "global",
					"literal": map[string]any{"Value": "went_RESULT"},
				}},
				{ID: "logDefault", Kind: "SetVar", Config: map[string]any{
					"VarName": "stepLog", "Scope": "global",
					"literal": map[string]any{"Value": "unknown"},
				}},
				{ID: "stop", Kind: "Stop"},
			},
			Edges: []container.GraphEdge{
				// Exec
				{From: "start.out", To: "setInitState.In"},
				{From: "setInitState.Done", To: "switch1.In"},
				{From: "switch1.Case1", To: "logIDLE.In"},
				{From: "switch1.Case2", To: "logFISHING.In"},
				{From: "switch1.Case3", To: "logRESULT.In"},
				{From: "switch1.Default", To: "logDefault.In"},
				{From: "logIDLE.Done", To: "stop.In"},
				{From: "logFISHING.Done", To: "stop.In"},
				{From: "logRESULT.Done", To: "stop.In"},
				{From: "logDefault.Done", To: "stop.In"},
				// Data: GetVar(state) → Switch.value
				{From: "getState.Value", To: "switch1.Value"},
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
