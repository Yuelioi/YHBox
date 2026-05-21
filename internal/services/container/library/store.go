package library

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"yhbox/internal/services/container"
)

// Store 库存储 (library/). 内部仅 subgraphs/<sgid>/ package 目录, 顶级无 templates/clips.
type Store struct {
	mu   sync.RWMutex
	root string
}

func NewStore(root string) (*Store, error) {
	if err := os.MkdirAll(filepath.Join(root, "subgraphs"), 0o755); err != nil {
		return nil, err
	}
	return &Store{root: root}, nil
}

// SubgraphPackage 一个 library subgraph package (root + 嵌入 callee + 共用 asset).
type SubgraphPackage struct {
	Root      container.Subgraph            `json:"root"`
	Embedded  map[string]container.Subgraph `json:"embedded"`  // sgID → callee subgraph
	Templates []string                      `json:"templates"` // key list
	Clips     []string                      `json:"clips"`     // clip id list
}

// ListSubgraphs 列 library 所有 package (按目录名).
func (s *Store) ListSubgraphs() ([]string, error) {
	dir := filepath.Join(s.root, "subgraphs")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(dir, e.Name(), "subgraph.json")); err != nil {
			continue
		}
		out = append(out, e.Name())
	}
	sort.Strings(out)
	return out, nil
}

// GetSubgraphPackage 读 1 个 package 完整内容.
func (s *Store) GetSubgraphPackage(sgID string) (*SubgraphPackage, error) {
	pkgDir := filepath.Join(s.root, "subgraphs", sgID)
	rootPath := filepath.Join(pkgDir, "subgraph.json")
	rootBytes, err := os.ReadFile(rootPath)
	if err != nil {
		return nil, fmt.Errorf("read root subgraph: %w", err)
	}
	var root container.Subgraph
	if err := json.Unmarshal(rootBytes, &root); err != nil {
		return nil, fmt.Errorf("parse root: %w", err)
	}

	embedded := map[string]container.Subgraph{}
	embDir := filepath.Join(pkgDir, "embedded")
	if entries, err := os.ReadDir(embDir); err == nil {
		for _, e := range entries {
			if filepath.Ext(e.Name()) != ".json" {
				continue
			}
			b, err := os.ReadFile(filepath.Join(embDir, e.Name()))
			if err != nil {
				continue
			}
			var sg container.Subgraph
			if err := json.Unmarshal(b, &sg); err != nil {
				continue
			}
			embedded[sg.ID] = sg
		}
	}

	var tplKeys []string
	tplDir := filepath.Join(pkgDir, "templates")
	if entries, err := os.ReadDir(tplDir); err == nil {
		for _, e := range entries {
			if filepath.Ext(e.Name()) != ".json" {
				continue
			}
			key := e.Name()[:len(e.Name())-len(".json")]
			tplKeys = append(tplKeys, key)
		}
	}

	var clipIDs []string
	clipDir := filepath.Join(pkgDir, "clips")
	if entries, err := os.ReadDir(clipDir); err == nil {
		for _, e := range entries {
			if filepath.Ext(e.Name()) != ".json" {
				continue
			}
			id := e.Name()[:len(e.Name())-len(".json")]
			clipIDs = append(clipIDs, id)
		}
	}

	return &SubgraphPackage{
		Root: root, Embedded: embedded,
		Templates: tplKeys, Clips: clipIDs,
	}, nil
}

// DeleteSubgraphPackage 删整个 package 目录.
func (s *Store) DeleteSubgraphPackage(sgID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return os.RemoveAll(filepath.Join(s.root, "subgraphs", sgID))
}

// copyFile helper.
func copyFile(src, dst string) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, b, 0o644)
}
