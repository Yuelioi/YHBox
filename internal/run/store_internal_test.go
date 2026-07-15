package run

import (
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/artifact"
)

func TestValidSuccessorRejectsSkippedAndTerminalTransitions(t *testing.T) {
	queuedAt := time.Date(2026, 7, 15, 1, 0, 0, 0, time.UTC)
	queued, err := NewQueuedRecord(Admission{
		RunID: "0190c7d4-1e40-7cc5-a783-57b16d5c8e3a", ProgramHash: testDigest(t, "program"),
		CatalogHash: testDigest(t, "catalog"), CapabilityPlanDigest: testDigest(t, "plan"), GrantDigest: testDigest(t, "grant"),
		PolicyGeneration: "policy-1", Principal: "user-1", QueuedAt: queuedAt,
	})
	if err != nil {
		t.Fatal(err)
	}
	running, err := queued.Start(queuedAt.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	failed, err := running.Fail(queuedAt.Add(2*time.Second), RunError{Code: "node.failed", Category: ErrorCategoryNode, GraphID: "main", NodeID: "node-1", Attempt: 1})
	if err != nil {
		t.Fatal(err)
	}
	if !validSuccessor(queued, running) || !validSuccessor(running, failed) {
		t.Fatal("valid RunRecord transition was rejected")
	}
	forgedDocument := failed.state.document
	forgedDocument.Generation = queued.Generation() + 1
	forged, err := sealRecord(forgedDocument, nil)
	if err != nil {
		t.Fatal(err)
	}
	if validSuccessor(queued, forged) {
		t.Fatal("queued RunRecord skipped directly to failed")
	}
	terminalDocument := failed.state.document
	terminalDocument.Generation++
	terminalSuccessor, err := sealRecord(terminalDocument, nil)
	if err != nil {
		t.Fatal(err)
	}
	if validSuccessor(failed, terminalSuccessor) {
		t.Fatal("terminal RunRecord accepted a successor")
	}
}

func testDigest(t *testing.T, label string) artifact.Digest {
	t.Helper()
	digest, err := artifact.Sum("test/run-store-internal/v1", []byte(label))
	if err != nil {
		t.Fatal(err)
	}
	return digest
}
