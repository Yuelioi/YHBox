package container

import (
	"testing"
	"time"
)

func newTestGraph() Graph {
	return Graph{
		ID:      "g1",
		Version: GraphSchemaVersion,
		Nodes: []GraphNode{
			{ID: "start", Kind: "Start", CreatedAt: time.Now().UTC()},
			{ID: "wait", Kind: "WaitTemplate", Config: map[string]any{"literal": map[string]any{"Template": "fish/onhook", "Threshold": "0.85"}}, CreatedAt: time.Now().UTC()},
			{ID: "click", Kind: "ClickTemplate", Config: map[string]any{"literal": map[string]any{"Template": "fish/onhook"}}, CreatedAt: time.Now().UTC()},
		},
		Edges: []GraphEdge{
			{From: "start.Done", To: "wait.in"},
			{From: "wait.found", To: "click.in"},
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
	if g.Edges[0].To != "wait-new.in" {
		t.Errorf("edge[0].To = %q, want wait-new.in", g.Edges[0].To)
	}
	if g.Edges[1].From != "wait-new.found" {
		t.Errorf("edge[1].From = %q, want wait-new.found", g.Edges[1].From)
	}
}

func TestGraphRewriter_RenameTemplateKey(t *testing.T) {
	g := newTestGraph()
	r := NewGraphRewriter()
	r.RenameTemplateKey("fish/onhook", "fish/onhook_2")
	r.Apply(&g)

	for _, n := range g.Nodes {
		if n.Kind == "WaitTemplate" || n.Kind == "ClickTemplate" {
			got := PinString(&n, "Template")
			if got != "fish/onhook_2" {
				t.Errorf("node %s template = %q, want fish/onhook_2", n.ID, got)
			}
		}
	}
}

func TestGraphRewriter_Combined(t *testing.T) {
	g := newTestGraph()
	r := NewGraphRewriter()
	r.RenameNodeID("wait", "w2")
	r.RenameTemplateKey("fish/onhook", "fish/onhook_v2")
	r.Apply(&g)

	if g.Edges[0].To != "w2.in" {
		t.Errorf("rename + rename: edge not updated: %q", g.Edges[0].To)
	}
	waitNode := g.Nodes[1]
	if waitNode.ID != "w2" || PinString(&waitNode, "Template") != "fish/onhook_v2" {
		t.Errorf("rename + rename: node not fully updated: %+v", waitNode)
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
