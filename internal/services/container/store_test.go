package container

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestContainerStore_SaveLoadList(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	c := &Container{
		SchemaVersion: 1, ID: "id-1", Name: "test",
		Graph: Graph{Nodes: []GraphNode{{ID: "n1", Kind: "Start"}}},
	}
	if err := s.Save(c); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "id-1", "container.json")); err != nil {
		t.Errorf("file not on disk: %v", err)
	}

	s2, _ := NewStore(dir)
	list := s2.List()
	if len(list) != 1 {
		t.Errorf("expected 1, got %d", len(list))
	}
	got, ok := s2.Get("id-1")
	if !ok {
		t.Fatal("Get failed")
	}
	if got.Name != "test" {
		t.Errorf("name lost")
	}
}

func TestContainerStore_Delete(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewStore(dir)
	_ = s.Save(&Container{SchemaVersion: 1, ID: "x", Name: "y", Graph: Graph{Nodes: []GraphNode{{ID: "n1", Kind: "Start"}}}})
	if err := s.Delete("x"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, ok := s.Get("x"); ok {
		t.Error("expected gone")
	}
	if _, err := os.Stat(filepath.Join(dir, "x")); !os.IsNotExist(err) {
		t.Error("dir should be gone")
	}
}

func TestContainerStore_Save_InvalidRejected(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewStore(dir)
	// 非空图缺 Start 节点，应被 v2 validator 拒绝
	c := &Container{
		SchemaVersion: 1, ID: "x", Name: "bad",
		Graph: Graph{Nodes: []GraphNode{{ID: "n1", Kind: "Sleep"}}},
	}
	if err := s.Save(c); err == nil {
		t.Error("expected validate error")
	}
}

func TestStore_CreatesSubdirsOnSave(t *testing.T) {
	root := t.TempDir()
	st, err := NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	c := &Container{
		SchemaVersion: CurrentSchemaVersion,
		ID:            "ctest",
		Name:          "test",
		Graph: Graph{
			ID:      "g",
			Version: GraphSchemaVersion,
			Nodes:   []GraphNode{{ID: "s", Kind: "Start", CreatedAt: time.Now().UTC()}},
		},
	}
	if err := st.Save(c); err != nil {
		t.Fatalf("save: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "ctest", "subgraphs")); os.IsNotExist(err) {
		t.Errorf("expected subgraphs/ dir created next to container.json")
	}
	if _, err := os.Stat(filepath.Join(root, "ctest", "templates")); os.IsNotExist(err) {
		t.Errorf("expected templates/ dir created next to container.json")
	}
}

func TestStore_LoadsSubgraphs(t *testing.T) {
	root := t.TempDir()
	cid := "with-subs"
	sgDir := filepath.Join(root, cid, "subgraphs")
	if err := os.MkdirAll(sgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cmain := Container{
		SchemaVersion: CurrentSchemaVersion,
		ID:            cid,
		Name:          "withsubs",
		Graph:         Graph{ID: "gm", Version: GraphSchemaVersion},
	}
	b, _ := json.Marshal(cmain)
	os.WriteFile(filepath.Join(root, cid, "container.json"), b, 0o644)

	sg := Subgraph{
		ID:    "sg-1",
		Label: "Sub1",
		Graph: Graph{ID: "g-sg1", Version: GraphSchemaVersion},
		OutputPins: []SubgraphOutputDecl{{ID: "d1", Name: "done"}},
		CreatedAt: time.Now().UTC(),
	}
	sgB, _ := json.Marshal(sg)
	os.WriteFile(filepath.Join(sgDir, "sg-1.json"), sgB, 0o644)

	st, err := NewStore(root)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	c, ok := st.Get(cid)
	if !ok {
		t.Fatal("container not loaded")
	}
	if len(c.Subgraphs) != 1 {
		t.Fatalf("expected 1 subgraph, got %d", len(c.Subgraphs))
	}
	if c.Subgraphs[0].ID != "sg-1" {
		t.Errorf("subgraph id wrong: %q", c.Subgraphs[0].ID)
	}
}
