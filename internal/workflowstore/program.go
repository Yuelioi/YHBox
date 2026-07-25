package workflowstore

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/configvalidator"
	"github.com/yottaapp/yotta/internal/durablefs"
	"github.com/yottaapp/yotta/internal/nodecatalog"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
)

const (
	programMarker         = ".yotta-program-cache"
	ProgramLayoutVersion  = "2"
	programMarkerContents = "yotta/program-cache/" + ProgramLayoutVersion + "\n"
)

var (
	ErrProgramNotFound = errors.New("program not found")
	ErrProgramChanged  = errors.New("program changed outside cache")
)

type ProgramStoreOptions struct {
	MaxPrograms int
	MaxBytes    int64
	Now         func() time.Time
}

type programCacheEntry struct {
	size       int64
	lastAccess time.Time
}

// ProgramStore is a rebuildable compiler cache scoped to one exact compiler
// build identity. It is never a backup or the authority for Workflow Source.
type ProgramStore struct {
	mu         sync.Mutex
	root       string
	max        int
	maxBytes   int64
	totalBytes int64
	now        func() time.Time
	catalog    nodecatalog.Snapshot
	build      artifact.Digest
	validators configvalidator.Registry
	programs   map[artifact.Digest]programCacheEntry
}

func OpenProgramStore(root string, catalog nodecatalog.Snapshot, validators configvalidator.Registry, build artifact.Digest, options ProgramStoreOptions) (*ProgramStore, error) {
	if strings.TrimSpace(root) == "" || !catalog.Valid() || !validators.Valid() || !build.Valid() ||
		options.MaxPrograms <= 0 || options.MaxBytes <= 0 {
		return nil, errors.New("program cache requires root, trusted Catalog/build, and positive count/byte quotas")
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	resolved, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(resolved, 0o700); err != nil {
		return nil, fmt.Errorf("create Program cache: %w", err)
	}
	entries, err := claimProgramCache(resolved)
	if err != nil {
		return nil, err
	}
	cacheIdentity, err := artifact.Sum(
		"yotta/program-cache-identity/v1",
		[]byte(build.String()+"\n"+catalog.Hash().String()),
	)
	if err != nil {
		return nil, fmt.Errorf("derive Program cache identity: %w", err)
	}
	generation := strings.TrimPrefix(cacheIdentity.String(), "sha256:")
	for _, entry := range entries {
		if entry.Name() == programMarker {
			continue
		}
		path := filepath.Join(resolved, entry.Name())
		if !entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || !validDigestHex(entry.Name()) {
			return nil, fmt.Errorf("program cache contains invalid generation %q", entry.Name())
		}
		if entry.Name() != generation {
			if err := os.RemoveAll(path); err != nil {
				return nil, fmt.Errorf("remove stale Program cache generation %q: %w", entry.Name(), err)
			}
		}
	}
	generationRoot := filepath.Join(resolved, generation)
	if err := os.Mkdir(generationRoot, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
		return nil, fmt.Errorf("create Program cache generation: %w", err)
	}
	info, err := os.Lstat(generationRoot)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("program cache generation is not a real directory")
	}
	store := &ProgramStore{
		root: generationRoot, max: options.MaxPrograms,
		maxBytes: options.MaxBytes, now: options.Now, catalog: catalog,
		validators: validators, build: build, programs: make(map[artifact.Digest]programCacheEntry),
	}
	if err := store.index(); err != nil {
		return nil, err
	}
	if err := store.evictLocked(0, ""); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *ProgramStore) Put(ctx context.Context, program compiler.ProgramSnapshot) error {
	if ctx == nil || !program.Valid() {
		return errors.New("program cache requires context and Program")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	trusted, err := compiler.OpenProgram(program.Artifact(), s.catalog, s.validators, s.build)
	if err != nil || trusted.Hash() != program.Hash() {
		return fmt.Errorf("program cache rejected untrusted Program: %w", err)
	}
	raw := trusted.Artifact()
	hash := trusted.Hash()
	if int64(len(raw)) > s.maxBytes {
		return errors.New("program exceeds cache byte quota")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.programs[hash]; ok {
		disk, err := os.ReadFile(s.programPath(hash))
		if err == nil && int64(len(disk)) == existing.size && bytes.Equal(disk, raw) {
			return s.touchLocked(hash, existing)
		}
		if removeErr := durablefs.Remove(s.programPath(hash)); removeErr != nil &&
			!errors.Is(removeErr, os.ErrNotExist) && !durablefs.Committed(removeErr) {
			return fmt.Errorf("replace corrupt Program cache entry: %w", removeErr)
		}
		delete(s.programs, hash)
		s.totalBytes -= existing.size
	}
	if err := s.evictLocked(int64(len(raw)), hash); err != nil {
		return err
	}
	path := s.programPath(hash)
	if _, statErr := os.Lstat(path); statErr == nil {
		if err := durablefs.Remove(path); err != nil && !durablefs.Committed(err) {
			return fmt.Errorf("remove unindexed Program cache entry: %w", err)
		}
	} else if !os.IsNotExist(statErr) {
		return statErr
	}
	err = durablefs.WriteFile(path, raw, 0o600)
	if err != nil && !durablefs.Committed(err) {
		return err
	}
	now := s.now().UTC()
	if touchErr := os.Chtimes(path, now, now); touchErr != nil {
		return errors.Join(err, touchErr)
	}
	s.programs[hash] = programCacheEntry{size: int64(len(raw)), lastAccess: now}
	s.totalBytes += int64(len(raw))
	return err
}

func (s *ProgramStore) Load(hash artifact.Digest) (compiler.ProgramSnapshot, error) {
	if !hash.Valid() {
		return compiler.ProgramSnapshot{}, errors.New("invalid program hash")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.programs[hash]
	if !ok {
		return compiler.ProgramSnapshot{}, ErrProgramNotFound
	}
	raw, err := os.ReadFile(s.programPath(hash))
	if err != nil {
		s.dropLocked(hash, entry)
		return compiler.ProgramSnapshot{}, ErrProgramNotFound
	}
	program, err := compiler.OpenProgram(raw, s.catalog, s.validators, s.build)
	if err != nil || program.Hash() != hash || int64(len(raw)) != entry.size {
		s.dropLocked(hash, entry)
		return compiler.ProgramSnapshot{}, ErrProgramNotFound
	}
	if err := s.touchLocked(hash, entry); err != nil {
		return compiler.ProgramSnapshot{}, err
	}
	return program, nil
}

func (s *ProgramStore) List() ([]compiler.ProgramSnapshot, error) {
	s.mu.Lock()
	hashes := make([]artifact.Digest, 0, len(s.programs))
	for hash := range s.programs {
		hashes = append(hashes, hash)
	}
	s.mu.Unlock()
	sort.Slice(hashes, func(i, j int) bool { return hashes[i] < hashes[j] })
	result := make([]compiler.ProgramSnapshot, 0, len(hashes))
	for _, hash := range hashes {
		program, err := s.Load(hash)
		if errors.Is(err, ErrProgramNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		result = append(result, program)
	}
	return result, nil
}

func (s *ProgramStore) index() error {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		path := filepath.Join(s.root, entry.Name())
		if abandonedStaging(entry.Name()) {
			if err := durablefs.Remove(path); err != nil {
				return fmt.Errorf("remove abandoned Program staging file: %w", err)
			}
			continue
		}
		hash, ok := digestFromProgramName(entry.Name())
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || !ok {
			if err := durablefs.Remove(path); err != nil {
				return fmt.Errorf("remove invalid Program cache entry %q: %w", entry.Name(), err)
			}
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		s.programs[hash] = programCacheEntry{size: info.Size(), lastAccess: info.ModTime().UTC()}
		s.totalBytes += info.Size()
	}
	return nil
}

func (s *ProgramStore) evictLocked(required int64, protected artifact.Digest) error {
	type candidate struct {
		hash artifact.Digest
		programCacheEntry
	}
	candidates := make([]candidate, 0, len(s.programs))
	for hash, entry := range s.programs {
		if hash != protected {
			candidates = append(candidates, candidate{hash: hash, programCacheEntry: entry})
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].lastAccess.Equal(candidates[j].lastAccess) {
			return candidates[i].hash < candidates[j].hash
		}
		return candidates[i].lastAccess.Before(candidates[j].lastAccess)
	})
	overCount := func() bool {
		if required == 0 {
			return len(s.programs) > s.max
		}
		return len(s.programs) >= s.max
	}
	for (overCount() || s.totalBytes+required > s.maxBytes) && len(candidates) != 0 {
		item := candidates[0]
		candidates = candidates[1:]
		if err := durablefs.Remove(s.programPath(item.hash)); err != nil && !durablefs.Committed(err) {
			return fmt.Errorf("evict Program %s: %w", item.hash, err)
		}
		delete(s.programs, item.hash)
		s.totalBytes -= item.size
	}
	if overCount() || s.totalBytes+required > s.maxBytes {
		return errors.New("program cache quota cannot admit artifact")
	}
	return nil
}

func (s *ProgramStore) touchLocked(hash artifact.Digest, entry programCacheEntry) error {
	now := s.now().UTC()
	if err := os.Chtimes(s.programPath(hash), now, now); err != nil {
		return err
	}
	entry.lastAccess = now
	s.programs[hash] = entry
	return nil
}

func (s *ProgramStore) dropLocked(hash artifact.Digest, entry programCacheEntry) {
	_ = durablefs.Remove(s.programPath(hash))
	delete(s.programs, hash)
	s.totalBytes -= entry.size
}

func (s *ProgramStore) programPath(hash artifact.Digest) string {
	return filepath.Join(s.root, strings.TrimPrefix(hash.String(), "sha256:")+".json")
}

func claimProgramCache(root string) ([]os.DirEntry, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	markerPath := filepath.Join(root, programMarker)
	if len(entries) == 0 {
		if err := durablefs.WriteFile(markerPath, []byte(programMarkerContents), 0o600); err != nil {
			return nil, fmt.Errorf("claim Program cache: %w", err)
		}
		return os.ReadDir(root)
	}
	info, statErr := os.Lstat(markerPath)
	marker, readErr := os.ReadFile(markerPath)
	if statErr != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() ||
		readErr != nil || string(marker) != programMarkerContents {
		return nil, errors.New("cache directory has an unsupported Program ownership marker")
	}
	return entries, nil
}

func digestFromProgramName(name string) (artifact.Digest, bool) {
	if filepath.Ext(name) != ".json" {
		return "", false
	}
	hexDigest := strings.TrimSuffix(name, ".json")
	if !validDigestHex(hexDigest) {
		return "", false
	}
	digest := artifact.Digest("sha256:" + hexDigest)
	return digest, digest.Valid()
}

func validDigestHex(value string) bool {
	if len(value) != 64 || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func abandonedStaging(name string) bool {
	return strings.HasPrefix(name, ".durable-") && strings.HasSuffix(name, ".tmp")
}
