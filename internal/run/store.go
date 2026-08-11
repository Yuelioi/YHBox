package run

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/runid"
	"github.com/yottaapp/yotta/internal/storage/catalog"
)

const LayoutVersion = "ledger-2"

var (
	ErrRunExists     = errors.New("run already exists")
	ErrRunNotFound   = errors.New("run not found")
	ErrRunConflict   = errors.New("run generation conflict")
	ErrRunIdentity   = errors.New("run identity changed")
	ErrRunTransition = errors.New("invalid run state transition")
)

type StoreOptions struct {
	MaxRecords int
}

// CommitOutcome is retained as the admission publication contract. SQLite
// commits either fail before publication or return as crash-durable commits.
type CommitOutcome uint8

const (
	CommitNotApplied CommitOutcome = iota
	CommitPublished
	CommitDurable
)

// Store is the deep Run domain boundary. It validates canonical Records and
// successor transitions, while RunRepository owns their append-oriented
// summary/event/value representation.
type Store struct {
	mu         sync.Mutex
	repository *catalog.RunRepository
	catalog    datatype.ValueTypeCatalog
	max        int
}

func OpenStore(
	repository *catalog.RunRepository,
	valueCatalog datatype.ValueTypeCatalog,
	options StoreOptions,
) (*Store, error) {
	if repository == nil || valueCatalog == nil || options.MaxRecords <= 0 {
		return nil, errors.New("run store requires a Run repository, trusted type catalog, and positive record limit")
	}
	count, err := repository.Count(context.Background())
	if err != nil {
		return nil, fmt.Errorf("open Run Ledger: %w", err)
	}
	if count > options.MaxRecords {
		return nil, errors.New("run store exceeds record limit")
	}
	return &Store{
		repository: repository,
		catalog:    valueCatalog,
		max:        options.MaxRecords,
	}, nil
}

