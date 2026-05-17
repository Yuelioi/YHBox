package container

import (
	"testing"
	"time"
)

func setupStoreWithContainer(t *testing.T) (*Store, string) {
	t.Helper()
	root := t.TempDir()
	st, err := NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	cid := "c1"
	c := &Container{
		SchemaVersion: CurrentSchemaVersion,
		ID:            cid,
		Name:          "c1",
		Graph: Graph{
			ID: "gm", Version: GraphSchemaVersion,
			Nodes: []GraphNode{
				{ID: "s", Kind: "Start", CreatedAt: time.Now().UTC()},
				{ID: "w", Kind: "WindowTarget", Config: map[string]any{"match": map[string]any{"title": "异环"}}, CreatedAt: time.Now().UTC()},
			},
		},
	}
	if err := st.Save(c); err != nil {
		t.Fatal(err)
	}
	return st, cid
}

func TestStore_SaveAndGetSubgraph(t *testing.T) {
	st, cid := setupStoreWithContainer(t)
	sg := &Subgraph{
		ID:    "sg-A",
		Label: "A",
		Graph: Graph{
			ID: "g-sgA", Version: GraphSchemaVersion,
			Nodes: []GraphNode{{ID: "in", Kind: "SubgraphInput", CreatedAt: time.Now().UTC()}},
		},
		OutputPins: []SubgraphOutputDecl{{ID: "d", Name: "done"}},
		CreatedAt:  time.Now().UTC(),
	}
	if err := st.SaveSubgraph(cid, sg); err != nil {
		t.Fatalf("save sg: %v", err)
	}
	got, ok := st.GetSubgraph(cid, "sg-A")
	if !ok {
		t.Fatal("not found")
	}
	if got.Label != "A" {
		t.Errorf("label = %q", got.Label)
	}
}

func TestStore_ListSubgraphs(t *testing.T) {
	st, cid := setupStoreWithContainer(t)
	for _, id := range []string{"sg-1", "sg-2", "sg-3"} {
		st.SaveSubgraph(cid, &Subgraph{
			ID: id, Label: id,
			Graph: Graph{ID: "g-" + id, Version: GraphSchemaVersion},
			OutputPins: []SubgraphOutputDecl{{ID: "d", Name: "done"}},
			CreatedAt: time.Now().UTC(),
		})
	}
	list := st.ListSubgraphs(cid)
	if len(list) != 3 {
		t.Errorf("expected 3, got %d", len(list))
	}
}

func TestStore_DeleteSubgraph(t *testing.T) {
	st, cid := setupStoreWithContainer(t)
	st.SaveSubgraph(cid, &Subgraph{
		ID: "sg-x", Label: "x",
		Graph: Graph{ID: "g-sgx", Version: GraphSchemaVersion},
		OutputPins: []SubgraphOutputDecl{{ID: "d", Name: "done"}},
		CreatedAt: time.Now().UTC(),
	})
	if err := st.DeleteSubgraph(cid, "sg-x"); err != nil {
		t.Fatal(err)
	}
	if _, ok := st.GetSubgraph(cid, "sg-x"); ok {
		t.Errorf("still found after delete")
	}
}

func TestStore_GetSubgraph_NotFound(t *testing.T) {
	st, cid := setupStoreWithContainer(t)
	if _, ok := st.GetSubgraph(cid, "nope"); ok {
		t.Errorf("nope should not exist")
	}
	if _, ok := st.GetSubgraph("no-such-container", "x"); ok {
		t.Errorf("no-such-container should not exist")
	}
}
