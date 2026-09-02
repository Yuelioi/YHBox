package run

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/storage/catalog"
)

const (
	LedgerSummaryFormat  = "yotta.run-summary"
	LedgerSummaryVersion = "1"
)

type ledgerSummaryDocument struct {
	Format               string          `json:"format"`
	Version              string          `json:"version"`
	RunID                string          `json:"runId"`
	WorkflowID           string          `json:"workflowId,omitempty"`
	SourceHash           artifact.Digest `json:"sourceHash,omitempty"`
	SourceRevision       int64           `json:"sourceRevision,omitempty"`
	ProgramHash          artifact.Digest `json:"programHash"`
	CatalogHash          artifact.Digest `json:"catalogHash"`
	CapabilityPlanDigest artifact.Digest `json:"capabilityPlanDigest"`
	GrantDigest          artifact.Digest `json:"grantDigest"`
	GrantArtifact        json.RawMessage `json:"grant"`
	PolicyGeneration     string          `json:"policyGeneration"`
	Principal            string          `json:"principal"`
	Status               Status          `json:"status"`
	QueuedAt             time.Time       `json:"queuedAt"`
	StartedAt            *time.Time      `json:"startedAt,omitempty"`
	EndedAt              *time.Time      `json:"endedAt,omitempty"`
	Error                *RunError       `json:"error,omitempty"`
}

// Summary is the bounded Run projection needed by timeline and list views.
// It intentionally excludes event and produced-value payloads.
type Summary struct {
	Admission  Admission
	Status     Status
	Generation uint64
	Digest     artifact.Digest
	Failure    *RunError
}

type TimelinePage struct {
	Summary Summary
	Entries []JournalEntry
	Page    int
	Pages   int
	Total   int
}

func ledgerRecord(record Record, valueCatalog datatype.ValueTypeCatalog) (catalog.RunLedgerRecord, error) {
	summary, err := ledgerSummary(record)
	if err != nil {
		return catalog.RunLedgerRecord{}, err
	}
	events := make([]catalog.RunEventRecord, 0, len(record.state.document.Journal))
	for _, entry := range record.state.document.Journal {
		event, err := ledgerEvent(entry)
		if err != nil {
			return catalog.RunLedgerRecord{}, err
		}
		events = append(events, event)
	}
	values, err := ledgerValues(record.state.document.Values, valueCatalog)
	if err != nil {
		return catalog.RunLedgerRecord{}, err
	}
	summary.JournalCount = uint64(len(events))
	return catalog.RunLedgerRecord{Summary: summary, Events: events, Values: values}, nil
}

func ledgerSummary(record Record) (catalog.RunSummaryRecord, error) {
	if !record.Valid() {
		return catalog.RunSummaryRecord{}, errors.New("run summary requires a valid Record")
	}
	document := record.state.document
	artifactBytes, err := artifact.Marshal(ledgerSummaryDocument{
		Format: LedgerSummaryFormat, Version: LedgerSummaryVersion,
		RunID: document.RunID, WorkflowID: document.WorkflowID, SourceHash: document.SourceHash, SourceRevision: document.SourceRevision,
		ProgramHash: document.ProgramHash,
		CatalogHash: document.CatalogHash, CapabilityPlanDigest: document.CapabilityPlanDigest,
		GrantDigest: document.GrantDigest, GrantArtifact: document.GrantArtifact,
		PolicyGeneration: document.PolicyGeneration, Principal: document.Principal,
		Status: document.Status, QueuedAt: document.QueuedAt, StartedAt: document.StartedAt,
		EndedAt: document.EndedAt, Error: document.Error,
	})
	if err != nil {
		return catalog.RunSummaryRecord{}, err
	}
	updatedAt := document.QueuedAt
	if document.StartedAt != nil {
		updatedAt = *document.StartedAt
	}
	if len(document.Journal) != 0 {
		updatedAt = document.Journal[len(document.Journal)-1].OccurredAt
	}
	if document.EndedAt != nil {
		updatedAt = *document.EndedAt
	}
	return catalog.RunSummaryRecord{
		RunID: document.RunID, Generation: document.Generation,
		Digest: document.RecordDigest, Status: string(document.Status),
		QueuedAt: document.QueuedAt, StartedAt: document.StartedAt,
		EndedAt: document.EndedAt, SummaryArtifact: artifactBytes,
		JournalCount: uint64(len(document.Journal)), UpdatedAt: updatedAt,
	}, nil
}

