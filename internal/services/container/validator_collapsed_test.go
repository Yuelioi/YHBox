package container

import "testing"

func TestValidateCollapsed_SubgraphCallToAnonymous_Errors(t *testing.T) {
	c := &Container{
		SchemaVersion: 1,
		Subgraphs: []Subgraph{
			{ID: "sg-anon", IsAnonymous: true},
		},
		Graph: Graph{
			Nodes: []GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "sg1", Kind: "Subgraph", Config: map[string]any{"subgraphId": "sg-anon"}},
			},
		},
	}
	errs := ValidateContainer(c)
	found := false
	for _, e := range errs {
		if e.Code == CodeCollapsedReferencedBySubgraphCall {
			found = true
		}
	}
	if !found {
		t.Fatalf("Subgraph kind ref to anonymous subgraph should error, got %+v", errs)
	}
}

func TestValidateCollapsed_CollapsedToMissing_Errors(t *testing.T) {
	c := &Container{
		SchemaVersion: 1,
		Graph: Graph{
			Nodes: []GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "c1", Kind: "CollapsedNode", Config: map[string]any{"subgraphId": "missing-id"}},
			},
		},
	}
	errs := ValidateContainer(c)
	found := false
	for _, e := range errs {
		if e.Code == CodeCollapsedPinBroken {
			found = true
		}
	}
	if !found {
		t.Fatalf("CollapsedNode → missing sg should COLLAPSED_PIN_BROKEN, got %+v", errs)
	}
}

func TestValidateCollapsed_CollapsedToNonAnonymous_Errors(t *testing.T) {
	c := &Container{
		SchemaVersion: 1,
		Subgraphs: []Subgraph{
			{ID: "sg-pub", IsAnonymous: false},
		},
		Graph: Graph{
			Nodes: []GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "c1", Kind: "CollapsedNode", Config: map[string]any{"subgraphId": "sg-pub"}},
			},
		},
	}
	errs := ValidateContainer(c)
	found := false
	for _, e := range errs {
		if e.Code == CodeCollapsedPinBroken {
			found = true
		}
	}
	if !found {
		t.Fatalf("CollapsedNode → non-anonymous sg should COLLAPSED_PIN_BROKEN, got %+v", errs)
	}
}

func TestValidateCollapsed_HappyPath(t *testing.T) {
	c := &Container{
		SchemaVersion: 1,
		Subgraphs: []Subgraph{
			{ID: "sg-anon", IsAnonymous: true},
		},
		Graph: Graph{
			Nodes: []GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "c1", Kind: "CollapsedNode", Config: map[string]any{"subgraphId": "sg-anon"}},
			},
		},
	}
	errs := ValidateContainer(c)
	for _, e := range errs {
		if e.Code == CodeCollapsedPinBroken || e.Code == CodeCollapsedReferencedBySubgraphCall {
			t.Errorf("happy path falsely flagged: %+v", e)
		}
	}
}
