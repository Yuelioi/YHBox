package container

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/node"
	"github.com/yottaapp/yotta/internal/nodes/control"
	"github.com/yottaapp/yotta/internal/services/asset"

	_ "github.com/yottaapp/yotta/internal/nodes/all"
)

type isolatedStoreNode struct{}

func (isolatedStoreNode) Spec() node.Spec {
	return node.Spec{
		Kind:    "IsolatedStoreNode",
		Inputs:  []node.InputSpec{{Name: "In", Type: node.TypeExec}},
		Outputs: []node.OutputSpec{{Name: "Done", Type: node.TypeExec}},
	}
}

func (isolatedStoreNode) Run(node.Ctx, node.Inputs) (node.Outputs, error) { return nil, nil }
func (isolatedStoreNode) Dependencies(node.Inputs) []node.Dependency {
	return []node.Dependency{{Kind: "template", Key: "custom-template"}}
}

func TestContainerStoreUsesExplicitRegistryForValidationAndDependencies(t *testing.T) {
	registry := node.NewRegistry()
	registry.Register(&control.Start{})
	registry.Register(isolatedStoreNode{})
	root := t.TempDir()
	store, err := NewStoreWithRegistry(root, registry.Snapshot())
	if err != nil {
		t.Fatal(err)
	}
	c := &Container{SchemaVersion: 1, ID: "custom-registry", Name: "custom", Graph: Graph{
		Nodes: []GraphNode{{ID: "start", Kind: "Start"}, {ID: "custom", Kind: "IsolatedStoreNode"}},
		Edges: []GraphEdge{{From: "start.Done", To: "custom.In"}},
	}}
	if err := store.Save(c); err != nil {
		t.Fatalf("Save with explicit registry: %v", err)
	}
	lock, err := readJSONFile[YottaLock](filepath.Join(root, c.ID, lockFile))
	if err != nil {
		t.Fatal(err)
	}
	if len(lock.Dependencies.Templates) != 1 || lock.Dependencies.Templates[0] != "custom-template" {
		t.Fatalf("custom dependency was not scanned: %+v", lock.Dependencies)
	}
}

func TestContainerStoreSnapshotsRegistryAtConstruction(t *testing.T) {
	registry := node.NewRegistry()
	registry.Register(&control.Start{})
	store, err := NewStoreWithRegistry(t.TempDir(), registry)
	if err != nil {
		t.Fatal(err)
	}
	registry.Register(isolatedStoreNode{})
	if _, ok := store.RegistrySnapshot().Get("IsolatedStoreNode"); ok {
		t.Fatal("store observed a node registered after construction")
	}
}

func transactionTestContainer(id, name string) *Container {
	return &Container{SchemaVersion: 1, ID: id, Name: name, Graph: Graph{Nodes: []GraphNode{
		{ID: "start", Kind: "Start"},
	}}}
}

func TestContainerStore_SaveUsesDeterministicLockLastOrder(t *testing.T) {
	s, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var order []string
	s.writeFileAtomic = func(path string, data []byte) error {
		order = append(order, filepath.Base(path))
		return writeContainerFileAtomic(path, data)
	}
	if err := s.Save(transactionTestContainer("ordered", "ordered")); err != nil {
		t.Fatal(err)
	}
	want := []string{packageFile, graphFile, installationFile, lockFile}
	if fmt.Sprint(order) != fmt.Sprint(want) {
		t.Fatalf("write order = %v, want %v", order, want)
	}
}

func TestContainerStore_SaveFailureRestoresLastCommittedGeneration(t *testing.T) {
	for failAt := 1; failAt <= 4; failAt++ {
		t.Run(fmt.Sprintf("write-%d", failAt), func(t *testing.T) {
			root := t.TempDir()
			s, err := NewStore(root)
			if err != nil {
				t.Fatal(err)
			}
			if err := s.Save(transactionTestContainer("txn", "old")); err != nil {
				t.Fatal(err)
			}
			calls := 0
			wantErr := errors.New("injected write failure")
			s.writeFileAtomic = func(path string, data []byte) error {
				calls++
				if calls == failAt {
					return wantErr
				}
				return writeContainerFileAtomic(path, data)
			}
			if err := s.Save(transactionTestContainer("txn", "new")); !errors.Is(err, wantErr) {
				t.Fatalf("Save error = %v", err)
			}
			if got, _ := s.Get("txn"); got.Name != "old" {
				t.Fatalf("memory cache published failed generation: %+v", got)
			}
			reloaded, err := NewStore(root)
			if err != nil {
				t.Fatal(err)
			}
			got, ok := reloaded.Get("txn")
			if !ok || got.Name != "old" || got.Status == StatusIncompatible {
				t.Fatalf("disk generation after rollback = %+v, ok=%v", got, ok)
			}
		})
	}
}