func (s *Store) Create(ctx context.Context, record Record) (CommitOutcome, error) {
	if ctx == nil {
		return CommitNotApplied, errors.New("run create requires a context")
	}
	if err := ctx.Err(); err != nil {
		return CommitNotApplied, err
	}
	if !record.Valid() || record.Status() != StatusQueued || record.Generation() != 1 {
		return CommitNotApplied, errors.New("run store can only create a queued generation-one record")
	}
	trusted, err := OpenRecord(record.Bytes(), s.catalog)
	if err != nil {
		return CommitNotApplied, fmt.Errorf("run store rejected untrusted record: %w", err)
	}
	if trusted.Digest() != record.Digest() {
		return CommitNotApplied, ErrRunIdentity
	}
	durable, err := ledgerRecord(trusted, s.catalog)
	if err != nil {
		return CommitNotApplied, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	count, err := s.repository.Count(ctx)
	if err != nil {
		return CommitNotApplied, err
	}
	if count >= s.max {
		return CommitNotApplied, errors.New("run store record limit reached")
	}
	if err := s.repository.Create(ctx, durable); err != nil {
		if errors.Is(err, catalog.ErrRunLedgerConflict) {
			return CommitNotApplied, ErrRunExists
		}
		return CommitNotApplied, err
	}
	return CommitDurable, nil
}

func (s *Store) Update(ctx context.Context, previous artifact.Digest, next Record) error {
	if ctx == nil {
		return errors.New("run update requires a context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if !previous.Valid() || !next.Valid() {
		return errors.New("run store update requires valid record identities")
	}
	trusted, err := OpenRecord(next.Bytes(), s.catalog)
	if err != nil {
		return fmt.Errorf("run store rejected untrusted record: %w", err)
	}
	if trusted.Digest() != next.Digest() {
		return ErrRunIdentity
	}
	next = trusted

	s.mu.Lock()
	defer s.mu.Unlock()
	current, err := s.load(ctx, next.Admission().RunID)
	if err != nil {
		return err
	}
	if current.Digest() != previous || next.Generation() != current.Generation()+1 {
		return ErrRunConflict
	}
	if current.Admission() != next.Admission() {
		return ErrRunIdentity
	}
	if !validSuccessor(current, next) {
		return ErrRunTransition
	}

	if current.Status() == StatusRunning && next.Status() == StatusRunning {
		entry := next.state.document.Journal[len(next.state.document.Journal)-1]
		event, err := ledgerEvent(entry)
		if err != nil {
			return err
		}
		err = s.repository.AppendEvent(
			ctx, next.Admission().RunID,
			current.Generation(), current.Digest(),
			next.Generation(), next.Digest(), event, entry.OccurredAt,
		)
		return mapRunRepositoryError(err)
	}
	summary, err := ledgerSummary(next)
	if err != nil {
		return err
	}
	values, err := ledgerValues(next.state.document.Values, s.catalog)
	if err != nil {
		return err
	}
	err = s.repository.Transition(ctx, current.Generation(), current.Digest(), summary, values)
	return mapRunRepositoryError(err)
}

// appendJournal commits the next locally sealed journal generation without
// reconstructing the complete ledger from SQLite. JournalWriter is the single
// owner of current; the repository CAS still rejects a stale generation or
// digest, while validSuccessor protects the immutable in-memory prefix.
//
// The ordinary Update boundary deliberately re-opens untrusted Records. Doing
// that for every runtime fact made each append scan and validate the entire
// persisted timeline, so latency grew linearly during long-running workflows.
func (s *Store) appendJournal(ctx context.Context, current, next Record) error {
	if ctx == nil {
		return errors.New("run journal append requires a context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if !current.Valid() || !next.Valid() || current.Status() != StatusRunning || next.Status() != StatusRunning ||
		!validSuccessor(current, next) {
		return ErrRunTransition
	}
	entry := next.state.document.Journal[len(next.state.document.Journal)-1]
	event, err := ledgerEvent(entry)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	err = s.repository.AppendEvent(
		ctx, next.Admission().RunID,
		current.Generation(), current.Digest(),
		next.Generation(), next.Digest(), event, entry.OccurredAt,
	)
	return mapRunRepositoryError(err)
}

func (s *Store) Load(runID string) (Record, error) {
	if err := runid.Validate(runID); err != nil {
		return Record{}, err
	}
	return s.load(context.Background(), runID)
}

func (s *Store) load(ctx context.Context, runID string) (Record, error) {
	stored, err := s.repository.Get(ctx, runID)
	if err != nil {
		return Record{}, mapRunRepositoryError(err)
	}
	record, err := openLedgerRecord(stored, s.catalog)
	if err != nil {
		return Record{}, fmt.Errorf("open Run Ledger record: %w", errors.Join(err, ErrRunConflict))
	}
	return record, nil
}

func (s *Store) List() ([]Record, error) {
	stored, err := s.repository.List(context.Background())
	if err != nil {
		return nil, err
	}
	result := make([]Record, 0, len(stored))
	for _, item := range stored {
		record, err := openLedgerRecord(item, s.catalog)
		if err != nil {
			return nil, fmt.Errorf("open Run Ledger record: %w", errors.Join(err, ErrRunConflict))
		}
		result = append(result, record)
	}
	return result, nil
}

func (s *Store) TimelinePage(ctx context.Context, runID string, page, pageSize int) (TimelinePage, error) {
	if ctx == nil {
		return TimelinePage{}, errors.New("run timeline requires a context")
	}
	if err := runid.Validate(runID); err != nil {
		return TimelinePage{}, err
	}
	summaryRecord, eventPage, err := s.repository.TimelinePage(ctx, runID, page, pageSize)
	if err != nil {
		return TimelinePage{}, mapRunRepositoryError(err)
	}
	summary, err := openLedgerSummary(summaryRecord)
	if err != nil {
		return TimelinePage{}, fmt.Errorf("open Run Ledger summary: %w", errors.Join(err, ErrRunConflict))
	}
	entries := make([]JournalEntry, 0, len(eventPage.Events))
	for _, stored := range eventPage.Events {
		entry, err := openLedgerEvent(stored)
		if err != nil {
			return TimelinePage{}, fmt.Errorf("open Run Ledger event: %w", errors.Join(err, ErrRunConflict))
		}
		entries = append(entries, journalEntryView(entry))
	}
	return TimelinePage{
		Summary: summary, Entries: entries,
		Page: eventPage.Page, Pages: eventPage.Pages, Total: eventPage.Total,
	}, nil
}

func (s *Store) BlobReferences(ctx context.Context) ([]blob.BlobRef, error) {
	if ctx == nil {
		return nil, errors.New("run Blob inventory requires a context")
	}
	return s.repository.BlobReferences(ctx)
}

func (s *Store) ArchiveTerminal(ctx context.Context, endedBefore, archivedAt time.Time, limit int) (int, error) {
	return s.repository.ArchiveTerminal(ctx, endedBefore, archivedAt, limit)
}

func (s *Store) PurgeArchived(ctx context.Context, archivedBefore time.Time, limit int) (int, error) {
	return s.repository.PurgeArchived(ctx, archivedBefore, limit)
}

// InterruptRunning durably terminates every Run left running after process
// restart. It never replays effects or silently requeues work.
func (s *Store) InterruptRunning(ctx context.Context, at time.Time) ([]Record, error) {
	if ctx == nil {
		return nil, errors.New("run recovery requires a context")
	}
	if at.Location() != time.UTC {
		return nil, errors.New("run recovery timestamp must be UTC")
	}
	summaries, err := s.repository.ListByStatus(ctx, string(StatusRunning))
	if err != nil {
		return nil, err
	}
	updated := make([]Record, 0, len(summaries))
	for _, summary := range summaries {
		if err := ctx.Err(); err != nil {
			return updated, err
		}
		current, err := s.load(ctx, summary.RunID)
		if err != nil {
			return updated, err
		}
		if current.Status() != StatusRunning {
			continue
		}
		next, err := current.Interrupt(at, RunError{
			Code: "runtime.process_interrupted", Category: ErrorCategoryInfrastructure,
		})
		if err != nil {
			return updated, err
		}
		if err := s.Update(ctx, current.Digest(), next); err != nil {
			return updated, err
		}
		updated = append(updated, next)
	}
	return updated, nil
}

// CancelQueued durably closes Runs that were admitted but never handed to a
// live worker before the previous process stopped.
func (s *Store) CancelQueued(ctx context.Context, at time.Time) ([]Record, error) {
	if ctx == nil {
		return nil, errors.New("run recovery requires a context")
	}
	if at.Location() != time.UTC {
		return nil, errors.New("run recovery timestamp must be UTC")
	}
	summaries, err := s.repository.ListByStatus(ctx, string(StatusQueued))
	if err != nil {
		return nil, err
	}
	updated := make([]Record, 0, len(summaries))
	for _, summary := range summaries {
		if err := ctx.Err(); err != nil {
			return updated, err
		}
		current, err := s.load(ctx, summary.RunID)
		if err != nil {
			return updated, err
		}
		if current.Status() != StatusQueued {
			continue
		}
		next, err := current.Cancel(at)
		if err != nil {
			return updated, err
		}
		if err := s.Update(ctx, current.Digest(), next); err != nil {
			return updated, err
		}
		updated = append(updated, next)
	}
	return updated, nil
}

func mapRunRepositoryError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, catalog.ErrRunLedgerNotFound):
		return ErrRunNotFound
	case errors.Is(err, catalog.ErrRunLedgerConflict):
		return ErrRunConflict
	default:
		return err
	}
}

func validSuccessor(current, next Record) bool {
	if !current.Valid() || !next.Valid() ||
		current.Admission() != next.Admission() ||
		next.Generation() != current.Generation()+1 {
		return false
	}
	currentJournal := current.state.document.Journal
	nextJournal := next.state.document.Journal
	if current.Status() == StatusRunning && next.Status() == StatusRunning {
		if len(nextJournal) != len(currentJournal)+1 ||
			!reflect.DeepEqual(nextJournal[:len(currentJournal)], currentJournal) {
			return false
		}
		currentDocument := current.state.document
		nextDocument := next.state.document
		currentDocument.RecordDigest, nextDocument.RecordDigest = "", ""
		currentDocument.Generation, nextDocument.Generation = 0, 0
		currentDocument.Journal, nextDocument.Journal = nil, nil
		return reflect.DeepEqual(currentDocument, nextDocument)
	}
	if !reflect.DeepEqual(currentJournal, nextJournal) {
		return false
	}
	switch current.Status() {
	case StatusQueued:
		return next.Status() == StatusRunning || next.Status() == StatusCancelled
	case StatusRunning:
		switch next.Status() {
		case StatusSucceeded, StatusFailed, StatusCancelled, StatusInterrupted:
			return true
		}
	}
	return false
}
