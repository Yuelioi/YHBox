// Package library 公共库 (library/subgraphs/ + library/templates/).
//
// Copy-on-use 语义: 本包负责 CRUD + 文件 IO, 实际拷贝到容器走 copy.go 的 CopyToContainer.
package library

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"yhbox/internal/services/container"
	"yhbox/internal/services/template"
)

var idRE = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// LibrarySubgraph 库 subgraph：在 container.Subgraph 基础上嵌入 requiredTemplates（meta），
// 落地拷贝时一起飞。
type LibrarySubgraph struct {
	container.Subgraph
	RequiredTemplates []template.TemplateMeta `json:"requiredTemplates,omitempty"`
}

// Store 库的文件存储。
type Store struct {
	mu        sync.RWMutex
	root      string
	subgraphs map[string]LibrarySubgraph
	templates map[string]template.TemplateMeta
}

func NewStore(root string) (*Store, error) {
	if err := EnsureBuiltinLibrary(root); err != nil {
		return nil, fmt.Errorf("ensure builtin library: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "subgraphs"), 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(root, "templates"), 0o755); err != nil {
		return nil, err
	}
	s := &Store{
		root:      root,
		subgraphs: map[string]LibrarySubgraph{},
		templates: map[string]template.TemplateMeta{},
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	sgDir := filepath.Join(s.root, "subgraphs")
	if _, err := os.Stat(sgDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	err := filepath.WalkDir(sgDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".json") {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		var sg LibrarySubgraph
		if err := json.Unmarshal(b, &sg); err != nil {
			return nil
		}
		s.subgraphs[sg.ID] = sg
		return nil
	})
	if err != nil {
		return err
	}

	tplIdxPath := filepath.Join(s.root, "templates", "_index.json")
	if b, err := os.ReadFile(tplIdxPath); err == nil {
		var idx template.TemplateIndex
		if err := json.Unmarshal(b, &idx); err == nil {
			for k, m := range idx.Templates {
				s.templates[k] = m
			}
		}
	}
	return nil
}

func (s *Store) ListSubgraphs() []LibrarySubgraph {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]LibrarySubgraph, 0, len(s.subgraphs))
	for _, sg := range s.subgraphs {
		out = append(out, sg)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

func (s *Store) GetSubgraph(id string) (LibrarySubgraph, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sg, ok := s.subgraphs[id]
	return sg, ok
}

func (s *Store) SaveSubgraph(sg *LibrarySubgraph) error {
	if !idRE.MatchString(sg.ID) {
		return fmt.Errorf("subgraph id %q 非法", sg.ID)
	}
	if sg.CreatedAt.IsZero() {
		sg.CreatedAt = time.Now().UTC()
	}
	b, err := json.MarshalIndent(sg, "", "  ")
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	path := filepath.Join(s.root, "subgraphs", sg.ID+".json")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	s.subgraphs[sg.ID] = *sg
	return nil
}

func (s *Store) DeleteSubgraph(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	path := filepath.Join(s.root, "subgraphs", id+".json")
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.RemoveAll(filepath.Join(s.root, "subgraphs", id)); err != nil {
		return err
	}
	delete(s.subgraphs, id)
	return nil
}

func (s *Store) ListTemplates() []TemplateEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]TemplateEntry, 0, len(s.templates))
	for k, m := range s.templates {
		out = append(out, TemplateEntry{Key: k, Meta: m})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Meta.CreatedAt.After(out[j].Meta.CreatedAt) })
	return out
}

// TemplateEntry list/get 时把 key 和 meta 一起返。
type TemplateEntry struct {
	Key  string                `json:"key"`
	Meta template.TemplateMeta `json:"meta"`
}

// DeleteTemplate 删除库 template（png + index 条目）。
func (s *Store) DeleteTemplate(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	pngPath := filepath.Join(s.root, "templates", key+".png")
	if err := os.Remove(pngPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	delete(s.templates, key)
	return s.writeTemplateIndex()
}

func (s *Store) writeTemplateIndex() error {
	idx := template.TemplateIndex{
		SchemaVersion: template.CurrentSchemaVersion,
		Templates:     s.templates,
	}
	b, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.root, "templates", "_index.json"), b, 0o644)
}
