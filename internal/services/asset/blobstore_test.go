// internal/services/asset/blobstore_test.go
package asset

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBlobStore_PutIdempotentDedup(t *testing.T) {
	dir := t.TempDir()
	blobDir := filepath.Join(dir, "blobs")

	bs, err := NewBlobStore(blobDir)
	if err != nil {
		t.Fatalf("NewBlobStore: %v", err)
	}

	data := []byte("hello")

	sha1, err := bs.Put(data)
	if err != nil {
		t.Fatalf("Put #1: %v", err)
	}
	sha2, err := bs.Put(data)
	if err != nil {
		t.Fatalf("Put #2: %v", err)
	}

	// 两次返回相同 sha。
	if sha1 != sha2 {
		t.Errorf("sha mismatch: %q vs %q", sha1, sha2)
	}

	// Has 应返回 true。
	if !bs.Has(sha1) {
		t.Errorf("Has(%q) = false, want true", sha1)
	}

	// Read 应还原原始内容。
	got, err := bs.Read(sha1)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if string(got) != "hello" {
		t.Errorf("Read: got %q want %q", got, "hello")
	}

	// blobs/ 目录应只有 1 个文件（去重）。
	entries, err := os.ReadDir(blobDir)
	if err != nil {
		t.Fatalf("ReadDir blobs/: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("blobs/ file count: got %d want 1", len(entries))
	}

	// 文件名应是裸 sha（无扩展名）。
	if entries[0].Name() != sha1 {
		t.Errorf("filename: got %q want %q", entries[0].Name(), sha1)
	}
}

func TestBlobStore_Has_Miss(t *testing.T) {
	dir := t.TempDir()
	bs, _ := NewBlobStore(filepath.Join(dir, "blobs"))
	if bs.Has("nonexistent") {
		t.Error("Has nonexistent blob should be false")
	}
}

func TestBlobStore_Read_Missing(t *testing.T) {
	dir := t.TempDir()
	bs, _ := NewBlobStore(filepath.Join(dir, "blobs"))
	_, err := bs.Read("nonexistent")
	if err == nil {
		t.Error("Read nonexistent should return error")
	}
}

func TestBlobStore_DifferentContent(t *testing.T) {
	dir := t.TempDir()
	bs, _ := NewBlobStore(filepath.Join(dir, "blobs"))

	sha1, _ := bs.Put([]byte("hello"))
	sha2, _ := bs.Put([]byte("world"))

	if sha1 == sha2 {
		t.Error("different content should produce different sha")
	}

	entries, _ := os.ReadDir(filepath.Join(dir, "blobs"))
	if len(entries) != 2 {
		t.Errorf("expected 2 blobs, got %d", len(entries))
	}
}
