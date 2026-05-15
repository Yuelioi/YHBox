package container

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

var idRE = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// validateID 检查 ID 是否安全做 folder 名（防 path traversal）。
// 接受 alphanumeric + 下划线 + 减号；拒绝空、含 / 含 \ 含 .. 含 : 等。
func validateID(id string) error {
	if id == "" {
		return errors.New("container.id 不能为空")
	}
	if !idRE.MatchString(id) {
		return fmt.Errorf("container.id %q 含非法字符（只允许字母/数字/_/-）", id)
	}
	return nil
}

// Store 文件系统存储：data/containers/<id>/container.json。
type Store struct {
	mu   sync.RWMutex
	root string
	byID map[string]Container
}

func NewStore(root string) (*Store, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir %s: %w", root, err)
	}
	s := &Store{root: root, byID: map[string]Container{}}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return fmt.Errorf("readdir: %w", err)
	}
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		path := filepath.Join(s.root, ent.Name(), "container.json")
		b, err := os.ReadFile(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		var c Container
		if err := json.Unmarshal(b, &c); err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		s.byID[c.ID] = c
	}
	return nil
}

// Save 持久化（含 validate）。
func (s *Store) Save(c *Container) error {
	if err := validateID(c.ID); err != nil {
		return err
	}
	// 本地副本：避免 mutation 通过指针泄漏到 caller，避免共享指针 race
	local := *c
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
	tmp := filepath.Join(dir, "container.json.tmp")
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, filepath.Join(dir, "container.json")); err != nil {
		_ = os.Remove(tmp) // 清理孤立 tmp
		return err
	}
	s.byID[local.ID] = local
	return nil
}

func (s *Store) Get(id string) (Container, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.byID[id]
	return c, ok
}

func (s *Store) List() []Container {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Container, 0, len(s.byID))
	for _, c := range s.byID {
		out = append(out, c)
	}
	return out
}

func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	dir := filepath.Join(s.root, id)
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	delete(s.byID, id)
	return nil
}