func ledgerEvent(entry journalEntry) (catalog.RunEventRecord, error) {
	raw, err := artifact.Marshal(entry)
	if err != nil {
		return catalog.RunEventRecord{}, err
	}
	return catalog.RunEventRecord{
		Sequence: entry.Sequence, Kind: string(entry.Kind),
		OccurredAt: entry.OccurredAt, Artifact: raw,
	}, nil
}

func ledgerValues(values []durableValue, valueCatalog datatype.ValueTypeCatalog) ([]catalog.RunValueRecord, error) {
	result := make([]catalog.RunValueRecord, 0, len(values))
	for index, value := range values {
		raw, err := artifact.Marshal(value)
		if err != nil {
			return nil, err
		}
		var reference *blob.BlobRef
		if valueCatalog != nil {
			envelope, err := datatype.OpenValueEnvelope(valueCatalog, value.Envelope)
			if err != nil {
				return nil, err
			}
			if found, ok := envelope.BlobRef(); ok {
				copy := found
				reference = &copy
			}
		}
		result = append(result, catalog.RunValueRecord{
			Ordinal: index, ValueID: value.ValueID, ValueDigest: value.ValueDigest,
			Artifact: raw, Blob: reference,
		})
	}
	return result, nil
}

func openLedgerRecord(record catalog.RunLedgerRecord, valueCatalog datatype.ValueTypeCatalog) (Record, error) {
	summary, err := openLedgerSummaryDocument(record.Summary.SummaryArtifact)
	if err != nil {
		return Record{}, err
	}
	events := make([]journalEntry, 0, len(record.Events))
	for _, stored := range record.Events {
		entry, err := openLedgerEvent(stored)
		if err != nil {
			return Record{}, err
		}
		events = append(events, entry)
	}
	values := make([]durableValue, 0, len(record.Values))
	for _, stored := range record.Values {
		value, err := openLedgerValue(stored)
		if err != nil {
			return Record{}, err
		}
		values = append(values, value)
	}
	document := summary.recordDocument(record.Summary.Generation, record.Summary.Digest, events, values)
	sealed, err := sealRecord(document, valueCatalog)
	if err != nil {
		return Record{}, err
	}
	if sealed.Digest() != record.Summary.Digest {
		return Record{}, errors.New("run Ledger head digest mismatch")
	}
	return sealed, nil
}

func openLedgerSummary(record catalog.RunSummaryRecord) (Summary, error) {
	document, err := openLedgerSummaryDocument(record.SummaryArtifact)
	if err != nil {
		return Summary{}, err
	}
	probe := document.recordDocument(record.Generation, record.Digest, []journalEntry{}, []durableValue{})
	if _, err := sealRecord(probe, nil); err != nil {
		return Summary{}, err
	}
	if document.RunID != record.RunID || document.Status != Status(record.Status) ||
		!document.QueuedAt.Equal(record.QueuedAt) ||
		!equalRunTime(document.StartedAt, record.StartedAt) ||
		!equalRunTime(document.EndedAt, record.EndedAt) {
		return Summary{}, errors.New("run Ledger summary columns disagree with its artifact")
	}
	var failure *RunError
	if document.Error != nil {
		copy := *document.Error
		failure = &copy
	}
	return Summary{
		Admission: Admission{
			RunID: document.RunID, WorkflowID: document.WorkflowID, SourceHash: document.SourceHash, SourceRevision: document.SourceRevision,
			ProgramHash: document.ProgramHash,
			CatalogHash: document.CatalogHash, CapabilityPlanDigest: document.CapabilityPlanDigest,
			GrantDigest: document.GrantDigest, PolicyGeneration: document.PolicyGeneration,
			Principal: document.Principal, QueuedAt: document.QueuedAt,
		},
		Status: document.Status, Generation: record.Generation,
		Digest: record.Digest, Failure: failure,
	}, nil
}

func openLedgerSummaryDocument(raw []byte) (ledgerSummaryDocument, error) {
	var document ledgerSummaryDocument
	if err := decodeCanonicalLedgerArtifact(raw, &document); err != nil {
		return ledgerSummaryDocument{}, fmt.Errorf("open Run summary: %w", err)
	}
	if document.Format != LedgerSummaryFormat || document.Version != LedgerSummaryVersion {
		return ledgerSummaryDocument{}, errors.New("unsupported Run summary artifact")
	}
	return document, nil
}

