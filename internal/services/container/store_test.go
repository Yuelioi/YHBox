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
		Graph: Graph{Nodes: []GraphNode{
			{ID: "n1", Kind: "Start"},
			{ID: "w", Kind: "WindowTarget", Config: map[string]any{"Title": "异环"}},
		}},
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
	_ = s.Save(&Container{SchemaVersion: 1, ID: "x", Name: "y", Graph: Graph{Nodes: []GraphNode{
		{ID: "n1", Kind: "Start"},
		{ID: "w", Kind: "WindowTarget", Config: map[string]any{"Title": "异环"}},
	}}})
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
			Nodes: []GraphNode{
				{ID: "s", Kind: "Start", CreatedAt: time.Now().UTC()},
				{ID: "w", Kind: "WindowTarget", Config: map[string]any{"Title": "异环"}, CreatedAt: time.Now().UTC()},
			},
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

func TestContainerStore_Reload(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewStore(dir)
	c := &Container{
		SchemaVersion: 1, ID: "r1", Name: "old",
		Graph: Graph{Nodes: []GraphNode{
			{ID: "n1", Kind: "Start"},
			{ID: "w", Kind: "WindowTarget", Config: map[string]any{"Title": "异环"}},
		}},
	}
	if err := s.Save(c); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// 模拟外部进程 (MCP / 手改) 直接改磁盘上的 container.json
	onDisk := filepath.Join(dir, "r1", "container.json")
	b, _ := os.ReadFile(onDisk)
	var dc Container
	if err := json.Unmarshal(b, &dc); err != nil {
		t.Fatal(err)
	}
	dc.Name = "new-from-disk"
	nb, _ := json.Marshal(&dc)
	if err := os.WriteFile(onDisk, nb, 0o644); err != nil {
		t.Fatal(err)
	}

	// 改盘后内存缓存仍是旧值
	if g, _ := s.Get("r1"); g.Name != "old" {
		t.Errorf("改盘后内存应仍是旧值, got %q", g.Name)
	}

	// Reload 后拿到新值, 且 byID 已更新
	got, err := s.Reload("r1")
	if err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if got.Name != "new-from-disk" {
		t.Errorf("Reload 返回名字 = %q, want new-from-disk", got.Name)
	}
	if g, _ := s.Get("r1"); g.Name != "new-from-disk" {
		t.Errorf("byID 未更新: %q", g.Name)
	}
}

func TestContainerStore_Reload_DeletedDir(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewStore(dir)
	if err := s.Save(&Container{
		SchemaVersion: 1, ID: "d1", Name: "x",
		Graph: Graph{Nodes: []GraphNode{
			{ID: "n1", Kind: "Start"},
			{ID: "w", Kind: "WindowTarget", Config: map[string]any{"Title": "异环"}},
		}},
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	// 外部删掉整个容器目录
	if err := os.RemoveAll(filepath.Join(dir, "d1")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Reload("d1"); err == nil {
		t.Error("目录已删, Reload 应返 not-found error")
	}
	if _, ok := s.Get("d1"); ok {
		t.Error("Reload 应把已删容器从 byID 移除")
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
