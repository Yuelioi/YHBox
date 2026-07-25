package catalog

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/blob"
)

func TestWorkflowRepositoryCommitsRevisionAndCASReferencesAtomically(t *testing.T) {
	foundation, err := Open(context.Background(), testRoots(t))
	if err != nil {
		t.Fatal(err)
	}
	defer foundation.Close()
	ctx := context.Background()
	repository := foundation.Workflows()
	objects := foundation.Objects()
	ref := catalogTestBlob("workflow-resource")
	if err := objects.Observe(ctx, blob.Object{Digest: ref.Digest, Size: ref.Size}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	created := testWorkflowRecord(t, "workflow-one", 0, "created", now)
	if err := repository.Commit(ctx, -1, created, []WorkflowReference{{Role: "blob/000000", Blob: ref}}); err != nil {
		t.Fatal(err)
	}
	plan, err := objects.PlanGC(ctx, nil, now.Add(time.Hour), 0)
	if err != nil || len(plan.Candidates) != 0 || plan.LiveCount != 1 {
		t.Fatalf("referenced object GC plan = %#v, %v", plan, err)
	}

	staleCreation := testWorkflowRecord(t, "workflow-one", 0, "stale", now.Add(time.Minute))
	if err := repository.Commit(ctx, -1, staleCreation, nil); !errors.Is(err, ErrWorkflowSourceConflict) {
		t.Fatalf("stale creation error = %v", err)
	}
	stale := testWorkflowRecord(t, "workflow-one", 1, "stale", now.Add(time.Minute))
	missing := catalogTestBlob("missing-workflow-resource")
	if err := repository.Commit(ctx, 0, stale, []WorkflowReference{{Role: "blob/000000", Blob: missing}}); err == nil {
		t.Fatal("commit accepted a reference absent from the active CAS inventory")
	}
	current, found, err := repository.Get(ctx, "workflow-one")
	if err != nil || !found || current.Revision != 0 || current.Hash != created.Hash {
		t.Fatalf("record after failed commit = %#v, %v, found=%v", current, err, found)
	}
	plan, err = objects.PlanGC(ctx, nil, now.Add(2*time.Hour), 0)
	if err != nil || len(plan.Candidates) != 0 || plan.LiveCount != 1 {
		t.Fatalf("failed commit changed CAS reachability = %#v, %v", plan, err)
	}
}

func testWorkflowRecord(
	t *testing.T,
	workflowID string,
	revision int64,
	label string,
	now time.Time,
) WorkflowSourceRecord {
	t.Helper()
	raw := []byte(label)
	hash, err := artifact.Sum("yotta/test/workflow-source/v1", raw)
	if err != nil {
		t.Fatal(err)
	}
	return WorkflowSourceRecord{
		WorkflowID: workflowID, Name: label, Revision: revision, Hash: hash,
		Format: "yotta.workflow", Version: "1", Artifact: raw,
		CreatedAt: now, UpdatedAt: now,
	}
}