func openLedgerEvent(stored catalog.RunEventRecord) (journalEntry, error) {
	var entry journalEntry
	if err := decodeCanonicalLedgerArtifact(stored.Artifact, &entry); err != nil {
		return journalEntry{}, fmt.Errorf("open Run event: %w", err)
	}
	if validateJournalFact(entry) != nil || entry.Sequence != stored.Sequence ||
		string(entry.Kind) != stored.Kind || !entry.OccurredAt.Equal(stored.OccurredAt) {
		return journalEntry{}, errors.New("run event columns disagree with its artifact")
	}
	return entry, nil
}

func openLedgerValue(stored catalog.RunValueRecord) (durableValue, error) {
	var value durableValue
	if err := decodeCanonicalLedgerArtifact(stored.Artifact, &value); err != nil {
		return durableValue{}, fmt.Errorf("open Run value: %w", err)
	}
	if value.ValueID != stored.ValueID || value.ValueDigest != stored.ValueDigest {
		return durableValue{}, errors.New("run value columns disagree with its artifact")
	}
	return value, nil
}

func decodeCanonicalLedgerArtifact(raw []byte, destination any) error {
	if len(raw) == 0 || len(raw) > MaxRecordBytes {
		return errors.New("run Ledger artifact exceeds byte budget")
	}
	canonical, err := artifact.Canonicalize(raw)
	if err != nil || !bytes.Equal(canonical, raw) {
		return errors.New("run Ledger artifact is not canonical")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return errors.New("run Ledger artifact contains trailing values")
	}
	return nil
}

func (d ledgerSummaryDocument) recordDocument(
	generation uint64,
	digest artifact.Digest,
	journal []journalEntry,
	values []durableValue,
) recordDocument {
	return recordDocument{
		Format: RecordFormat, Version: RecordVersion, RecordDigest: digest,
		RunID: d.RunID, WorkflowID: d.WorkflowID, SourceHash: d.SourceHash, SourceRevision: d.SourceRevision,
		Generation: generation, ProgramHash: d.ProgramHash,
		CatalogHash: d.CatalogHash, CapabilityPlanDigest: d.CapabilityPlanDigest,
		GrantDigest: d.GrantDigest, GrantArtifact: append([]byte(nil), d.GrantArtifact...),
		PolicyGeneration: d.PolicyGeneration, Principal: d.Principal,
		Status: d.Status, QueuedAt: d.QueuedAt, StartedAt: cloneRunTime(d.StartedAt),
		EndedAt: cloneRunTime(d.EndedAt), Error: cloneRunError(d.Error),
		Journal: journal, Values: values,
	}
}

func journalEntryView(entry journalEntry) JournalEntry {
	counters := make(map[string]int64, len(entry.Summary.Counters))
	for _, counter := range entry.Summary.Counters {
		counters[counter.Name] = counter.Value
	}
	facts := make(map[string]string, len(entry.Summary.Facts))
	for _, fact := range entry.Summary.Facts {
		facts[fact.Name] = fact.Value
	}
	return JournalEntry{
		Sequence: entry.Sequence, Kind: entry.Kind,
		GraphPath: append([]string(nil), entry.GraphPath...),
		NodeID:    entry.NodeID, EffectID: entry.EffectID, Attempt: entry.Attempt,
		Action: entry.Action, AttemptOutcome: entry.AttemptOutcome,
		ActionOutcome: entry.ActionOutcome, OccurredAt: entry.OccurredAt,
		ErrorCode: entry.ErrorCode, StatusCode: entry.StatusCode,
		ErrorParams:    append(json.RawMessage(nil), entry.ErrorParams...),
		StatusCategory: entry.StatusCategory,
		Summary: RedactedSummaryView{
			Code: entry.Summary.Code, Counters: counters, Facts: facts,
		},
	}
}

func cloneRunTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func cloneRunError(value *RunError) *RunError {
	if value == nil {
		return nil
	}
	copy := *value
	copy.Params = append(json.RawMessage(nil), value.Params...)
	return &copy
}

func equalRunTime(left, right *time.Time) bool {
	if left == nil || right == nil {
		return left == right
	}
	return left.Equal(*right)
}
