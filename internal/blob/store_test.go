package blob_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yottaapp/yotta/internal/artifact"

	"github.com/yottaapp/yotta/internal/blob"
)

func TestStoreSealsAndReadsContentAddressedBlob(t *testing.T) {
	store, err := blob.Open(t.TempDir(), blob.Limits{MaxBlobBytes: 1024, MaxTotalBytes: 4096})
	if err != nil {
		t.Fatal(err)
	}
	ref, err := store.Put(context.Background(), "text/plain", bytes.NewReader([]byte("hello")))
	if err != nil {
		t.Fatal(err)
	}
	if ref.Digest.String() != "sha256:2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824" || ref.Size != 5 || ref.MediaType != "text/plain" {
		t.Fatalf("ref = %#v", ref)
	}
	got, err := store.ReadRange(context.Background(), ref, 0, ref.Size)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, []byte("hello")) {
		t.Fatalf("read = %q", got)
	}
}

func TestStoreRejectsTamperingTraversalAndQuotaAmplification(t *testing.T) {
	root := t.TempDir()
	store, err := blob.Open(root, blob.Limits{MaxBlobBytes: 5, MaxTotalBytes: 5})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Put(context.Background(), "text/plain", strings.NewReader("123456")); err == nil {
		t.Fatal("accepted a blob above the per-object quota")
	}
	ref, err := store.Put(context.Background(), "text/plain", strings.NewReader("hello"))
	if err != nil {
		t.Fatal(err)
	}
	if got, err := store.ReadRange(context.Background(), ref, 1, 3); err != nil || string(got) != "ell" {
		t.Fatalf("range=%q err=%v", got, err)
	}
	forged := blob.BlobRef{MediaType: "text/plain", Digest: artifact.Digest("sha256:" + strings.Repeat(".", 64)), Size: 1}
	if _, err := store.ReadRange(context.Background(), forged, 0, 1); err == nil {
		t.Fatal("accepted a forged blob path")
	}
	name := strings.TrimPrefix(ref.Digest.String(), "sha256:")
	path := filepath.Join(root, name[:2], name[2:4], name)
	if err := os.WriteFile(path, []byte("HELLO"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ReadRange(context.Background(), ref, 0, ref.Size); err == nil {
		t.Fatal("accepted tampered blob bytes")
	}
	if _, err := store.Put(context.Background(), "text/plain", strings.NewReader("hello")); err == nil {
		t.Fatal("reused an existing object whose bytes do not match its digest")
	}
}

func TestStoreEnforcesTotalQuotaAcrossDistinctObjects(t *testing.T) {
	store, err := blob.Open(t.TempDir(), blob.Limits{MaxBlobBytes: 4, MaxTotalBytes: 5})
	if err != nil {
		t.Fatal(err)
	}
	ref, err := store.Put(context.Background(), "text/plain", strings.NewReader("one"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Verify(context.Background(), ref); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Put(context.Background(), "text/plain", strings.NewReader("two")); err == nil {
		t.Fatal("accepted distinct objects above the total quota")
	}
}

func TestStoreSweepOwnsObjectLifecycle(t *testing.T) {
	store, err := blob.Open(t.TempDir(), blob.Limits{MaxBlobBytes: 1024, MaxTotalBytes: 4096})
	if err != nil {
		t.Fatal(err)
	}
	live, err := store.Put(context.Background(), "text/plain", strings.NewReader("live"))
	if err != nil {
		t.Fatal(err)
	}
	orphan, err := store.Put(context.Background(), "text/plain", strings.NewReader("orphan"))
	if err != nil {
		t.Fatal(err)
	}
	reclaimed, err := store.Sweep([]blob.BlobRef{live})
	if err != nil {
		t.Fatal(err)
	}
	if reclaimed != 1 {
		t.Fatalf("reclaimed = %d, want 1", reclaimed)
	}
	if _, err := store.ReadRange(context.Background(), live, 0, live.Size); err != nil {
		t.Fatalf("live object was removed: %v", err)
	}
	if _, err := store.ReadRange(context.Background(), orphan, 0, orphan.Size); err == nil {
		t.Fatal("orphan object survived sweep")
	}
}

func TestRetainedPutProtectsObjectUntilDurableReferencePublication(t *testing.T) {
	store, err := blob.Open(t.TempDir(), blob.Limits{MaxBlobBytes: 1024, MaxTotalBytes: 4096})
	if err != nil {
		t.Fatal(err)
	}
	ref, retention, err := store.PutRetained(context.Background(), "text/plain", strings.NewReader("pending"))
	if err != nil {
		t.Fatal(err)
	}
	if reclaimed, err := store.Sweep(nil); err != nil || reclaimed != 0 {
		t.Fatalf("Sweep while retained = %d, %v", reclaimed, err)
	}
	if err := store.Verify(context.Background(), ref); err != nil {
		t.Fatalf("retained object was removed: %v", err)
	}
	retention.Release()
	if reclaimed, err := store.Sweep(nil); err != nil || reclaimed != 1 {
		t.Fatalf("Sweep after release = %d, %v", reclaimed, err)
	}
}

func TestStoreRejectsUnownedOrEmptyRoot(t *testing.T) {
	limits := blob.Limits{MaxBlobBytes: 1024, MaxTotalBytes: 4096}
	if _, err := blob.Open("", limits); err == nil {
		t.Fatal("accepted an empty blob root")
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "unrelated.txt"), []byte("mine"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := blob.Open(root, limits); err == nil {
		t.Fatal("claimed a non-empty directory without an ownership marker")
	}
}

func TestStoreMigrationUpgradesLegacyOwnershipMarker(t *testing.T) {
	root := t.TempDir()
	markerPath := filepath.Join(root, ".yotta-blob-store")
	if err := os.WriteFile(markerPath, []byte("yotta/blob-store/1\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := blob.MigrateLayoutOneToTwo(context.Background(), root, nil); err != nil {
		t.Fatalf("migrate legacy Blob Store: %v", err)
	}
	if _, err := blob.Open(root, blob.Limits{MaxBlobBytes: 1024, MaxTotalBytes: 4096}); err != nil {
		t.Fatalf("open legacy Blob Store: %v", err)
	}

	marker, err := os.ReadFile(markerPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(marker) != "yotta/blob-store/2\n" {
		t.Fatalf("marker = %q", marker)
	}
}

func TestStoreMigrationPreservesLegacyObjectsAndRollsBack(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(root, ".yotta-blob-store"),
		[]byte("yotta/blob-store/1\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	const name = "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	legacyPath := filepath.Join(root, name)
	if err := os.WriteFile(legacyPath, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	inventory := newMemoryInventory()

	report, err := blob.MigrateLayoutOneToTwo(context.Background(), root, inventory)
	if err != nil {
		t.Fatal(err)
	}
	if report.Version != blob.LayoutVersion || report.Objects != 1 || report.Bytes != 5 {
		t.Fatalf("migration report = %#v", report)
	}
	store, err := blob.Open(
		root,
		blob.Limits{MaxBlobBytes: 1024, MaxTotalBytes: 4096},
		inventory,
	)
	if err != nil {
		t.Fatal(err)
	}
	ref := blob.BlobRef{
		MediaType: "text/plain",
		Digest:    artifact.Digest("sha256:" + name),
		Size:      5,
	}
	if err := store.Verify(context.Background(), ref); err != nil {
		t.Fatalf("migrated object: %v", err)
	}

	if err := blob.RollbackLayoutTwoToOne(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	if raw, err := os.ReadFile(legacyPath); err != nil || string(raw) != "hello" {
		t.Fatalf("rolled-back object = %q, %v", raw, err)
	}
	if marker, err := os.ReadFile(filepath.Join(root, ".yotta-blob-store")); err != nil ||
		string(marker) != "yotta/blob-store/1\n" {
		t.Fatalf("rolled-back marker = %q, %v", marker, err)
	}
}

func TestStoreMigrationReconcilesAlreadyShardedObjects(t *testing.T) {
	root := t.TempDir()
	store, err := blob.Open(root, blob.Limits{MaxBlobBytes: 1024, MaxTotalBytes: 4096})
	if err != nil {
		t.Fatal(err)
	}
	ref, err := store.Put(context.Background(), "text/plain", strings.NewReader("indexed"))
	if err != nil {
		t.Fatal(err)
	}
	inventory := newMemoryInventory()

	if _, err := blob.MigrateLayoutOneToTwo(context.Background(), root, inventory); err != nil {
		t.Fatal(err)
	}
	if object, found, err := inventory.Object(context.Background(), ref.Digest); err != nil ||
		!found || object.Size != ref.Size {
		t.Fatalf("reconciled object = %#v, %v, %v", object, found, err)
	}
}
