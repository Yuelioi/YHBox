package container

import (
	"testing"
	"time"
)

func TestSync_OnlyMainCalibrationNodePatched(t *testing.T) {
	st, _ := setupStoreWithContainer(t)
	cid := "c1"
	c, _ := st.Get(cid)
	c.Graph.Nodes = append(c.Graph.Nodes,
		GraphNode{
			ID:        "cal",
			Kind:      "MouseCalibration",
			Config:    map[string]any{"Counts360": 1800},
			CreatedAt: time.Now().UTC(),
		},
	)
	c.Graph.Nodes = append(c.Graph.Nodes,
		GraphNode{ID: "m", Kind: "MouseMoveRel", Config: map[string]any{"dx": "100"}, CreatedAt: time.Now().UTC()},
	)

	// Save container first with the new nodes
	st.Save(&c)

	// Now save the subgraph (need SubgraphOutput node to pass validation)
	sg := Subgraph{
		ID:    "sg-1",
		Label: "录制",
		Graph: Graph{
			ID: "g-sg1", Version: GraphSchemaVersion,
			Nodes: []GraphNode{
				{ID: "in", Kind: "SubgraphInput", CreatedAt: time.Now().UTC()},
				{ID: "out", Kind: "SubgraphOutput", CreatedAt: time.Now().UTC()},
			},
		},
		OutputPins: []SubgraphOutputDecl{{ID: "d", Name: "done"}},
		RecordingContext: &RecordingContext{MouseCounts360: 4120},
		CreatedAt:        time.Now().UTC(),
	}
	st.SaveSubgraph(cid, &sg)

	res, err := SyncLocalMouseCalibration(st, 2200)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Updated) != 1 || res.Updated[0] != cid {
		t.Errorf("Updated = %v, want [c1]", res.Updated)
	}
	c2, _ := st.Get(cid)
	for _, n := range c2.Graph.Nodes {
		if n.ID == "cal" {
			counts, _ := intFromConfig(n.Config, "Counts360")
			if counts != 2200 {
				t.Errorf("main cal.counts360 = %d, want 2200", counts)
			}
		}
	}
	sg2, _ := st.GetSubgraph(cid, "sg-1")
	if sg2.RecordingContext == nil || sg2.RecordingContext.MouseCounts360 != 4120 {
		t.Errorf("subgraph RecordingContext was tampered: %+v", sg2.RecordingContext)
	}
}

func TestSync_SkipContainerWithoutCalibration(t *testing.T) {
	st, _ := setupStoreWithContainer(t)
	res, err := SyncLocalMouseCalibration(st, 2200)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Updated) != 0 {
		t.Errorf("Updated should be empty for container without MouseCalibration")
	}
	if len(res.Skipped) != 1 || res.Skipped[0] != "c1" {
		t.Errorf("expected c1 skipped, got %+v", res)
	}
}

func TestSync_SkipIncompatible(t *testing.T) {
	st, _ := setupStoreWithContainer(t)
	c, _ := st.Get("c1")
	c.Status = StatusIncompatible
	c.IncompatibleReason = "test"
	res, err := SyncLocalMouseCalibration(st, 2200)
	if err != nil {
		t.Fatal(err)
	}
	_ = res
}
