package container

import (
	"testing"
	"time"
)

func newTestGraph() Graph {
	return Graph{
		ID:            "g1",
		SchemaVersion: GraphSchemaVersion,
		Nodes: []GraphNode{
			{ID: "start", Kind: "Start", CreatedAt: time.Now().UTC()},
			{ID: "wait", Kind: "Sleep", Config: map[string]any{"literal": map[string]any{"Duration": 100}}, CreatedAt: time.Now().UTC()},
			{ID: "click", Kind: "Log", Config: map[string]any{"literal": map[string]any{"Message": "done"}}, CreatedAt: time.Now().UTC()},
		},
		Edges: []GraphEdge{
			{From: "start.Done", To: "wait.In"},
			{From: "wait.Done", To: "click.In"},
		},
	}
}

func TestGraphRewriter_RenameNodeID(t *testing.T) {
	g := newTestGraph()
	r := NewGraphRewriter()
	r.RenameNodeID("wait", "wait-new")
	r.Apply(&g)

	found := false
	for _, n := range g.Nodes {
		if n.ID == "wait-new" {
			found = true
		}
		if n.ID == "wait" {
			t.Errorf("old node id 'wait' still present")
		}
	}
	if !found {
		t.Errorf("renamed node 'wait-new' not found")
	}
	if g.Edges[0].To != "wait-new.In" {
		t.Errorf("edge[0].To = %q, want wait-new.In", g.Edges[0].To)
	}
	if g.Edges[1].From != "wait-new.Done" {
		t.Errorf("edge[1].From = %q, want wait-new.Done", g.Edges[1].From)
	}
}

func TestGraphRewriter_NoOp(t *testing.T) {
	g := newTestGraph()
	original := g
	r := NewGraphRewriter()
	r.Apply(&g)
	if len(g.Nodes) != len(original.Nodes) {
		t.Errorf("no-op rewriter changed node count")
	}
	if g.Edges[0] != original.Edges[0] {
		t.Errorf("no-op rewriter changed edge: %+v vs %+v", g.Edges[0], original.Edges[0])
	}
}
