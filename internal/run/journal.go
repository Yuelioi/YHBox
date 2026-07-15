package run

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/durablefs"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

const MaxJournalEntries = 65536

var ErrJournalOrder = errors.New("invalid Run journal order")

type JournalKind string

const (
	JournalNodeAttempt   JournalKind = "node-attempt"
	JournalAdapterAction JournalKind = "adapter-action"
	JournalNodeStatus    JournalKind = "node-status"
)

type AttemptOutcome string

const (
	AttemptStarted   AttemptOutcome = "started"
	AttemptSucceeded AttemptOutcome = "succeeded"
	AttemptFailed    AttemptOutcome = "failed"
	AttemptCancelled AttemptOutcome = "cancelled"
	AttemptRouted    AttemptOutcome = "routed"
)

type ActionOutcome string

const (
	ActionSucceeded ActionOutcome = "succeeded"
	ActionFailed    ActionOutcome = "failed"
	ActionCancelled ActionOutcome = "cancelled"
)

type summaryCounter struct {
	Name  string `json:"name"`
	Value int64  `json:"value"`
}

type summaryFact struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type redactedSummaryDocument struct {
	Code     string           `json:"code"`
	Counters []summaryCounter `json:"counters"`
	Facts    []summaryFact    `json:"facts"`
}

type RedactedSummary struct{ document redactedSummaryDocument }

type RedactedSummaryView struct {
	Code     string
	Counters map[string]int64
	Facts    map[string]string
}

func NewRedactedSummary(code string, counters map[string]int64, facts map[string]string) (RedactedSummary, error) {
	if !errorCodePattern.MatchString(code) || len(counters) > 64 || len(facts) > 64 {
		return RedactedSummary{}, errors.New("invalid redacted summary")
	}
	names := make([]string, 0, len(counters))
	for name, value := range counters {
		if !runFieldPattern.MatchString(name) || value < 0 {
			return RedactedSummary{}, errors.New("invalid redacted summary counter")
		}
		names = append(names, name)
	}
	sort.Strings(names)
	factNames := make([]string, 0, len(facts))
	for name, value := range facts {
		if !runFieldPattern.MatchString(name) || !validSummaryFactValue(value) {
			return RedactedSummary{}, errors.New("invalid redacted summary fact")
		}
		factNames = append(factNames, name)
	}
	sort.Strings(factNames)
	document := redactedSummaryDocument{
		Code: code, Counters: make([]summaryCounter, 0, len(names)), Facts: make([]summaryFact, 0, len(factNames)),
	}
	for _, name := range names {
		document.Counters = append(document.Counters, summaryCounter{Name: name, Value: counters[name]})
	}
	for _, name := range factNames {
		document.Facts = append(document.Facts, summaryFact{Name: name, Value: facts[name]})
	}
	return RedactedSummary{document: document}, nil
}

type NodeAttemptInput struct {
	GraphPath  []string
	NodeID     string
	Attempt    int
	Outcome    AttemptOutcome
	OccurredAt time.Time
	ErrorCode  string
	Summary    RedactedSummary
}

type AdapterActionInput struct {
	GraphPath  []string
	NodeID     string
	EffectID   string
	Attempt    int
	Action     string
	Outcome    ActionOutcome
	OccurredAt time.Time
	ErrorCode  string
	Summary    RedactedSummary
}

type NodeStatusInput struct {
	GraphPath  []string
	NodeID     string
	Attempt    int
	Code       string
	Category   nodecontract.StatusCategory
	OccurredAt time.Time
	Summary    RedactedSummary
}

