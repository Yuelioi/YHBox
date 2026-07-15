package run_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/datatype"
	run31 "github.com/yottaapp/yotta/internal/run"
)

func TestRunStorePersistsGenerationsAndRejectsStaleUpdates(t *testing.T) {
	catalog, _ := stringValueCatalog(t)
	root := t.TempDir()
	store, err := run31.OpenStore(root, catalog, run31.StoreOptions{MaxRecords: 8})
	if err != nil {
		t.Fatal(err)
	}
	queuedAt := time.Date(2026, 7, 15, 1, 0, 0, 0, time.UTC)
	queued := queuedRecord(t, queuedAt)
	if err := store.Create(context.Background(), queued); err != nil {
		t.Fatal(err)
	}
	running, err := queued.Start(queuedAt.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Update(context.Background(), queued.Digest(), running); err != nil {
		t.Fatal(err)
	}
	if err := store.Update(context.Background(), queued.Digest(), running); !errors.Is(err, run31.ErrRunConflict) {
		t.Fatalf("stale update = %v", err)
	}
	reopened, err := run31.OpenStore(root, catalog, run31.StoreOptions{MaxRecords: 8})
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := reopened.Load(testRunID)
	if err != nil || loaded.Digest() != running.Digest() || loaded.Generation() != 2 {
		t.Fatalf("loaded = %s generation %d, %v", loaded.Digest(), loaded.Generation(), err)
	}
}

func TestRunStoreRejectsOutOfBandRecordMutation(t *testing.T) {
	catalog, _ := stringValueCatalog(t)
	root := t.TempDir()
	store, err := run31.OpenStore(root, catalog, run31.StoreOptions{MaxRecords: 8})
	if err != nil {
		t.Fatal(err)
	}
	queued := queuedRecord(t, time.Date(2026, 7, 15, 1, 0, 0, 0, time.UTC))
	if err := store.Create(context.Background(), queued); err != nil {
		t.Fatal(err)
	}
	running, err := queued.Start(queued.Admission().QueuedAt.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Update(context.Background(), queued.Digest(), running); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, testRunID+".json")
	if err := os.WriteFile(path, append(running.Bytes(), '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(testRunID); !errors.Is(err, run31.ErrRunConflict) {
		t.Fatalf("tampered Load = %v", err)
	}
	if _, err := store.InterruptRunning(context.Background(), queued.Admission().QueuedAt.Add(2*time.Second)); !errors.Is(err, run31.ErrRunConflict) {
		t.Fatalf("tampered recovery = %v", err)
	}
}

func TestRunStoreRejectsTerminalRecordSealedAgainstAnotherCatalog(t *testing.T) {
	recordCatalog, definition := stringValueCatalog(t)
	store, err := run31.OpenStore(t.TempDir(), valueCatalog{}, run31.StoreOptions{MaxRecords: 8})
	if err != nil {
		t.Fatal(err)
	}
	queuedAt := time.Date(2026, 7, 15, 1, 0, 0, 0, time.UTC)
	queued := queuedRecord(t, queuedAt)
	if err := store.Create(context.Background(), queued); err != nil {
		t.Fatal(err)
	}
	running, err := queued.Start(queuedAt.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Update(context.Background(), queued.Digest(), running); err != nil {
		t.Fatal(err)
	}
	envelope, err := datatype.SealInlineJSON(recordCatalog, datatype.RefResolvedType(definition.TypeRef()), []byte(`"done"`))
	if err != nil {
		t.Fatal(err)
	}
	succeeded, err := running.Succeed(queuedAt.Add(2*time.Second), recordCatalog, []run31.ProducedValue{{
		ValueID: "value-1", GraphID: "main", NodeID: "node-1", PortID: "result", Attempt: 1, Envelope: envelope,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Update(context.Background(), running.Digest(), succeeded); err == nil {
		t.Fatal("run store accepted a terminal record sealed against another catalog")
	}
}

func TestRunStoreInterruptsOrphanedRunningRecordsWithoutReplay(t *testing.T) {
	catalog, _ := stringValueCatalog(t)
	store, err := run31.OpenStore(t.TempDir(), catalog, run31.StoreOptions{MaxRecords: 8})
	if err != nil {
		t.Fatal(err)
	}
	queuedAt := time.Date(2026, 7, 15, 1, 0, 0, 0, time.UTC)
	queued := queuedRecord(t, queuedAt)
	if err := store.Create(context.Background(), queued); err != nil {
		t.Fatal(err)
	}
	running, err := queued.Start(queuedAt.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Update(context.Background(), queued.Digest(), running); err != nil {
		t.Fatal(err)
	}
	recovered, err := store.InterruptRunning(context.Background(), queuedAt.Add(2*time.Second))
	if err != nil || len(recovered) != 1 || recovered[0].Status() != run31.StatusInterrupted {
		t.Fatalf("recovered = %#v, %v", recovered, err)
	}
	again, err := store.InterruptRunning(context.Background(), queuedAt.Add(3*time.Second))
	if err != nil || len(again) != 0 {
		t.Fatalf("second recovery = %#v, %v", again, err)
	}
}

func queuedRecord(t *testing.T, queuedAt time.Time) run31.Record {
	t.Helper()
	record, err := run31.NewQueuedRecord(run31.Admission{
		RunID: testRunID, ProgramHash: digest("program"), CatalogHash: digest("catalog"),
		CapabilityPlanDigest: digest("plan"), GrantDigest: digest("grant"), PolicyGeneration: "policy-1",
		Principal: "user-1", QueuedAt: queuedAt,
	})
	if err != nil {
		t.Fatal(err)
	}
	return record
}
