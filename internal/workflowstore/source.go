// Package workflowstore owns the durable Workflow Source and Program facts
// consumed by the application command surface.
package workflowstore

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/durablefs"
	"github.com/yottaapp/yotta/internal/workflow/schema"
)

const (
	sourceMarker         = ".yotta-workflow-source-store"
	SourceLayoutVersion  = "1"
	sourceMarkerContents = "yotta/workflow-source-store/" + SourceLayoutVersion + "\n"
	sourceRecoveryDir    = ".recovery"
	sourceRecoverySchema = 1
)

var (
	ErrSourceNotFound = errors.New("workflow source not found")
	ErrSourceConflict = errors.New("workflow source revision conflict")
	ErrSourceChanged  = errors.New("workflow source changed outside store")
)

type InvalidSourceError struct{ Diagnostics []schema.Diagnostic }

func (e *InvalidSourceError) Error() string { return "workflow source is invalid" }

type SourceStoreOptions struct{ MaxSources int }

// SourceRecovery is an invalid Workflow Source isolated from the active
// store. The Store keeps the original bytes so a user can repair or delete one
// bad object without deleting the workspace.
type SourceRecovery struct {
	ID           artifact.Digest
	OriginalName string
	Reason       string
	raw          []byte
}

func (r SourceRecovery) Artifact() []byte { return append([]byte(nil), r.raw...) }

type sourceRecoveryEnvelope struct {
	SchemaVersion int             `json:"schemaVersion"`
	ID            artifact.Digest `json:"id"`
	OriginalName  string          `json:"originalName"`
	Reason        string          `json:"reason"`
	Artifact      []byte          `json:"artifact"`
}

type SourceSnapshot struct {
	workflowID string
	revision   int64
	hash       artifact.Digest
	raw        []byte
}

func (s SourceSnapshot) Valid() bool {
	return s.workflowID != "" && s.revision >= 0 && s.hash.Valid() && len(s.raw) != 0
}
func (s SourceSnapshot) WorkflowID() string    { return s.workflowID }
func (s SourceSnapshot) Revision() int64       { return s.revision }
func (s SourceSnapshot) Hash() artifact.Digest { return s.hash }
func (s SourceSnapshot) Artifact() []byte      { return append([]byte(nil), s.raw...) }

type SourceStore struct {
	mu         sync.RWMutex
	root       string
	max        int
	sources    map[string]SourceSnapshot
	recoveries map[artifact.Digest]SourceRecovery
}

func OpenSourceStore(root string, options SourceStoreOptions) (*SourceStore, error) {
	migrations, err := currentSourceMigrationPlan()
	if err != nil {
		return nil, err
	}
	return openSourceStore(root, options, migrations)
}

