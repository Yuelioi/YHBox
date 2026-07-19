package snippet

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/yottaapp/yotta/internal/durablefs"
)

var idPattern = regexp.MustCompile(`^snippet-[a-f0-9-]{36}$`)

type Store struct {
	mu       sync.RWMutex
	root     string
	byID     map[string]Snippet
	warnings []LoadWarning
}

func NewStore(root string) (*Store, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create snippet store: %w", err)
	}
	store := &Store{root: root, byID: map[string]Snippet{}, warnings: []LoadWarning{}}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) load() error {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return fmt.Errorf("list snippet store: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		value, err := readSnippet(filepath.Join(s.root, entry.Name()))
		if err != nil {
			s.warnings = append(s.warnings, LoadWarning{File: entry.Name(), Error: err.Error()})
			continue
		}
		fileID := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		if value.ID != fileID {
			s.warnings = append(s.warnings, LoadWarning{File: entry.Name(), Error: "snippet id does not match its filename"})
			continue
		}
		s.byID[value.ID] = value
	}
	return nil
}

func readSnippet(path string) (Snippet, error) {
	file, err := os.Open(path)
	if err != nil {
		return Snippet{}, err
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, 1<<20))
	decoder.DisallowUnknownFields()
	var value Snippet
	if err := decoder.Decode(&value); err != nil {
		return Snippet{}, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return Snippet{}, errors.New("expected exactly one JSON value")
	}
	if err := validate(value); err != nil {
		return Snippet{}, err
	}
	return value, nil
}

func (s *Store) Save(value Snippet) error {
	if err := validate(value); err != nil {
		return err
	}
	local := clone(value)
	content, err := json.MarshalIndent(local, "", "  ")
	if err != nil {
		return fmt.Errorf("encode snippet: %w", err)
	}
	path := filepath.Join(s.root, local.ID+".json")
	s.mu.Lock()
	defer s.mu.Unlock()
	err = durablefs.WriteFile(path, content, 0o600)
	if err == nil || durablefs.Committed(err) {
		s.byID[local.ID] = local
	}
	return err
}

func (s *Store) Get(id string) (Snippet, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.byID[id]
	return clone(value), ok
}

func (s *Store) List() ListResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]Summary, 0, len(s.byID))
	for _, value := range s.byID {
		items = append(items, summary(value))
	}
	sort.Slice(items, func(left, right int) bool {
		if items[left].UpdatedAt.Equal(items[right].UpdatedAt) {
			return items[left].Name < items[right].Name
		}
		return items[left].UpdatedAt.After(items[right].UpdatedAt)
	})
	return ListResult{Items: items, Warnings: append([]LoadWarning(nil), s.warnings...)}
}

func (s *Store) Delete(id string) error {
	if !idPattern.MatchString(id) {
		return errors.New("invalid snippet id")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.byID[id]; !ok {
		return fmt.Errorf("snippet %q not found", id)
	}
	if err := durablefs.Remove(filepath.Join(s.root, id+".json")); err != nil {
		return err
	}
	delete(s.byID, id)
	return nil
}