func TestContainerStore_RollbackFailureLeavesDiskExplicitlyIncompatible(t *testing.T) {
	root := t.TempDir()
	s, _ := NewStore(root)
	if err := s.Save(transactionTestContainer("rollback-fail", "old")); err != nil {
		t.Fatal(err)
	}
	calls := 0
	primaryErr := errors.New("installation write failed")
	rollbackErr := errors.New("graph rollback failed")
	s.writeFileAtomic = func(path string, data []byte) error {
		calls++
		switch calls {
		case 3:
			return primaryErr
		case 4:
			return rollbackErr
		default:
			return writeContainerFileAtomic(path, data)
		}
	}
	err := s.Save(transactionTestContainer("rollback-fail", "new"))
	if !errors.Is(err, primaryErr) || !errors.Is(err, rollbackErr) {
		t.Fatalf("Save error = %v", err)
	}
	if got, _ := s.Get("rollback-fail"); got.Status != StatusIncompatible || got.IncompatibleReason == "" {
		t.Fatalf("rollback failure was not isolated in cache: %+v", got)
	}
	reloaded, loadErr := NewStore(root)
	if loadErr != nil {
		t.Fatal(loadErr)
	}
	got, ok := reloaded.Get("rollback-fail")
	if !ok || got.Status != StatusIncompatible {
		t.Fatalf("mixed disk generation was accepted: %+v, ok=%v", got, ok)
	}
}

func TestContainerStore_LoadRejectsMixedOrUncommittedGeneration(t *testing.T) {
	mutations := map[string]func(t *testing.T, dir string){
		"package-schema": func(t *testing.T, dir string) {
			path := filepath.Join(dir, packageFile)
			v, _ := readJSONFile[PackageManifest](path)
			v.SchemaVersion = PackageSchemaVersion + 1
			if err := writeJSONAtomic(path, v); err != nil {
				t.Fatal(err)
			}
			updateTestLockHash(t, dir, func(lock *YottaLock) { lock.ManifestHash, _ = hashJSON(v) })
		},
		"package-kind": func(t *testing.T, dir string) {
			path := filepath.Join(dir, packageFile)
			v, _ := readJSONFile[PackageManifest](path)
			v.Kind = "yotta.other"
			if err := writeJSONAtomic(path, v); err != nil {
				t.Fatal(err)
			}
			updateTestLockHash(t, dir, func(lock *YottaLock) { lock.ManifestHash, _ = hashJSON(v) })
		},
		"entry-graph": func(t *testing.T, dir string) {
			path := filepath.Join(dir, packageFile)
			v, _ := readJSONFile[PackageManifest](path)
			v.Yotta.EntryGraph = "other.json"
			if err := writeJSONAtomic(path, v); err != nil {
				t.Fatal(err)
			}
			updateTestLockHash(t, dir, func(lock *YottaLock) { lock.ManifestHash, _ = hashJSON(v) })
		},
		"installation-schema": func(t *testing.T, dir string) {
			path := filepath.Join(dir, installationFile)
			v, _ := readJSONFile[Installation](path)
			v.SchemaVersion = InstallationSchemaVersion + 1
			if err := writeJSONAtomic(path, v); err != nil {
				t.Fatal(err)
			}
			updateTestLockHash(t, dir, func(lock *YottaLock) { lock.InstallationHash, _ = hashJSON(v) })
		},
		"closure": func(t *testing.T, dir string) {
			updateTestLockHash(t, dir, func(lock *YottaLock) {
				lock.Dependencies.Templates = []string{"tampered"}
			})
		},
		"package": func(t *testing.T, dir string) {
			path := filepath.Join(dir, packageFile)
			v, _ := readJSONFile[PackageManifest](path)
			v.DisplayName = "tampered"
			if err := writeJSONAtomic(path, v); err != nil {
				t.Fatal(err)
			}
		},
		"graph": func(t *testing.T, dir string) {
			path := filepath.Join(dir, graphFile)
			v, _ := readJSONFile[Graph](path)
			v.ID = "tampered"
			if err := writeJSONAtomic(path, v); err != nil {
				t.Fatal(err)
			}
		},
		"installation": func(t *testing.T, dir string) {
			path := filepath.Join(dir, installationFile)
			v, _ := readJSONFile[Installation](path)
			v.Display.Alias = "tampered"
			if err := writeJSONAtomic(path, v); err != nil {
				t.Fatal(err)
			}
		},
		"installation-identity": func(t *testing.T, dir string) {
			installationPath := filepath.Join(dir, installationFile)
			installation, _ := readJSONFile[Installation](installationPath)
			installation.PackageID = "pkg_other"
			if err := writeJSONAtomic(installationPath, installation); err != nil {
				t.Fatal(err)
			}
			lockPath := filepath.Join(dir, lockFile)
			lock, _ := readJSONFile[YottaLock](lockPath)
			lock.InstallationHash, _ = hashJSON(installation)
			if err := writeJSONAtomic(lockPath, lock); err != nil {
				t.Fatal(err)
			}
		},
		"missing-lock": func(t *testing.T, dir string) {
			if err := os.Remove(filepath.Join(dir, lockFile)); err != nil {
				t.Fatal(err)
			}
		},
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			s, _ := NewStore(root)
			if err := s.Save(transactionTestContainer("mixed", "original")); err != nil {
				t.Fatal(err)
			}
			mutate(t, filepath.Join(root, "mixed"))
			reloaded, err := NewStore(root)
			if err != nil {
				t.Fatal(err)
			}
			got, ok := reloaded.Get("mixed")
			if !ok || got.Status != StatusIncompatible || got.IncompatibleReason == "" {
				t.Fatalf("mixed generation was accepted: %+v, ok=%v", got, ok)
			}
		})
	}
}