type journalEntry struct {
	Sequence       uint64                      `json:"sequence"`
	Kind           JournalKind                 `json:"kind"`
	GraphPath      []string                    `json:"graphPath"`
	NodeID         string                      `json:"nodeId"`
	EffectID       string                      `json:"effectId,omitempty"`
	Attempt        int                         `json:"attempt"`
	Action         string                      `json:"action,omitempty"`
	AttemptOutcome AttemptOutcome              `json:"attemptOutcome,omitempty"`
	ActionOutcome  ActionOutcome               `json:"actionOutcome,omitempty"`
	OccurredAt     time.Time                   `json:"occurredAt"`
	ErrorCode      string                      `json:"errorCode,omitempty"`
	StatusCode     string                      `json:"statusCode,omitempty"`
	StatusCategory nodecontract.StatusCategory `json:"statusCategory,omitempty"`
	Summary        redactedSummaryDocument     `json:"summary"`
}

type JournalFact struct{ entry journalEntry }

type JournalEntry struct {
	Sequence       uint64
	Kind           JournalKind
	GraphPath      []string
	NodeID         string
	EffectID       string
	Attempt        int
	Action         string
	AttemptOutcome AttemptOutcome
	ActionOutcome  ActionOutcome
	OccurredAt     time.Time
	ErrorCode      string
	StatusCode     string
	StatusCategory nodecontract.StatusCategory
	Summary        RedactedSummaryView
}

// JournalWriter is the single CAS writer for one running RunRecord journal.
// It never retries conflicts because another writer means the Run has lost
// its single-owner invariant.
type JournalWriter struct {
	mu      sync.Mutex
	store   *Store
	current Record
}

func (s *Store) OpenJournal(runID string) (*JournalWriter, error) {
	current, err := s.Load(runID)
	if err != nil {
		return nil, err
	}
	if current.Status() != StatusRunning {
		return nil, errors.New("run journal requires a running record")
	}
	return &JournalWriter{store: s, current: current}, nil
}

