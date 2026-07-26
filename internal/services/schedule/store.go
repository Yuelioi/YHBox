package schedule

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
	"time"

	"github.com/yottaapp/yotta/internal/durablefs"
)

const (
	pauseJournalFilename = ".pause-installation-transaction"
	pauseJournalVersion  = "1"
)

// idRE allowed ID 字符集：alphanumeric + _ + -。
var idRE = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// validateID 拒绝空、含 / 含 \ 含 .. 含 : 等。
func validateID(id string) error {
	if id == "" {
		return errors.New("schedule.id 不能为空")
	}
	if !idRE.MatchString(id) {
		return fmt.Errorf("schedule.id %q 含非法字符（只允许字母/数字/_/-）", id)
	}
	return nil
}

// Store 平铺单文件：data/schedules/<id>.json。
type Store struct {
	mu     sync.RWMutex
	root   string
	byID   map[string]Schedule
	faults storeFaults
}

func NewStore(root string) (*Store, error) {
	return newStore(root, storeFaults{})
}

type storeFaults struct {
	beforePauseCommit func() error
	afterPauseWrite   func(completed int) error
}

type pauseJournal struct {
	Version   string     `json:"version"`
	Schedules []Schedule `json:"schedules"`
}

type committedStoreError struct{ err error }

func (e *committedStoreError) Error() string { return e.err.Error() }
func (e *committedStoreError) Unwrap() error { return e.err }
func (e *committedStoreError) Committed() bool {
	return true
}

func newStore(root string, faults storeFaults) (*Store, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir: %w", err)
	}
	s := &Store{root: root, byID: map[string]Schedule{}, faults: faults}
	return s, s.load()
}

func (s *Store) load() error {
	if err := s.recoverPauseJournal(); err != nil {
		return err
	}
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return err
	}
	for _, ent := range entries {
		if ent.IsDir() || filepath.Ext(ent.Name()) != ".json" {
			continue
		}
		path := filepath.Join(s.root, ent.Name())
		b, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		var sc Schedule
		decoder := json.NewDecoder(strings.NewReader(string(b)))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&sc); err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
			return fmt.Errorf("parse %s: expected exactly one JSON value", path)
		}
		if sc.SchemaVersion == "1" {
			for index := range sc.Targets {
				if sc.Targets[index].Kind != "workflow" {
					return fmt.Errorf("migrate %s: legacy target kind %q is invalid", path, sc.Targets[index].Kind)
				}
				sc.Targets[index].Kind = TargetWorkflowInstallation
			}
			// A legacy Workflow Source ID cannot be proven to identify one
			// Installation. Preserve it for repair, but never keep the old
			// schedule armed across the semantic migration.
			sc.SchemaVersion = CurrentSchemaVersion
			sc.Enabled = false
		}
		if err := sc.Validate(); err != nil {
			return fmt.Errorf("validate %s: %w", path, err)
		}
		fileID := strings.TrimSuffix(ent.Name(), filepath.Ext(ent.Name()))
		if sc.ID != fileID {
			return fmt.Errorf("validate %s: schedule.id %q does not match filename %q", path, sc.ID, fileID)
		}
		s.byID[sc.ID] = sc
	}
	return nil
}

