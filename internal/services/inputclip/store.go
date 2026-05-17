package inputclip

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"sync"
)

var idRE = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// Store filesystem CRUD for InputClip.
// 路径模式: <root>/<clipID>.iclp
// 容器级 / 库级共用同一 Store, 不同 root.
type Store struct {
	mu    sync.RWMutex
	root  string
	cache map[string]*InputClip // id → clip (含 events)
}

func NewStore(root string) *Store {
	s := &Store{root: root, cache: map[string]*InputClip{}}
	_ = os.MkdirAll(root, 0o755)
	s.loadAll()
	return s
}

func (s *Store) loadAll() {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return
	}
	for _, ent := range entries {
		if ent.IsDir() || filepath.Ext(ent.Name()) != ".iclp" {
			continue
		}
		path := filepath.Join(s.root, ent.Name())
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		clip, err := Decode(f)
		f.Close()
		if err != nil {
			continue
		}
		s.cache[clip.ID] = clip
	}
}

func (s *Store) Save(clip *InputClip) error {
	if !idRE.MatchString(clip.ID) {
		return fmt.Errorf("clip id %q 非法", clip.ID)
	}
	var buf bytes.Buffer
	if err := Encode(&buf, clip); err != nil {
		return err
	}
	path := filepath.Join(s.root, clip.ID+".iclp")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write tmp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	s.mu.Lock()
	s.cache[clip.ID] = clip
	s.mu.Unlock()
	return nil
}

func (s *Store) Get(id string) (*InputClip, error) {
	s.mu.RLock()
	clip, ok := s.cache[id]
	s.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("clip %q not found", id)
	}
	return clip, nil
}

// List 列所有 clip metadata (不含 events, list view 用)
func (s *Store) List() []ClipSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]ClipSummary, 0, len(s.cache))
	for _, c := range s.cache {
		out = append(out, ClipSummary{
			ID: c.ID, Label: c.Label, Description: c.Description,
			Tags: c.Tags, DurationUs: c.DurationUs, CreatedAt: c.CreatedAt,
			Meta: c.Meta, EventCount: len(c.Events),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt > out[j].CreatedAt })
	return out
}

func (s *Store) Delete(id string) error {
	path := filepath.Join(s.root, id+".iclp")
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	s.mu.Lock()
	delete(s.cache, id)
	s.mu.Unlock()
	return nil
}

// ClipSummary 列表项 — 不含 events, 给 UI list view.
type ClipSummary struct {
	ID          string   `json:"id"`
	Label       string   `json:"label"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	DurationUs  uint64   `json:"durationUs"`
	CreatedAt   string   `json:"createdAt"`
	Meta        ClipMeta `json:"meta"`
	EventCount  int      `json:"eventCount"`
}
