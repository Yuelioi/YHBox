package run_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecontract"
	run "github.com/yottaapp/yotta/internal/run"
	"github.com/yottaapp/yotta/internal/storage"
	storagecatalog "github.com/yottaapp/yotta/internal/storage/catalog"
	"github.com/yottaapp/yotta/internal/stream"
)

func TestRunStorePersistsGenerationsAndRejectsStaleUpdates(t *testing.T) {
	catalog, _ := stringValueCatalog(t)
	root := t.TempDir()
	store, err := openRunStore(t, root, catalog, run.StoreOptions{MaxRecords: 8})
	if err != nil {
		t.Fatal(err)
	}
	queuedAt := time.Date(2026, 7, 15, 1, 0, 0, 0, time.UTC)
	queued := queuedRecord(t, queuedAt)
	commit, err := store.Create(context.Background(), queued)
	if err != nil || commit != run.CommitDurable {
		t.Fatalf("Create commit = %v, error = %v", commit, err)
	}
	running, err := queued.Start(queuedAt.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Update(context.Background(), queued.Digest(), running); err != nil {
		t.Fatal(err)
	}
	if err := store.Update(context.Background(), queued.Digest(), running); !errors.Is(err, run.ErrRunConflict) {
		t.Fatalf("stale update = %v", err)
	}
	reopened, err := openRunStore(t, root, catalog, run.StoreOptions{MaxRecords: 8})
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := reopened.Load(testRunID)
	if err != nil || loaded.Digest() != running.Digest() || loaded.Generation() != 2 {
		t.Fatalf("loaded = %s generation %d, %v", loaded.Digest(), loaded.Generation(), err)
	}
}

func TestJournalAppendKeepsPersistedHeadConsistentAcrossTimeline(t *testing.T) {
	catalog, _ := stringValueCatalog(t)
	store, err := openRunStore(t, t.TempDir(), catalog, run.StoreOptions{MaxRecords: 8})
	if err != nil {
		t.Fatal(err)
	}
	queuedAt := time.Date(2026, 8, 1, 1, 0, 0, 0, time.UTC)
	queued := queuedRecord(t, queuedAt)
	if _, err := store.Create(context.Background(), queued); err != nil {
		t.Fatal(err)
	}
	running, err := queued.Start(queuedAt.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Update(context.Background(), queued.Digest(), running); err != nil {
		t.Fatal(err)
	}
	journal, err := store.OpenJournal(testRunID)
	if err != nil {
		t.Fatal(err)
	}
	summary, err := run.NewRedactedSummary("node.execute", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	started, err := run.NewNodeAttemptFact(run.NodeAttemptInput{
		GraphPath: []string{"main"}, NodeID: "latency-probe", Attempt: 1,
		Outcome: run.AttemptStarted, OccurredAt: queuedAt.Add(2 * time.Second), Summary: summary,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := journal.Append(context.Background(), started); err != nil {
		t.Fatal(err)
	}

	const samples = 160
	for index := range samples {
		fact, err := run.NewNodeStatusFact(run.NodeStatusInput{
			GraphPath: []string{"main"}, NodeID: "latency-probe", Attempt: 1,
			Code: "runtime.latency_probe", Category: nodecontract.StatusProgress,
			OccurredAt: queuedAt.Add(time.Duration(index+3) * time.Second), Summary: summary,
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := journal.Append(context.Background(), fact); err != nil {
			t.Fatal(err)
		}
	}
	current := journal.Current()
	persisted, err := store.Load(testRunID)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Generation() != current.Generation() || persisted.Digest() != current.Digest() {
		t.Fatalf("persisted head = generation %d digest %s, current = generation %d digest %s",
			persisted.Generation(), persisted.Digest(), current.Generation(), current.Digest())
	}
	if got, want := len(persisted.Journal()), samples+1; got != want {
		t.Fatalf("persisted journal entries = %d, want %d", got, want)
	}
}

func TestRunStoreImportsLegacyJSONRecordsIdempotently(t *testing.T) {
	valueCatalog, _ := stringValueCatalog(t)
	legacyRoot := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(legacyRoot, ".yotta-run-store"),
		[]byte("yotta/run-store/1\n"), 0o600,
	); err != nil {
		t.Fatal(err)
	}
	queued := queuedRecord(t, time.Date(2026, 7, 15, 1, 0, 0, 0, time.UTC))
	if err := os.WriteFile(
		filepath.Join(legacyRoot, testRunID+".json"),
		queued.Bytes(), 0o600,
	); err != nil {
		t.Fatal(err)
	}
	ledgerRoot := t.TempDir()
	options := run.StoreOptions{MaxRecords: 8}
	roots, err := storage.Resolve(filepath.Join(ledgerRoot, "profile"))
	if err != nil {
		t.Fatal(err)
	}
	foundation, err := storagecatalog.Open(context.Background(), roots)
	if err != nil {
		t.Fatal(err)
	}
	if err := run.ImportLegacyStore(
		context.Background(), foundation.Runs(), valueCatalog, legacyRoot, options.MaxRecords,
	); err != nil {
		_ = foundation.Close()
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(legacyRoot, testRunID+".json"),
		[]byte("{broken"), 0o600,
	); err != nil {
		_ = foundation.Close()
		t.Fatal(err)
	}
	if err := run.ImportLegacyStore(
		context.Background(), foundation.Runs(), valueCatalog, legacyRoot, options.MaxRecords,
	); err != nil {
		_ = foundation.Close()
		t.Fatalf("repeat import consulted retired Run JSON after durable marker: %v", err)
	}
	if err := foundation.Close(); err != nil {
		t.Fatal(err)
	}
	store, err := openRunStore(t, ledgerRoot, valueCatalog, options)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load(testRunID)
	if err != nil || loaded.Digest() != queued.Digest() {
		t.Fatalf("Load(imported) = %s, %v", loaded.Digest(), err)
	}
	if _, err := os.Stat(filepath.Join(legacyRoot, ".yotta-run-ledger-imported")); err != nil {
		t.Fatalf("legacy import marker: %v", err)
	}
	reopened, err := openRunStore(t, ledgerRoot, valueCatalog, options)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err = reopened.Load(testRunID)
	if err != nil || loaded.Digest() != queued.Digest() {
		t.Fatalf("Load(reopened) = %s, %v", loaded.Digest(), err)
	}
}

func TestRunStoreRejectsOutOfBandRecordMutation(t *testing.T) {
	catalog, _ := stringValueCatalog(t)
	root := t.TempDir()
	store, err := openRunStore(t, root, catalog, run.StoreOptions{MaxRecords: 8})
	if err != nil {
		t.Fatal(err)
	}
	queued := queuedRecord(t, time.Date(2026, 7, 15, 1, 0, 0, 0, time.UTC))
	if _, err := store.Create(context.Background(), queued); err != nil {
		t.Fatal(err)
	}
	running, err := queued.Start(queued.Admission().QueuedAt.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Update(context.Background(), queued.Digest(), running); err != nil {
		t.Fatal(err)
	}
	roots, err := storage.Resolve(filepath.Join(root, "profile"))
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", filepath.Join(roots.State, storagecatalog.RunFilename))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("UPDATE runs SET summary_artifact = CAST(summary_artifact || x'0A' AS BLOB) WHERE run_id = ?", testRunID); err != nil {
		_ = db.Close()
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(testRunID); !errors.Is(err, run.ErrRunConflict) {
		t.Fatalf("tampered Load = %v", err)
	}
	if _, err := store.InterruptRunning(context.Background(), queued.Admission().QueuedAt.Add(2*time.Second)); !errors.Is(err, run.ErrRunConflict) {
		t.Fatalf("tampered recovery = %v", err)
	}
}

func TestRunStoreRejectsTerminalRecordSealedAgainstAnotherCatalog(t *testing.T) {
	recordCatalog, definition := stringValueCatalog(t)
	store, err := openRunStore(t, t.TempDir(), valueCatalog{}, run.StoreOptions{MaxRecords: 8})
	if err != nil {
		t.Fatal(err)
	}
	queuedAt := time.Date(2026, 7, 15, 1, 0, 0, 0, time.UTC)
	queued := queuedRecord(t, queuedAt)
	if _, err := store.Create(context.Background(), queued); err != nil {
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
	succeeded, err := running.Succeed(queuedAt.Add(2*time.Second), recordCatalog, []run.ProducedValue{{
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
	store, err := openRunStore(t, t.TempDir(), catalog, run.StoreOptions{MaxRecords: 8})
	if err != nil {
		t.Fatal(err)
	}
	queuedAt := time.Date(2026, 7, 15, 1, 0, 0, 0, time.UTC)
	queued := queuedRecord(t, queuedAt)
	if _, err := store.Create(context.Background(), queued); err != nil {
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
	if err != nil || len(recovered) != 1 || recovered[0].Status() != run.StatusInterrupted {
		t.Fatalf("recovered = %#v, %v", recovered, err)
	}
	again, err := store.InterruptRunning(context.Background(), queuedAt.Add(3*time.Second))
	if err != nil || len(again) != 0 {
		t.Fatalf("second recovery = %#v, %v", again, err)
	}
}

func TestRunStoreCancelsUndeliveredQueuedRecordsWithoutStartingThem(t *testing.T) {
	catalog, _ := stringValueCatalog(t)
	queuedAt := time.Date(2026, 7, 15, 9, 0, 0, 0, time.UTC)
	store, err := openRunStore(t, t.TempDir(), catalog, run.StoreOptions{MaxRecords: 4})
	if err != nil {
		t.Fatal(err)
	}
	queued := queuedRecord(t, queuedAt)
	if outcome, err := store.Create(context.Background(), queued); err != nil || outcome != run.CommitDurable {
		t.Fatalf("Create = %v, %v", outcome, err)
	}
	cancelled, err := store.CancelQueued(context.Background(), queuedAt.Add(time.Second))
	if err != nil || len(cancelled) != 1 || cancelled[0].Status() != run.StatusCancelled || cancelled[0].Generation() != 2 {
		t.Fatalf("CancelQueued = %#v, %v", cancelled, err)
	}
	loaded, err := store.Load(queued.Admission().RunID)
	if err != nil || loaded.Status() != run.StatusCancelled || len(loaded.Journal()) != 0 {
		t.Fatalf("Load = %#v, %v", loaded, err)
	}
	if again, err := store.CancelQueued(context.Background(), queuedAt.Add(2*time.Second)); err != nil || len(again) != 0 {
		t.Fatalf("second CancelQueued = %#v, %v", again, err)
	}
}

func openRunStore(
	t *testing.T,
	root string,
	valueCatalog datatype.ValueTypeCatalog,
	options run.StoreOptions,
) (*run.Store, error) {
	t.Helper()
	roots, err := storage.Resolve(filepath.Join(root, "profile"))
	if err != nil {
		return nil, err
	}
	foundation, err := storagecatalog.Open(context.Background(), roots)
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() {
		if err := foundation.Close(); err != nil {
			t.Errorf("close test Run Ledger: %v", err)
		}
	})
	return run.OpenStore(foundation.Runs(), valueCatalog, options)
}

func queuedRecord(t *testing.T, queuedAt time.Time) run.Record {
	t.Helper()
	definition := streamCapability(t)
	plan := streamPlan(t, definition)
	grant, err := capability.SealRunGrant(capability.GrantRequest{
		ProgramHash: digest("program"), Plan: plan, RunID: testRunID, Principal: "user-1", PolicyGeneration: "policy-1",
		IssuedAt: queuedAt, Bindings: []capability.Binding{{
			GraphID: "main", NodeID: "producer", RequirementID: "stream", ProviderID: stream.ProviderID,
			ProviderArtifactDigest: streamProviderDigest(t), ProviderABI: stream.ProviderABI,
			TargetID: "memory", TargetKind: "stream-session", ResourceKind: stream.Kind, PluginInstanceID: "builtin", SessionID: "session-1",
		}},
	}, catalog{definition.Ref().CapabilityID: definition})
	if err != nil {
		t.Fatal(err)
	}
	record, err := run.NewQueuedRecord(run.QueueRequest{
		ProgramHash: digest("program"), CatalogHash: digest("catalog"), CapabilityPlanDigest: plan.Digest(), Grant: grant, QueuedAt: queuedAt,
	})
	if err != nil {
		t.Fatal(err)
	}
	return record
}
