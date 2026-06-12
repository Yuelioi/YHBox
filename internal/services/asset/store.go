// internal/services/asset/store.go
package asset

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Store 全局资产记录库。平铺布局 (类型即目录, kind 是权威、目录是索引):
//
//	<dataRoot>/templates/<guid>.json   (kind=template)
//	<dataRoot>/clips/<guid>.json       (kind=clip)
//	<dataRoot>/blobs/<sha256>
type Store struct {
	mu    sync.RWMutex
	root  string
	recs  map[string]AssetRecord
	blobs *BlobStore
}

// kindDir kind → 顶层目录名。未知 kind 返回 ""。
func kindDir(kind string) string {
	switch kind {
	case KindTemplate:
		return "templates"
	case KindClip:
		return "clips"
	}
	return ""
}

// NewStore 初始化目录结构，preload 已有记录，坏 JSON 警告跳过不 fail;
// kind↔目录不一致 / schemaVersion 超出支持 → 硬错误 (数据损坏, fail fast)。
func NewStore(dataRoot string) (*Store, error) {
	for _, d := range []string{"templates", "clips"} {
		if err := os.MkdirAll(filepath.Join(dataRoot, d), 0o755); err != nil {
			return nil, fmt.Errorf("asset mkdir %s: %w", d, err)
		}
	}

	bs, err := NewBlobStore(filepath.Join(dataRoot, "blobs"))
	if err != nil {
		return nil, err
	}

	s := &Store{
		root:  dataRoot,
		recs:  map[string]AssetRecord{},
		blobs: bs,
	}
	for _, kind := range []string{KindTemplate, KindClip} {
		if err := s.preload(kind); err != nil {
			return nil, fmt.Errorf("asset preload %s: %w", kindDir(kind), err)
		}
	}
	return s, nil
}

// preload 扫单个 kind 目录的 *.json → 填 recs。坏文件 stderr 跳过;
// kind 与目录不符 / 版本超前 → 返回错误让启动 fail fast。
func (s *Store) preload(expectKind string) error {
	dir := filepath.Join(s.root, kindDir(expectKind))
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if len(name) < 5 || name[len(name)-5:] != ".json" {
			continue
		}
		path := filepath.Join(dir, name)
		b, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[asset.Store] skip %q: read error: %v\n", name, err)
			continue
		}
		var rec AssetRecord
		if err := json.Unmarshal(b, &rec); err != nil {
			fmt.Fprintf(os.Stderr, "[asset.Store] skip %q: bad json: %v\n", name, err)
			continue
		}
		if rec.GUID == "" {
			fmt.Fprintf(os.Stderr, "[asset.Store] skip %q: empty GUID\n", name)
			continue
		}
		if rec.SchemaVersion > RecordSchemaVersion {
			return fmt.Errorf("%s: schemaVersion %d 超出本程序支持 (%d)", path, rec.SchemaVersion, RecordSchemaVersion)
		}
		if rec.Kind != expectKind {
			return fmt.Errorf("%s: kind %q 与所在目录 %s/ 不符 (kind 是权威, 文件放错位置)", path, rec.Kind, kindDir(expectKind))
		}
		s.recs[rec.GUID] = rec
	}
	return nil
}

// recordPath 返回记录对应的 JSON 路径 (按 kind 分目录)。
func (s *Store) recordPath(kind, guid string) string {
	return filepath.Join(s.root, kindDir(kind), guid+".json")
}

// writeRecord 原子写单条记录到磁盘（temp+rename）。统一盖 schemaVersion。调用方负责持锁。
func (s *Store) writeRecord(rec AssetRecord) error {
	if kindDir(rec.Kind) == "" {
		return fmt.Errorf("writeRecord: 未知 kind %q (guid %s)", rec.Kind, rec.GUID)
	}
	rec.SchemaVersion = RecordSchemaVersion
	b, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	return atomicWriteFile(s.recordPath(rec.Kind, rec.GUID), b)
}

// Get 返回 guid 对应的记录。
func (s *Store) Get(guid string) (AssetRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.recs[guid]
	return rec, ok
}

// List 返回所有记录的快照切片。
func (s *Store) List() []AssetRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]AssetRecord, 0, len(s.recs))
	for _, r := range s.recs {
		out = append(out, r)
	}
	return out
}

// PutRecord 写入（新建或全量替换）一条记录。
func (s *Store) PutRecord(rec AssetRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.writeRecord(rec); err != nil {
		return fmt.Errorf("PutRecord write: %w", err)
	}
	rec.SchemaVersion = RecordSchemaVersion
	s.recs[rec.GUID] = rec
	return nil
}

// PutRecordMeta 仅更新 Name 和 Tags，其余字段不变。
func (s *Store) PutRecordMeta(guid, name string, tags []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.recs[guid]
	if !ok {
		return fmt.Errorf("PutRecordMeta: guid %q not found", guid)
	}
	rec.Name = name
	rec.Tags = tags
	if err := s.writeRecord(rec); err != nil {
		return fmt.Errorf("PutRecordMeta write: %w", err)
	}
	s.recs[guid] = rec
	return nil
}

// DeleteRecord 删除记录（内存 + 磁盘）。
func (s *Store) DeleteRecord(guid string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if rec, ok := s.recs[guid]; ok {
		if err := os.Remove(s.recordPath(rec.Kind, guid)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("DeleteRecord rm: %w", err)
		}
	}
	delete(s.recs, guid)
	return nil
}

// Blobs 返回底层 BlobStore。
func (s *Store) Blobs() *BlobStore {
	return s.blobs
}

// PutVariant 锁内读记录 → 按 Resolution upsert 单条 Variant → 写回。
func (s *Store) PutVariant(guid string, res [2]int, blobSha string, bbox [4]int, regions [][4]int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.recs[guid]
	if !ok {
		return fmt.Errorf("PutVariant: guid %q not found", guid)
	}
	v := Variant{
		Resolution: res,
		BBox:       bbox,
		Regions:    regions,
		Blob:       blobSha,
	}
	// clone 后再改：rec.Variants 与 map value / 历史 List() 结果共享 backing array,
	// 原地改会污染别处持有的快照。
	vs := make([]Variant, len(rec.Variants), len(rec.Variants)+1)
	copy(vs, rec.Variants)
	// 同 Resolution 覆盖，否则追加。
	found := false
	for i, existing := range vs {
		if existing.Resolution == res {
			vs[i] = v
			found = true
			break
		}
	}
	if !found {
		vs = append(vs, v)
	}
	rec.Variants = vs
	if err := s.writeRecord(rec); err != nil {
		return fmt.Errorf("PutVariant write: %w", err)
	}
	s.recs[guid] = rec
	return nil
}

// RemoveVariant 锁内删除指定 Resolution 的 Variant。
func (s *Store) RemoveVariant(guid string, res [2]int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.recs[guid]
	if !ok {
		return fmt.Errorf("RemoveVariant: guid %q not found", guid)
	}
	// 新建 slice, 不复用 rec.Variants 的 backing array (避免污染 List() 快照)。
	filtered := make([]Variant, 0, len(rec.Variants))
	for _, v := range rec.Variants {
		if v.Resolution != res {
			filtered = append(filtered, v)
		}
	}
	rec.Variants = filtered
	if err := s.writeRecord(rec); err != nil {
		return fmt.Errorf("RemoveVariant write: %w", err)
	}
	s.recs[guid] = rec
	return nil
}

// atomicWriteFile temp+rename 原子写（复用同包 blobstore 模式）。
// 注意：blobstore.go 没有导出此函数，这里同名私有 helper。
func atomicWriteFile(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
