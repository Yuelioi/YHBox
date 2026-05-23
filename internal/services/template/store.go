// internal/services/template/store.go
package template

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Store 单容器模板存储 (containers/<id>/templates/).
// 目录-per-key, 启动期 preload in-memory index, PickBest O(1).
type Store struct {
	mu    sync.RWMutex
	root  string
	metas map[string]KeyMeta
	vars  map[string]map[[2]int]VariantMeta
}

func NewStore(root string) (*Store, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir %s: %w", root, err)
	}
	s := &Store{
		root:  root,
		metas: map[string]KeyMeta{},
		vars:  map[string]map[[2]int]VariantMeta{},
	}
	if err := s.preload(); err != nil {
		return nil, fmt.Errorf("preload: %w", err)
	}
	return s, nil
}

// preload 扫 root/*/  → 填 metas + vars. 坏文件 log warning 跳过, 不 fail.
func (s *Store) preload() error {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		key := e.Name()
		if err := ValidateKey(key); err != nil {
			fmt.Fprintf(os.Stderr, "[template.Store] skip invalid key dir %q: %v\n", key, err)
			continue
		}
		keyDir := filepath.Join(s.root, key)

		metaBytes, err := os.ReadFile(filepath.Join(keyDir, "_meta.json"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "[template.Store] skip key %q: missing _meta.json: %v\n", key, err)
			continue
		}
		var km KeyMeta
		if err := json.Unmarshal(metaBytes, &km); err != nil {
			fmt.Fprintf(os.Stderr, "[template.Store] skip key %q: bad _meta.json: %v\n", key, err)
			continue
		}
		s.metas[key] = km

		varEntries, err := os.ReadDir(keyDir)
		if err != nil {
			continue
		}
		s.vars[key] = map[[2]int]VariantMeta{}
		for _, ve := range varEntries {
			name := ve.Name()
			if !strings.HasSuffix(name, ".json") || name == "_meta.json" {
				continue
			}
			res, ok := parseResolutionFromFilename(name[:len(name)-len(".json")])
			if !ok {
				fmt.Fprintf(os.Stderr, "[template.Store] skip variant %q/%q: filename not WxH\n", key, name)
				continue
			}
			vb, err := os.ReadFile(filepath.Join(keyDir, name))
			if err != nil {
				continue
			}
			var vm VariantMeta
			if err := json.Unmarshal(vb, &vm); err != nil {
				fmt.Fprintf(os.Stderr, "[template.Store] skip variant %q/%q: bad json: %v\n", key, name, err)
				continue
			}
			if vm.Resolution != res {
				fmt.Fprintf(os.Stderr, "[template.Store] skip variant %q/%q: resolution mismatch (filename %v vs json %v)\n", key, name, res, vm.Resolution)
				continue
			}
			s.vars[key][res] = vm
		}
	}
	return nil
}

func parseResolutionFromFilename(s string) ([2]int, bool) {
	parts := strings.SplitN(s, "x", 2)
	if len(parts) != 2 {
		return [2]int{}, false
	}
	w, err := strconv.Atoi(parts[0])
	if err != nil || w <= 0 {
		return [2]int{}, false
	}
	h, err := strconv.Atoi(parts[1])
	if err != nil || h <= 0 {
		return [2]int{}, false
	}
	return [2]int{w, h}, true
}

func resolutionToFilename(res [2]int) string {
	return fmt.Sprintf("%dx%d", res[0], res[1])
}

