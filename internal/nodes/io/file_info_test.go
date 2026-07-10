package io

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/yottaapp/yotta/internal/node"
)

func runFileInfo(t *testing.T, cfg map[string]any) node.RunResult {
	t.Helper()
	node.ResetRegistryForTest()
	node.Register(&FileInfo{})
	rn, _ := node.Get("FileInfo")
	return node.RunNode(context.Background(), rn, nil, cfg, nil, node.StubServices(), false)
}

func TestFileInfo_PathInput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sample.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	res := runFileInfo(t, map[string]any{"Path": path})
	if res.Error != nil || res.ExitName != "Done" {
		t.Fatalf("exit=%q err=%v", res.ExitName, res.Error)
	}
	file, ok := res.OutputData["File"].(node.File)
	if !ok {
		t.Fatalf("File = %#v, want node.File", res.OutputData["File"])
	}
	if file.Path != path || file.Name != "sample.txt" || file.Ext != ".txt" || file.Size != 5 {
		t.Fatalf("File metadata = %+v", file)
	}
	if got := res.OutputData["Path"]; got != path {
		t.Fatalf("Path = %#v, want %q", got, path)
	}
	if got := res.OutputData["Name"]; got != "sample.txt" {
		t.Fatalf("Name = %#v", got)
	}
	if got := res.OutputData["Ext"]; got != ".txt" {
		t.Fatalf("Ext = %#v", got)
	}
	if got := res.OutputData["Size"]; got != 5 {
		t.Fatalf("Size = %#v", got)
	}
	if got := res.OutputData["IsDir"]; got != false {
		t.Fatalf("IsDir = %#v", got)
	}
	if got := res.OutputData["ModTimeMs"].(int64); got <= 0 {
		t.Fatalf("ModTimeMs = %d, want > 0", got)
	}
}

func TestFileInfo_FileInputWins(t *testing.T) {
	dir := t.TempDir()
	pathA := filepath.Join(dir, "a.txt")
	pathB := filepath.Join(dir, "b.txt")
	if err := os.WriteFile(pathA, []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pathB, []byte("bb"), 0o644); err != nil {
		t.Fatal(err)
	}
	file, err := node.FileFromPath(pathB)
	if err != nil {
		t.Fatal(err)
	}

	res := runFileInfo(t, map[string]any{"Path": pathA, "File": file})
	if res.Error != nil || res.ExitName != "Done" {
		t.Fatalf("exit=%q err=%v", res.ExitName, res.Error)
	}
	if got := res.OutputData["Path"]; got != pathB {
		t.Fatalf("Path = %#v, want File input path %q", got, pathB)
	}
	if got := res.OutputData["Size"]; got != 2 {
		t.Fatalf("Size = %#v, want File input size", got)
	}
}

func TestFileInfo_MissingPathFails(t *testing.T) {
	res := runFileInfo(t, map[string]any{"Path": filepath.Join(t.TempDir(), "missing.txt")})
	if res.Error == nil {
		t.Fatal("missing file should fail")
	}
}
