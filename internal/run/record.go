package run

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/runid"
)

const (
	RecordFormat       = "yotta.run-record"
	RecordVersion      = "3.1"
	MaxRecordBytes     = 16 << 20
	recordDigestDomain = "yotta/run-record/v1"
)

var (
	runFieldPattern  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]{0,255}$`)
	errorCodePattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[._/-][a-z0-9]+)+$`)
)

type Status string

const (
	StatusQueued      Status = "queued"
	StatusRunning     Status = "running"
	StatusSucceeded   Status = "succeeded"
	StatusFailed      Status = "failed"
	StatusCancelled   Status = "cancelled"
	StatusInterrupted Status = "interrupted"
)

type Admission struct {
	RunID                string
	ProgramHash          artifact.Digest
	CatalogHash          artifact.Digest
	CapabilityPlanDigest artifact.Digest
	GrantDigest          artifact.Digest
	PolicyGeneration     string
	Principal            string
	QueuedAt             time.Time
}

type RunError struct {
	Code      string `json:"code"`
	Category  string `json:"category"`
	Retryable bool   `json:"retryable"`
	GraphID   string `json:"graphId,omitempty"`
	NodeID    string `json:"nodeId,omitempty"`
	Attempt   int    `json:"attempt,omitempty"`
}

const (
	ErrorCategoryNode           = "node"
	ErrorCategoryAdapter        = "adapter"
	ErrorCategoryPolicy         = "policy"
	ErrorCategoryInfrastructure = "infrastructure"
)

type ProducedValue struct {
	ValueID  string
	GraphID  string
	NodeID   string
	PortID   string
	Attempt  int
	Envelope datatype.ValueEnvelope
}

type durableValue struct {
	ValueID     string          `json:"valueId"`
	GraphID     string          `json:"graphId"`
	NodeID      string          `json:"nodeId"`
	PortID      string          `json:"portId"`
	Attempt     int             `json:"attempt"`
	ValueDigest artifact.Digest `json:"valueDigest"`
	Envelope    json.RawMessage `json:"envelope"`
}

type recordDocument struct {
	Format               string          `json:"format"`
	Version              string          `json:"version"`
	RecordDigest         artifact.Digest `json:"recordDigest"`
	RunID                string          `json:"runId"`
	Generation           uint64          `json:"generation"`
	ProgramHash          artifact.Digest `json:"programHash"`
	CatalogHash          artifact.Digest `json:"catalogHash"`
	CapabilityPlanDigest artifact.Digest `json:"capabilityPlanDigest"`
	GrantDigest          artifact.Digest `json:"grantDigest"`
	PolicyGeneration     string          `json:"policyGeneration"`
	Principal            string          `json:"principal"`
	Status               Status          `json:"status"`
	QueuedAt             time.Time       `json:"queuedAt"`
	StartedAt            *time.Time      `json:"startedAt,omitempty"`
	EndedAt              *time.Time      `json:"endedAt,omitempty"`
	Error                *RunError       `json:"error,omitempty"`
	Values               []durableValue  `json:"values"`
}

type recordState struct {
	document recordDocument
	bytes    []byte
}

type Record struct{ state *recordState }

func NewQueuedRecord(admission Admission) (Record, error) {
	document := recordDocument{
		Format: RecordFormat, Version: RecordVersion, RunID: admission.RunID, Generation: 1,
		ProgramHash: admission.ProgramHash, CatalogHash: admission.CatalogHash,
		CapabilityPlanDigest: admission.CapabilityPlanDigest, GrantDigest: admission.GrantDigest,
		PolicyGeneration: admission.PolicyGeneration, Principal: admission.Principal,
		Status: StatusQueued, QueuedAt: admission.QueuedAt, Values: []durableValue{},
	}
	return sealRecord(document, nil)
}

