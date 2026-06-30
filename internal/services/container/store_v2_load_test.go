package container

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestStore_LoadGracefulVersionMismatch(t *testing.T) {
	root := t.TempDir()
	cid := "future-container"
	if err := os.MkdirAll(filepath.Join(root, cid), 0o755); err != nil {
		t.Fatal(err)
	}
	writePackageStoreFixture(t, root, cid, "from-the-future", 999)

	st, err := NewStore(root)
	if err != nil {
		t.Fatalf("store should not error on version mismatch, got %v", err)
	}
	list := st.List()
	if len(list) != 1 {
		t.Fatalf("expected 1 container loaded (incompatible), got %d", len(list))
	}
	if list[0].Status != StatusIncompatible {
		t.Errorf("expected Status=Incompatible, got %q", list[0].Status)
	}
	if list[0].IncompatibleReason == "" {
		t.Errorf("expected IncompatibleReason populated")
	}
}

func TestStore_LoadOKMatchingVersion(t *testing.T) {
	root := t.TempDir()
	cid := "normal"
	os.MkdirAll(filepath.Join(root, cid), 0o755)
	writePackageStoreFixture(t, root, cid, "ok", GraphSchemaVersion)

	st, err := NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	list := st.List()
	if len(list) != 1 || list[0].Status == StatusIncompatible {
		t.Errorf("expected ok container, got %+v", list)
	}
}

func writePackageStoreFixture(t *testing.T, root, cid, displayName string, graphSchemaVersion int) {
	t.Helper()
	manifest := PackageManifest{
		SchemaVersion: PackageSchemaVersion,
		Kind:          PackageKindContainer,
		Name:          "@local/" + cid,
		DisplayName:   displayName,
		Version:       "0.1.0",
		Author:        PackagePerson{},
		Publisher:     PackagePublisher{ID: "local"},
		Yotta: PackageYotta{
			PackageID:  "pkg_" + cid,
			EntryGraph: "graph.json",
			Publication: Publication{
				State:      PublicationDraft,
				Visibility: VisibilityPrivate,
			},
		},
	}
	graph := Graph{
		ID:            "g",
		SchemaVersion: graphSchemaVersion,
		Nodes:         []GraphNode{},
		Edges:         []GraphEdge{},
	}
	installation := Installation{
		SchemaVersion:    InstallationSchemaVersion,
		InstanceID:       cid,
		PackageID:        manifest.Yotta.PackageID,
		PackageName:      manifest.Name,
		InstalledVersion: manifest.Version,
		RuntimeOverrides: RuntimeOverrides{},
	}
	writeJSONFixture(t, filepath.Join(root, cid, "package.json"), manifest)
	writeJSONFixture(t, filepath.Join(root, cid, "graph.json"), graph)
	writeJSONFixture(t, filepath.Join(root, cid, "installation.json"), installation)
}

func writeJSONFixture(t *testing.T, path string, v any) {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
}
