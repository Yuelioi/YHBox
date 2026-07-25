package catalog

import (
	"context"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/blob"
)

func TestObjectRepositoryGraceReferencesAndDurableLeases(t *testing.T) {
	foundation, err := Open(context.Background(), testRoots(t))
	if err != nil {
		t.Fatal(err)
	}
	defer foundation.Close()
	objects := foundation.Objects()
	ctx := context.Background()
	ref := catalogTestBlob("gc-object")
	object := blob.Object{Digest: ref.Digest, Size: ref.Size}
	if err := objects.Observe(ctx, object); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Add(time.Hour)
	plan, err := objects.PlanGC(ctx, nil, now, time.Hour)
	if err != nil || len(plan.Candidates) != 0 || len(plan.Objects) != 1 {
		t.Fatalf("first PlanGC() = %#v, %v", plan, err)
	}
	plan, err = objects.PlanGC(ctx, nil, now.Add(2*time.Hour), time.Hour)
	if err != nil || len(plan.Candidates) != 1 {
		t.Fatalf("eligible PlanGC() = %#v, %v", plan, err)
	}
	if err := objects.Lease(
		ctx,
		"lease-1",
		ref.Digest,
		"run",
		"run-1",
		now.Add(4*time.Hour),
	); err != nil {
		t.Fatal(err)
	}
	plan, err = objects.PlanGC(ctx, nil, now.Add(3*time.Hour), time.Hour)
	if err != nil || len(plan.Candidates) != 0 {
		t.Fatalf("leased PlanGC() = %#v, %v", plan, err)
	}
	if err := objects.ReleaseLease(ctx, "lease-1"); err != nil {
		t.Fatal(err)
	}
	plan, err = objects.PlanGC(ctx, []blob.BlobRef{ref}, now.Add(5*time.Hour), time.Hour)
	if err != nil || len(plan.Candidates) != 0 {
		t.Fatalf("externally referenced PlanGC() = %#v, %v", plan, err)
	}
	plan, err = objects.PlanGC(ctx, nil, now.Add(6*time.Hour), time.Hour)
	if err != nil || len(plan.Candidates) != 0 {
		t.Fatalf("newly unreachable PlanGC() = %#v, %v", plan, err)
	}
	plan, err = objects.PlanGC(ctx, nil, now.Add(8*time.Hour), time.Hour)
	if err != nil || len(plan.Candidates) != 1 {
		t.Fatalf("second eligible PlanGC() = %#v, %v", plan, err)
	}
}

func TestObjectRepositoryRejectsForgetWhileReferenced(t *testing.T) {
	foundation, err := Open(context.Background(), testRoots(t))
	if err != nil {
		t.Fatal(err)
	}
	defer foundation.Close()
	ref := catalogTestBlob("referenced")
	record := AssetRecord{
		GUID:      "asset-ref",
		Kind:      AssetKindClip,
		Name:      "referenced",
		Origin:    AssetOrigin{Kind: "user"},
		CreatedAt: time.Now().UTC(),
		Blob:      pointerToBlob(ref),
	}
	if err := foundation.Objects().Observe(context.Background(), blob.Object{
		Digest: ref.Digest,
		Size:   ref.Size,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := foundation.Assets().Put(context.Background(), record); err != nil {
		t.Fatal(err)
	}
	if err := foundation.Objects().Forget(context.Background(), ref.Digest); err == nil {
		t.Fatal("Forget() removed an object with a durable Catalog reference")
	}
}

func TestObjectRepositoryRejectsLeaseForUnobservedObject(t *testing.T) {
	foundation, err := Open(context.Background(), testRoots(t))
	if err != nil {
		t.Fatal(err)
	}
	defer foundation.Close()
	ref := catalogTestBlob("missing-lease")
	if err := foundation.Objects().Lease(
		context.Background(),
		"missing",
		ref.Digest,
		"run",
		"run-1",
		time.Now().UTC().Add(time.Hour),
	); err == nil {
		t.Fatal("Lease() accepted an object outside the CAS inventory")
	}
}

func TestIndexedBlobStartupUsesCatalogQuotaForOneHundredThousandObjects(t *testing.T) {
	roots := testRoots(t)
	foundation, err := Open(context.Background(), roots)
	if err != nil {
		t.Fatal(err)
	}
	defer foundation.Close()
	tx, err := foundation.content.db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	statement, err := tx.Prepare(`
		INSERT INTO gc_objects(
			digest, size, physical_generation, state, observed_at
		) VALUES (?, 1, 1, 'active', ?)
	`)
	if err != nil {
		_ = tx.Rollback()
		t.Fatal(err)
	}
	observed := time.Date(2026, 7, 25, 9, 0, 0, 0, time.UTC).Format(time.RFC3339Nano)
	for index := range 100_000 {
		digest := catalogTestBlob(string(rune(index)) + "-" + time.Unix(int64(index), 0).String()).Digest
		if _, err := statement.Exec(digest.String(), observed); err != nil {
			_ = statement.Close()
			_ = tx.Rollback()
			t.Fatal(err)
		}
	}
	if err := statement.Close(); err != nil {
		_ = tx.Rollback()
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if _, err := blob.Open(
		roots.Objects,
		blob.Limits{MaxBlobBytes: 1 << 10, MaxTotalBytes: 200_000},
		foundation.Objects(),
	); err != nil {
		t.Fatalf("indexed startup with 100k object metadata rows: %v", err)
	}
}
