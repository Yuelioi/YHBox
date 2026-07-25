package catalog

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestRunRepositoryAppendsPagesRetainsAndInventoriesPayloads(t *testing.T) {
	foundation, err := Open(context.Background(), testRoots(t))
	if err != nil {
		t.Fatal(err)
	}
	defer foundation.Close()
	repository := foundation.Runs()
	ctx := context.Background()
	queuedAt := time.Date(2026, 7, 25, 8, 0, 0, 0, time.UTC)
	runID := "0190c7d4-1e40-7cc5-a783-57b16d5c8e3a"
	digest := func(generation int) string {
		return fmt.Sprintf("run-%d", generation)
	}
	first := RunSummaryRecord{
		RunID: runID, Generation: 1, Digest: catalogTestBlob(digest(1)).Digest,
		Status: "queued", QueuedAt: queuedAt, SummaryArtifact: []byte(`{"status":"queued"}`),
		UpdatedAt: queuedAt,
	}
	if err := repository.Create(ctx, RunLedgerRecord{Summary: first}); err != nil {
		t.Fatal(err)
	}
	startedAt := queuedAt.Add(time.Second)
	running := first
	running.Generation = 2
	running.Digest = catalogTestBlob(digest(2)).Digest
	running.Status = "running"
	running.StartedAt = &startedAt
	running.SummaryArtifact = []byte(`{"status":"running"}`)
	running.UpdatedAt = startedAt
	if err := repository.Transition(ctx, first.Generation, first.Digest, running, nil); err != nil {
		t.Fatal(err)
	}
	current := running
	for sequence := 1; sequence <= 3; sequence++ {
		occurredAt := startedAt.Add(time.Duration(sequence) * time.Second)
		nextDigest := catalogTestBlob(digest(sequence + 2)).Digest
		event := RunEventRecord{
			Sequence: uint64(sequence), Kind: "node-attempt", OccurredAt: occurredAt,
			Artifact: []byte(fmt.Sprintf(`{"sequence":%d}`, sequence)),
		}
		if err := repository.AppendEvent(
			ctx, runID, current.Generation, current.Digest,
			current.Generation+1, nextDigest, event, occurredAt,
		); err != nil {
			t.Fatal(err)
		}
		current.Generation++
		current.Digest = nextDigest
		current.JournalCount++
		current.UpdatedAt = occurredAt
	}
	if err := repository.AppendEvent(
		ctx, runID, running.Generation, running.Digest,
		running.Generation+1, catalogTestBlob("stale").Digest,
		RunEventRecord{
			Sequence: 1, Kind: "node-attempt", OccurredAt: startedAt.Add(time.Second),
			Artifact: []byte(`{"sequence":1}`),
		}, startedAt.Add(time.Second),
	); !errors.Is(err, ErrRunLedgerConflict) {
		t.Fatalf("stale AppendEvent = %v", err)
	}
	summary, page, err := repository.TimelinePage(ctx, runID, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Generation != current.Generation || page.Total != 3 ||
		page.Pages != 2 || len(page.Events) != 2 ||
		page.Events[0].Sequence != 2 || page.Events[1].Sequence != 3 {
		t.Fatalf("TimelinePage() = %#v / %#v", summary, page)
	}

	endedAt := startedAt.Add(5 * time.Second)
	terminal := current
	terminal.Generation++
	terminal.Digest = catalogTestBlob(digest(int(terminal.Generation))).Digest
	terminal.Status = "succeeded"
	terminal.EndedAt = &endedAt
	terminal.SummaryArtifact = []byte(`{"status":"succeeded"}`)
	terminal.UpdatedAt = endedAt
	payload := catalogTestBlob("run payload")
	values := []RunValueRecord{{
		Ordinal: 0, ValueID: "result", ValueDigest: catalogTestBlob("value").Digest,
		Artifact: []byte(`{"valueId":"result"}`), Blob: &payload,
	}}
	if err := repository.Transition(
		ctx, current.Generation, current.Digest, terminal, values,
	); err != nil {
		t.Fatal(err)
	}
	references, err := repository.BlobReferences(ctx)
	if err != nil || len(references) != 1 || references[0] != payload {
		t.Fatalf("BlobReferences() = %#v, %v", references, err)
	}
	archivedAt := endedAt.Add(time.Hour)
	archived, err := repository.ArchiveTerminal(ctx, endedAt.Add(time.Second), archivedAt, 10)
	if err != nil || archived != 1 {
		t.Fatalf("ArchiveTerminal() = %d, %v", archived, err)
	}
	purged, err := repository.PurgeArchived(ctx, archivedAt.Add(time.Second), 10)
	if err != nil || purged != 1 {
		t.Fatalf("PurgeArchived() = %d, %v", purged, err)
	}
	if _, err := repository.Get(ctx, runID); !errors.Is(err, ErrRunLedgerNotFound) {
		t.Fatalf("Get(purged) = %v", err)
	}
}

func TestRunRepositoryGetNeverMixesAnAppendingHeadAndEventSet(t *testing.T) {
	foundation, err := Open(context.Background(), testRoots(t))
	if err != nil {
		t.Fatal(err)
	}
	defer foundation.Close()
	repository := foundation.Runs()
	ctx := context.Background()
	startedAt := time.Date(2026, 7, 25, 9, 0, 0, 0, time.UTC)
	runID := "0190c7d4-1e40-7cc5-a783-57b16d5c8e3b"
	current := RunSummaryRecord{
		RunID: runID, Generation: 1, Digest: catalogTestBlob("snapshot-1").Digest,
		Status: "running", QueuedAt: startedAt, StartedAt: &startedAt,
		SummaryArtifact: []byte(`{"status":"running"}`), UpdatedAt: startedAt,
	}
	if err := repository.Create(ctx, RunLedgerRecord{Summary: current}); err != nil {
		t.Fatal(err)
	}

	const appends = 50
	var wait sync.WaitGroup
	wait.Add(1)
	writerDone := make(chan struct{})
	writerErrors := make(chan error, 1)
	go func() {
		defer wait.Done()
		defer close(writerDone)
		head := current
		for sequence := 1; sequence <= appends; sequence++ {
			at := startedAt.Add(time.Duration(sequence) * time.Millisecond)
			nextDigest := catalogTestBlob(fmt.Sprintf("snapshot-%d", sequence+1)).Digest
			err := repository.AppendEvent(
				ctx, runID, head.Generation, head.Digest,
				head.Generation+1, nextDigest,
				RunEventRecord{
					Sequence: uint64(sequence), Kind: "node-status", OccurredAt: at,
					Artifact: []byte(fmt.Sprintf(`{"sequence":%d}`, sequence)),
				}, at,
			)
			if err != nil {
				writerErrors <- err
				return
			}
			head.Generation++
			head.Digest = nextDigest
		}
	}()
	for {
		select {
		case <-writerDone:
			wait.Wait()
			select {
			case err := <-writerErrors:
				t.Fatal(err)
			default:
			}
			record, err := repository.Get(ctx, runID)
			if err != nil || len(record.Events) != appends ||
				record.Summary.JournalCount != appends {
				t.Fatalf("final Get() = %#v, %v", record.Summary, err)
			}
			return
		default:
			if _, err := repository.Get(ctx, runID); err != nil {
				t.Fatalf("concurrent Get() = %v", err)
			}
		}
	}
}