func updateTestLockHash(t *testing.T, dir string, mutate func(*YottaLock)) {
	t.Helper()
	path := filepath.Join(dir, lockFile)
	lock, err := readJSONFile[YottaLock](path)
	if err != nil {
		t.Fatal(err)
	}
	mutate(&lock)
	if err := writeJSONAtomic(path, lock); err != nil {
		t.Fatal(err)
	}
}

func TestContainerStore_LoadUpgradesLegacyV1Lock(t *testing.T) {
	root := t.TempDir()
	s, _ := NewStore(root)
	if err := s.Save(transactionTestContainer("legacy-lock", "legacy")); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "legacy-lock", lockFile)
	lock, _ := readJSONFile[YottaLock](path)
	lock.SchemaVersion = 1
	lock.InstallationHash = ""
	if err := writeJSONAtomic(path, lock); err != nil {
		t.Fatal(err)
	}
	reloaded, err := NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := reloaded.Get("legacy-lock")
	if !ok || got.Status == StatusIncompatible {
		t.Fatalf("legacy lock load = %+v, ok=%v", got, ok)
	}
	upgraded, err := readJSONFile[YottaLock](path)
	if err != nil || upgraded.SchemaVersion != LockSchemaVersion || upgraded.InstallationHash == "" {
		t.Fatalf("upgraded lock = %+v, err=%v", upgraded, err)
	}
}

func TestContainerStore_LegacyV1MigrationFailureDoesNotBlockLoad(t *testing.T) {
	root := t.TempDir()
	s, _ := NewStore(root)
	if err := s.Save(transactionTestContainer("legacy-readonly", "legacy")); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "legacy-readonly", lockFile)
	lock, _ := readJSONFile[YottaLock](path)
	lock.SchemaVersion = 1
	lock.InstallationHash = ""
	if err := writeJSONAtomic(path, lock); err != nil {
		t.Fatal(err)
	}
	s.writeFileAtomic = func(string, []byte) error { return errors.New("read-only directory") }
	got, err := s.loadOne("legacy-readonly")
	if err != nil || got.Status == StatusIncompatible {
		t.Fatalf("valid v1 lock should remain readable: got=%+v err=%v", got, err)
	}
	unchanged, err := readJSONFile[YottaLock](path)
	if err != nil || unchanged.SchemaVersion != 1 {
		t.Fatalf("failed best-effort migration changed lock: %+v err=%v", unchanged, err)
	}
}

