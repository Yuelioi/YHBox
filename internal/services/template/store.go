package template

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Store 单容器模板存储 (containers/<id>/templates/). 每模板 = 1 PNG + 1 JSON.
// 不再有 global instance, ContainerRunner 启动期为自己容器构造一个.
type Store struct {
	mu   sync.RWMutex
	root string // 绝对路径 containers/<id>/templates
}

func NewStore(root string) (*Store, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir %s: %w", root, err)
	}
	return &Store{root: root}, nil
}

// Save 写一个模板 (PNG + JSON 元数据). key 必须符合 namespace 格式.
// sha256 / createdAt 自动填充 (空时).
func (s *Store) Save(key string, pngData []byte, meta TemplateMeta) (TemplateMeta, error) {
	if err := ValidateKey(key); err != nil {
		return TemplateMeta{}, err
	}
	if len(pngData) == 0 {
		return TemplateMeta{}, errors.New("pngData empty")
	}
	h := sha256.Sum256(pngData)
	meta.SHA256 = hex.EncodeToString(h[:])
	if meta.CreatedAt.IsZero() {
		meta.CreatedAt = time.Now().UTC()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	pngPath := filepath.Join(s.root, key+".png")
	jsonPath := filepath.Join(s.root, key+".json")
	if err := os.WriteFile(pngPath, pngData, 0o644); err != nil {
		return TemplateMeta{}, fmt.Errorf("write png: %w", err)
	}
	metaJSON, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return TemplateMeta{}, fmt.Errorf("marshal meta: %w", err)
	}
	if err := os.WriteFile(jsonPath, metaJSON, 0o644); err != nil {
		_ = os.Remove(pngPath)
		return TemplateMeta{}, fmt.Errorf("write json: %w", err)
	}
	return meta, nil
}

// Get 取一个模板元数据. 找不到返 (zero, false).
func (s *Store) Get(key string) (TemplateMeta, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	jsonPath := filepath.Join(s.root, key+".json")
	b, err := os.ReadFile(jsonPath)
	if err != nil {
		return TemplateMeta{}, false
	}
	var m TemplateMeta
	if err := json.Unmarshal(b, &m); err != nil {
		return TemplateMeta{}, false
	}
	return m, true
}

// ReadPng 读 PNG bytes.
func (s *Store) ReadPng(key string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return os.ReadFile(filepath.Join(s.root, key+".png"))
}

// List 列所有模板 key (扫 *.json files).
func (s *Store) List() map[string]TemplateMeta {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := map[string]TemplateMeta{}
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return out
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if filepath.Ext(name) != ".json" {
			continue
		}
		key := name[:len(name)-len(".json")]
		b, err := os.ReadFile(filepath.Join(s.root, name))
		if err != nil {
			continue
		}
		var m TemplateMeta
		if err := json.Unmarshal(b, &m); err != nil {
			continue
		}
		out[key] = m
	}
	return out
}

// Delete 删 PNG + JSON.
func (s *Store) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.Remove(filepath.Join(s.root, key+".png")); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("rm png: %w", err)
	}
	if err := os.Remove(filepath.Join(s.root, key+".json")); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("rm json: %w", err)
	}
	return nil
}

// PngPath 返 PNG 文件绝对路径 (不检查存在).
func (s *Store) PngPath(key string) (string, error) {
	if err := ValidateKey(key); err != nil {
		return "", err
	}
	return filepath.Join(s.root, key+".png"), nil
}
