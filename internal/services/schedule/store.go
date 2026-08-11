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

	"github.com/yottaapp/yotta/internal/durablefs"
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
	loaded := make(map[string]Schedule)
	type pendingMigration struct {
		path    string
		content []byte
	}
	migrations := make([]pendingMigration, 0)
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
		migrated, err := migrateToCurrent(&sc)
		if err != nil {
			return fmt.Errorf("migrate %s: %w", path, err)
		}
		normalizeMetadata(&sc)
		if err := sc.Validate(); err != nil {
			return fmt.Errorf("validate %s: %w", path, err)
		}
		fileID := strings.TrimSuffix(ent.Name(), filepath.Ext(ent.Name()))
		if sc.ID != fileID {
			return fmt.Errorf("validate %s: schedule.id %q does not match filename %q", path, sc.ID, fileID)
		}
		if _, duplicate := loaded[sc.ID]; duplicate {
			return fmt.Errorf("validate %s: duplicate schedule.id %q", path, sc.ID)
		}
		loaded[sc.ID] = sc
		if migrated {
			content, err := json.MarshalIndent(sc, "", "  ")
			if err != nil {
				return fmt.Errorf("encode migrated %s: %w", path, err)
			}
			migrations = append(migrations, pendingMigration{path: path, content: content})
		}
	}
	// All source files have now been parsed, migrated on copies, identity
	// checked, and validated. Each replacement is crash-atomic; a restart can
	// safely resume any remaining adjacent migrations.
	for _, migration := range migrations {
		if err := durablefs.WriteFile(migration.path, migration.content, 0o600); err != nil {
			return fmt.Errorf("publish migrated %s: %w", migration.path, err)
		}
	}
	s.byID = loaded
	return nil
}

func migrateToCurrent(sc *Schedule) (bool, error) {
	if sc == nil {
		return false, errors.New("schedule migration requires a value")
	}
	sourceVersion := sc.SchemaVersion
	for sc.SchemaVersion != CurrentSchemaVersion {
		switch sc.SchemaVersion {
		case "1":
			if err := migrateV1ToV2(sc); err != nil {
				return false, err
			}
		case "2":
			if err := migrateV2ToV3(sc); err != nil {
				return false, err
			}
		case "3":
			sc.SchemaVersion = "4"
		case "4":
			sc.TargetIntervalSeconds = 0
			sc.SchemaVersion = "5"
		default:
			return false, fmt.Errorf("schemaVersion %q is unsupported", sc.SchemaVersion)
		}
	}
	return sourceVersion != CurrentSchemaVersion, nil
}

func migrateV1ToV2(sc *Schedule) error {
	for index := range sc.Targets {
		if sc.Targets[index].Kind != "workflow" {
			return fmt.Errorf("v1 target kind %q is invalid", sc.Targets[index].Kind)
		}
		sc.Targets[index].Kind = "workflow-installation"
	}
	// A Workflow Source ID cannot be proven to identify an Installation.
	// Preserve the reference for repair, but never keep the schedule armed.
	sc.Enabled = false
	sc.SchemaVersion = "2"
	return nil
}

func migrateV2ToV3(sc *Schedule) error {
	for index := range sc.Targets {
		if sc.Targets[index].Kind != "workflow-installation" {
			return fmt.Errorf("v2 target kind %q is invalid", sc.Targets[index].Kind)
		}
		sc.Targets[index].Kind = TargetWorkflow
	}
	// V2 stored Installation IDs. Keep the reference visible for repair, but
	// do not automatically fire a potentially stale Workflow target.
	sc.Enabled = false
	sc.SchemaVersion = "3"
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
	writeErr := durablefs.WriteFile(path, raw, 0o600)
	if writeErr == nil || durablefs.Committed(writeErr) {
		s.byID[local.ID] = local
	}
	return writeErr
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
	if err := durablefs.Remove(filepath.Join(s.root, id+".json")); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	delete(s.byID, id)
	return nil
}
