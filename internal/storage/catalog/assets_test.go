package catalog

import (
	"context"
	"crypto/sha256"
	"fmt"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/blob"
)

func TestAssetRepositoryPersistsQueriesAndReferences(t *testing.T) {
	roots := testRoots(t)
	foundation, err := Open(context.Background(), roots)
	if err != nil {
		t.Fatal(err)
	}
	defer foundation.Close()
	repository := foundation.Assets()
	ctx := context.Background()
	created := time.Date(2026, 7, 25, 1, 2, 3, 0, time.UTC)
	first := AssetRecord{
		GUID: "asset-a", Kind: AssetKindTemplate, Name: "购买按钮",
		Description: "钓鱼商店", Category: "钓鱼", Tags: []string{"按钮", "Fishing", "按钮"},
		Origin: AssetOrigin{Kind: "user"}, CreatedAt: created,
		Variants: []AssetVariant{{
			Resolution: [2]int{1920, 1080}, BBox: [4]int{1, 2, 30, 40},
			Regions: [][4]int{{1, 2, 3, 4}}, Blob: catalogTestBlob("first"),
		}},
	}
	observeCatalogAssetBlobs(t, foundation.Objects(), first)
	saved, err := repository.Put(ctx, first)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Revision != 1 || len(saved.Tags) != 2 {
		t.Fatalf("Put() = %#v", saved)
	}
	second := AssetRecord{
		GUID: "asset-b", Kind: AssetKindMacro, Name: "收杆",
		Description: "自动宏", Category: "钓鱼", Tags: []string{"Fishing"},
		Origin: AssetOrigin{Kind: "user"}, CreatedAt: created.Add(time.Minute),
		Blob: pointerToBlob(catalogTestBlob("second")),
	}
	observeCatalogAssetBlobs(t, foundation.Objects(), second)
	if _, err := repository.Put(ctx, second); err != nil {
		t.Fatal(err)
	}

	page, err := repository.Query(ctx, AssetQuery{
		Search: "按钮", Kind: AssetKindTemplate, Tags: []string{"fishing"},
		Sort: "created_desc", Page: 1, PageSize: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || len(page.Records) != 1 || page.Records[0].GUID != first.GUID ||
		page.Revision != 2 {
		t.Fatalf("Query() = %#v", page)
	}
	if len(page.Categories) != 1 || page.Categories[0].Value != "钓鱼" ||
		len(page.Tags) != 2 {
		t.Fatalf("facets = %#v/%#v", page.Categories, page.Tags)
	}
	matches, err := repository.ResolveBinding(ctx, first.Variants[0].Blob)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 || matches[0].GUID != first.GUID ||
		matches[0].Resolution != first.Variants[0].Resolution {
		t.Fatalf("ResolveBinding() = %#v", matches)
	}

	if err := foundation.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(ctx, roots)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	got, ok, err := reopened.Assets().Get(ctx, first.GUID)
	if err != nil || !ok || got.Name != first.Name || len(got.Variants) != 1 {
		t.Fatalf("Get(reopened) = %#v, %v, %v", got, ok, err)
	}
}

func TestAssetRepositoryUpdateDeleteAndStableRecentSort(t *testing.T) {
	foundation, err := Open(context.Background(), testRoots(t))
	if err != nil {
		t.Fatal(err)
	}
	defer foundation.Close()
	repository := foundation.Assets()
	ctx := context.Background()
	for index, guid := range []string{"a", "b", "c"} {
		record := AssetRecord{
			GUID: guid, Kind: AssetKindClip, Name: "same",
			Origin: AssetOrigin{Kind: "user"}, CreatedAt: time.Unix(int64(index+1), 0).UTC(),
			Blob: pointerToBlob(catalogTestBlob(guid)),
		}
		observeCatalogAssetBlobs(t, foundation.Objects(), record)
		if _, err := repository.Put(ctx, record); err != nil {
			t.Fatal(err)
		}
	}
	page, err := repository.Query(ctx, AssetQuery{
		Sort: "recent_desc", RecentGUIDs: []string{"c", "a"}, Page: 1, PageSize: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if page.Records[0].GUID != "c" || page.Records[1].GUID != "a" {
		t.Fatalf("recent order = %s, %s", page.Records[0].GUID, page.Records[1].GUID)
	}
	record, ok, err := repository.Get(ctx, "a")
	if err != nil || !ok {
		t.Fatal(err)
	}
	record.Name = "updated"
	record, err = repository.Put(ctx, record)
	if err != nil {
		t.Fatal(err)
	}
	if record.Revision != 2 {
		t.Fatalf("updated revision = %d", record.Revision)
	}
	deleted, err := repository.Delete(ctx, "a")
	if err != nil || !deleted {
		t.Fatalf("Delete() = %v, %v", deleted, err)
	}
	if revision, err := repository.Revision(ctx); err != nil || revision != 5 {
		t.Fatalf("Revision() = %d, %v", revision, err)
	}
}

func TestAssetRepositoryRejectsConflictingObjectSize(t *testing.T) {
	foundation, err := Open(context.Background(), testRoots(t))
	if err != nil {
		t.Fatal(err)
	}
	defer foundation.Close()
	repository := foundation.Assets()
	ref := catalogTestBlob("same")
	first := AssetRecord{
		GUID: "first", Kind: AssetKindClip, Name: "first", Origin: AssetOrigin{Kind: "user"},
		CreatedAt: time.Now().UTC(), Blob: pointerToBlob(ref),
	}
	observeCatalogAssetBlobs(t, foundation.Objects(), first)
	if _, err := repository.Put(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	ref.Size++
	second := AssetRecord{
		GUID: "second", Kind: AssetKindClip, Name: "second", Origin: AssetOrigin{Kind: "user"},
		CreatedAt: time.Now().UTC(), Blob: pointerToBlob(ref),
	}
	if _, err := repository.Put(context.Background(), second); err == nil {
		t.Fatal("Put() accepted conflicting size for one digest")
	}
}

func TestAssetRepositoryRejectsReferenceBeforeCASObservation(t *testing.T) {
	foundation, err := Open(context.Background(), testRoots(t))
	if err != nil {
		t.Fatal(err)
	}
	defer foundation.Close()
	record := AssetRecord{
		GUID: "missing", Kind: AssetKindClip, Name: "missing",
		Origin: AssetOrigin{Kind: "user"}, CreatedAt: time.Now().UTC(),
		Blob: pointerToBlob(catalogTestBlob("not-published")),
	}
	if _, err := foundation.Assets().Put(context.Background(), record); err == nil {
		t.Fatal("Put() committed a reference before the CAS object was observed")
	}
	if _, found, err := foundation.Assets().Get(context.Background(), record.GUID); err != nil || found {
		t.Fatalf("failed Put() left asset metadata behind: found=%v, err=%v", found, err)
	}
}

func TestAssetRepositoryQueriesTenThousandMetadataRowsByPage(t *testing.T) {
	foundation, err := Open(context.Background(), testRoots(t))
	if err != nil {
		t.Fatal(err)
	}
	defer foundation.Close()
	tx, err := foundation.content.db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	created := time.Date(2026, 7, 25, 0, 0, 0, 0, time.UTC).Format(time.RFC3339Nano)
	for index := range 10_000 {
		guid := fmt.Sprintf("scale-%05d", index)
		category := "general"
		if index%10 == 0 {
			category = "target"
		}
		if _, err := tx.Exec(`
			INSERT INTO assets(
				guid, kind, name, description, category, origin_kind, origin_source_id,
				created_at, record_revision
			) VALUES (?, 'template', ?, ?, ?, 'user', '', ?, 1)
		`, guid, fmt.Sprintf("Asset %05d", index), "scale metadata", category, created); err != nil {
			_ = tx.Rollback()
			t.Fatal(err)
		}
		if index%10 == 0 {
			if _, err := tx.Exec(`
				INSERT INTO asset_tags(asset_guid, ordinal, tag, normalized_tag)
				VALUES (?, 0, 'Selected', 'selected')
			`, guid); err != nil {
				_ = tx.Rollback()
				t.Fatal(err)
			}
		}
	}
	if _, err := tx.Exec("UPDATE meta SET value = '10000' WHERE key = 'asset_revision'"); err != nil {
		_ = tx.Rollback()
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	page, err := foundation.Assets().Query(context.Background(), AssetQuery{
		Search:   "scale metadata",
		Kind:     AssetKindTemplate,
		Category: "target",
		Tags:     []string{"selected"},
		Sort:     "name_desc",
		Page:     7,
		PageSize: 25,
	})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1_000 || len(page.Records) != 25 || page.Page != 7 ||
		page.Revision != 10_000 {
		t.Fatalf("scale query page = total %d, records %d, page %d, revision %d",
			page.Total, len(page.Records), page.Page, page.Revision)
	}
}

func catalogTestBlob(label string) blob.BlobRef {
	sum := sha256.Sum256([]byte(label))
	return blob.BlobRef{
		MediaType: "application/octet-stream",
		Digest:    artifact.Digest(fmt.Sprintf("sha256:%x", sum)),
		Size:      int64(len(label)),
	}
}

func pointerToBlob(ref blob.BlobRef) *blob.BlobRef { return &ref }

func observeCatalogAssetBlobs(t *testing.T, objects *ObjectRepository, record AssetRecord) {
	t.Helper()
	refs := make([]blob.BlobRef, 0, len(record.Variants)+1)
	if record.Blob != nil {
		refs = append(refs, *record.Blob)
	}
	for _, variant := range record.Variants {
		refs = append(refs, variant.Blob)
	}
	for _, ref := range refs {
		if err := objects.Observe(context.Background(), blob.Object{
			Digest: ref.Digest,
			Size:   ref.Size,
		}); err != nil {
			t.Fatal(err)
		}
	}
}
