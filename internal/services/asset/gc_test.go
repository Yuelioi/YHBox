// internal/services/asset/gc_test.go
package asset

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGCBlobs_ReclaimsOrphans(t *testing.T) {
	s, dir := newTestStore(t)
	bs := s.Blobs()

	// 写两个 blob：live（被记录引用）和 orphan（无引用）。
	liveSha, err := bs.Put([]byte("live content"))
	if err != nil {
		t.Fatalf("Put live: %v", err)
	}
	orphanSha, err := bs.Put([]byte("orphan content"))
	if err != nil {
		t.Fatalf("Put orphan: %v", err)
	}

	// 建一条 template 记录，只引用 live blob。
	rec := makeRecord("gc1", "GC Test", KindTemplate)
	if err := s.PutRecord(rec); err != nil {
		t.Fatalf("PutRecord: %v", err)
	}
	if err := s.PutVariant("gc1", [2]int{1920, 1080}, liveSha, [4]int{}, nil); err != nil {
		t.Fatalf("PutVariant: %v", err)
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
	if bs.Has(orphanSha) {
		t.Error("orphan blob should be deleted")
	}
	// live 还在。
	if !bs.Has(liveSha) {
		t.Error("live blob should survive GC")
	}

	// Case: clip 记录的 Blob 也算 live，不被回收。
	clipSha, err := bs.Put([]byte("clip content"))
	if err != nil {
		t.Fatalf("Put clip blob: %v", err)
	}
	orphan2Sha, err := bs.Put([]byte("orphan2 content"))
	if err != nil {
		t.Fatalf("Put orphan2: %v", err)
	}

	clipRec := makeRecord("clip1", "Clip", KindClip)
	clipRec.Blob = clipSha
	if err := s.PutRecord(clipRec); err != nil {
		t.Fatalf("PutRecord clip: %v", err)
	}

	n2, err := s.GCBlobs()
	if err != nil {
		t.Fatalf("GCBlobs round2: %v", err)
	}
	// 只有 orphan2 被回收。
	if n2 != 1 {
		t.Errorf("GCBlobs round2: reclaimed %d, want 1", n2)
	}
	if bs.Has(orphan2Sha) {
		t.Error("orphan2 blob should be deleted")
	}
	if !bs.Has(clipSha) {
		t.Error("clip blob should survive GC")
	}

	// Case: .tmp 残留不被误删（也不计入回收数）。
	blobDir := filepath.Join(dir, "blobs")
	tmpPath := filepath.Join(blobDir, "deadbeef.tmp")
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
