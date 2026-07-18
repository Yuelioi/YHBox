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

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/configvalidator"
	"github.com/yottaapp/yotta/internal/durablefs"
	"github.com/yottaapp/yotta/internal/nodecatalog"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
)

const (
	programMarker         = ".yotta-program-store"
	programMarkerContents = "yotta/program-store/3.1\n"
)

var (
	ErrProgramNotFound = errors.New("program not found")
	ErrProgramChanged  = errors.New("program changed outside store")
)

type ProgramStoreOptions struct{ MaxPrograms int }

type ProgramStore struct {
	mu         sync.RWMutex
	root       string
	max        int
	catalog    nodecatalog.Snapshot
	build      artifact.Digest
	validators configvalidator.Registry
	programs   map[artifact.Digest][]byte
}

func OpenProgramStore(root string, catalog nodecatalog.Snapshot, validators configvalidator.Registry, build artifact.Digest, options ProgramStoreOptions) (*ProgramStore, error) {
	if strings.TrimSpace(root) == "" || !catalog.Valid() || !validators.Valid() || !build.Valid() || options.MaxPrograms <= 0 {
		return nil, errors.New("program store requires root, trusted Catalog/build, and positive program limit")
	}
	resolved, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(resolved, 0o700); err != nil {
		return nil, fmt.Errorf("create Program Store: %w", err)
	}
	entries, err := claimDirectory(resolved, programMarker, programMarkerContents)
	if err != nil {
		return nil, err
	}
	programs := make(map[artifact.Digest][]byte)
	for _, entry := range entries {
		if entry.Name() == programMarker {
			continue
		}
		path := filepath.Join(resolved, entry.Name())
		if abandonedStaging(entry.Name()) {
			if err := durablefs.Remove(path); err != nil {
				return nil, fmt.Errorf("remove abandoned Program staging file: %w", err)
			}
			continue
		}
		want, ok := digestFromProgramName(entry.Name())
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || !ok {
			return nil, fmt.Errorf("program store contains invalid entry %q", entry.Name())
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		program, err := compiler.OpenProgram(raw, catalog, validators, build)
		if err != nil || program.Hash() != want {
			// Program artifacts are derived caches. A Catalog/compiler upgrade or
			// a corrupt cache entry must never make healthy Workflow Sources
			// unavailable; remove the stale artifact and let the next compile
			// repopulate the content-addressed store.
			removeErr := durablefs.Remove(path)
			if removeErr != nil && !durablefs.Committed(removeErr) {
				return nil, fmt.Errorf("remove stale Program %q: %w", entry.Name(), removeErr)
			}
			continue
		}
		programs[want] = append([]byte(nil), raw...)
		if len(programs) > options.MaxPrograms {
			return nil, errors.New("program store exceeds program limit")
		}
	}
	return &ProgramStore{root: resolved, max: options.MaxPrograms, catalog: catalog, validators: validators, build: build, programs: programs}, nil
}

func (s *ProgramStore) Put(ctx context.Context, program compiler.ProgramSnapshot) error {
	if ctx == nil || !program.Valid() {
		return errors.New("program store requires context and Program")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	trusted, err := compiler.OpenProgram(program.Artifact(), s.catalog, s.validators, s.build)
	if err != nil || trusted.Hash() != program.Hash() {
		return fmt.Errorf("program store rejected untrusted Program: %w", err)
	}
	raw := trusted.Artifact()
	hash := trusted.Hash()
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.programs[hash]; ok {
		if !bytes.Equal(existing, raw) {
			return ErrProgramChanged
		}
		disk, err := os.ReadFile(s.programPath(hash))
		if err != nil || !bytes.Equal(disk, existing) {
			return fmt.Errorf("%w: %v", ErrProgramChanged, err)
		}
		return nil
	}
	if len(s.programs) >= s.max {
		return errors.New("program store program limit reached")
	}
	if _, statErr := os.Lstat(s.programPath(hash)); statErr == nil || !os.IsNotExist(statErr) {
		return fmt.Errorf("%w: unindexed program path", ErrProgramChanged)
	}
	err = durablefs.WriteFile(s.programPath(hash), raw, 0o600)
	if err == nil || durablefs.Committed(err) {
		s.programs[hash] = append([]byte(nil), raw...)
	}
	return err
}

func (s *ProgramStore) Load(hash artifact.Digest) (compiler.ProgramSnapshot, error) {
	if !hash.Valid() {
		return compiler.ProgramSnapshot{}, errors.New("invalid program hash")
	}
	s.mu.RLock()
	expected, ok := s.programs[hash]
	s.mu.RUnlock()
	if !ok {
		return compiler.ProgramSnapshot{}, ErrProgramNotFound
	}
	raw, err := os.ReadFile(s.programPath(hash))
	if err != nil {
		return compiler.ProgramSnapshot{}, err
	}
	program, err := compiler.OpenProgram(raw, s.catalog, s.validators, s.build)
	if err != nil || program.Hash() != hash || !bytes.Equal(raw, expected) {
		if err != nil {
			return compiler.ProgramSnapshot{}, fmt.Errorf("%w: %v", ErrProgramChanged, err)
		}
		return compiler.ProgramSnapshot{}, ErrProgramChanged
	}
	return program, nil
}

func (s *ProgramStore) List() ([]compiler.ProgramSnapshot, error) {
	s.mu.RLock()
	hashes := make([]artifact.Digest, 0, len(s.programs))
	for hash := range s.programs {
		hashes = append(hashes, hash)
	}
	s.mu.RUnlock()
	sort.Slice(hashes, func(i, j int) bool { return hashes[i] < hashes[j] })
	result := make([]compiler.ProgramSnapshot, 0, len(hashes))
	for _, hash := range hashes {
		program, err := s.Load(hash)
		if err != nil {
			return nil, err
		}
		result = append(result, program)
	}
	return result, nil
}

func (s *ProgramStore) programPath(hash artifact.Digest) string {
	return filepath.Join(s.root, strings.TrimPrefix(hash.String(), "sha256:")+".json")
}

func digestFromProgramName(name string) (artifact.Digest, bool) {
	if filepath.Ext(name) != ".json" {
		return "", false
	}
	hexDigest := strings.TrimSuffix(name, ".json")
	if len(hexDigest) != 64 {
		return "", false
	}
	if _, err := hex.DecodeString(hexDigest); err != nil || strings.ToLower(hexDigest) != hexDigest {
		return "", false
	}
	digest := artifact.Digest("sha256:" + hexDigest)
	return digest, digest.Valid()
}
