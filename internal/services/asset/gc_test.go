// internal/services/asset/gc_test.go
package asset

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/yottaapp/yotta/internal/blob"
)

func TestGCBlobs_ReclaimsOrphans(t *testing.T) {
	s, dir := newTestStore(t)

	// 写两个 blob：live（被记录引用）和 orphan（无引用）。
	liveRef, err := s.CommitRecordBlob(context.Background(), "application/octet-stream", bytes.NewReader([]byte("live content")), func(ref blob.Ref) AssetRecord {
		rec := makeRecord("gc1", "GC Test", KindTemplate)
		rec.Variants = []Variant{{Resolution: [2]int{1920, 1080}, Blob: ref}}
		return rec
	})
	if err != nil {
		t.Fatalf("Put live: %v", err)
	}
	orphanRef, err := s.blobs.Put(context.Background(), "application/octet-stream", bytes.NewReader([]byte("orphan content")))
	if err != nil {
		t.Fatalf("Put orphan: %v", err)
	}

	// 跑 GC。
	n, err := s.GCBlobs()
	if err != nil {
		t.Fatalf("GCBlobs: %v", err)
	}
	if n != 1 {
		t.Errorf("GCBlobs: reclaimed %d, want 1", n)
	}

	// orphan 没了。
	if _, err := s.ReadBlob(context.Background(), orphanRef); err == nil {
		t.Error("orphan blob should be deleted")
	}
	// live 还在。
	if _, err := s.ReadBlob(context.Background(), liveRef); err != nil {
		t.Errorf("live blob should survive GC: %v", err)
	}

	// Case: clip 记录的 Blob 也算 live，不被回收。
	clipRef, err := s.CommitRecordBlob(context.Background(), "application/octet-stream", bytes.NewReader([]byte("clip content")), func(ref blob.Ref) AssetRecord {
		clipRec := makeRecord("clip1", "Clip", KindClip)
		clipRec.Blob = blobPtr(ref)
		return clipRec
	})
	if err != nil {
		t.Fatalf("Put clip blob: %v", err)
	}
	orphan2Ref, err := s.blobs.Put(context.Background(), "application/octet-stream", bytes.NewReader([]byte("orphan2 content")))
	if err != nil {
		t.Fatalf("Put orphan2: %v", err)
	}

	n2, err := s.GCBlobs()
	if err != nil {
		t.Fatalf("GCBlobs round2: %v", err)
	}
	// 只有 orphan2 被回收。
	if n2 != 1 {
		t.Errorf("GCBlobs round2: reclaimed %d, want 1", n2)
	}
	if _, err := s.ReadBlob(context.Background(), orphan2Ref); err == nil {
		t.Error("orphan2 blob should be deleted")
	}
	if _, err := s.ReadBlob(context.Background(), clipRef); err != nil {
		t.Errorf("clip blob should survive GC: %v", err)
	}

	// Case: .tmp 残留不被误删（也不计入回收数）。
	blobDir := filepath.Join(dir, "blobs")
	tmpPath := filepath.Join(blobDir, ".tmp-active")
	if err := os.WriteFile(tmpPath, []byte("tmp"), 0o644); err != nil {
		t.Fatalf("write .tmp: %v", err)
	}
	n3, err := s.GCBlobs()
	if err != nil {
		t.Fatalf("GCBlobs round3: %v", err)
	}
	if n3 != 0 {
		t.Errorf("GCBlobs round3: reclaimed %d, want 0 (.tmp should be skipped)", n3)
	}
	if _, statErr := os.Stat(tmpPath); statErr != nil {
		t.Error(".tmp file should NOT be deleted by GC")
	}
}
