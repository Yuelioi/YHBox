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
	forged := blob.Ref{MediaType: "text/plain", Digest: artifact.Digest("sha256:" + strings.Repeat(".", 64)), Size: 1}
	if _, err := store.ReadRange(context.Background(), forged, 0, 1); err == nil {
		t.Fatal("accepted a forged blob path")
	}
	path := filepath.Join(root, strings.TrimPrefix(ref.Digest.String(), "sha256:"))
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
	reclaimed, err := store.Sweep([]blob.Ref{live})
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
