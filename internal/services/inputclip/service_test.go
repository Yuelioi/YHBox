package inputclip

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/services/asset"
	"github.com/yottaapp/yotta/internal/storage"
	"github.com/yottaapp/yotta/internal/storage/catalog"
)

func TestServiceMetadataUpdatePreservesNominalBlobIdentity(t *testing.T) {
	root := t.TempDir()
	assets, _ := newInputClipAssetStore(t, root)
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
	list, err := service.List()
	if err != nil || len(list) != 1 || list[0].Blob != wantRef {
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

func TestServiceListReportsUnsupportedCarrierInsteadOfHidingClip(t *testing.T) {
	assets, _ := newInputClipAssetStore(t, t.TempDir())
	if _, err := assets.CommitRecordBlob(
		context.Background(), MediaType, bytes.NewBufferString("not-an-input-clip"),
		func(ref blob.BlobRef) asset.AssetRecord {
			return asset.AssetRecord{
				GUID: "clip-broken", Kind: asset.KindClip, Name: "Broken", Origin: asset.Origin{Kind: "user"},
				Blob: &ref, CreatedAt: time.Now().UTC(),
			}
		},
	); err != nil {
		t.Fatal(err)
	}
	if _, err := NewService(assets).List(); err == nil || !strings.Contains(err.Error(), "clip-broken") {
		t.Fatalf("List error = %v", err)
	}
}

func TestServiceEmitsUnifiedAssetInvalidation(t *testing.T) {
	root := t.TempDir()
	assets, _ := newInputClipAssetStore(t, root)
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

func newInputClipAssetStore(t *testing.T, root string) (*asset.Store, *blob.Store) {
	t.Helper()
	roots, err := storage.Resolve(filepath.Join(root, "profile"))
	if err != nil {
		t.Fatal(err)
	}
	foundation, err := catalog.Open(context.Background(), roots)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = foundation.Close() })
	blobs, err := blob.Open(
		filepath.Join(root, "blobs"),
		blob.Limits{MaxBlobBytes: 1 << 20, MaxTotalBytes: 4 << 20},
		foundation.Objects(),
	)
	if err != nil {
		t.Fatal(err)
	}
	store, err := asset.NewStore(
		foundation.Assets(),
		foundation.Objects(),
		blobs,
		asset.WithGCGracePeriod(0),
	)
	if err != nil {
		t.Fatal(err)
	}
	return store, blobs
}
