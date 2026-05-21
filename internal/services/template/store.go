package template

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Store 模板库的文件系统存储层。
//
// 布局：
//   root/
//   ├── _index.json                  # 全部元数据
//   ├── battle/skill1-cd.png         # key="battle/skill1-cd" 的 png
//   └── ui/enter-button.png          # key="ui/enter-button" 的 png
type Store struct {
	mu    sync.RWMutex
	root  string
	index TemplateIndex
}

// NewStore 打开 root 目录的模板库；不存在则创建空索引。
func NewStore(root string) (*Store, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir %s: %w", root, err)
	}
	s := &Store{root: root}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	path := filepath.Join(s.root, "_index.json")
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		s.index = TemplateIndex{SchemaVersion: CurrentSchemaVersion, Templates: map[string]TemplateMeta{}}
		return nil
	}
	if err != nil {
		return fmt.Errorf("read _index.json: %w", err)
	}
	if err := json.Unmarshal(b, &s.index); err != nil {
		return fmt.Errorf("parse _index.json: %w", err)
	}
	if s.index.Templates == nil {
		s.index.Templates = map[string]TemplateMeta{}
	}
	return nil
}

// cloneIndex 浅拷贝索引（map 重建）。
func cloneIndex(src TemplateIndex) TemplateIndex {
	out := TemplateIndex{
		SchemaVersion: src.SchemaVersion,
		Templates:     make(map[string]TemplateMeta, len(src.Templates)),
	}
	for k, v := range src.Templates {
		out.Templates[k] = v
	}
	return out
}

// persistIndex 把传入索引序列化到 _index.json（temp+rename）。
// 不改 s.index；调用方在 persistIndex 成功后再做内存替换。
func (s *Store) persistIndex(idx TemplateIndex) error {
	idx.SchemaVersion = CurrentSchemaVersion
	b, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}
	tmp := filepath.Join(s.root, "_index.json.tmp")
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(s.root, "_index.json"))
}

// validateKey 检查 key 是否符合 path/name 规则。
// 允许：a / a/b / a/b/c。禁：前导/尾随斜杠、空段、.. 段、含 .png 后缀。
func validateKey(key string) error {
	if key == "" {
		return errors.New("template key 不能为空")
	}
	if strings.HasPrefix(key, "/") || strings.HasSuffix(key, "/") {
		return fmt.Errorf("template key %q 不能以 / 开头或结尾", key)
	}
	if strings.HasSuffix(key, ".png") {
		return fmt.Errorf("template key %q 不应含 .png 后缀", key)
	}
	parts := strings.Split(key, "/")
	for _, p := range parts {
		if p == "" {
			return fmt.Errorf("template key %q 含空段", key)
		}
		if p == "." || p == ".." {
			return fmt.Errorf("template key %q 含 . 或 ..", key)
		}
	}
	return nil
}

// Save 写入或覆盖一个模板。meta.SHA256 / meta.CreatedAt 会被自动填充（若空）。
// 调用方 (GUI 录制) 不知道 Regions 字段, 传入 meta.Regions == nil. 这里如果 old 有 Regions
// 则保留 (multi-slot 配置非 GUI 编辑路径, e.g. bait_product 在 _index.json 手动加的 6 槽).
func (s *Store) Save(key string, pngData []byte, meta TemplateMeta) (TemplateMeta, error) {
	if err := validateKey(key); err != nil {
		return TemplateMeta{}, err
	}
	if len(pngData) == 0 {
		return TemplateMeta{}, errors.New("pngData 不能为空")
	}

	hash := sha256.Sum256(pngData)
	meta.SHA256 = hex.EncodeToString(hash[:])
	if meta.CreatedAt.IsZero() {
		meta.CreatedAt = time.Now().UTC()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// 保留 old.Regions 当调用方没传 (GUI 录制场景).
	if meta.Regions == nil {
		if old, ok := s.index.Templates[key]; ok && len(old.Regions) > 0 {
			meta.Regions = old.Regions
		}
	}

	pngPath := filepath.Join(s.root, key+".png")
	if err := os.MkdirAll(filepath.Dir(pngPath), 0o755); err != nil {
		return TemplateMeta{}, fmt.Errorf("mkdir: %w", err)
	}
	// 先写 PNG。失败直接返。
	if err := os.WriteFile(pngPath, pngData, 0o644); err != nil {
		return TemplateMeta{}, fmt.Errorf("write png: %w", err)
	}
	// 写索引：用副本算新值再持久化。persistIndex 失败 → rollback PNG（best-effort）。
	newIndex := cloneIndex(s.index)
	newIndex.Templates[key] = meta
	if err := s.persistIndex(newIndex); err != nil {
		_ = os.Remove(pngPath) // best-effort rollback
		return TemplateMeta{}, err
	}
	s.index = newIndex
	return meta, nil
}

func (s *Store) Get(key string) (TemplateMeta, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, ok := s.index.Templates[key]
	return m, ok
}

func (s *Store) List() map[string]TemplateMeta {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]TemplateMeta, len(s.index.Templates))
	for k, v := range s.index.Templates {
		out[k] = v
	}
	return out
}

func (s *Store) Delete(key string) error {
	if err := validateKey(key); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.index.Templates[key]; !ok {
		return nil
	}
	newIndex := cloneIndex(s.index)
	delete(newIndex.Templates, key)
	if err := s.persistIndex(newIndex); err != nil {
		return err
	}
	s.index = newIndex
	pngPath := filepath.Join(s.root, key+".png")
	if err := os.Remove(pngPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove png: %w", err)
	}
	return nil
}

// UpdateMeta 只改索引里的 meta，不动 PNG。保留 SHA256/CreatedAt/Width/Height/Region。
func (s *Store) UpdateMeta(key string, meta TemplateMeta) error {
	if err := validateKey(key); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	old, ok := s.index.Templates[key]
	if !ok {
		return fmt.Errorf("template %q not found", key)
	}
	// 强制保留物理字段，调用方只能改 Name/Description
	meta.SHA256 = old.SHA256
	meta.CreatedAt = old.CreatedAt
	meta.Width = old.Width
	meta.Height = old.Height
	meta.RecordedResolution = old.RecordedResolution
	meta.Region = old.Region
	if meta.Regions == nil {
		meta.Regions = old.Regions
	}
	newIndex := cloneIndex(s.index)
	newIndex.Templates[key] = meta
	if err := s.persistIndex(newIndex); err != nil {
		return err
	}
	s.index = newIndex
	return nil
}

// PngPath 返某 key 的绝对 png 路径（供外部 file:// URL 或读 bytes 用）。
// 调 validateKey；非法 key 返 error。不检查文件是否存在。
func (s *Store) PngPath(key string) (string, error) {
	if err := validateKey(key); err != nil {
		return "", err
	}
	return filepath.Join(s.root, key+".png"), nil
}

func (s *Store) ReadPng(key string) ([]byte, error) {
	if err := validateKey(key); err != nil {
		return nil, err
	}
	return os.ReadFile(filepath.Join(s.root, key+".png"))
}
