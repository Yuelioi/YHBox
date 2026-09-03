package macro

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/services/asset"
	"github.com/yottaapp/yotta/internal/storage"
	"github.com/yottaapp/yotta/internal/storage/catalog"
)

func TestServiceGetDurablyMigratesVersion1(t *testing.T) {
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
	blobs, err := blob.Open(filepath.Join(dir, "blobs"), blob.Limits{MaxBlobBytes: 1 << 20, MaxTotalBytes: 8 << 20}, foundation.Objects())
	if err != nil {
		t.Fatal(err)
	}
	assets, err := asset.NewStore(foundation.Assets(), foundation.Objects(), blobs)
	if err != nil {
		t.Fatal(err)
	}
	createdAt := time.Date(2026, time.July, 1, 2, 3, 4, 0, time.UTC)
	legacyRef, err := assets.CommitRecordBlob(context.Background(), MediaType, bytes.NewBufferString(legacyV1Carrier), func(ref blob.BlobRef) asset.AssetRecord {
		return asset.AssetRecord{
			SchemaVersion: asset.RecordSchemaVersion,
			GUID:          "macro-legacy",
			Kind:          asset.KindMacro,
			Name:          "Legacy",
			Origin:        asset.Origin{Kind: "user"},
			Blob:          &ref,
			CreatedAt:     createdAt,
		}
	})
	if err != nil {
		t.Fatal(err)
	}

	value, err := NewService(assets).Get("macro-legacy")
	if err != nil {
		t.Fatal(err)
	}
	if value.Document.SchemaVersion != SchemaVersion || value.Document.Meta != legacyV1Meta() {
		t.Fatalf("migrated document = %#v", value.Document)
	}
	record, found := assets.Get("macro-legacy")
	if !found || record.Blob == nil || *record.Blob == legacyRef {
		t.Fatalf("record after migration = %#v, legacy ref = %#v", record, legacyRef)
	}
	migratedRef := *record.Blob
	if err := foundation.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := catalog.Open(context.Background(), roots)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	reopenedBlobs, err := blob.Open(filepath.Join(dir, "blobs"), blob.Limits{MaxBlobBytes: 1 << 20, MaxTotalBytes: 8 << 20}, reopened.Objects())
	if err != nil {
		t.Fatal(err)
	}
	reopenedAssets, err := asset.NewStore(reopened.Assets(), reopened.Objects(), reopenedBlobs)
	if err != nil {
		t.Fatal(err)
	}
	reloaded, err := NewService(reopenedAssets).Get("macro-legacy")
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Document.SchemaVersion != SchemaVersion {
		t.Fatalf("reloaded schemaVersion = %d", reloaded.Document.SchemaVersion)
	}
	reloadedRecord, found := reopenedAssets.Get("macro-legacy")
	if !found || reloadedRecord.Blob == nil || *reloadedRecord.Blob != migratedRef {
		t.Fatalf("reloaded record = %#v, want blob %#v", reloadedRecord, migratedRef)
	}
}

func TestServiceCRUD(t *testing.T) {
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
	blobs, err := blob.Open(filepath.Join(dir, "blobs"), blob.Limits{MaxBlobBytes: 1 << 20, MaxTotalBytes: 8 << 20}, foundation.Objects())
	if err != nil {
		t.Fatal(err)
	}
	assets, err := asset.NewStore(foundation.Assets(), foundation.Objects(), blobs)
	if err != nil {
		t.Fatal(err)
	}
	changed := 0
	service := NewService(assets, func(string, any) { changed++ })
	value := &Macro{
		Label: " Test ", Description: " Description ", Category: " Category ", Tags: []string{" One ", "one", "Two"},
		Document: Document{SchemaVersion: SchemaVersion, BaseResolution: [2]int{800, 600}, Meta: DefaultMeta(), Actions: []Action{{ID: "sleep", Kind: ActionSleep, DurationUs: 20_000}}},
	}
	saved, err := service.Save(value)
	if err != nil {
		t.Fatal(err)
	}
	if saved.ID == "" || saved.Label != "Test" || len(saved.Tags) != 2 || changed != 2 {
		t.Fatalf("saved = %#v, changed=%d", saved, changed)
	}
	items, err := service.List()
	if err != nil || len(items) != 1 || service.Analyze(saved.Document).DurationUs != 20_000 {
		t.Fatalf("list/analyze = %#v, %v", items, err)
	}
	copy := cloneMacro(saved)
	copy.Tags[0] = "changed"
	if saved.Tags[0] == "changed" {
		t.Fatal("clone shares tags")
	}
	if err := service.Delete(saved.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Get(saved.ID); err == nil {
		t.Fatal("deleted macro still loads")
	}
}