func openSourceStore(root string, options SourceStoreOptions, migrations sourceMigrationPlan) (*SourceStore, error) {
	if strings.TrimSpace(root) == "" || options.MaxSources <= 0 {
		return nil, errors.New("workflow source store requires root and positive source limit")
	}
	resolved, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(resolved, 0o700); err != nil {
		return nil, fmt.Errorf("create Workflow Source Store: %w", err)
	}
	entries, err := claimDirectory(resolved, sourceMarker, sourceMarkerContents)
	if err != nil {
		return nil, err
	}
	recoveries, err := openSourceRecoveries(resolved)
	if err != nil {
		return nil, err
	}
	sources := make(map[string]SourceSnapshot)
	for _, entry := range entries {
		if entry.Name() == sourceMarker || entry.Name() == sourceRecoveryDir {
			continue
		}
		path := filepath.Join(resolved, entry.Name())
		if abandonedStaging(entry.Name()) {
			if err := durablefs.Remove(path); err != nil {
				return nil, fmt.Errorf("remove abandoned Workflow Source staging file: %w", err)
			}
			continue
		}
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || filepath.Ext(entry.Name()) != ".json" {
			return nil, fmt.Errorf("workflow source store contains invalid entry %q", entry.Name())
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		candidate, migrated, err := migrations.Migrate(raw)
		if err != nil {
			if !errors.Is(err, errSourceMigrationUnavailable) {
				return nil, fmt.Errorf("migrate Workflow Source %q: %w", entry.Name(), err)
			}
			recovery, quarantineErr := quarantineSource(resolved, entry.Name(), raw, err)
			if quarantineErr != nil {
				return nil, fmt.Errorf("quarantine Workflow Source %q: %w", entry.Name(), quarantineErr)
			}
			recoveries[recovery.ID] = recovery
			continue
		}
		snapshot, err := openSourceArtifact(candidate, !migrated)
		if err != nil {
			if migrated {
				return nil, fmt.Errorf("validate migrated Workflow Source %q: %w", entry.Name(), err)
			}
			recovery, quarantineErr := quarantineSource(resolved, entry.Name(), raw, err)
			if quarantineErr != nil {
				return nil, fmt.Errorf("quarantine Workflow Source %q: %w", entry.Name(), quarantineErr)
			}
			recoveries[recovery.ID] = recovery
			continue
		}
		if entry.Name() != snapshot.workflowID+".json" {
			if migrated {
				return nil, fmt.Errorf("migrated Workflow Source %q changed its durable identity to %q", entry.Name(), snapshot.workflowID)
			}
			recovery, quarantineErr := quarantineSource(resolved, entry.Name(), raw, errors.New("workflow source filename does not match its identity"))
			if quarantineErr != nil {
				return nil, fmt.Errorf("quarantine Workflow Source %q: %w", entry.Name(), quarantineErr)
			}
			recoveries[recovery.ID] = recovery
			continue
		}
		if migrated {
			if err := durablefs.WriteFile(path, snapshot.raw, 0o600); err != nil {
				return nil, fmt.Errorf("publish migrated Workflow Source %q: %w", entry.Name(), err)
			}
		}
		sources[snapshot.workflowID] = snapshot
		if len(sources) > options.MaxSources {
			return nil, errors.New("workflow source store exceeds source limit")
		}
	}
	return &SourceStore{root: resolved, max: options.MaxSources, sources: sources, recoveries: recoveries}, nil
}

// ListRecoveries returns the isolated invalid Sources in stable order.
func (s *SourceStore) ListRecoveries() []SourceRecovery {
	s.mu.RLock()
	result := make([]SourceRecovery, 0, len(s.recoveries))
	for _, recovery := range s.recoveries {
		recovery.raw = append([]byte(nil), recovery.raw...)
		result = append(result, recovery)
	}
	s.mu.RUnlock()
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

// RepairRecovery validates replacement bytes against the current Source
// contract before atomically returning the object to the active store.
func (s *SourceStore) RepairRecovery(ctx context.Context, recoveryID artifact.Digest, raw []byte) (SourceSnapshot, error) {
	if ctx == nil || !recoveryID.Valid() {
		return SourceSnapshot{}, errors.New("workflow source recovery requires context and recovery identity")
	}
	if err := ctx.Err(); err != nil {
		return SourceSnapshot{}, err
	}
	next, err := openSourceArtifact(raw, false)
	if err != nil {
		return SourceSnapshot{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.recoveries[recoveryID]; !exists {
		return SourceSnapshot{}, errors.New("workflow source recovery not found")
	}
	if _, exists := s.sources[next.workflowID]; exists || len(s.sources) >= s.max {
		return SourceSnapshot{}, ErrSourceConflict
	}
	if _, statErr := os.Lstat(s.sourcePath(next.workflowID)); statErr == nil || !os.IsNotExist(statErr) {
		return SourceSnapshot{}, fmt.Errorf("%w: recovery destination exists", ErrSourceChanged)
	}
	writeErr := durablefs.WriteFile(s.sourcePath(next.workflowID), next.raw, 0o600)
	if writeErr != nil && !durablefs.Committed(writeErr) {
		return SourceSnapshot{}, writeErr
	}
	s.sources[next.workflowID] = next
	removeErr := durablefs.Remove(s.recoveryPath(recoveryID))
	if removeErr == nil || durablefs.Committed(removeErr) {
		delete(s.recoveries, recoveryID)
	}
	if writeErr != nil {
		return cloneSource(next), fmt.Errorf("workflow source repair committed without confirmed durability: %w", writeErr)
	}
	if removeErr != nil {
		return cloneSource(next), fmt.Errorf("remove repaired recovery object: %w", removeErr)
	}
	return cloneSource(next), nil
}

// DeleteRecovery removes one exact isolated object. It never touches healthy
// Sources or other workspace stores.
func (s *SourceStore) DeleteRecovery(ctx context.Context, recoveryID artifact.Digest) error {
	if ctx == nil || !recoveryID.Valid() {
		return errors.New("workflow source recovery delete requires context and recovery identity")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.recoveries[recoveryID]; !exists {
		return errors.New("workflow source recovery not found")
	}
	err := durablefs.Remove(s.recoveryPath(recoveryID))
	if err == nil || durablefs.Committed(err) {
		delete(s.recoveries, recoveryID)
	}
	return err
}

// Save applies one explicit revision transition. Creation requires
// baseRevision=-1 and source revision 0; updates require source revision to be
// exactly baseRevision+1.
func (s *SourceStore) Save(ctx context.Context, raw []byte, baseRevision int64) (SourceSnapshot, error) {
	if ctx == nil {
		return SourceSnapshot{}, errors.New("workflow source save context is required")
	}
	if err := ctx.Err(); err != nil {
		return SourceSnapshot{}, err
	}
	next, err := openSourceArtifact(raw, false)
	if err != nil {
		return SourceSnapshot{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	current, exists := s.sources[next.workflowID]
	if !exists {
		if baseRevision != -1 || next.revision != 0 || len(s.sources) >= s.max {
			return SourceSnapshot{}, ErrSourceConflict
		}
		if _, statErr := os.Lstat(s.sourcePath(next.workflowID)); statErr == nil || !os.IsNotExist(statErr) {
			return SourceSnapshot{}, fmt.Errorf("%w: unindexed source path", ErrSourceChanged)
		}
	} else {
		if baseRevision != current.revision || next.revision != current.revision+1 {
			return SourceSnapshot{}, ErrSourceConflict
		}
		if err := s.verifyLocked(current); err != nil {
			return SourceSnapshot{}, err
		}
	}
	err = durablefs.WriteFile(s.sourcePath(next.workflowID), next.raw, 0o600)
	if err == nil || durablefs.Committed(err) {
		s.sources[next.workflowID] = next
	}
	return cloneSource(next), err
}

func (s *SourceStore) Load(workflowID string) (SourceSnapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	current, ok := s.sources[workflowID]
	if !ok {
		return SourceSnapshot{}, ErrSourceNotFound
	}
	raw, err := os.ReadFile(s.sourcePath(workflowID))
	if err != nil {
		return SourceSnapshot{}, err
	}
	durable, err := openSourceArtifact(raw, true)
	if err != nil || durable.hash != current.hash || durable.revision != current.revision || !bytes.Equal(durable.raw, current.raw) {
		return SourceSnapshot{}, changedSource(err)
	}
	return cloneSource(durable), nil
}

func (s *SourceStore) List() []SourceSnapshot {
	s.mu.RLock()
	ids := make([]string, 0, len(s.sources))
	for id := range s.sources {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	result := make([]SourceSnapshot, 0, len(ids))
	for _, id := range ids {
		result = append(result, cloneSource(s.sources[id]))
	}
	s.mu.RUnlock()
	return result
}

// Delete removes one exact Source revision. The revision and hash form a CAS
// boundary so a library action cannot delete a Source edited after the user
// reviewed the confirmation dialog.
func (s *SourceStore) Delete(ctx context.Context, workflowID string, revision int64, hash artifact.Digest) error {
	if ctx == nil || strings.TrimSpace(workflowID) == "" || revision < 0 || !hash.Valid() {
		return errors.New("workflow source delete requires context and exact source identity")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	current, exists := s.sources[workflowID]
	if !exists {
		return ErrSourceNotFound
	}
	if current.revision != revision || current.hash != hash {
		return ErrSourceConflict
	}
	if err := s.verifyLocked(current); err != nil {
		return err
	}
	err := durablefs.Remove(s.sourcePath(workflowID))
	if err == nil || durablefs.Committed(err) {
		delete(s.sources, workflowID)
	}
	return err
}

func (s *SourceStore) verifyLocked(current SourceSnapshot) error {
	raw, err := os.ReadFile(s.sourcePath(current.workflowID))
	if err != nil {
		return err
	}
	durable, err := openSourceArtifact(raw, true)
	if err != nil || durable.hash != current.hash || !bytes.Equal(durable.raw, current.raw) {
		return changedSource(err)
	}
	return nil
}

func (s *SourceStore) sourcePath(workflowID string) string {
	return filepath.Join(s.root, workflowID+".json")
}

func (s *SourceStore) recoveryPath(recoveryID artifact.Digest) string {
	return filepath.Join(s.root, sourceRecoveryDir, strings.TrimPrefix(recoveryID.String(), "sha256:")+".json")
}

func openSourceRecoveries(root string) (map[artifact.Digest]SourceRecovery, error) {
	result := make(map[artifact.Digest]SourceRecovery)
	directory := filepath.Join(root, sourceRecoveryDir)
	info, err := os.Lstat(directory)
	if errors.Is(err, os.ErrNotExist) {
		return result, nil
	}
	if err != nil {
		return nil, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("workflow source recovery path is not a directory")
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || filepath.Ext(entry.Name()) != ".json" {
			return nil, fmt.Errorf("workflow source recovery contains invalid entry %q", entry.Name())
		}
		raw, err := os.ReadFile(filepath.Join(directory, entry.Name()))
		if err != nil {
			return nil, err
		}
		var envelope sourceRecoveryEnvelope
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&envelope); err != nil {
			return nil, fmt.Errorf("open Workflow Source recovery %q: %w", entry.Name(), err)
		}
		canonical, err := artifact.Marshal(envelope)
		if err != nil || !bytes.Equal(raw, canonical) {
			return nil, fmt.Errorf("open Workflow Source recovery %q: recovery envelope is not canonical", entry.Name())
		}
		if err := validateRecoveryEnvelope(entry.Name(), envelope); err != nil {
			return nil, fmt.Errorf("open Workflow Source recovery %q: %w", entry.Name(), err)
		}
		result[envelope.ID] = SourceRecovery{
			ID: envelope.ID, OriginalName: envelope.OriginalName,
			Reason: envelope.Reason, raw: append([]byte(nil), envelope.Artifact...),
		}
	}
	return result, nil
}

func quarantineSource(root, originalName string, raw []byte, cause error) (SourceRecovery, error) {
	identityInput := make([]byte, 0, len(originalName)+1+len(raw))
	identityInput = append(identityInput, originalName...)
	identityInput = append(identityInput, 0)
	identityInput = append(identityInput, raw...)
	id, err := artifact.Sum("yotta/workflow-source-recovery/v1", identityInput)
	if err != nil {
		return SourceRecovery{}, err
	}
	recovery := SourceRecovery{ID: id, OriginalName: originalName, Reason: cause.Error(), raw: append([]byte(nil), raw...)}
	envelope := sourceRecoveryEnvelope{
		SchemaVersion: sourceRecoverySchema, ID: id, OriginalName: originalName,
		Reason: recovery.Reason, Artifact: recovery.raw,
	}
	encoded, err := artifact.Marshal(envelope)
	if err != nil {
		return SourceRecovery{}, err
	}
	directory := filepath.Join(root, sourceRecoveryDir)
	if err := ensureRecoveryDirectory(directory); err != nil {
		return SourceRecovery{}, err
	}
	destination := filepath.Join(directory, strings.TrimPrefix(id.String(), "sha256:")+".json")
	if existing, readErr := os.ReadFile(destination); readErr == nil {
		if !bytes.Equal(existing, encoded) {
			return SourceRecovery{}, errors.New("workflow source recovery identity collision")
		}
	} else if !errors.Is(readErr, os.ErrNotExist) {
		return SourceRecovery{}, readErr
	} else if err := durablefs.WriteFile(destination, encoded, 0o600); err != nil {
		return SourceRecovery{}, err
	}
	if err := durablefs.Remove(filepath.Join(root, originalName)); err != nil {
		return SourceRecovery{}, err
	}
	return recovery, nil
}

func ensureRecoveryDirectory(directory string) error {
	info, err := os.Lstat(directory)
	if errors.Is(err, os.ErrNotExist) {
		return os.Mkdir(directory, 0o700)
	}
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("workflow source recovery path is not a directory")
	}
	return nil
}

func validateRecoveryEnvelope(filename string, envelope sourceRecoveryEnvelope) error {
	if envelope.SchemaVersion != sourceRecoverySchema || !envelope.ID.Valid() ||
		strings.TrimSpace(envelope.OriginalName) == "" || envelope.OriginalName != filepath.Base(envelope.OriginalName) ||
		strings.TrimSpace(envelope.Reason) == "" || len(envelope.Artifact) == 0 {
		return errors.New("workflow source recovery envelope is invalid")
	}
	if filename != strings.TrimPrefix(envelope.ID.String(), "sha256:")+".json" {
		return errors.New("workflow source recovery filename does not match identity")
	}
	identityInput := make([]byte, 0, len(envelope.OriginalName)+1+len(envelope.Artifact))
	identityInput = append(identityInput, envelope.OriginalName...)
	identityInput = append(identityInput, 0)
	identityInput = append(identityInput, envelope.Artifact...)
	want, err := artifact.Sum("yotta/workflow-source-recovery/v1", identityInput)
	if err != nil || want != envelope.ID {
		return errors.New("workflow source recovery content identity is invalid")
	}
	return nil
}

func openSourceArtifact(raw []byte, requireCanonical bool) (SourceSnapshot, error) {
	document, canonical, digest, diagnostics, err := schema.CanonicalSource(raw)
	if err != nil {
		return SourceSnapshot{}, err
	}
	if len(diagnostics) != 0 {
		return SourceSnapshot{}, &InvalidSourceError{Diagnostics: append([]schema.Diagnostic(nil), diagnostics...)}
	}
	if requireCanonical && !bytes.Equal(raw, canonical) {
		return SourceSnapshot{}, errors.New("workflow source artifact is not canonical")
	}
	return SourceSnapshot{
		workflowID: document.Workflow.ID, revision: document.Revision,
		hash: digest, raw: append([]byte(nil), canonical...),
	}, nil
}

func cloneSource(source SourceSnapshot) SourceSnapshot {
	source.raw = append([]byte(nil), source.raw...)
	return source
}

func changedSource(cause error) error {
	if cause == nil {
		return ErrSourceChanged
	}
	return fmt.Errorf("%w: %v", ErrSourceChanged, cause)
}

func claimDirectory(root, markerName, markerContents string) ([]os.DirEntry, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	markerPath := filepath.Join(root, markerName)
	if len(entries) == 0 {
		if err := durablefs.WriteFile(markerPath, []byte(markerContents), 0o600); err != nil {
			return nil, fmt.Errorf("claim store directory: %w", err)
		}
		return os.ReadDir(root)
	}
	info, statErr := os.Lstat(markerPath)
	marker, readErr := os.ReadFile(markerPath)
	if statErr != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || readErr != nil || string(marker) != markerContents {
		return nil, errors.New("store directory has an unsupported Workflow Source ownership marker")
	}
	return entries, nil
}

func abandonedStaging(name string) bool {
	return strings.HasPrefix(name, ".durable-") && strings.HasSuffix(name, ".tmp")
}