func atomicWriteFile(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (s *Store) SaveMeta(key string, m KeyMeta) error {
	if err := ValidateKey(key); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	keyDir := filepath.Join(s.root, key)
	if err := os.MkdirAll(keyDir, 0o755); err != nil {
		return fmt.Errorf("mkdir key dir: %w", err)
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	if err := atomicWriteFile(filepath.Join(keyDir, "_meta.json"), b); err != nil {
		return err
	}
	s.metas[key] = m
	if _, ok := s.vars[key]; !ok {
		s.vars[key] = map[[2]int]VariantMeta{}
	}
	return nil
}

func (s *Store) GetMeta(key string) (KeyMeta, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, ok := s.metas[key]
	return m, ok
}

func (s *Store) SaveVariant(key string, pngData []byte, v VariantMeta) (VariantMeta, error) {
	if err := ValidateKey(key); err != nil {
		return VariantMeta{}, err
	}
	if len(pngData) == 0 {
		return VariantMeta{}, errors.New("pngData empty")
	}
	if v.Resolution[0] <= 0 || v.Resolution[1] <= 0 {
		return VariantMeta{}, errors.New("VariantMeta.Resolution must be positive WxH")
	}

	h := sha256.Sum256(pngData)
	v.SHA256 = hex.EncodeToString(h[:])
	if v.CreatedAt.IsZero() {
		v.CreatedAt = time.Now().UTC()
	}
	v.Width = v.BBox[2] - v.BBox[0]
	v.Height = v.BBox[3] - v.BBox[1]

	s.mu.Lock()
	defer s.mu.Unlock()

	keyDir := filepath.Join(s.root, key)
	if err := os.MkdirAll(keyDir, 0o755); err != nil {
		return VariantMeta{}, err
	}

	base := resolutionToFilename(v.Resolution)
	pngPath := filepath.Join(keyDir, base+".png")
	jsonPath := filepath.Join(keyDir, base+".json")

	if err := atomicWriteFile(pngPath, pngData); err != nil {
		return VariantMeta{}, fmt.Errorf("write png: %w", err)
	}
	jb, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		_ = os.Remove(pngPath)
		return VariantMeta{}, err
	}
	if err := atomicWriteFile(jsonPath, jb); err != nil {
		_ = os.Remove(pngPath)
		return VariantMeta{}, fmt.Errorf("write json: %w", err)
	}

	if _, ok := s.vars[key]; !ok {
		s.vars[key] = map[[2]int]VariantMeta{}
	}
	s.vars[key][v.Resolution] = v
	return v, nil
}

func (s *Store) GetVariant(key string, resolution [2]int) (VariantMeta, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	vs, ok := s.vars[key]
	if !ok {
		return VariantMeta{}, false
	}
	v, ok := vs[resolution]
	return v, ok
}

func (s *Store) ListVariants(key string) []VariantMeta {
	s.mu.RLock()
	defer s.mu.RUnlock()
	vs, ok := s.vars[key]
	if !ok {
		return nil
	}
	out := make([]VariantMeta, 0, len(vs))
	for _, v := range vs {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool {
		ai := out[i].Resolution[0] * out[i].Resolution[1]
		aj := out[j].Resolution[0] * out[j].Resolution[1]
		if ai != aj {
			return ai > aj
		}
		if out[i].Resolution[0] != out[j].Resolution[0] {
			return out[i].Resolution[0] > out[j].Resolution[0]
		}
		return out[i].Resolution[1] > out[j].Resolution[1]
	})
	return out
}

func (s *Store) PickBest(key string, frameW, frameH int) (VariantMeta, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	vs, ok := s.vars[key]
	if !ok {
		return VariantMeta{}, false
	}
	v, ok := vs[[2]int{frameW, frameH}]
	return v, ok
}

func (s *Store) ReadVariantPng(key string, resolution [2]int) ([]byte, error) {
	if err := ValidateKey(key); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	base := resolutionToFilename(resolution)
	return os.ReadFile(filepath.Join(s.root, key, base+".png"))
}

func (s *Store) DeleteVariant(key string, resolution [2]int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	base := resolutionToFilename(resolution)
	keyDir := filepath.Join(s.root, key)
	if err := os.Remove(filepath.Join(keyDir, base+".png")); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("rm png: %w", err)
	}
	if err := os.Remove(filepath.Join(keyDir, base+".json")); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("rm json: %w", err)
	}
	if vs, ok := s.vars[key]; ok {
		delete(vs, resolution)
	}
	return nil
}

func (s *Store) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.RemoveAll(filepath.Join(s.root, key)); err != nil {
		return fmt.Errorf("rm key dir: %w", err)
	}
	delete(s.metas, key)
	delete(s.vars, key)
	return nil
}

func (s *Store) List() map[string]KeyMeta {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]KeyMeta, len(s.metas))
	for k, v := range s.metas {
		out[k] = v
	}
	return out
}
