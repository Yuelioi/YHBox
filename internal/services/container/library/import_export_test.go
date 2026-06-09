package library

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	// blank import 真节点 — dependency scanner 走 nodepkg.Get(kind).Dependencies.
	_ "yotta/internal/nodes/detect" // CheckTemplate / ClickTemplate / WaitTemplate
	_ "yotta/internal/nodes/io"     // PlayClip
	_ "yotta/internal/nodes/system" // Subgraph / CollapsedNode

	"yotta/internal/services/asset"
	"yotta/internal/services/container"
)

// newAssetStore 建一个临时 asset.Store.
func newAssetStore(t *testing.T) *asset.Store {
	t.Helper()
	s, err := asset.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// writeBundlePkg 手工拼一个 library bundle 目录 (root subgraph + 一条 template 资产 + blob).
// 返回 bundleID. blob 内容由 caller 给, sha 自算.
func writeBundlePkg(t *testing.T, libRoot, bundleID string, rec asset.AssetRecord, blobs map[string][]byte) {
	t.Helper()
	pkg := filepath.Join(libRoot, "subgraphs", bundleID)
	os.MkdirAll(filepath.Join(pkg, "assets"), 0o755)
	os.MkdirAll(filepath.Join(pkg, "blobs"), 0o755)
	os.MkdirAll(filepath.Join(pkg, "embedded"), 0o755)
	rootSg := container.Subgraph{ID: bundleID, Label: "Root"}
	rb, _ := json.Marshal(rootSg)
	os.WriteFile(filepath.Join(pkg, "subgraph.json"), rb, 0o644)
	ab, _ := json.Marshal(rec)
	os.WriteFile(filepath.Join(pkg, "assets", rec.GUID+".json"), ab, 0o644)
	for sha, data := range blobs {
		os.WriteFile(filepath.Join(pkg, "blobs", sha), data, 0o644)
	}
}

func TestImportToContainer_CleanAndIdempotent(t *testing.T) {
	libRoot := t.TempDir()
	as := newAssetStore(t)
	// blob sha 必须真实 (importAsset 校验 declared==computed). 用 store.Blobs() 借算一次, 再删.
	sha, _ := as.Blobs().Put([]byte("pixels-1080"))
	rec := asset.AssetRecord{
		GUID: "g1", Kind: asset.KindTemplate, Name: "登录",
		Variants: []asset.Variant{{Resolution: [2]int{1920, 1080}, Blob: sha}},
	}
	writeBundlePkg(t, libRoot, "foo", rec, map[string][]byte{sha: []byte("pixels-1080")})

	// 用一个全新 asset store 做 import 目标 (确保 blob 从 bundle 落池).
	target := newAssetStore(t)
	s, _ := NewStore(libRoot)
	containerRoot := t.TempDir()

	r, err := s.ImportToContainer("foo", containerRoot, target, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(containerRoot, "subgraphs", "foo.json")); err != nil {
		t.Errorf("foo.json not imported: %v", err)
	}
	if rg, ok := target.Get("g1"); !ok || len(rg.Variants) != 1 {
		t.Fatalf("asset g1 not imported correctly: %+v", rg)
	}
	if !target.Blobs().Has(sha) {
		t.Error("blob not put to target pool")
	}
	_ = r

	// 幂等重放: 再导一次, 资产仍 1 变体, blob 不重复.
	if _, err := s.ImportToContainer("foo", containerRoot, target, nil); err != nil {
		t.Fatal(err)
	}
	if rg, _ := target.Get("g1"); len(rg.Variants) != 1 {
		t.Fatalf("re-import should be idempotent, got %d variants", len(rg.Variants))
	}
}

func TestImportToContainer_AdditiveVariantMerge(t *testing.T) {
	libRoot := t.TempDir()
	// 目标本地已有 g1@1080.
	target := newAssetStore(t)
	localSha, _ := target.Blobs().Put([]byte("local-1080"))
	target.PutRecord(asset.AssetRecord{GUID: "g1", Kind: asset.KindTemplate, Name: "本地名"})
	target.PutVariant("g1", [2]int{1920, 1080}, localSha, [4]int{0, 0, 1, 1}, nil)

	// bundle 含同 GUID 的 720 新档 + 一个 1080 (不同 blob, 应被忽略保留本地).
	tmp := newAssetStore(t)
	sha720, _ := tmp.Blobs().Put([]byte("bundle-720"))
	sha1080b, _ := tmp.Blobs().Put([]byte("bundle-1080-different"))
	rec := asset.AssetRecord{
		GUID: "g1", Kind: asset.KindTemplate, Name: "bundle名",
		Variants: []asset.Variant{
			{Resolution: [2]int{1280, 720}, Blob: sha720},
			{Resolution: [2]int{1920, 1080}, Blob: sha1080b},
		},
	}
	writeBundlePkg(t, libRoot, "foo", rec, map[string][]byte{
		sha720:   []byte("bundle-720"),
		sha1080b: []byte("bundle-1080-different"),
	})

	s, _ := NewStore(libRoot)
	if _, err := s.ImportToContainer("foo", t.TempDir(), target, nil); err != nil {
		t.Fatal(err)
	}
	rg, _ := target.Get("g1")
	if len(rg.Variants) != 2 {
		t.Fatalf("expected additive merge to 2 variants, got %d", len(rg.Variants))
	}
	// 本地 1080 保留 (localSha), name 保留本地.
	if rg.Name != "本地名" {
		t.Errorf("name should stay local, got %q", rg.Name)
	}
	for _, v := range rg.Variants {
		if v.Resolution == [2]int{1920, 1080} && v.Blob != localSha {
			t.Errorf("local 1080 variant overwritten: blob=%s want %s", v.Blob, localSha)
		}
		if v.Resolution == [2]int{1280, 720} && v.Blob != sha720 {
			t.Errorf("720 variant blob wrong: %s", v.Blob)
		}
	}
}

func TestExportSubgraph_WithAsset(t *testing.T) {
	libRoot := t.TempDir()
	srcRoot := t.TempDir()
	os.MkdirAll(filepath.Join(srcRoot, "subgraphs"), 0o755)

	// 子图含一个引用 template GUID 的 CheckTemplate 节点.
	sg := container.Subgraph{
		ID: "root",
		Graph: container.Graph{Nodes: []container.GraphNode{
			{ID: "n1", Kind: "CheckTemplate", Config: map[string]any{"literal": map[string]any{"Templates": []any{"g1"}}}},
		}},
	}
	rb, _ := json.Marshal(sg)
	os.WriteFile(filepath.Join(srcRoot, "subgraphs", "root.json"), rb, 0o644)

	as := newAssetStore(t)
	sha, _ := as.Blobs().Put([]byte("pix"))
	as.PutRecord(asset.AssetRecord{GUID: "g1", Kind: asset.KindTemplate, Name: "t"})
	as.PutVariant("g1", [2]int{1920, 1080}, sha, [4]int{0, 0, 1, 1}, nil)

	s, _ := NewStore(libRoot)
	if err := s.ExportSubgraph(srcRoot, "root", as, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(libRoot, "subgraphs", "root", "subgraph.json")); err != nil {
		t.Errorf("root subgraph not exported: %v", err)
	}
	if _, err := os.Stat(filepath.Join(libRoot, "subgraphs", "root", "assets", "g1.json")); err != nil {
		t.Errorf("asset record not exported: %v", err)
	}
	if _, err := os.Stat(filepath.Join(libRoot, "subgraphs", "root", "blobs", sha)); err != nil {
		t.Errorf("blob not exported: %v", err)
	}
}

func TestExportSubgraph_MissingAssetRecord(t *testing.T) {
	libRoot := t.TempDir()
	srcRoot := t.TempDir()
	os.MkdirAll(filepath.Join(srcRoot, "subgraphs"), 0o755)
	sg := container.Subgraph{
		ID: "root",
		Graph: container.Graph{Nodes: []container.GraphNode{
			{ID: "n1", Kind: "CheckTemplate", Config: map[string]any{"literal": map[string]any{"Templates": []any{"ghost"}}}},
		}},
	}
	rb, _ := json.Marshal(sg)
	os.WriteFile(filepath.Join(srcRoot, "subgraphs", "root.json"), rb, 0o644)

	as := newAssetStore(t) // 没有 "ghost" 记录
	s, _ := NewStore(libRoot)
	if err := s.ExportSubgraph(srcRoot, "root", as, false); err == nil {
		t.Error("expected error when graph references missing asset GUID")
	}
	// 不写半包.
	if _, err := os.Stat(filepath.Join(libRoot, "subgraphs", "root")); err == nil {
		t.Error("partial bundle should not be created on missing-record error")
	}
}

func TestExportSubgraph_NoOverwrite(t *testing.T) {
	libRoot := t.TempDir()
	srcRoot := t.TempDir()
	os.MkdirAll(filepath.Join(srcRoot, "subgraphs"), 0o755)
	sg := container.Subgraph{ID: "root"}
	rb, _ := json.Marshal(sg)
	os.WriteFile(filepath.Join(srcRoot, "subgraphs", "root.json"), rb, 0o644)
	as := newAssetStore(t)
	s, _ := NewStore(libRoot)

	if err := s.ExportSubgraph(srcRoot, "root", as, false); err != nil {
		t.Fatal(err)
	}
	if err := s.ExportSubgraph(srcRoot, "root", as, false); err == nil {
		t.Error("expected error on second export without overwrite")
	}
	if err := s.ExportSubgraph(srcRoot, "root", as, true); err != nil {
		t.Errorf("overwrite export failed: %v", err)
	}
}
