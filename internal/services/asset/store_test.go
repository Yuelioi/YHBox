// internal/services/asset/store_test.go
package asset

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// ---- helpers ----

func newTestStore(t *testing.T) (*Store, string) {
	t.Helper()
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	return s, dir
}

func makeRecord(guid, name, kind string) AssetRecord {
	return AssetRecord{
		GUID:      guid,
		Kind:      kind,
		Name:      name,
		Origin:    Origin{Kind: "user"},
		CreatedAt: time.Now().UTC(),
	}
}

// ---- Task 0.3 tests ----

func TestStore_PreloadSkipsCorrupt(t *testing.T) {
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

	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore with corrupt file should not fail: %v", err)
	}

	rec, ok := s.Get("good")
	if !ok {
		t.Error("Get(good) should hit")
	}
	if rec.GUID != "good" {
		t.Errorf("Get(good).GUID = %q", rec.GUID)
	}

	_, ok = s.Get("bad")
	if ok {
		t.Error("Get(bad) should miss (corrupt file skipped)")
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
	s1, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	rec := makeRecord("persist1", "Persistent", KindClip)
	rec.Blob = "deadbeef"
	if err := s1.PutRecord(rec); err != nil {
		t.Fatal(err)
	}

	// 新实例 preload
	s2, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := s2.Get("persist1")
	if !ok {
		t.Fatal("preload should restore record")
	}
	if got.Blob != "deadbeef" {
		t.Errorf("Blob: %q", got.Blob)
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

func TestStore_Blobs(t *testing.T) {
	s, _ := newTestStore(t)
	bs := s.Blobs()
	if bs == nil {
		t.Fatal("Blobs() should not be nil")
	}
	sha, err := bs.Put([]byte("blob data"))
	if err != nil {
		t.Fatalf("Blobs().Put: %v", err)
	}
	if !bs.Has(sha) {
		t.Error("Blobs().Has should be true after Put")
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
		if err := s.PutVariant("g", res1, "sha1", [4]int{0, 0, 100, 100}, nil); err != nil {
			t.Errorf("PutVariant res1: %v", err)
		}
	}()
	go func() {
		defer wg.Done()
		if err := s.PutVariant("g", res2, "sha2", [4]int{0, 0, 200, 200}, nil); err != nil {
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
	s.PutVariant("u1", res, "sha-old", [4]int{0, 0, 100, 100}, nil)
	s.PutVariant("u1", res, "sha-new", [4]int{5, 5, 95, 95}, nil)

	got, _ := s.Get("u1")
	if len(got.Variants) != 1 {
		t.Errorf("upsert same res should stay 1 variant, got %d", len(got.Variants))
	}
	if got.Variants[0].Blob != "sha-new" {
		t.Errorf("upsert should overwrite blob: %q", got.Variants[0].Blob)
	}
}

func TestStore_RemoveVariant(t *testing.T) {
	s, _ := newTestStore(t)
	s.PutRecord(makeRecord("rv", "RemoveVar", KindTemplate))

	res1 := [2]int{1920, 1080}
	res2 := [2]int{1280, 720}
	s.PutVariant("rv", res1, "sha1", [4]int{}, nil)
	s.PutVariant("rv", res2, "sha2", [4]int{}, nil)

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
	err := s.PutVariant("nope", [2]int{1920, 1080}, "sha", [4]int{}, nil)
	if err == nil {
		t.Error("PutVariant on missing guid should error")
	}
}
