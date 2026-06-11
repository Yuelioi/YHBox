// Package codesnippet — Script/Expr 编辑器用户代码片段的极简持久化:
// 单文件 <dataDir>/snippets.json 整存整取 (List/SaveAll), 不做 per-item CRUD —
// 片段量小、改动低频, 前端持全量列表, 每次变更整体回写.
package codesnippet

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type Snippet struct {
	ID          string `json:"id"`
	Lang        string `json:"lang"`   // "script" | "expr"
	Prefix      string `json:"prefix"` // 补全触发词 (VSCode snippet prefix 同义)
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Body        string `json:"body"`
}

type Service struct {
	mu   sync.Mutex
	path string
}

func NewService(path string) *Service {
	return &Service{path: path}
}

func (s *Service) List() ([]Snippet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return []Snippet{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", s.path, err)
	}
	var out []Snippet
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("parse %s: %w", s.path, err)
	}
	return out, nil
}

func (s *Service) SaveAll(list []Snippet) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if list == nil {
		list = []Snippet{}
	}
	b, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, s.path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