func TestContainerStore_GetAndListReturnDeepSnapshots(t *testing.T) {
	s, _ := NewStore(t.TempDir())
	c := transactionTestContainer("snapshot", "snapshot")
	c.Graph.Nodes[0].Config = map[string]any{"literal": map[string]any{"Value": "original"}}
	if err := s.Save(c); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Get("snapshot")
	got.Name = "mutated"
	got.Graph.Nodes[0].Config["literal"].(map[string]any)["Value"] = "mutated"
	list := s.List()
	list[0].Graph.Nodes[0].Config["literal"].(map[string]any)["Value"] = "list-mutated"
	again, _ := s.Get("snapshot")
	if again.Name != "snapshot" || PinString(&again.Graph.Nodes[0], "Value") != "original" {
		t.Fatalf("store cache escaped: %+v", again)
	}
}

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

func TestContainerStore_AggregatedBindingSlotsAreNotUnknownLiteralPins(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	c := &Container{
		SchemaVersion: 1,
		ID:            "binding-validation",
		Name:          "binding validation",
		Graph: Graph{Nodes: []GraphNode{
			{ID: "start", Kind: "Start"},
			{
				ID:   "2c0266df-4595-4118-9a7b-2549e4e7eeb3",
				Kind: "Win32WindowTarget",
				Config: map[string]any{
					"Title": "Blue Archive",
				},
			},
			{
				ID:   "android-target",
				Kind: "AndroidTarget",
				Config: map[string]any{
					"Serial": "emulator-5554",
				},
			},
		}},
	}
	if err := s.Save(c); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, ok := s.Get(c.ID)
	if !ok {
		t.Fatal("Get failed")
	}
	for _, nodeID := range []string{"2c0266df-4595-4118-9a7b-2549e4e7eeb3", "android-target"} {
		if errs := validateUnknownLiteralPins(&got, nil); hasCodeForNode(errs, CodeUnknownLiteralPin, nodeID) {
			t.Fatalf("package binding metadata must not be reported as an unknown pin for %s: %+v", nodeID, errs)
		}
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
	assetRoot := filepath.Join(dir, "assets")
	assetBlobs, err := blob.Open(filepath.Join(assetRoot, "blobs"), blob.Limits{MaxBlobBytes: 1 << 20, MaxTotalBytes: 8 << 20})
	if err != nil {
		t.Fatal(err)
	}
	assetStore, _ := asset.NewStore(assetRoot, assetBlobs)
	s.SetAssetStore(assetStore)

	templateBlob, err := assetStore.CommitRecordBlob(context.Background(), "image/png", bytes.NewReader([]byte("template-png")), func(ref blob.BlobRef) asset.AssetRecord {
		return asset.AssetRecord{
			GUID:   "tpl-1",
			Kind:   asset.KindTemplate,
			Name:   "Template",
			Origin: asset.Origin{Kind: "user"},
			Variants: []asset.Variant{{
				Resolution: [2]int{1280, 720},
				BBox:       [4]int{1, 2, 3, 4},
				Blob:       ref,
			}},
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Save(&Container{
		SchemaVersion: 1, ID: "zip-assets", Name: "zip assets",
		Graph: Graph{Nodes: []GraphNode{
			{ID: "start", Kind: "Start"},
			{ID: "target", Kind: "Win32WindowTarget", Config: map[string]any{"Title": "Game"}},
			{ID: "check", Kind: "CheckTemplate", Config: map[string]any{"literal": map[string]any{"Templates": []any{"tpl-1"}}}},
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
		"assets/blobs/" + blobObjectName(templateBlob),
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

	// 未同步更新 lock 的外部改盘属于未完成提交；Reload 必须暴露 incompatible，
	// 不能把混合代当成可运行容器。
	got, err := s.Reload("r1")
	if err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if got.Name != "new-from-disk" {
		t.Errorf("Reload 返回名字 = %q, want new-from-disk", got.Name)
	}
	if got.Status != StatusIncompatible || got.IncompatibleReason == "" {
		t.Fatalf("Reload should reject stale lock: %+v", got)
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
