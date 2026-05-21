package library

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"yhbox/internal/services/container"
	"yhbox/internal/services/container/dependency"
)

func TestImportToContainer_Clean(t *testing.T) {
	dir := t.TempDir()
	pkg := filepath.Join(dir, "subgraphs", "foo")
	os.MkdirAll(filepath.Join(pkg, "templates"), 0o755)
	os.MkdirAll(filepath.Join(pkg, "embedded"), 0o755)
	rootSg := container.Subgraph{ID: "foo", Label: "Foo"}
	rb, _ := json.Marshal(rootSg)
	os.WriteFile(filepath.Join(pkg, "subgraph.json"), rb, 0o644)
	calleeSg := container.Subgraph{ID: "bar", Label: "Bar"}
	cb, _ := json.Marshal(calleeSg)
	os.WriteFile(filepath.Join(pkg, "embedded", "bar.json"), cb, 0o644)
	os.WriteFile(filepath.Join(pkg, "templates", "ns.x.png"), []byte{1}, 0o644)
	os.WriteFile(filepath.Join(pkg, "templates", "ns.x.json"), []byte("{}"), 0o644)

	s, _ := NewStore(dir)
	containerRoot := t.TempDir()
	result, err := s.ImportToContainer("foo", containerRoot, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Conflicts) != 0 {
		t.Errorf("unexpected conflicts: %v", result.Conflicts)
	}
	if _, err := os.Stat(filepath.Join(containerRoot, "subgraphs", "foo.json")); err != nil {
		t.Errorf("foo.json not imported: %v", err)
	}
	if _, err := os.Stat(filepath.Join(containerRoot, "subgraphs", "bar.json")); err != nil {
		t.Errorf("bar.json (embedded) not imported: %v", err)
	}
	if _, err := os.Stat(filepath.Join(containerRoot, "templates", "ns.x.png")); err != nil {
		t.Errorf("template not imported: %v", err)
	}
}

func TestImportToContainer_SkipAll(t *testing.T) {
	dir := t.TempDir()
	pkg := filepath.Join(dir, "subgraphs", "foo")
	os.MkdirAll(pkg, 0o755)
	rootSg := container.Subgraph{ID: "foo", Label: "Foo"}
	rb, _ := json.Marshal(rootSg)
	os.WriteFile(filepath.Join(pkg, "subgraph.json"), rb, 0o644)

	s, _ := NewStore(dir)
	containerRoot := t.TempDir()

	// pre-populate conflict
	sgDir := filepath.Join(containerRoot, "subgraphs")
	os.MkdirAll(sgDir, 0o755)
	os.WriteFile(filepath.Join(sgDir, "foo.json"), []byte(`{"id":"foo"}`), 0o644)

	result, err := s.ImportToContainer("foo", containerRoot, "skip_all")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Conflicts) != 1 || result.Conflicts[0].Key != "foo" {
		t.Errorf("expected conflict for foo, got %v", result.Conflicts)
	}
	// foo.json should still be the old content (skipped)
	b, _ := os.ReadFile(filepath.Join(sgDir, "foo.json"))
	if string(b) != `{"id":"foo"}` {
		t.Errorf("foo.json was overwritten despite skip_all, content: %s", b)
	}
}

func TestImportToContainer_Cancel(t *testing.T) {
	dir := t.TempDir()
	pkg := filepath.Join(dir, "subgraphs", "foo")
	os.MkdirAll(pkg, 0o755)
	rootSg := container.Subgraph{ID: "foo", Label: "Foo"}
	rb, _ := json.Marshal(rootSg)
	os.WriteFile(filepath.Join(pkg, "subgraph.json"), rb, 0o644)

	s, _ := NewStore(dir)
	containerRoot := t.TempDir()
	sgDir := filepath.Join(containerRoot, "subgraphs")
	os.MkdirAll(sgDir, 0o755)
	os.WriteFile(filepath.Join(sgDir, "foo.json"), []byte(`{"id":"foo"}`), 0o644)

	result, err := s.ImportToContainer("foo", containerRoot, "cancel")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Conflicts) != 1 {
		t.Errorf("expected 1 conflict, got %v", result.Conflicts)
	}
	if len(result.Imported) != 0 {
		t.Errorf("cancel should import nothing, got %v", result.Imported)
	}
}

func TestExportFromContainer(t *testing.T) {
	dir := t.TempDir()
	srcRoot := t.TempDir()
	os.MkdirAll(filepath.Join(srcRoot, "subgraphs"), 0o755)
	os.MkdirAll(filepath.Join(srcRoot, "templates"), 0o755)

	rootSg := container.Subgraph{ID: "root"}
	rb, _ := json.Marshal(rootSg)
	os.WriteFile(filepath.Join(srcRoot, "subgraphs", "root.json"), rb, 0o644)

	os.WriteFile(filepath.Join(srcRoot, "templates", "ns.t.png"), []byte{1}, 0o644)
	os.WriteFile(filepath.Join(srcRoot, "templates", "ns.t.json"), []byte("{}"), 0o644)

	stubScanner := func(id string, get func(string) ([]dependency.NodeInfo, error)) ([]dependency.Dependency, error) {
		return []dependency.Dependency{
			{Kind: dependency.KindSubgraph, Key: "root"},
			{Kind: dependency.KindTemplate, Key: "ns.t"},
		}, nil
	}
	s, _ := NewStore(dir)
	if err := s.ExportFromContainer(srcRoot, "root", stubScanner, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "subgraphs", "root", "subgraph.json")); err != nil {
		t.Errorf("root subgraph not exported: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "subgraphs", "root", "templates", "ns.t.png")); err != nil {
		t.Errorf("template not exported: %v", err)
	}
}

func TestExportFromContainer_NoOverwrite(t *testing.T) {
	dir := t.TempDir()
	srcRoot := t.TempDir()
	os.MkdirAll(filepath.Join(srcRoot, "subgraphs"), 0o755)
	rootSg := container.Subgraph{ID: "root"}
	rb, _ := json.Marshal(rootSg)
	os.WriteFile(filepath.Join(srcRoot, "subgraphs", "root.json"), rb, 0o644)

	stubScanner := func(id string, get func(string) ([]dependency.NodeInfo, error)) ([]dependency.Dependency, error) {
		return []dependency.Dependency{{Kind: dependency.KindSubgraph, Key: "root"}}, nil
	}
	s, _ := NewStore(dir)

	// first export succeeds
	if err := s.ExportFromContainer(srcRoot, "root", stubScanner, false); err != nil {
		t.Fatal(err)
	}
	// second export without overwrite should fail
	if err := s.ExportFromContainer(srcRoot, "root", stubScanner, false); err == nil {
		t.Error("expected error on second export without overwrite, got nil")
	}
	// with overwrite it should succeed
	if err := s.ExportFromContainer(srcRoot, "root", stubScanner, true); err != nil {
		t.Errorf("overwrite export failed: %v", err)
	}
}
