// internal/services/asset/store_test.go
package asset

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/blob"
)

// ---- helpers ----

func newTestStore(t *testing.T) (*Store, string) {
	t.Helper()
	dir := t.TempDir()
	s, err := NewStore(dir, newTestBlobStore(t, dir))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	return s, dir
}

func newTestBlobStore(t *testing.T, dir string) *blob.Store {
	t.Helper()
	store, err := blob.Open(filepath.Join(dir, "blobs"), blob.Limits{MaxBlobBytes: 1 << 20, MaxTotalBytes: 8 << 20})
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func TestStoreRequiresSharedBlobStore(t *testing.T) {
	if _, err := NewStore(t.TempDir(), nil); err == nil {
		t.Fatal("NewStore accepted an implicit Blob Store")
	}
}

func makeRecord(guid, name, kind string) AssetRecord {
	return AssetRecord{
		SchemaVersion: RecordSchemaVersion,
		GUID:          guid,
		Kind:          kind,
		Name:          name,
		Origin:        Origin{Kind: "user"},
		CreatedAt:     time.Now().UTC(),
	}
}

func testBlobRef(label string) blob.BlobRef {
	digest := sha256.Sum256([]byte(label))
	return blob.BlobRef{
		MediaType: "application/octet-stream",
		Digest:    artifact.Digest(fmt.Sprintf("sha256:%x", digest)),
		Size:      int64(len(label)),
	}
}

func blobPtr(ref blob.BlobRef) *blob.BlobRef { return &ref }

// ---- Task 0.3 tests ----

func TestStore_PreloadRejectsCorrupt(t *testing.T) {
	dir := t.TempDir()
	recDir := filepath.Join(dir, "templates")
	if err := os.MkdirAll(recDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// 合法记录
	good := makeRecord("good", "Good Asset", KindTemplate)
	b, _ := json.MarshalIndent(good, "", "  ")
	if err := os.WriteFile(filepath.Join(recDir, "good.json"), b, 0o644); err != nil {
		t.Fatal(err)
	}

	// 坏 JSON
	if err := os.WriteFile(filepath.Join(recDir, "bad.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := NewStore(dir, newTestBlobStore(t, dir)); err == nil {
		t.Fatal("NewStore accepted corrupt persisted contract data")
	}
}

func TestStore_PreloadRejectsOldSchema(t *testing.T) {
	dir := t.TempDir()
	recDir := filepath.Join(dir, "templates")
	if err := os.MkdirAll(recDir, 0o755); err != nil {
		t.Fatal(err)
	}
	rec := makeRecord("old", "Old", KindTemplate)
	rec.SchemaVersion = RecordSchemaVersion - 1
	b, _ := json.Marshal(rec)
	if err := os.WriteFile(filepath.Join(recDir, "old.json"), b, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewStore(dir, newTestBlobStore(t, dir)); err == nil {
		t.Fatal("NewStore accepted an old schema through a compatibility path")
	}
}

func TestStore_PreloadRejectsDanglingBlobReference(t *testing.T) {
	dir := t.TempDir()
	first, err := NewStore(dir, newTestBlobStore(t, dir))
	if err != nil {
		t.Fatal(err)
	}
	rec := makeRecord("dangling", "Dangling", KindTemplate)
	ref := testBlobRef("missing")
	rec.Variants = []Variant{{Resolution: [2]int{1, 1}, Blob: ref}}
	if err := first.putRecord(rec); err != nil {
		t.Fatal(err)
	}
	if _, err := NewStore(dir, newTestBlobStore(t, dir)); err == nil {
		t.Fatal("NewStore accepted a durable reference to a missing blob")
	}
}

func TestStore_PreloadRejectsUnboundedAmbiguousOrUnknownJSON(t *testing.T) {
	tests := []struct {
		name string
		raw  []byte
	}{
		{name: "unknown field", raw: []byte(`{"schemaVersion":2,"guid":"bad","guidExtra":"x","kind":"template","name":"Bad","origin":{"kind":"user"},"createdAt":"2026-07-15T00:00:00Z"}`)},
		{name: "duplicate field", raw: []byte(`{"schemaVersion":2,"guid":"bad","guid":"other","kind":"template","name":"Bad","origin":{"kind":"user"},"createdAt":"2026-07-15T00:00:00Z"}`)},
		{name: "oversized", raw: bytes.Repeat([]byte{' '}, maxAssetRecordBytes+1)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			recordDir := filepath.Join(dir, "templates")
			if err := os.MkdirAll(recordDir, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(recordDir, "bad.json"), test.raw, 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := NewStore(dir, newTestBlobStore(t, dir)); err == nil {
				t.Fatalf("NewStore accepted %s record", test.name)
			}
		})
	}
}

func TestStore_PutRecord_And_Get(t *testing.T) {
	s, _ := newTestStore(t)

	rec := makeRecord("r1", "Test", KindTemplate)
	if err := s.PutRecord(rec); err != nil {
		t.Fatalf("PutRecord: %v", err)
	}

	got, ok := s.Get("r1")
	if !ok {
		t.Fatal("Get should hit after PutRecord")
	}
	if got.Name != "Test" {
		t.Errorf("Name: %q", got.Name)
	}
}

func TestStore_PutRecord_PersistAndReload(t *testing.T) {
	dir := t.TempDir()
	s1, err := NewStore(dir, newTestBlobStore(t, dir))
	if err != nil {
		t.Fatal(err)
	}
	rec := makeRecord("persist1", "Persistent", KindClip)
	ref, err := s1.CommitRecordBlob(context.Background(), "application/octet-stream", bytes.NewReader([]byte("deadbeef")), func(ref blob.BlobRef) AssetRecord {
		rec.Blob = &ref
		return rec
	})
	if err != nil {
		t.Fatal(err)
	}

	// 新实例 preload
	s2, err := NewStore(dir, newTestBlobStore(t, dir))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := s2.Get("persist1")
	if !ok {
		t.Fatal("preload should restore record")
	}
	if got.Blob == nil || *got.Blob != ref {
		t.Errorf("Blob: %#v", got.Blob)
	}
}

func TestStore_List(t *testing.T) {
	s, _ := newTestStore(t)
	s.PutRecord(makeRecord("a", "A", KindTemplate))
	s.PutRecord(makeRecord("b", "B", KindClip))

	list := s.List()
	if len(list) != 2 {
		t.Errorf("List len: got %d want 2", len(list))
	}
}

func TestStore_PutRecordMeta(t *testing.T) {
	s, _ := newTestStore(t)
	s.PutRecord(makeRecord("m1", "Old Name", KindTemplate))

	if err := s.PutRecordMeta("m1", "New Name", "新描述", "战斗", []string{"tag1"}); err != nil {
		t.Fatalf("PutRecordMeta: %v", err)
	}
	got, _ := s.Get("m1")
	if got.Name != "New Name" {
		t.Errorf("Name: %q", got.Name)
	}
	if got.Description != "新描述" {
		t.Errorf("Description: %q", got.Description)
	}
	if got.Category != "战斗" {
		t.Errorf("Category: %q", got.Category)
	}
	if len(got.Tags) != 1 || got.Tags[0] != "tag1" {
		t.Errorf("Tags: %v", got.Tags)
	}
}

func TestStore_PutRecordMeta_NotFound(t *testing.T) {
	s, _ := newTestStore(t)
	err := s.PutRecordMeta("nope", "X", "", "", nil)
	if err == nil {
		t.Error("PutRecordMeta on missing guid should error")
	}
}

func TestStore_DeleteRecord(t *testing.T) {
	s, dir := newTestStore(t)
	s.PutRecord(makeRecord("d1", "Delete Me", KindTemplate))

	if err := s.DeleteRecord("d1"); err != nil {
		t.Fatalf("DeleteRecord: %v", err)
	}

	_, ok := s.Get("d1")
	if ok {
		t.Error("Get after delete should miss")
	}

	// 磁盘文件也删掉了
	_, err := os.Stat(filepath.Join(dir, "templates", "d1.json"))
	if err == nil {
		t.Error("record file should be removed from disk")
	}
}

func TestStore_CommitRecordBlob(t *testing.T) {
	s, _ := newTestStore(t)
	ref, err := s.CommitRecordBlob(context.Background(), "application/octet-stream", bytes.NewReader([]byte("blob data")), func(ref blob.BlobRef) AssetRecord {
		rec := makeRecord("blob", "Blob", KindClip)
		rec.Blob = &ref
		return rec
	})
	if err != nil {
		t.Fatalf("CommitRecordBlob: %v", err)
	}
	if _, err := s.ReadBlob(context.Background(), ref); err != nil {
		t.Errorf("stored blob is not readable: %v", err)
	}
}

// ---- Task 0.4 tests ----

func TestStore_PutVariantConcurrentNoLostUpdate(t *testing.T) {
	s, _ := newTestStore(t)
	s.PutRecord(makeRecord("g", "Concurrent", KindTemplate))

	res1 := [2]int{1920, 1080}
	res2 := [2]int{1280, 720}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		if err := s.putVariant("g", res1, testBlobRef("sha1"), [4]int{0, 0, 100, 100}, nil); err != nil {
			t.Errorf("PutVariant res1: %v", err)
		}
	}()
	go func() {
		defer wg.Done()
		if err := s.putVariant("g", res2, testBlobRef("sha2"), [4]int{0, 0, 200, 200}, nil); err != nil {
			t.Errorf("PutVariant res2: %v", err)
		}
	}()
	wg.Wait()

	got, ok := s.Get("g")
	if !ok {
		t.Fatal("Get after PutVariant should hit")
	}
	if len(got.Variants) != 2 {
		t.Errorf("Variants len: got %d want 2 (lost update!)", len(got.Variants))
	}
}

func TestStore_PutVariant_Upsert(t *testing.T) {
	s, _ := newTestStore(t)
	s.PutRecord(makeRecord("u1", "Upsert", KindTemplate))

	res := [2]int{1920, 1080}
	s.putVariant("u1", res, testBlobRef("sha-old"), [4]int{0, 0, 100, 100}, nil)
	s.putVariant("u1", res, testBlobRef("sha-new"), [4]int{5, 5, 95, 95}, nil)

	got, _ := s.Get("u1")
	if len(got.Variants) != 1 {
		t.Errorf("upsert same res should stay 1 variant, got %d", len(got.Variants))
	}
	if got.Variants[0].Blob != testBlobRef("sha-new") {
		t.Errorf("upsert should overwrite blob: %#v", got.Variants[0].Blob)
	}
}

func TestStore_RemoveVariant(t *testing.T) {
	s, _ := newTestStore(t)
	s.PutRecord(makeRecord("rv", "RemoveVar", KindTemplate))

	res1 := [2]int{1920, 1080}
	res2 := [2]int{1280, 720}
	s.putVariant("rv", res1, testBlobRef("sha1"), [4]int{}, nil)
	s.putVariant("rv", res2, testBlobRef("sha2"), [4]int{}, nil)

	if err := s.RemoveVariant("rv", res1); err != nil {
		t.Fatalf("RemoveVariant: %v", err)
	}

	got, _ := s.Get("rv")
	if len(got.Variants) != 1 {
		t.Errorf("after remove: got %d variants, want 1", len(got.Variants))
	}
	if got.Variants[0].Resolution != res2 {
		t.Errorf("remaining variant should be res2, got %v", got.Variants[0].Resolution)
	}
}

func TestStore_PutVariant_NotFound(t *testing.T) {
	s, _ := newTestStore(t)
	err := s.putVariant("nope", [2]int{1920, 1080}, testBlobRef("sha"), [4]int{}, nil)
	if err == nil {
		t.Error("PutVariant on missing guid should error")
	}
}
