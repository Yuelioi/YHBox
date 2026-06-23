package container

import "testing"

func aiRefNode(id, conn string) GraphNode {
	return GraphNode{ID: id, Kind: "AI", Config: map[string]any{
		"literal": map[string]any{"Connection": conn},
	}}
}

func TestScanAIConnectionRefs(t *testing.T) {
	containers := []Container{
		{ID: "ct1", Name: "Fishing", Graph: Graph{Nodes: []GraphNode{
			aiRefNode("n1", "c1"),
			{ID: "n2", Kind: "Sleep"},
		}}},
		{ID: "ct2", Name: "Other", Graph: Graph{Nodes: []GraphNode{
			aiRefNode("n3", "c2"),
			aiRefNode("n4", "c1"),
		}}},
	}

	refs := ScanAIConnectionRefs(containers, "c1")
	if len(refs) != 2 {
		t.Fatalf("c1 refs = %d, want 2 (%+v)", len(refs), refs)
	}
	if refs[0].ContainerID != "ct1" || refs[0].NodeID != "n1" || refs[0].ContainerName != "Fishing" {
		t.Errorf("ref[0] = %+v", refs[0])
	}
	if refs[1].ContainerID != "ct2" || refs[1].NodeID != "n4" {
		t.Errorf("ref[1] = %+v", refs[1])
	}

	if got := ScanAIConnectionRefs(containers, "nope"); len(got) != 0 {
		t.Errorf("unknown id should yield no refs, got %+v", got)
	}
	if got := ScanAIConnectionRefs(containers, ""); got != nil {
		t.Errorf("empty id should yield nil, got %+v", got)
	}
}