func (w *JournalWriter) Append(ctx context.Context, fact JournalFact) (Record, error) {
	if w == nil || w.store == nil || ctx == nil {
		return Record{}, errors.New("run journal writer is required")
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	next, err := w.current.AppendJournal(fact)
	if err != nil {
		return Record{}, err
	}
	return w.commitLocked(ctx, next)
}

func (w *JournalWriter) Current() Record {
	if w == nil {
		return Record{}
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.current
}

func (w *JournalWriter) Succeed(ctx context.Context, at time.Time, catalog datatype.ValueTypeCatalog, produced []ProducedValue) (Record, error) {
	if w == nil || w.store == nil || ctx == nil {
		return Record{}, errors.New("run journal writer is required")
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	next, err := w.current.Succeed(at, catalog, produced)
	if err != nil {
		return Record{}, err
	}
	return w.commitLocked(ctx, next)
}

func (w *JournalWriter) Fail(ctx context.Context, at time.Time, runError RunError) (Record, error) {
	if w == nil || w.store == nil || ctx == nil {
		return Record{}, errors.New("run journal writer is required")
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	next, err := w.current.Fail(at, runError)
	if err != nil {
		return Record{}, err
	}
	return w.commitLocked(ctx, next)
}

func (w *JournalWriter) Cancel(ctx context.Context, at time.Time) (Record, error) {
	if w == nil || w.store == nil || ctx == nil {
		return Record{}, errors.New("run journal writer is required")
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	next, err := w.current.Cancel(at)
	if err != nil {
		return Record{}, err
	}
	return w.commitLocked(ctx, next)
}

func (w *JournalWriter) commitLocked(ctx context.Context, next Record) (Record, error) {
	err := w.store.Update(ctx, w.current.Digest(), next)
	if err == nil || durablefs.Committed(err) {
		w.current = next
	}
	return next, err
}

func NewNodeAttemptFact(input NodeAttemptInput) (JournalFact, error) {
	entry := journalEntry{Kind: JournalNodeAttempt, GraphPath: append([]string(nil), input.GraphPath...), NodeID: input.NodeID,
		Attempt: input.Attempt, AttemptOutcome: input.Outcome, OccurredAt: input.OccurredAt, ErrorCode: input.ErrorCode,
		Summary: cloneSummary(input.Summary.document)}
	if err := validateJournalFact(entry); err != nil {
		return JournalFact{}, err
	}
	return JournalFact{entry: entry}, nil
}

func NewAdapterActionFact(input AdapterActionInput) (JournalFact, error) {
	entry := journalEntry{Kind: JournalAdapterAction, GraphPath: append([]string(nil), input.GraphPath...), NodeID: input.NodeID,
		EffectID: input.EffectID, Attempt: input.Attempt, Action: input.Action, ActionOutcome: input.Outcome,
		OccurredAt: input.OccurredAt, ErrorCode: input.ErrorCode, Summary: cloneSummary(input.Summary.document)}
	if err := validateJournalFact(entry); err != nil {
		return JournalFact{}, err
	}
	return JournalFact{entry: entry}, nil
}

func NewNodeStatusFact(input NodeStatusInput) (JournalFact, error) {
	entry := journalEntry{
		Kind: JournalNodeStatus, GraphPath: append([]string(nil), input.GraphPath...), NodeID: input.NodeID,
		Attempt: input.Attempt, OccurredAt: input.OccurredAt, StatusCode: input.Code, StatusCategory: input.Category,
		Summary: cloneSummary(input.Summary.document),
	}
	if err := validateJournalFact(entry); err != nil {
		return JournalFact{}, err
	}
	return JournalFact{entry: entry}, nil
}

func validateJournalFact(entry journalEntry) error {
	if len(entry.GraphPath) == 0 || len(entry.GraphPath) > 32 || !attributionPattern.MatchString(entry.NodeID) || entry.Attempt < 1 || entry.OccurredAt.Location() != time.UTC || !validSummary(entry.Summary) {
		return errors.New("invalid Run journal attribution")
	}
	for _, graphID := range entry.GraphPath {
		if !attributionPattern.MatchString(graphID) {
			return errors.New("invalid Run journal graph path")
		}
	}
	switch entry.Kind {
	case JournalNodeAttempt:
		if entry.EffectID != "" || entry.Action != "" || entry.ActionOutcome != "" || entry.StatusCode != "" || entry.StatusCategory != "" ||
			!validAttemptOutcome(entry.AttemptOutcome) || !validOutcomeError(string(entry.AttemptOutcome), entry.ErrorCode) {
			return errors.New("invalid NodeAttempt fact")
		}
	case JournalAdapterAction:
		if !validAttribution(entry.EffectID) || !runFieldPattern.MatchString(entry.Action) || entry.AttemptOutcome != "" || entry.StatusCode != "" || entry.StatusCategory != "" ||
			!validActionOutcome(entry.ActionOutcome) || !validOutcomeError(string(entry.ActionOutcome), entry.ErrorCode) {
			return errors.New("invalid AdapterAction fact")
		}
	case JournalNodeStatus:
		if entry.EffectID != "" || entry.Action != "" || entry.AttemptOutcome != "" || entry.ActionOutcome != "" || entry.ErrorCode != "" ||
			!errorCodePattern.MatchString(entry.StatusCode) || !validStatusCategory(entry.StatusCategory) {
			return errors.New("invalid NodeStatus fact")
		}
	default:
		return errors.New("invalid Run journal kind")
	}
	return nil
}

func validateJournal(entries []journalEntry, startedAt *time.Time, requireClosed, requireSucceeded bool) error {
	if len(entries) > MaxJournalEntries || len(entries) > 0 && startedAt == nil {
		return ErrJournalOrder
	}
	active := map[string]struct{}{}
	latest := map[string]int{}
	terminal := map[string]AttemptOutcome{}
	actions := map[string]struct{ failed, cancelled bool }{}
	var previous time.Time
	for index, entry := range entries {
		if entry.Sequence != uint64(index+1) || validateJournalFact(entry) != nil || entry.OccurredAt.Before(*startedAt) || index > 0 && entry.OccurredAt.Before(previous) {
			return ErrJournalOrder
		}
		previous = entry.OccurredAt
		nodeKey := strings.Join(entry.GraphPath, "\x00") + "\x00" + entry.NodeID
		attemptKey := fmt.Sprintf("%s\x00%d", nodeKey, entry.Attempt)
		if entry.Kind == JournalNodeAttempt {
			switch entry.AttemptOutcome {
			case AttemptStarted:
				if _, exists := active[attemptKey]; exists || entry.Attempt != latest[nodeKey]+1 {
					return ErrJournalOrder
				}
				active[attemptKey] = struct{}{}
				latest[nodeKey] = entry.Attempt
			default:
				if _, exists := active[attemptKey]; !exists {
					return ErrJournalOrder
				}
				actionState := actions[attemptKey]
				if entry.AttemptOutcome == AttemptSucceeded && (actionState.failed || actionState.cancelled) ||
					entry.AttemptOutcome == AttemptCancelled && actionState.failed {
					return ErrJournalOrder
				}
				delete(active, attemptKey)
				terminal[nodeKey] = entry.AttemptOutcome
			}
		} else if entry.Kind == JournalAdapterAction {
			if _, exists := active[attemptKey]; !exists {
				return ErrJournalOrder
			}
			actionState := actions[attemptKey]
			actionState.failed = actionState.failed || entry.ActionOutcome == ActionFailed
			actionState.cancelled = actionState.cancelled || entry.ActionOutcome == ActionCancelled
			actions[attemptKey] = actionState
		} else if _, exists := active[attemptKey]; !exists {
			return ErrJournalOrder
		}
	}
	if requireClosed && len(active) != 0 {
		return ErrJournalOrder
	}
	if requireSucceeded {
		for _, outcome := range terminal {
			if outcome != AttemptSucceeded && outcome != AttemptRouted {
				return ErrJournalOrder
			}
		}
	}
	return nil
}

func validSummary(summary redactedSummaryDocument) bool {
	if !errorCodePattern.MatchString(summary.Code) || len(summary.Counters) > 64 || len(summary.Facts) > 64 {
		return false
	}
	previous := ""
	for _, counter := range summary.Counters {
		if !runFieldPattern.MatchString(counter.Name) || counter.Value < 0 || counter.Name <= previous {
			return false
		}
		previous = counter.Name
	}
	previous = ""
	for _, fact := range summary.Facts {
		if !runFieldPattern.MatchString(fact.Name) || !validSummaryFactValue(fact.Value) || fact.Name <= previous {
			return false
		}
		previous = fact.Name
	}
	return true
}

func validSummaryFactValue(value string) bool {
	if len(value) == 0 || len(value) > 256 {
		return false
	}
	for _, character := range value {
		if character < 0x20 || character > 0x7e {
			return false
		}
	}
	return true
}

func validAttemptOutcome(value AttemptOutcome) bool {
	return value == AttemptStarted || value == AttemptSucceeded || value == AttemptFailed || value == AttemptCancelled || value == AttemptRouted
}
func validActionOutcome(value ActionOutcome) bool {
	return value == ActionSucceeded || value == ActionFailed || value == ActionCancelled
}
func validOutcomeError(outcome, code string) bool {
	if outcome == string(AttemptFailed) || outcome == string(AttemptRouted) || outcome == string(ActionFailed) {
		return errorCodePattern.MatchString(code)
	}
	return code == ""
}

func validStatusCategory(category nodecontract.StatusCategory) bool {
	switch category {
	case nodecontract.StatusProgress, nodecontract.StatusWaiting, nodecontract.StatusConnection:
		return true
	default:
		return false
	}
}
func cloneSummary(source redactedSummaryDocument) redactedSummaryDocument {
	return redactedSummaryDocument{
		Code: source.Code, Counters: append([]summaryCounter(nil), source.Counters...), Facts: append([]summaryFact(nil), source.Facts...),
	}
}
