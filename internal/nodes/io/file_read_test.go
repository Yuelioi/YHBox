package io

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/text/encoding/simplifiedchinese"

	"yotta/internal/node"
)

func runReadTextFile(t *testing.T, cfg map[string]any) node.RunResult {
	t.Helper()
	node.ResetRegistryForTest()
	node.Register(&ReadTextFile{})
	rn, _ := node.Get("ReadTextFile")
	return node.RunNode(context.Background(), rn, nil, cfg, nil, node.StubServices(), false)
}

func runReadJsonFile(t *testing.T, cfg map[string]any) node.RunResult {
	t.Helper()
	node.ResetRegistryForTest()
	node.Register(&ReadJsonFile{})
	rn, _ := node.Get("ReadJsonFile")
	return node.RunNode(context.Background(), rn, nil, cfg, nil, node.StubServices(), false)
}

func writeTempFile(t *testing.T, dir, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestReadTextFile_UTF8AbsolutePath(t *testing.T) {
	path := writeTempFile(t, t.TempDir(), "notes.txt", []byte("hello\n世界"))

	res := runReadTextFile(t, map[string]any{"Path": path})
	if res.Error != nil || res.ExitName != "Done" {
		t.Fatalf("exit=%q err=%v", res.ExitName, res.Error)
	}
	if got := res.OutputData["Text"]; got != "hello\n世界" {
		t.Fatalf("Text = %#v, want utf-8 content", got)
	}
	if got := res.OutputData["Size"].(int); got <= 0 {
		t.Fatalf("Size = %d, want > 0", got)
	}
	if got := res.OutputData["ModTimeMs"].(int64); got <= 0 {
		t.Fatalf("ModTimeMs = %d, want > 0", got)
	}
}

func TestReadTextFile_GBK(t *testing.T) {
	gbkText, err := simplifiedchinese.GBK.NewEncoder().String("中文 cookie")
	if err != nil {
		t.Fatal(err)
	}
	path := writeTempFile(t, t.TempDir(), "gbk.txt", []byte(gbkText))

	res := runReadTextFile(t, map[string]any{"Path": path, "Encoding": "gbk"})
	if res.Error != nil || res.ExitName != "Done" {
		t.Fatalf("exit=%q err=%v", res.ExitName, res.Error)
	}
	if got := res.OutputData["Text"]; got != "中文 cookie" {
		t.Fatalf("Text = %#v, want decoded GBK", got)
	}
}

func TestReadTextFile_RelativePathUsesDataDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("YOTTA_DATA_DIR", dir)
	writeTempFile(t, dir, "cookies.txt", []byte("a=b"))

	res := runReadTextFile(t, map[string]any{"Path": "cookies.txt"})
	if res.Error != nil || res.ExitName != "Done" {
		t.Fatalf("relative path should read from data dir: exit=%q err=%v", res.ExitName, res.Error)
	}
	if got := res.OutputData["Text"]; got != "a=b" {
		t.Fatalf("Text = %#v, want data-dir file content", got)
	}
}

func TestReadTextFile_FileInput(t *testing.T) {
	path := writeTempFile(t, t.TempDir(), "from-file.txt", []byte("from file value"))
	file, err := node.FileFromPath(path)
	if err != nil {
		t.Fatal(err)
	}

	res := runReadTextFile(t, map[string]any{"File": file})
	if res.Error != nil || res.ExitName != "Done" {
		t.Fatalf("File input should read: exit=%q err=%v", res.ExitName, res.Error)
	}
	if got := res.OutputData["Text"]; got != "from file value" {
		t.Fatalf("Text = %#v, want file content", got)
	}
	outFile, ok := res.OutputData["File"].(node.File)
	if !ok || outFile.Path != file.Path {
		t.Fatalf("File output = %#v, want %q", res.OutputData["File"], file.Path)
	}
}

func TestReadTextFile_MissingFileFails(t *testing.T) {
	res := runReadTextFile(t, map[string]any{"Path": filepath.Join(t.TempDir(), "missing.txt")})
	if res.Error == nil {
		t.Fatal("missing file should fail")
	}
}

func TestReadJsonFile_ArrayValue(t *testing.T) {
	path := writeTempFile(t, t.TempDir(), "items.json", []byte(`[{"url":"https://example.test"}, 2]`))

	res := runReadJsonFile(t, map[string]any{"Path": path})
	if res.Error != nil || res.ExitName != "Done" {
		t.Fatalf("exit=%q err=%v", res.ExitName, res.Error)
	}
	arr, ok := res.OutputData["JSON"].([]any)
	if !ok || len(arr) != 2 {
		t.Fatalf("JSON = %#v, want array value", res.OutputData["JSON"])
	}
	if got := res.OutputData["Text"]; got == "" {
		t.Fatal("Text should keep original JSON text")
	}
}

func TestReadJsonFile_OutputsFile(t *testing.T) {
	path := writeTempFile(t, t.TempDir(), "items.json", []byte(`{"ok":true}`))

	res := runReadJsonFile(t, map[string]any{"Path": path})
	if res.Error != nil || res.ExitName != "Done" {
		t.Fatalf("exit=%q err=%v", res.ExitName, res.Error)
	}
	file, ok := res.OutputData["File"].(node.File)
	if !ok {
		t.Fatalf("File output = %#v, want node.File", res.OutputData["File"])
	}
	if file.Name != "items.json" || file.Ext != ".json" || file.Size <= 0 {
		t.Fatalf("File metadata = %+v", file)
	}
}

func TestReadJsonFile_InvalidJSONFails(t *testing.T) {
	path := writeTempFile(t, t.TempDir(), "bad.json", []byte(`{"bad"`))

	res := runReadJsonFile(t, map[string]any{"Path": path})
	if res.Error == nil {
		t.Fatal("invalid JSON should fail")
	}
}

func TestReadJsonFile_TrailingGarbageFails(t *testing.T) {
	path := writeTempFile(t, t.TempDir(), "bad.json", []byte(`{"ok":true} trailing`))

	res := runReadJsonFile(t, map[string]any{"Path": path})
	if res.Error == nil {
		t.Fatal("JSON with trailing garbage should fail")
	}
}
