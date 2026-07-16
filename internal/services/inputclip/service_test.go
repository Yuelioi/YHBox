package inputclip

import (
	"path/filepath"
	"testing"

	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/services/asset"
)

func TestServiceMetadataUpdatePreservesNominalBlobIdentity(t *testing.T) {
	root := t.TempDir()
	blobs, err := blob.Open(filepath.Join(root, "blobs"), blob.Limits{MaxBlobBytes: 1 << 20, MaxTotalBytes: 4 << 20})
	if err != nil {
		t.Fatal(err)
	}
	assets, err := asset.NewStore(root, blobs)
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(assets)
	clip := &InputClip{
		ID: "clip-test", Label: "Before", Description: "old", Category: "demo", Tags: []string{"a"},
		Meta: ClipMeta{MouseMode: "absolute", BaseResolution: [2]int{1920, 1080}},
		Events: []Event{{TUs: 0, Type: EventTypeKeyDown, A: 0x41}},
	}
	clip.UpdateDuration()
	if err := service.Save(clip); err != nil {
		t.Fatal(err)
	}
	wantRef := clip.Blob
	if err := service.Update(clip.ID, "After", "new", "updated", []string{"b"}); err != nil {
		t.Fatal(err)
	}
	got, err := service.Get(clip.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Blob != wantRef || got.Label != "After" || got.Description != "new" || got.Category != "updated" || len(got.Tags) != 1 || got.Tags[0] != "b" {
		t.Fatalf("updated clip = %#v, original BlobRef = %#v", got, wantRef)
	}
	list := service.List()
	if len(list) != 1 || list[0].Blob != wantRef {
		t.Fatalf("clip summaries = %#v", list)
	}
}
