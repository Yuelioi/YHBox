package schedule

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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

// Store 平铺单文件：data/schedules/<id>.json。
type Store struct {
	mu   sync.RWMutex
	root string
	byID map[string]Schedule
}

func NewStore(root string) (*Store, error) {
	return newStore(root)
}

func newStore(root string) (*Store, error) {
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
				sc.Targets[index].Kind = TargetWorkflow
			}
			sc.SchemaVersion = CurrentSchemaVersion
		}
		if sc.SchemaVersion == "2" {
			for index := range sc.Targets {
				if sc.Targets[index].Kind != "workflow-installation" {
					return fmt.Errorf("migrate %s: v2 target kind %q is invalid", path, sc.Targets[index].Kind)
				}
				sc.Targets[index].Kind = TargetWorkflow
			}
			// V2 stored Installation IDs. Keep the reference visible for repair,
			// but do not automatically fire a potentially stale target.
			sc.SchemaVersion = CurrentSchemaVersion
			sc.Enabled = false
		}
		if sc.SchemaVersion == "3" {
			sc.SchemaVersion = CurrentSchemaVersion
		}
		if sc.SchemaVersion == "4" {
			sc.TargetIntervalSeconds = 0
			sc.SchemaVersion = CurrentSchemaVersion
		}
		normalizeMetadata(&sc)
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
	local.Targets = append([]TargetRef(nil), sc.Targets...)
	local.Tags = append([]string(nil), sc.Tags...)
	normalizeMetadata(&local)
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

func normalizeMetadata(sc *Schedule) {
	sc.Name = strings.TrimSpace(sc.Name)
	sc.Description = strings.TrimSpace(sc.Description)
	sc.Category = strings.TrimSpace(sc.Category)
	seen := make(map[string]struct{}, len(sc.Tags))
	tags := make([]string, 0, len(sc.Tags))
	for _, raw := range sc.Tags {
		tag := strings.TrimSpace(raw)
		key := strings.ToLower(tag)
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		tags = append(tags, tag)
	}
	sc.Tags = tags
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
	if err := os.Remove(filepath.Join(s.root, id+".json")); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	delete(s.byID, id)
	return nil
}
