// internal/services/asset/store_test.go
package asset

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/storage"
	"github.com/yottaapp/yotta/internal/storage/catalog"
)

// ---- helpers ----

func newTestStore(t *testing.T) (*Store, string) {
	t.Helper()
	dir := t.TempDir()
	roots, err := storage.Resolve(filepath.Join(dir, "profile"))
	if err != nil {
		t.Fatal(err)
	}
	foundation, err := catalog.Open(context.Background(), roots)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = foundation.Close() })
	blobs := newTestBlobStore(t, dir, foundation.Objects())
	s, err := NewStore(
		foundation.Assets(),
		foundation.Objects(),
		blobs,
		WithGCGracePeriod(0),
	)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	return s, dir
}

func newTestBlobStore(
	t *testing.T,
	dir string,
	objects *catalog.ObjectRepository,
) *blob.Store {
	t.Helper()
	store, err := blob.Open(
		filepath.Join(dir, "blobs"),
		blob.Limits{MaxBlobBytes: 1 << 20, MaxTotalBytes: 8 << 20},
		objects,
	)
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func TestStoreRequiresSharedBlobStore(t *testing.T) {
	if _, err := NewStore(nil, nil, nil); err == nil {
		t.Fatal("NewStore accepted an implicit Content Catalog")
	}
	roots, err := storage.Resolve(filepath.Join(t.TempDir(), "profile"))
	if err != nil {
		t.Fatal(err)
	}
	foundation, err := catalog.Open(context.Background(), roots)
	if err != nil {
		t.Fatal(err)
	}
	defer foundation.Close()
	if _, err := NewStore(foundation.Assets(), foundation.Objects(), nil); err == nil {
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

func observeTestBlob(t *testing.T, store *Store, ref blob.BlobRef) {
	t.Helper()
	if err := store.objects.Observe(context.Background(), blob.Object{
		Digest: ref.Digest,
		Size:   ref.Size,
	}); err != nil {
		t.Fatal(err)
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
	roots, err := storage.Resolve(filepath.Join(dir, "profile"))
	if err != nil {
		t.Fatal(err)
	}
	foundation, err := catalog.Open(context.Background(), roots)
	if err != nil {
		t.Fatal(err)
	}
	blobs := newTestBlobStore(t, dir, foundation.Objects())
	s1, err := NewStore(foundation.Assets(), foundation.Objects(), blobs)
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
	if err := foundation.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := catalog.Open(context.Background(), roots)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	reopenedBlobs := newTestBlobStore(t, dir, reopened.Objects())
	s2, err := NewStore(reopened.Assets(), reopened.Objects(), reopenedBlobs)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := s2.Get("persist1")
	if !ok {
		t.Fatal("Catalog reopen should restore record")
	}
	if got.Blob == nil || *got.Blob != ref {
		t.Errorf("Blob: %#v", got.Blob)
	}
	if err := s2.blobs.Verify(context.Background(), ref); err != nil {
		t.Fatalf("CAS reopen did not restore bytes: %v", err)
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
	s, _ := newTestStore(t)
	s.PutRecord(makeRecord("d1", "Delete Me", KindTemplate))

	if err := s.DeleteRecord("d1"); err != nil {
		t.Fatalf("DeleteRecord: %v", err)
	}

	_, ok := s.Get("d1")
	if ok {
		t.Error("Get after delete should miss")
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

func TestStoreCatalogFailureLeavesOnlyGraceCollectableOrphan(t *testing.T) {
	s, _ := newTestStore(t)
	content := []byte("published-before-invalid-metadata")
	if _, err := s.CommitRecordBlob(
		context.Background(),
		"application/octet-stream",
		bytes.NewReader(content),
		func(ref blob.BlobRef) AssetRecord {
			record := makeRecord("", "invalid", KindClip)
			record.Blob = &ref
			return record
		},
	); err == nil {
		t.Fatal("CommitRecordBlob() accepted invalid Catalog metadata")
	}
	sum := sha256.Sum256(content)
	orphan := blob.BlobRef{
		MediaType: "application/octet-stream",
		Digest:    artifact.Digest(fmt.Sprintf("sha256:%x", sum)),
		Size:      int64(len(content)),
	}
	if err := s.blobs.Verify(context.Background(), orphan); err != nil {
		t.Fatalf("CAS publish was not recoverable after Catalog failure: %v", err)
	}
	service := NewService(s, nil, nil)
	preview, err := service.PreviewCleanup()
	if err != nil || preview.CandidateCount != 1 {
		t.Fatalf("PreviewCleanup() = %#v, %v", preview, err)
	}
	result, err := service.CommitCleanup(preview.Token)
	if err != nil || result.Reclaimed != 1 {
		t.Fatalf("CommitCleanup() = %#v, %v", result, err)
	}
	if err := s.blobs.Verify(context.Background(), orphan); err == nil {
		t.Fatal("Catalog failure orphan survived grace-period cleanup")
	}
}

// ---- Task 0.4 tests ----

func TestStore_PutVariantConcurrentNoLostUpdate(t *testing.T) {
	s, _ := newTestStore(t)
	s.PutRecord(makeRecord("g", "Concurrent", KindTemplate))

	res1 := [2]int{1920, 1080}
	res2 := [2]int{1280, 720}
	ref1 := testBlobRef("sha1")
	ref2 := testBlobRef("sha2")
	observeTestBlob(t, s, ref1)
	observeTestBlob(t, s, ref2)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		if err := s.putVariant("g", res1, ref1, [4]int{0, 0, 100, 100}, nil); err != nil {
			t.Errorf("PutVariant res1: %v", err)
		}
	}()
	go func() {
		defer wg.Done()
		if err := s.putVariant("g", res2, ref2, [4]int{0, 0, 200, 200}, nil); err != nil {
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
	oldRef := testBlobRef("sha-old")
	newRef := testBlobRef("sha-new")
	observeTestBlob(t, s, oldRef)
	observeTestBlob(t, s, newRef)
	s.putVariant("u1", res, oldRef, [4]int{0, 0, 100, 100}, nil)
	s.putVariant("u1", res, newRef, [4]int{5, 5, 95, 95}, nil)

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
	ref1 := testBlobRef("sha1")
	ref2 := testBlobRef("sha2")
	observeTestBlob(t, s, ref1)
	observeTestBlob(t, s, ref2)
	s.putVariant("rv", res1, ref1, [4]int{}, nil)
	s.putVariant("rv", res2, ref2, [4]int{}, nil)

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