func (s *Store) Save(sc *Schedule) error {
	if err := validateID(sc.ID); err != nil {
		return err
	}
	// 本地副本，避免 mutation 通过指针泄漏 + 避免共享指针 race
	local := *sc
	if err := local.Validate(); err != nil {
		return err
	}
	now := time.Now().UTC()
	if local.CreatedAt.IsZero() {
		local.CreatedAt = now
	}
	local.UpdatedAt = now

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.recoverPauseJournal(); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(local, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(s.root, local.ID+".json")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	s.byID[local.ID] = local
	return nil
}

// PauseInstallation durably commits one logical batch. The journal is the
// commit point; schedule files are a recoverable materialization of it.
func (s *Store) PauseInstallation(installationID string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.recoverPauseJournal(); err != nil {
		return nil, err
	}

	paused := make([]string, 0)
	next := make([]Schedule, 0)
	now := time.Now().UTC()
	for _, current := range s.byID {
		if !current.Enabled || !scheduleTargetsInstallation(current, installationID) {
			continue
		}
		current.Enabled = false
		current.UpdatedAt = now
		paused = append(paused, current.ID)
		next = append(next, current)
	}
	sort.Strings(paused)
	sort.Slice(next, func(i, j int) bool { return next[i].ID < next[j].ID })
	if len(next) == 0 {
		return paused, nil
	}
	if s.faults.beforePauseCommit != nil {
		if err := s.faults.beforePauseCommit(); err != nil {
			return nil, err
		}
	}
	raw, err := json.Marshal(pauseJournal{Version: pauseJournalVersion, Schedules: next})
	if err != nil {
		return nil, err
	}
	journalPath := filepath.Join(s.root, pauseJournalFilename)
	journalErr := durablefs.WriteFile(journalPath, raw, 0o600)
	if journalErr != nil && !durablefs.Committed(journalErr) {
		return nil, journalErr
	}
	for _, schedule := range next {
		s.byID[schedule.ID] = schedule
	}
	if journalErr != nil {
		return paused, &committedStoreError{err: fmt.Errorf("commit schedule pause journal: %w", journalErr)}
	}
	for index, schedule := range next {
		if err := s.writeSchedule(schedule); err != nil {
			return paused, &committedStoreError{err: fmt.Errorf("materialize paused schedule %q: %w", schedule.ID, err)}
		}
		if s.faults.afterPauseWrite != nil {
			if err := s.faults.afterPauseWrite(index + 1); err != nil {
				return paused, &committedStoreError{err: err}
			}
		}
	}
	if err := durablefs.Remove(journalPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return paused, &committedStoreError{err: fmt.Errorf("retire schedule pause journal: %w", err)}
	}
	return paused, nil
}

func (s *Store) recoverPauseJournal() error {
	path := filepath.Join(s.root, pauseJournalFilename)
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read schedule pause journal: %w", err)
	}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	var journal pauseJournal
	if err := decoder.Decode(&journal); err != nil {
		return fmt.Errorf("parse schedule pause journal: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("parse schedule pause journal: expected exactly one JSON value")
	}
	if journal.Version != pauseJournalVersion || len(journal.Schedules) == 0 {
		return errors.New("schedule pause journal is invalid")
	}
	seen := make(map[string]struct{}, len(journal.Schedules))
	for _, schedule := range journal.Schedules {
		if _, duplicate := seen[schedule.ID]; duplicate {
			return errors.New("schedule pause journal contains duplicate identities")
		}
		seen[schedule.ID] = struct{}{}
		if schedule.Enabled || schedule.Validate() != nil {
			return errors.New("schedule pause journal contains an invalid paused schedule")
		}
		if err := s.writeSchedule(schedule); err != nil {
			return fmt.Errorf("recover paused schedule %q: %w", schedule.ID, err)
		}
		s.byID[schedule.ID] = schedule
	}
	if err := durablefs.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("retire recovered schedule pause journal: %w", err)
	}
	return nil
}

func (s *Store) writeSchedule(schedule Schedule) error {
	raw, err := json.MarshalIndent(schedule, "", "  ")
	if err != nil {
		return err
	}
	return durablefs.WriteFile(filepath.Join(s.root, schedule.ID+".json"), raw, 0o644)
}

func (s *Store) Get(id string) (Schedule, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sc, ok := s.byID[id]
	return sc, ok
}

func (s *Store) List() []Schedule {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Schedule, 0, len(s.byID))
	for _, sc := range s.byID {
		out = append(out, sc)
	}
	return out
}

func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := validateID(id); err != nil {
		return err
	}
	if err := s.recoverPauseJournal(); err != nil {
		return err
	}
	if err := os.Remove(filepath.Join(s.root, id+".json")); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	delete(s.byID, id)
	return nil
}
