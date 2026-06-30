package container

import (
	"archive/zip"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"yotta/internal/services/asset"

	_ "yotta/internal/nodes/all"
)

func TestContainerStore_SaveLoadList(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	c := &Container{
		SchemaVersion: 1, ID: "id-1", Name: "test",
		Version:  "1.2.3",
		Category: "daily",
		Keywords: []string{"daily", "fish"},
		Author:   PackagePerson{Name: "yl"},
		Graph: Graph{Nodes: []GraphNode{
			{ID: "n1", Kind: "Start"},
			{ID: "w", Kind: "Win32WindowTarget", Config: map[string]any{"Title": "异环"}},
		}},
	}
	if err := s.Save(c); err != nil {
		t.Fatalf("Save: %v", err)
	}
	for _, name := range []string{"package.json", "graph.json", "installation.json", "yotta-lock.json"} {
		if _, err := os.Stat(filepath.Join(dir, "id-1", name)); err != nil {
			t.Errorf("%s not on disk: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "id-1", "container.json")); !os.IsNotExist(err) {
		t.Errorf("legacy container.json should not be written, stat err=%v", err)
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
	if got.Version != "1.2.3" || got.Category != "daily" || got.Author.Name != "yl" {
		t.Errorf("package fields lost: version=%q category=%q author=%+v", got.Version, got.Category, got.Author)
	}
	if len(got.Keywords) != 2 || got.Keywords[0] != "daily" || len(got.Tags) != 2 {
		t.Errorf("keywords/tags lost: keywords=%v tags=%v", got.Keywords, got.Tags)
	}
	if title := PinString(&got.Graph.Nodes[1], "Title"); title != "异环" {
		t.Errorf("aggregated graph should rehydrate window title, got %q", title)
	}
}

func TestContainerStore_SaveSplitsPortableGraphAndInstallationBindings(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewStore(dir)

	c := &Container{
		SchemaVersion: 1, ID: "split-1", Name: "split",
		Graph: Graph{Nodes: []GraphNode{
			{ID: "start", Kind: "Start"},
			{ID: "w", Kind: "Win32WindowTarget", Config: map[string]any{
				"Title": "Blue Archive", "Class": "QtWindow", "ProcessName": "game.exe", "TitleMatch": "contains",
			}},
		}},
	}
	if err := s.Save(c); err != nil {
		t.Fatalf("Save: %v", err)
	}

	graph, err := readJSONFile[Graph](filepath.Join(dir, "split-1", "graph.json"))
	if err != nil {
		t.Fatal(err)
	}
	if got := PinString(&graph.Nodes[1], "Target"); got != "game" {
		t.Fatalf("portable graph should keep logical target slot, got %q config=%v", got, graph.Nodes[1].Config)
	}
	if title := PinString(&graph.Nodes[1], "Title"); title != "" {
		t.Fatalf("portable graph must not keep local window title, got %q", title)
	}

	manifest, err := readJSONFile[PackageManifest](filepath.Join(dir, "split-1", "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Yotta.Targets["game"].Kind != "win32-window" {
		t.Fatalf("package target slot missing: %+v", manifest.Yotta.Targets)
	}

	installation, err := readJSONFile[Installation](filepath.Join(dir, "split-1", "installation.json"))
	if err != nil {
		t.Fatal(err)
	}
	binding := installation.TargetBindings["game"]
	if binding.Kind != "win32-window" || binding.Match["title"] != "Blue Archive" || binding.Match["processName"] != "game.exe" {
		t.Fatalf("installation target binding missing local match: %+v", binding)
	}
}

func TestContainerStore_ExportPackageZipExcludesInstallation(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewStore(dir)
	if err := s.Save(&Container{
		SchemaVersion: 1, ID: "zip-1", Name: "zip",
		Graph: Graph{Nodes: []GraphNode{
			{ID: "start", Kind: "Start"},
			{ID: "target", Kind: "Win32WindowTarget", Config: map[string]any{"Title": "Game"}},
		}},
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	out := filepath.Join(t.TempDir(), "zip-1.yotta-container.zip")
	if err := s.ExportPackageZip("zip-1", out); err != nil {
		t.Fatalf("ExportPackageZip: %v", err)
	}
	zr, err := zip.OpenReader(out)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	names := map[string]bool{}
	for _, f := range zr.File {
		names[f.Name] = true
	}
	for _, want := range []string{"package.json", "graph.json", "yotta-lock.json"} {
		if !names[want] {
			t.Fatalf("zip missing %s; names=%v", want, names)
		}
	}
	if names["installation.json"] {
		t.Fatalf("zip must not include installation.json")
	}
}

func TestContainerStore_ExportPackageZipIncludesAssetClosure(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewStore(filepath.Join(dir, "containers"))
	assetStore, _ := asset.NewStore(filepath.Join(dir, "assets"))
	s.SetAssetStore(assetStore)

	templateBlob, err := assetStore.Blobs().Put([]byte("template-png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := assetStore.PutRecord(asset.AssetRecord{
		GUID:   "tpl-1",
		Kind:   asset.KindTemplate,
		Name:   "Template",
		Origin: asset.Origin{Kind: "user"},
		Variants: []asset.Variant{{
			Resolution: [2]int{1280, 720},
			BBox:       [4]int{1, 2, 3, 4},
			Blob:       templateBlob,
		}},
	}); err != nil {
		t.Fatal(err)
	}
	clipBlob, err := assetStore.Blobs().Put([]byte("clip-bytes"))
	if err != nil {
		t.Fatal(err)
	}
	if err := assetStore.PutRecord(asset.AssetRecord{
		GUID:   "clip-1",
		Kind:   asset.KindClip,
		Name:   "Clip",
		Origin: asset.Origin{Kind: "user"},
		Blob:   clipBlob,
	}); err != nil {
		t.Fatal(err)
	}

	if err := s.Save(&Container{
		SchemaVersion: 1, ID: "zip-assets", Name: "zip assets",
		Graph: Graph{Nodes: []GraphNode{
			{ID: "start", Kind: "Start"},
			{ID: "target", Kind: "Win32WindowTarget", Config: map[string]any{"Title": "Game"}},
			{ID: "check", Kind: "CheckTemplate", Config: map[string]any{"literal": map[string]any{"Templates": []any{"tpl-1"}}}},
			{ID: "clip", Kind: "PlayClip", Config: map[string]any{"literal": map[string]any{"ClipID": "clip-1"}}},
		}},
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	out := filepath.Join(t.TempDir(), "zip-assets.yotta-container.zip")
	if err := s.ExportPackageZip("zip-assets", out); err != nil {
		t.Fatalf("ExportPackageZip: %v", err)
	}
	zr, err := zip.OpenReader(out)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	names := map[string]bool{}
	for _, f := range zr.File {
		names[f.Name] = true
	}
	for _, want := range []string{
		"assets/records/tpl-1.json",
		"assets/blobs/" + templateBlob,
		"clips/clip-1.json",
		"clips/blobs/" + clipBlob,
	} {
		if !names[want] {
			t.Fatalf("zip missing %s; names=%v", want, names)
		}
	}
}

func TestContainerStore_ExportPackageZipRejectsStaleLock(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewStore(dir)
	if err := s.Save(&Container{
		SchemaVersion: 1, ID: "stale-lock", Name: "stale",
		Graph: Graph{Nodes: []GraphNode{
			{ID: "start", Kind: "Start"},
			{ID: "target", Kind: "Win32WindowTarget", Config: map[string]any{"Title": "Game"}},
		}},
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	manifestPath := filepath.Join(dir, "stale-lock", "package.json")
	manifest, err := readJSONFile[PackageManifest](manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	manifest.Version = "9.9.9"
	if err := writeJSONAtomic(manifestPath, manifest); err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(t.TempDir(), "stale-lock.yotta-container.zip")
	if err := s.ExportPackageZip("stale-lock", out); err == nil {
		t.Fatal("ExportPackageZip should reject stale yotta-lock.json")
	}
}

func TestContainerStore_Delete(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewStore(dir)
	_ = s.Save(&Container{SchemaVersion: 1, ID: "x", Name: "y", Graph: Graph{Nodes: []GraphNode{
		{ID: "n1", Kind: "Start"},
		{ID: "w", Kind: "Win32WindowTarget", Config: map[string]any{"Title": "异环"}},
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

func TestContainerStore_Reload(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewStore(dir)
	c := &Container{
		SchemaVersion: 1, ID: "r1", Name: "old",
		Graph: Graph{Nodes: []GraphNode{
			{ID: "n1", Kind: "Start"},
			{ID: "w", Kind: "Win32WindowTarget", Config: map[string]any{"Title": "异环"}},
		}},
	}
	if err := s.Save(c); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// 模拟外部进程 (MCP / 手改) 直接改磁盘上的 package.json
	onDisk := filepath.Join(dir, "r1", "package.json")
	b, _ := os.ReadFile(onDisk)
	var manifest PackageManifest
	if err := json.Unmarshal(b, &manifest); err != nil {
		t.Fatal(err)
	}
	manifest.DisplayName = "new-from-disk"
	nb, _ := json.Marshal(&manifest)
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
			{ID: "w", Kind: "Win32WindowTarget", Config: map[string]any{"Title": "异环"}},
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
