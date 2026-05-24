package runtime

import (
	"strings"
	"testing"
	"time"

	"yhbox/internal/services/container"
)

func TestResolveSubgraphCall_OK(t *testing.T) {
	c := &container.Container{
		ID: "c1",
		Subgraphs: []container.Subgraph{
			{ID: "sg-A", Label: "A"},
		},
	}
	callNode := &container.GraphNode{
		ID:        "call",
		Kind:      "Subgraph",
		Config:    map[string]any{"SubgraphID": "sg-A"},
		CreatedAt: time.Now().UTC(),
	}
	sg, err := ResolveSubgraphCall(c, callNode)
	if err != nil {
		t.Fatalf("expected ok, got %v", err)
	}
	if sg.ID != "sg-A" {
		t.Errorf("got %q", sg.ID)
	}
}

func TestResolveSubgraphCall_MissingConfig(t *testing.T) {
	c := &container.Container{ID: "c1"}
	callNode := &container.GraphNode{ID: "call", Kind: "Subgraph", Config: map[string]any{}, CreatedAt: time.Now().UTC()}
	_, err := ResolveSubgraphCall(c, callNode)
	if err == nil || !strings.Contains(err.Error(), "SubgraphID") {
		t.Errorf("expected SubgraphID err, got %v", err)
	}
}

func TestResolveSubgraphCall_NotFound(t *testing.T) {
	c := &container.Container{ID: "c1", Subgraphs: nil}
	callNode := &container.GraphNode{ID: "call", Kind: "Subgraph", Config: map[string]any{"SubgraphID": "ghost"}, CreatedAt: time.Now().UTC()}
	_, err := ResolveSubgraphCall(c, callNode)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected not found err, got %v", err)
	}
}

func TestResolveSubgraphOutputDestEdge(t *testing.T) {
	parentEdges := []container.GraphEdge{
		{From: "call-node.decl-1", To: "next-node.in"},
		{From: "call-node.decl-2", To: "other.in"},
	}
	got := FindParentDownstreamByDeclID(parentEdges, "call-node", "decl-1")
	if got != "next-node.in" {
		t.Errorf("got %q want next-node.in", got)
	}
	got2 := FindParentDownstreamByDeclID(parentEdges, "call-node", "decl-3-missing")
	if got2 != "" {
		t.Errorf("expected empty for missing decl, got %q", got2)
	}
}
