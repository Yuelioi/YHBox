package runtime

import (
	"context"
	"testing"
	"time"

	"yotta/internal/services/container"
	"yotta/internal/services/execution"
	"yotta/internal/services/expr"
)

// TestSubgraph_InputParams_PullFromLiteral:
// Parent calls subgraph with InputParam hp; literal pin value flows into frame.LocalParams,
// inner GetParam(hp) reads it back, SetVar(result, global) writes it for verification.
func TestSubgraph_InputParams_PullFromLiteral(t *testing.T) {
	subgraphInput := &container.GraphNode{ID: "sgi", Kind: "SubgraphInput"}
	getParam := &container.GraphNode{ID: "gp", Kind: "GetParam", Config: map[string]any{"ParamName": "hp"}}
	setVar := &container.GraphNode{
		ID:   "sv",
		Kind: "SetVar",
		Config: map[string]any{
			"VarName": "result",
			"Scope":   "global",
		},
	}
	sgo := &container.GraphNode{
		ID:     "sgo",
		Kind:   "SubgraphOutput",
		Config: map[string]any{"DeclID": "done"},
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
				{From: "sgi.Done", To: "sv.In"},
				{From: "sv.Done", To: "sgo.In"},
				// data: GetParam(hp).value → SetVar(result).value
				{From: "gp.Value", To: "sv.Value"},
			},
		},
	}

	c := &container.Container{
		SchemaVersion: 1, ID: "v4-params-e2e", Name: "v4-params-e2e",
		Vars: []container.VarDecl{
			{Name: "result", Type: "number", Default: 0.0},
		},
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "call", Kind: "Subgraph",
					Config: map[string]any{
						"SubgraphID": "sg-hp",
						// Literal pin "hp" sends 0.42 into the subgraph's input.
						"literal": map[string]any{"hp": 0.42},
					}},
				{ID: "stop", Kind: "Stop"},
			},
			Edges: []container.GraphEdge{
				{From: "start.Done", To: "call.In"},
				{From: "call.Done", To: "stop.In"},
			},
		},
	}

	rtCtx := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, nil, nil, nil, 0)
	rtCtx.Subgraphs = []container.Subgraph{sg}
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
