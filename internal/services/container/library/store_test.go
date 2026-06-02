package library

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"yotta/internal/services/container"
)

func TestStore_ListGetPackage(t *testing.T) {
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "subgraphs", "test_pkg")
	os.MkdirAll(filepath.Join(pkgDir, "embedded"), 0o755)
	os.MkdirAll(filepath.Join(pkgDir, "templates"), 0o755)

	root := container.Subgraph{ID: "test_pkg", Label: "Test"}
	rootBytes, _ := json.Marshal(root)
	os.WriteFile(filepath.Join(pkgDir, "subgraph.json"), rootBytes, 0o644)

	callee := container.Subgraph{ID: "callee_1", Label: "Callee"}
	calleeBytes, _ := json.Marshal(callee)
	os.WriteFile(filepath.Join(pkgDir, "embedded", "callee_1.json"), calleeBytes, 0o644)

	os.WriteFile(filepath.Join(pkgDir, "templates", "ns.x.png"), []byte{1}, 0o644)
	os.WriteFile(filepath.Join(pkgDir, "templates", "ns.x.json"), []byte("{}"), 0o644)

	s, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	list, err := s.ListSubgraphs()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0] != "test_pkg" {
		t.Fatalf("ListSubgraphs = %v, want [test_pkg]", list)
	}

	pkg, err := s.GetSubgraphPackage("test_pkg")
	if err != nil {
		t.Fatal(err)
	}
	if pkg.Root.ID != "test_pkg" {
		t.Errorf("Root.ID = %q", pkg.Root.ID)
	}
	if _, ok := pkg.Embedded["callee_1"]; !ok {
		t.Error("missing embedded callee_1")
	}
	if len(pkg.Templates) != 1 || pkg.Templates[0] != "ns.x" {
		t.Errorf("Templates = %v", pkg.Templates)
	}
}

func TestStore_DeleteSubgraphPackage(t *testing.T) {
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "subgraphs", "pkg_to_delete")
	os.MkdirAll(pkgDir, 0o755)
	sg := container.Subgraph{ID: "pkg_to_delete", Label: "Delete Me"}
	b, _ := json.Marshal(sg)
	os.WriteFile(filepath.Join(pkgDir, "subgraph.json"), b, 0o644)

	s, _ := NewStore(dir)
	list, _ := s.ListSubgraphs()
	if len(list) != 1 {
		t.Fatalf("expected 1 package before delete, got %d", len(list))
	}

	if err := s.DeleteSubgraphPackage("pkg_to_delete"); err != nil {
		t.Fatal(err)
	}
	list, _ = s.ListSubgraphs()
	if len(list) != 0 {
		t.Errorf("expected 0 packages after delete, got %d", len(list))
	}
	if _, err := os.Stat(pkgDir); err == nil {
		t.Error("package dir still exists after delete")
	}
}
