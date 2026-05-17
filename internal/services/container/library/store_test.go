package library

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"yhbox/internal/services/container"
)

func newStore(t *testing.T) (*Store, string) {
	root := t.TempDir()
	st, err := NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	return st, root
}

func TestLibrary_CreateAndListSubgraph(t *testing.T) {
	st, _ := newStore(t)
	sg := container.Subgraph{
		ID:    "lib-sg-1",
		Label: "上钩等待",
		Graph: container.Graph{ID: "g", Version: container.GraphSchemaVersion},
		OutputPins: []container.SubgraphOutputDecl{{ID: "d", Name: "done"}},
		Tags:       []string{"钓鱼"},
		CreatedAt:  time.Now().UTC(),
	}
	if err := st.SaveSubgraph(&LibrarySubgraph{Subgraph: sg}); err != nil {
		t.Fatal(err)
	}
	list := st.ListSubgraphs()
	if len(list) != 1 || list[0].ID != "lib-sg-1" {
		t.Errorf("list = %+v", list)
	}
}

func TestLibrary_DeleteSubgraph(t *testing.T) {
	st, root := newStore(t)
	st.SaveSubgraph(&LibrarySubgraph{Subgraph: container.Subgraph{
		ID:    "lib-sg-x",
		Label: "x",
		Graph: container.Graph{ID: "g", Version: container.GraphSchemaVersion},
		OutputPins: []container.SubgraphOutputDecl{{ID: "d", Name: "done"}},
		CreatedAt:  time.Now().UTC(),
	}})
	if err := st.DeleteSubgraph("lib-sg-x"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "subgraphs", "lib-sg-x.json")); err == nil {
		t.Errorf("file still exists after delete")
	}
	if _, ok := st.GetSubgraph("lib-sg-x"); ok {
		t.Errorf("still found in memory")
	}
}

func TestLibrary_LoadAcrossRestart(t *testing.T) {
	root := t.TempDir()
	sgDir := filepath.Join(root, "subgraphs")
	os.MkdirAll(sgDir, 0o755)
	sg := LibrarySubgraph{
		Subgraph: container.Subgraph{
			ID:    "preset",
			Label: "Preset",
			Graph: container.Graph{ID: "g", Version: container.GraphSchemaVersion},
			OutputPins: []container.SubgraphOutputDecl{{ID: "d", Name: "done"}},
			CreatedAt:  time.Now().UTC(),
		},
	}
	b, _ := json.MarshalIndent(sg, "", "  ")
	os.WriteFile(filepath.Join(sgDir, "preset.json"), b, 0o644)

	st, err := NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := st.GetSubgraph("preset"); !ok {
		t.Errorf("preset not loaded after restart")
	}
}
