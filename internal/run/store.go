package run

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/durablefs"
	"github.com/yottaapp/yotta/internal/runid"
)

const (
	runStoreMarker         = ".yotta-run-store"
	runStoreMarkerContents = "yotta/run-store/3.1\n"
)

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

// CommitOutcome distinguishes a Run that was never published, one whose
// authoritative directory entry was published but whose directory sync could
// not be confirmed, and one whose publication is crash-durable.
type CommitOutcome uint8

const (
	CommitNotApplied CommitOutcome = iota
	CommitPublished
	CommitDurable
)

// Store is the single durable owner of RunRecord generations. Each update is
// compare-and-swap against the previous record digest and atomically replaces
// one canonical record file.
type Store struct {
	mu      sync.RWMutex
	root    string
	catalog datatype.ValueTypeCatalog
	max     int
	records map[string]Record
}

func OpenStore(root string, catalog datatype.ValueTypeCatalog, options StoreOptions) (*Store, error) {
	if strings.TrimSpace(root) == "" || catalog == nil || options.MaxRecords <= 0 {
		return nil, errors.New("run store requires root, trusted type catalog, and positive record limit")
	}
	resolved, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(resolved, 0o700); err != nil {
		return nil, fmt.Errorf("create Run Store: %w", err)
	}
	entries, err := os.ReadDir(resolved)
	if err != nil {
		return nil, err
	}
	markerPath := filepath.Join(resolved, runStoreMarker)
	if len(entries) == 0 {
		if err := durablefs.WriteFile(markerPath, []byte(runStoreMarkerContents), 0o600); err != nil {
			return nil, fmt.Errorf("claim Run Store directory: %w", err)
		}
		entries, err = os.ReadDir(resolved)
		if err != nil {
			return nil, err
		}
	} else {
		markerInfo, statErr := os.Lstat(markerPath)
		marker, err := os.ReadFile(markerPath)
		if statErr != nil || markerInfo.Mode()&os.ModeSymlink != 0 || !markerInfo.Mode().IsRegular() || err != nil || string(marker) != runStoreMarkerContents {
			return nil, errors.New("run store directory is not owned by Yotta 3.1")
		}
	}
	records := make(map[string]Record)
	for _, entry := range entries {
		name := entry.Name()
		if name == runStoreMarker {
			continue
		}
		path := filepath.Join(resolved, name)
		if strings.HasPrefix(name, ".durable-") && strings.HasSuffix(name, ".tmp") {
			if err := durablefs.Remove(path); err != nil {
				return nil, fmt.Errorf("remove abandoned Run staging file: %w", err)
			}
			continue
		}
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || filepath.Ext(name) != ".json" {
			return nil, fmt.Errorf("run store contains invalid entry %q", name)
		}
		runID := strings.TrimSuffix(name, ".json")
		if err := runid.Validate(runID); err != nil {
			return nil, fmt.Errorf("run store contains invalid record name %q", name)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		record, err := OpenRecord(raw, catalog)
		if err != nil || record.Admission().RunID != runID {
			return nil, fmt.Errorf("open Run record %q: %w", name, errors.Join(err, ErrRunIdentity))
		}
		records[runID] = record
		if len(records) > options.MaxRecords {
			return nil, errors.New("run store exceeds record limit")
		}
	}
	return &Store{root: resolved, catalog: catalog, max: options.MaxRecords, records: records}, nil
}

func (s *Store) Create(ctx context.Context, record Record) (CommitOutcome, error) {
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
	record = trusted
	runID := record.Admission().RunID
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.records[runID]; exists {
		return CommitNotApplied, ErrRunExists
	}
	if len(s.records) >= s.max {
		return CommitNotApplied, errors.New("run store record limit reached")
	}
	if _, err := os.Lstat(s.recordPath(runID)); err == nil || !os.IsNotExist(err) {
		return CommitNotApplied, ErrRunExists
	}
	err = durablefs.WriteFile(s.recordPath(runID), record.Bytes(), 0o600)
	if err == nil {
		s.records[runID] = record
		return CommitDurable, nil
	}
	if durablefs.Committed(err) {
		s.records[runID] = record
		return CommitPublished, err
	}
	return CommitNotApplied, err
}

func (s *Store) Update(ctx context.Context, previous artifact.Digest, next Record) error {
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
	admission := next.Admission()
	s.mu.Lock()
	defer s.mu.Unlock()
	current, exists := s.records[admission.RunID]
	if !exists {
		return ErrRunNotFound
	}
	raw, err := os.ReadFile(s.recordPath(admission.RunID))
	if err != nil {
		return err
	}
	durable, err := OpenRecord(raw, s.catalog)
	if err != nil || durable.Digest() != current.Digest() || !bytes.Equal(durable.Bytes(), current.Bytes()) {
		return fmt.Errorf("run store record changed outside store: %w", errors.Join(err, ErrRunConflict))
	}
	if current.Digest() != previous || next.Generation() != current.Generation()+1 {
		return ErrRunConflict
	}
	if current.Admission() != admission {
		return ErrRunIdentity
	}
	if !validSuccessor(current, next) {
		return ErrRunTransition
	}
	err = durablefs.WriteFile(s.recordPath(admission.RunID), next.Bytes(), 0o600)
	if err == nil || durablefs.Committed(err) {
		s.records[admission.RunID] = next
	}
	return err
}

func (s *Store) Load(runID string) (Record, error) {
	if err := runid.Validate(runID); err != nil {
		return Record{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, ok := s.records[runID]
	path := s.recordPath(runID)
	catalog := s.catalog
	if !ok {
		return Record{}, ErrRunNotFound
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return Record{}, err
	}
	durable, err := OpenRecord(raw, catalog)
	if err != nil || durable.Digest() != record.Digest() || !bytes.Equal(durable.Bytes(), record.Bytes()) {
		return Record{}, fmt.Errorf("run store record changed outside store: %w", errors.Join(err, ErrRunConflict))
	}
	return durable, nil
}

func (s *Store) List() ([]Record, error) {
	s.mu.RLock()
	ids := make([]string, 0, len(s.records))
	for id := range s.records {
		ids = append(ids, id)
	}
	s.mu.RUnlock()
	sort.Strings(ids)
	result := make([]Record, 0, len(ids))
	for _, id := range ids {
		record, err := s.Load(id)
		if err != nil {
			return nil, err
		}
		result = append(result, record)
	}
	return result, nil
}

// InterruptRunning durably terminates every Run left running after process
// restart. It never replays effects or silently requeues work.
func (s *Store) InterruptRunning(ctx context.Context, at time.Time) ([]Record, error) {
	if at.Location() != time.UTC {
		return nil, errors.New("run recovery timestamp must be UTC")
	}
	s.mu.RLock()
	ids := make([]string, 0)
	for id, record := range s.records {
		if record.Status() == StatusRunning {
			ids = append(ids, id)
		}
	}
	s.mu.RUnlock()
	sort.Strings(ids)
	updated := make([]Record, 0, len(ids))
	for _, id := range ids {
		if err := ctx.Err(); err != nil {
			return updated, err
		}
		current, err := s.Load(id)
		if err != nil {
			return updated, err
		}
		if current.Status() != StatusRunning {
			continue
		}
		next, err := current.Interrupt(at, RunError{Code: "runtime.process_interrupted", Category: ErrorCategoryInfrastructure})
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
// live worker before the previous process stopped. Startup never guesses that
// a stale in-memory notification was delivered.
func (s *Store) CancelQueued(ctx context.Context, at time.Time) ([]Record, error) {
	if at.Location() != time.UTC {
		return nil, errors.New("run recovery timestamp must be UTC")
	}
	s.mu.RLock()
	ids := make([]string, 0)
	for id, record := range s.records {
		if record.Status() == StatusQueued {
			ids = append(ids, id)
		}
	}
	s.mu.RUnlock()
	sort.Strings(ids)
	updated := make([]Record, 0, len(ids))
	for _, id := range ids {
		if err := ctx.Err(); err != nil {
			return updated, err
		}
		current, err := s.Load(id)
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

func (s *Store) recordPath(runID string) string { return filepath.Join(s.root, runID+".json") }

func validSuccessor(current, next Record) bool {
	if !current.Valid() || !next.Valid() || current.Admission() != next.Admission() || next.Generation() != current.Generation()+1 {
		return false
	}
	currentJournal := current.state.document.Journal
	nextJournal := next.state.document.Journal
	if current.Status() == StatusRunning && next.Status() == StatusRunning {
		if len(nextJournal) != len(currentJournal)+1 || !reflect.DeepEqual(nextJournal[:len(currentJournal)], currentJournal) {
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
