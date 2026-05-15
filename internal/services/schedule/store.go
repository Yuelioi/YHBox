package schedule

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"
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

// Store 跟 container.Store 同构：data/schedules/<id>/schedule.json。
type Store struct {
	mu   sync.RWMutex
	root string
	byID map[string]Schedule
}

func NewStore(root string) (*Store, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir: %w", err)
	}
	s := &Store{root: root, byID: map[string]Schedule{}}
	return s, s.load()
}

func (s *Store) load() error {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return err
	}
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		path := filepath.Join(s.root, ent.Name(), "schedule.json")
		b, err := os.ReadFile(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		var sc Schedule
		if err := json.Unmarshal(b, &sc); err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
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
	local.Normalize()
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

	dir := filepath.Join(s.root, local.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(local, "", "  ")
	if err != nil {
		return err
	}
	tmp := filepath.Join(dir, "schedule.json.tmp")
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, filepath.Join(dir, "schedule.json")); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	s.byID[local.ID] = local
	return nil
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
	if err := os.RemoveAll(filepath.Join(s.root, id)); err != nil {
		return err
	}
	delete(s.byID, id)
	return nil
}
