// Package workflowstore owns the durable Workflow Source and Program facts
// consumed by the Yotta 3.1 application command surface.
package workflowstore

import (
	"bytes"
	"context"
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
	sourceMarkerContents = "yotta/workflow-source-store/3.1\n"
)

var (
	ErrSourceNotFound = errors.New("workflow source not found")
	ErrSourceConflict = errors.New("workflow source revision conflict")
	ErrSourceChanged  = errors.New("workflow source changed outside store")
)

type InvalidSourceError struct{ Diagnostics []schema.Diagnostic }

func (e *InvalidSourceError) Error() string { return "workflow source is invalid" }

type SourceStoreOptions struct{ MaxSources int }

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
	mu      sync.RWMutex
	root    string
	max     int
	sources map[string]SourceSnapshot
}

func OpenSourceStore(root string, options SourceStoreOptions) (*SourceStore, error) {
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
	sources := make(map[string]SourceSnapshot)
	for _, entry := range entries {
		if entry.Name() == sourceMarker {
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
		snapshot, err := openSourceArtifact(raw, true)
		if err != nil {
			return nil, fmt.Errorf("open Workflow Source %q: %w", entry.Name(), err)
		}
		if entry.Name() != snapshot.workflowID+".json" {
			return nil, fmt.Errorf("workflow source %q has mismatched identity", entry.Name())
		}
		sources[snapshot.workflowID] = snapshot
		if len(sources) > options.MaxSources {
			return nil, errors.New("workflow source store exceeds source limit")
		}
	}
	return &SourceStore{root: resolved, max: options.MaxSources, sources: sources}, nil
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
		return nil, errors.New("store directory is not owned by Yotta 3.1")
	}
	return entries, nil
}

func abandonedStaging(name string) bool {
	return strings.HasPrefix(name, ".durable-") && strings.HasSuffix(name, ".tmp")
}
