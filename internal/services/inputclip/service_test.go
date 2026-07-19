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
		Meta: ClipMeta{RecordingMode: RecordingModeSimple, MouseMode: "absolute", BaseResolution: [2]int{1920, 1080}},
		Events: []Event{
			{TUs: 0, Type: EventTypeKeyDown, A: 0x41},
			{TUs: 1, Seq: 1, Type: EventTypeKeyUp, A: 0x41},
		},
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
	summary, err := service.Summary(clip.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(summary.Tracks) != 1 || summary.Tracks[0].Kind != "keyboard" || summary.Tracks[0].Count != 2 {
		t.Fatalf("summary tracks = %#v", summary.Tracks)
	}
	page, err := service.Events(clip.ID, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 2 || len(page.Items) != 1 || page.Items[0].Type != EventTypeKeyUp {
		t.Fatalf("event page = %#v", page)
	}
	if _, err := service.Events(clip.ID, 0, maxEventPageSize+1); err == nil {
		t.Fatal("oversized event page succeeded")
	}
}

func TestServiceEmitsUnifiedAssetInvalidation(t *testing.T) {
	root := t.TempDir()
	blobs, err := blob.Open(filepath.Join(root, "blobs"), blob.Limits{MaxBlobBytes: 1 << 20, MaxTotalBytes: 4 << 20})
	if err != nil {
		t.Fatal(err)
	}
	assets, err := asset.NewStore(root, blobs)
	if err != nil {
		t.Fatal(err)
	}
	events := make([]string, 0)
	service := NewService(assets, func(name string, _ any) { events = append(events, name) })
	clip := &InputClip{
		ID: "clip-events", Label: "Events",
		Meta: ClipMeta{RecordingMode: RecordingModeSimple, MouseMode: "absolute", BaseResolution: [2]int{1280, 720}},
		Events: []Event{
			{TUs: 0, Type: EventTypeKeyDown, A: 0x41},
			{TUs: 1, Seq: 1, Type: EventTypeKeyUp, A: 0x41},
		},
	}
	clip.UpdateDuration()
	if err := service.Save(clip); err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[0] != "asset:changed" || events[1] != "clip:changed" {
		t.Fatalf("events = %v", events)
	}
}