func OpenRecord(raw []byte, catalog datatype.ValueTypeCatalog) (Record, error) {
	if len(raw) == 0 || len(raw) > MaxRecordBytes || catalog == nil {
		return Record{}, errors.New("RunRecord exceeds byte budget or lacks a trusted type catalog")
	}
	if err := artifact.InspectJSONBudget(raw, 128, 1048576, 1<<20); err != nil {
		return Record{}, err
	}
	canonical, err := artifact.Canonicalize(raw)
	if err != nil || !bytes.Equal(canonical, raw) {
		return Record{}, errors.New("RunRecord is not canonical")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var document recordDocument
	if err := decoder.Decode(&document); err != nil {
		return Record{}, fmt.Errorf("decode RunRecord: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return Record{}, errors.New("RunRecord contains trailing values")
	}
	sealed, err := sealRecord(document, catalog)
	if err != nil {
		return Record{}, err
	}
	if sealed.Digest() != document.RecordDigest || !bytes.Equal(sealed.Bytes(), raw) {
		return Record{}, errors.New("RunRecord digest mismatch")
	}
	return sealed, nil
}

func sealRecord(document recordDocument, catalog datatype.ValueTypeCatalog) (Record, error) {
	if err := validateRecord(document, catalog); err != nil {
		return Record{}, err
	}
	body := document
	body.RecordDigest = ""
	bodyBytes, err := artifact.Marshal(body)
	if err != nil {
		return Record{}, err
	}
	digest, err := artifact.Sum(recordDigestDomain, bodyBytes)
	if err != nil {
		return Record{}, err
	}
	document.RecordDigest = digest
	raw, err := artifact.Marshal(document)
	if err != nil {
		return Record{}, err
	}
	if len(raw) > MaxRecordBytes {
		return Record{}, errors.New("RunRecord exceeds byte budget")
	}
	return Record{state: &recordState{document: document, bytes: raw}}, nil
}

func validateRecord(document recordDocument, catalog datatype.ValueTypeCatalog) error {
	if document.Format != RecordFormat || document.Version != RecordVersion || document.Generation == 0 ||
		runid.Validate(document.RunID) != nil || !runFieldPattern.MatchString(document.PolicyGeneration) || !runFieldPattern.MatchString(document.Principal) ||
		!document.ProgramHash.Valid() || !document.CatalogHash.Valid() || !document.CapabilityPlanDigest.Valid() || !document.GrantDigest.Valid() ||
		document.QueuedAt.Location() != time.UTC {
		return errors.New("invalid RunRecord identity")
	}
	if catalog == nil && len(document.Values) != 0 {
		return errors.New("trusted type catalog required for Run values")
	}
	switch document.Status {
	case StatusQueued:
		if document.StartedAt != nil || document.EndedAt != nil || document.Error != nil || len(document.Values) != 0 {
			return errors.New("malformed queued RunRecord")
		}
	case StatusRunning:
		if !validTimeAfter(document.StartedAt, document.QueuedAt) || document.EndedAt != nil || document.Error != nil || len(document.Values) != 0 {
			return errors.New("malformed running RunRecord")
		}
	case StatusSucceeded:
		if !validTerminalTimes(document) || document.Error != nil {
			return errors.New("malformed succeeded RunRecord")
		}
	case StatusFailed, StatusInterrupted:
		if !validTerminalTimes(document) || document.Error == nil || !validRunError(*document.Error) || len(document.Values) != 0 {
			return errors.New("malformed failed RunRecord")
		}
	case StatusCancelled:
		validTimes := document.StartedAt == nil && validTimeAfter(document.EndedAt, document.QueuedAt) ||
			document.StartedAt != nil && validTerminalTimes(document)
		if !validTimes || document.Error != nil || len(document.Values) != 0 {
			return errors.New("malformed cancelled RunRecord")
		}
	default:
		return errors.New("unsupported RunRecord status")
	}
	seen := make(map[string]struct{}, len(document.Values))
	for _, value := range document.Values {
		if !runFieldPattern.MatchString(value.ValueID) || !validAttribution(value.GraphID) || !validAttribution(value.NodeID) ||
			!runFieldPattern.MatchString(value.PortID) || value.Attempt < 1 || !value.ValueDigest.Valid() {
			return errors.New("invalid durable Run value provenance")
		}
		if _, duplicate := seen[value.ValueID]; duplicate {
			return errors.New("duplicate durable Run value id")
		}
		seen[value.ValueID] = struct{}{}
		envelope, err := datatype.OpenValueEnvelope(catalog, value.Envelope)
		if err != nil || !envelope.Durable() || envelope.Digest() != value.ValueDigest || envelope.Artifact() == nil {
			return errors.New("RunRecord contains invalid or runtime-only value authority")
		}
	}
	return nil
}

func validTimeAfter(value *time.Time, minimum time.Time) bool {
	return value != nil && value.Location() == time.UTC && !value.Before(minimum)
}
func validTerminalTimes(document recordDocument) bool {
	return validTimeAfter(document.StartedAt, document.QueuedAt) && document.EndedAt != nil && document.EndedAt.Location() == time.UTC && !document.EndedAt.Before(*document.StartedAt)
}
func validRunError(value RunError) bool {
	if !errorCodePattern.MatchString(value.Code) || !validErrorCategory(value.Category) || value.Attempt < 0 {
		return false
	}
	if value.GraphID == "" && value.NodeID == "" {
		return value.Attempt == 0
	}
	return validAttribution(value.GraphID) && validAttribution(value.NodeID) && value.Attempt > 0
}

func validErrorCategory(value string) bool {
	switch value {
	case ErrorCategoryNode, ErrorCategoryAdapter, ErrorCategoryPolicy, ErrorCategoryInfrastructure:
		return true
	default:
		return false
	}
}

func (r Record) Start(at time.Time) (Record, error) {
	if !r.Valid() || r.state.document.Status != StatusQueued || at.Location() != time.UTC || at.Before(r.state.document.QueuedAt) {
		return Record{}, errors.New("invalid Run start transition")
	}
	document := r.state.document
	document.Generation++
	document.Status = StatusRunning
	document.StartedAt = timePtr(at)
	return sealRecord(document, nil)
}

func (r Record) Succeed(at time.Time, catalog datatype.ValueTypeCatalog, produced []ProducedValue) (Record, error) {
	if !r.Valid() || r.state.document.Status != StatusRunning || !validEnd(at, r.state.document.StartedAt) || catalog == nil {
		return Record{}, errors.New("invalid Run success transition")
	}
	values := make([]durableValue, 0, len(produced))
	for _, value := range produced {
		if !value.Envelope.Valid() || !value.Envelope.Durable() || value.Envelope.Artifact() == nil {
			return Record{}, errors.New("run success cannot persist runtime-only authority")
		}
		values = append(values, durableValue{
			ValueID: value.ValueID, GraphID: value.GraphID, NodeID: value.NodeID, PortID: value.PortID, Attempt: value.Attempt,
			ValueDigest: value.Envelope.Digest(), Envelope: value.Envelope.Artifact(),
		})
	}
	sort.Slice(values, func(i, j int) bool { return values[i].ValueID < values[j].ValueID })
	document := r.state.document
	document.Generation++
	document.Status = StatusSucceeded
	document.EndedAt = timePtr(at)
	document.Values = values
	return sealRecord(document, catalog)
}

func (r Record) Fail(at time.Time, runError RunError) (Record, error) {
	return r.endWithError(StatusFailed, at, runError)
}
func (r Record) Interrupt(at time.Time, runError RunError) (Record, error) {
	return r.endWithError(StatusInterrupted, at, runError)
}
func (r Record) endWithError(status Status, at time.Time, runError RunError) (Record, error) {
	if !r.Valid() || r.state.document.Status != StatusRunning || !validEnd(at, r.state.document.StartedAt) || !validRunError(runError) {
		return Record{}, errors.New("invalid Run error transition")
	}
	document := r.state.document
	document.Generation++
	document.Status = status
	document.EndedAt = timePtr(at)
	document.Error = &runError
	document.Values = []durableValue{}
	return sealRecord(document, nil)
}

func (r Record) Cancel(at time.Time) (Record, error) {
	if !r.Valid() || at.Location() != time.UTC {
		return Record{}, errors.New("invalid Run cancel transition")
	}
	status := r.state.document.Status
	validTime := status == StatusQueued && !at.Before(r.state.document.QueuedAt) ||
		status == StatusRunning && validEnd(at, r.state.document.StartedAt)
	if !validTime {
		return Record{}, errors.New("invalid Run cancel transition")
	}
	document := r.state.document
	document.Generation++
	document.Status = StatusCancelled
	document.EndedAt = timePtr(at)
	document.Error = nil
	document.Values = []durableValue{}
	return sealRecord(document, nil)
}

func validEnd(at time.Time, started *time.Time) bool {
	return started != nil && at.Location() == time.UTC && !at.Before(*started)
}
func timePtr(value time.Time) *time.Time { clone := value; return &clone }

func validAttribution(value string) bool {
	if value == "" || len(value) > 256 || strings.TrimSpace(value) != value {
		return false
	}
	for _, character := range value {
		if unicode.IsControl(character) || character == '\u061c' || character == '\u200e' || character == '\u200f' ||
			character >= '\u202a' && character <= '\u202e' || character >= '\u2066' && character <= '\u2069' {
			return false
		}
	}
	return true
}

func (r Record) Valid() bool { return r.state != nil && r.state.document.RecordDigest.Valid() }
func (r Record) Digest() artifact.Digest {
	if !r.Valid() {
		return ""
	}
	return r.state.document.RecordDigest
}
func (r Record) Status() Status {
	if !r.Valid() {
		return ""
	}
	return r.state.document.Status
}
func (r Record) Generation() uint64 {
	if !r.Valid() {
		return 0
	}
	return r.state.document.Generation
}
func (r Record) Admission() Admission {
	if !r.Valid() {
		return Admission{}
	}
	document := r.state.document
	return Admission{
		RunID: document.RunID, ProgramHash: document.ProgramHash, CatalogHash: document.CatalogHash,
		CapabilityPlanDigest: document.CapabilityPlanDigest, GrantDigest: document.GrantDigest,
		PolicyGeneration: document.PolicyGeneration, Principal: document.Principal, QueuedAt: document.QueuedAt,
	}
}
func (r Record) Bytes() []byte {
	if !r.Valid() {
		return nil
	}
	return append([]byte(nil), r.state.bytes...)
}
