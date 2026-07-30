package run

import (
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/stream"
)

func TestValidSuccessorRejectsSkippedAndTerminalTransitions(t *testing.T) {
	queuedAt := time.Date(2026, 7, 15, 1, 0, 0, 0, time.UTC)
	queued := queuedRecordInternal(t, queuedAt)
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
	queued := queuedRecordInternal(t, queuedAt)
	running, err := queued.Start(queuedAt.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	summary, err := NewRedactedSummary("node.execute", nil, nil)
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

type testCapabilityCatalog map[string]capability.Definition

func (c testCapabilityCatalog) LookupCapability(id string) (capability.Definition, bool) {
	definition, ok := c[id]
	return definition, ok
}

func queuedRecordInternal(t *testing.T, queuedAt time.Time) Record {
	t.Helper()
	const capabilityID = "https://schemas.yotta.dev/capabilities/test/stream/v1"
	definition, err := capability.SealDefinition(capability.DefinitionDraft{
		CapabilityID: capabilityID, Operations: []string{stream.OperationSend}, TargetKinds: []string{"stream-session"},
		ScopeSchemaRoot: capabilityID + "/scope", ScopeSchemaBundle: []datatype.SchemaResource{{
			ID: capabilityID + "/scope", Schema: []byte(`{"$id":"https://schemas.yotta.dev/capabilities/test/stream/v1/scope","$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","additionalProperties":false}`),
		}}, Credential: capability.CredentialNone, Risk: capability.RiskLow, Consent: capability.ConsentNone,
		ProviderABI: stream.ProviderABI,
	})
	if err != nil {
		t.Fatal(err)
	}
	requirement, err := definition.NormalizeRequirement(capability.Requirement{
		ID: "stream", Capability: definition.Ref(), Operations: []string{stream.OperationSend}, TargetSlot: "stream", Scope: []byte(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := capability.SealPlan([]capability.PlanEntry{{GraphID: "main", NodeID: "node-1", Requirement: requirement}})
	if err != nil {
		t.Fatal(err)
	}
	grant, err := capability.SealRunGrant(capability.GrantRequest{
		ProgramHash: testDigest(t, "program"), Plan: plan, RunID: "0190c7d4-1e40-7cc5-a783-57b16d5c8e3a",
		Principal: "user-1", PolicyGeneration: "policy-1", IssuedAt: queuedAt,
		Bindings: []capability.Binding{{
			GraphID: "main", NodeID: "node-1", RequirementID: "stream", ProviderID: stream.ProviderID,
			ProviderArtifactDigest: testDigest(t, "provider"), ProviderABI: stream.ProviderABI,
			TargetID: "memory", TargetKind: "stream-session", ResourceKind: stream.Kind, PluginInstanceID: "builtin", SessionID: "session-1",
		}},
	}, testCapabilityCatalog{definition.Ref().CapabilityID: definition})
	if err != nil {
		t.Fatal(err)
	}
	record, err := NewQueuedRecord(QueueRequest{
		ProgramHash: testDigest(t, "program"), CatalogHash: testDigest(t, "catalog"), CapabilityPlanDigest: plan.Digest(), Grant: grant, QueuedAt: queuedAt,
	})
	if err != nil {
		t.Fatal(err)
	}
	return record
}

func testDigest(t *testing.T, label string) artifact.Digest {
	t.Helper()
	digest, err := artifact.Sum("test/run-store-internal/v1", []byte(label))
	if err != nil {
		t.Fatal(err)
	}
	return digest
}
