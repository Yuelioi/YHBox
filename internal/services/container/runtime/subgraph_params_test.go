package runtime

import (
	"context"
	"testing"
	"time"

	"yhbox/internal/services/container"
	"yhbox/internal/services/execution"
	"yhbox/internal/services/expr"
)

// TestSubgraph_InputParams_PullFromLiteral:
// Parent calls subgraph with InputParam hp; literal pin value flows into frame.LocalParams,
// inner GetParam(hp) reads it back, SetVar(result, global) writes it for verification.
func TestSubgraph_InputParams_PullFromLiteral(t *testing.T) {
	subgraphInput := &container.GraphNode{ID: "sgi", Kind: "SubgraphInput"}
	getParam := &container.GraphNode{ID: "gp", Kind: "GetParam", Config: map[string]any{"paramName": "hp"}}
	setVar := &container.GraphNode{
		ID:   "sv",
		Kind: "SetVar",
		Config: map[string]any{
			"varName": "result",
			"scope":   "global",
		},
	}
	sgo := &container.GraphNode{
		ID:     "sgo",
		Kind:   "SubgraphOutput",
		Config: map[string]any{"declID": "done"},
	}

	sg := container.Subgraph{
		ID: "sg-hp",
		InputParams: []container.SubgraphInputParam{
			{Name: "hp", Type: "number"},
		},
		OutputPins: []container.SubgraphOutputDecl{{ID: "done", Name: "done"}},
		Graph: container.Graph{
			Nodes: []container.GraphNode{*subgraphInput, *getParam, *setVar, *sgo},
			Edges: []container.GraphEdge{
				{From: "sgi.out", To: "sv.in"},
				{From: "sv.out", To: "sgo.in"},
				// data: GetParam(hp).value → SetVar(result).value
				{From: "gp.value", To: "sv.value", Kind: "data"},
			},
		},
	}

	c := &container.Container{
		SchemaVersion: 1, ID: "v4-params-e2e", Name: "v4-params-e2e",
		Vars: []container.VarDecl{
			{Name: "result", Type: "number", Default: 0.0},
		},
		Subgraphs: []container.Subgraph{sg},
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "call", Kind: "Subgraph",
					Config: map[string]any{
						"subgraphId": "sg-hp",
						// Literal pin "hp" sends 0.42 into the subgraph's input.
						"literal": map[string]any{"hp": 0.42},
					}},
				{ID: "stop", Kind: "Stop"},
			},
			Edges: []container.GraphEdge{
				{From: "start.out", To: "call.in"},
				{From: "call.done", To: "stop.in"},
			},
		},
	}

	rtCtx := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, NoopColorDetector{}, nil, nil, nil, nil, 0)
	stubRuntimeWindowAndInput(rtCtx)
	r := NewContainerRunner(rtCtx)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := r.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}

	got, _ := expr.AsNumber(rtCtx.Vars()["result"])
	if got != 0.42 {
		t.Fatalf("result via InputParam: want 0.42, got %v", got)
	}
}
