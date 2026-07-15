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

func TestValidSuccessorRejectsJournalHistoryMutation(t *testing.T) {
	queuedAt := time.Date(2026, 7, 15, 3, 0, 0, 0, time.UTC)
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
	summary, err := NewRedactedSummary("node.execute", nil)
	if err != nil {
		t.Fatal(err)
	}
	fact, err := NewNodeAttemptFact(NodeAttemptInput{
		GraphPath: []string{"main"}, NodeID: "node-1", Attempt: 1, Outcome: AttemptStarted,
		OccurredAt: queuedAt.Add(2 * time.Second), Summary: summary,
	})
	if err != nil {
		t.Fatal(err)
	}
	withStart, err := running.AppendJournal(fact)
	if err != nil {
		t.Fatal(err)
	}
	forgedDocument := withStart.state.document
	forgedDocument.Journal = append([]journalEntry(nil), forgedDocument.Journal...)
	forgedDocument.Generation++
	forgedDocument.Journal[0].NodeID = "other-node"
	second, err := NewNodeAttemptFact(NodeAttemptInput{
		GraphPath: []string{"main"}, NodeID: "other-node", Attempt: 1, Outcome: AttemptSucceeded,
		OccurredAt: queuedAt.Add(3 * time.Second), Summary: summary,
	})
	if err != nil {
		t.Fatal(err)
	}
	secondEntry := second.entry
	secondEntry.Sequence = 2
	forgedDocument.Journal = append(forgedDocument.Journal, secondEntry)
	forged, err := sealRecord(forgedDocument, nil)
	if err != nil {
		t.Fatal(err)
	}
	if validSuccessor(withStart, forged) {
		t.Fatal("Run Store accepted a rewritten journal prefix")
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
