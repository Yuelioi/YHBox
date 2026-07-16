// internal/services/asset/store.go
package asset

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/durablefs"
)

const maxAssetRecordBytes = 1 << 20

var assetIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

// Store 全局资产记录库。平铺布局 (类型即目录, kind 是权威、目录是索引):
//
//	<dataRoot>/templates/<guid>.json   (kind=template)
//	<dataRoot>/clips/<guid>.json       (kind=clip)
//	<dataRoot>/blobs/<sha256>
type Store struct {
	mu            sync.RWMutex
	blobLifecycle sync.RWMutex
	root          string
	recs          map[string]AssetRecord
	blobs         *blob.Store
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

// NewStore initializes the exact asset schema. Corrupt, mismatched, or old
// records fail startup; there is no compatibility reader.
func NewStore(dataRoot string, blobs *blob.Store) (*Store, error) {
	if strings.TrimSpace(dataRoot) == "" {
		return nil, errors.New("asset store root is required")
	}
	if blobs == nil {
		return nil, errors.New("asset store requires the shared content-addressed Blob Store")
	}
	resolvedRoot, err := filepath.Abs(dataRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve asset store root: %w", err)
	}
	dataRoot = resolvedRoot
	for _, d := range []string{"templates", "clips"} {
		if err := os.MkdirAll(filepath.Join(dataRoot, d), 0o755); err != nil {
			return nil, fmt.Errorf("asset mkdir %s: %w", d, err)
		}
	}

	s := &Store{
		root:  dataRoot,
		recs:  map[string]AssetRecord{},
		blobs: blobs,
	}
	for _, kind := range []string{KindTemplate, KindClip} {
		if err := s.preload(kind); err != nil {
			return nil, fmt.Errorf("asset preload %s: %w", kindDir(kind), err)
		}
	}
	if err := s.verifyBlobReferences(context.Background()); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) verifyBlobReferences(ctx context.Context) error {
	seen := make(map[artifact.Digest]int64)
	for _, rec := range s.recs {
		refs := make([]blob.BlobRef, 0, len(rec.Variants)+1)
		for _, variant := range rec.Variants {
			refs = append(refs, variant.Blob)
		}
		if rec.Blob != nil {
			refs = append(refs, *rec.Blob)
		}
		for _, ref := range refs {
			if size, ok := seen[ref.Digest]; ok {
				if size != ref.Size {
					return fmt.Errorf("asset %q gives blob %s conflicting sizes", rec.GUID, ref.Digest)
				}
				continue
			}
			if err := s.blobs.Verify(ctx, ref); err != nil {
				return fmt.Errorf("asset %q references invalid blob %s: %w", rec.GUID, ref.Digest, err)
			}
			seen[ref.Digest] = ref.Size
		}
	}
	return nil
}

// preload treats every persisted record as authoritative contract data. There
// is no skip-corrupt or old-schema fallback path.
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
		if e.IsDir() || e.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("%s: unexpected asset store entry", filepath.Join(dir, e.Name()))
		}
		name := e.Name()
		if len(name) < 5 || name[len(name)-5:] != ".json" {
			return fmt.Errorf("%s: unexpected asset store entry", filepath.Join(dir, name))
		}
		path := filepath.Join(dir, name)
		info, err := e.Info()
		if err != nil {
			return fmt.Errorf("inspect %s: %w", path, err)
		}
		if info.Size() > maxAssetRecordBytes {
			return fmt.Errorf("%s: record exceeds byte budget", path)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		if err := artifact.InspectJSONBudget(b, 64, 65536, maxAssetRecordBytes); err != nil {
			return fmt.Errorf("inspect JSON %s: %w", path, err)
		}
		if _, err := artifact.Canonicalize(b); err != nil {
			return fmt.Errorf("canonical JSON %s: %w", path, err)
		}
		if err := rejectDuplicateJSONKeys(b); err != nil {
			return fmt.Errorf("decode %s: %w", path, err)
		}
		var rec AssetRecord
		decoder := json.NewDecoder(bytes.NewReader(b))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&rec); err != nil {
			return fmt.Errorf("decode %s: %w", path, err)
		}
		var trailing any
		if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
			return fmt.Errorf("decode %s: trailing JSON value", path)
		}
		if rec.SchemaVersion != RecordSchemaVersion {
			return fmt.Errorf("%s: schemaVersion %d, require %d", path, rec.SchemaVersion, RecordSchemaVersion)
		}
		if err := rec.validate(); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		if rec.Kind != expectKind {
			return fmt.Errorf("%s: kind %q 与所在目录 %s/ 不符 (kind 是权威, 文件放错位置)", path, rec.Kind, kindDir(expectKind))
		}
		if name != rec.GUID+".json" {
			return fmt.Errorf("%s: filename does not match asset GUID %q", path, rec.GUID)
		}
		if _, exists := s.recs[rec.GUID]; exists {
			return fmt.Errorf("%s: duplicate asset GUID %q", path, rec.GUID)
		}
		s.recs[rec.GUID] = rec
	}
	return nil
}

func rejectDuplicateJSONKeys(raw []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := scanJSONValue(decoder); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return errors.New("trailing JSON value")
	}
	return nil
}

func scanJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delimiter {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return errors.New("JSON object key is not a string")
			}
			if _, exists := seen[key]; exists {
				return fmt.Errorf("duplicate JSON field %q", key)
			}
			seen[key] = struct{}{}
			if err := scanJSONValue(decoder); err != nil {
				return err
			}
		}
		_, err = decoder.Token()
		return err
	case '[':
		for decoder.More() {
			if err := scanJSONValue(decoder); err != nil {
				return err
			}
		}
		_, err = decoder.Token()
		return err
	default:
		return fmt.Errorf("unexpected JSON delimiter %q", delimiter)
	}
}

// recordPath 返回记录对应的 JSON 路径 (按 kind 分目录)。
func (s *Store) recordPath(kind, guid string) string {
	return filepath.Join(s.root, kindDir(kind), guid+".json")
}

// writeRecord 原子写单条记录到磁盘（temp+rename）。统一盖 schemaVersion。调用方负责持锁。
func (s *Store) writeRecord(rec AssetRecord) error {
	if !assetIDPattern.MatchString(rec.GUID) {
		return fmt.Errorf("invalid asset GUID %q", rec.GUID)
	}
	if err := rec.validate(); err != nil {
		return err
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
	return cloneRecord(rec), ok
}

// List 返回所有记录的快照切片。
func (s *Store) List() []AssetRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]AssetRecord, 0, len(s.recs))
	for _, r := range s.recs {
		out = append(out, cloneRecord(r))
	}
	return out
}

// PutRecord 写入（新建或全量替换）一条记录。
func (s *Store) PutRecord(rec AssetRecord) error {
	if len(rec.Variants) != 0 || rec.Blob != nil {
		return errors.New("records with blob references require CommitRecordBlob")
	}
	return s.putRecord(rec)
}

func (s *Store) putRecord(rec AssetRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	writeErr := s.writeRecord(rec)
	if writeErr != nil && !durablefs.Committed(writeErr) {
		return fmt.Errorf("PutRecord write: %w", writeErr)
	}
	rec.SchemaVersion = RecordSchemaVersion
	s.recs[rec.GUID] = cloneRecord(rec)
	if writeErr != nil {
		return fmt.Errorf("PutRecord write committed without confirmed durability: %w", writeErr)
	}
	return nil
}

// PutRecordMeta 仅更新 Name 和 Tags，其余字段不变。
func (s *Store) PutRecordMeta(guid, name, description, category string, tags []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.recs[guid]
	if !ok {
		return fmt.Errorf("PutRecordMeta: guid %q not found", guid)
	}
	rec.Name = name
	rec.Description = description
	rec.Category = category
	rec.Tags = append([]string(nil), tags...)
	writeErr := s.writeRecord(rec)
	if writeErr != nil && !durablefs.Committed(writeErr) {
		return fmt.Errorf("PutRecordMeta write: %w", writeErr)
	}
	s.recs[guid] = rec
	if writeErr != nil {
		return fmt.Errorf("PutRecordMeta write committed without confirmed durability: %w", writeErr)
	}
	return nil
}

// DeleteRecord 删除记录（内存 + 磁盘）。
func (s *Store) DeleteRecord(guid string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if rec, ok := s.recs[guid]; ok {
		removeErr := durablefs.Remove(s.recordPath(rec.Kind, guid))
		if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) && !durablefs.Committed(removeErr) {
			return fmt.Errorf("DeleteRecord rm: %w", removeErr)
		}
		delete(s.recs, guid)
		if removeErr != nil && durablefs.Committed(removeErr) {
			return fmt.Errorf("DeleteRecord committed without confirmed durability: %w", removeErr)
		}
		return nil
	}
	delete(s.recs, guid)
	return nil
}

// CommitRecordBlob is the only way to introduce a record-level blob reference.
func (s *Store) CommitRecordBlob(ctx context.Context, mediaType string, source io.Reader, build func(blob.BlobRef) AssetRecord) (blob.BlobRef, error) {
	if build == nil {
		return blob.BlobRef{}, errors.New("blob record builder is required")
	}
	s.blobLifecycle.RLock()
	defer s.blobLifecycle.RUnlock()
	ref, err := s.blobs.Put(ctx, mediaType, source)
	if err != nil {
		return blob.BlobRef{}, err
	}
	if err := s.putRecord(build(ref)); err != nil {
		return blob.BlobRef{}, err
	}
	return ref, nil
}

// CommitVariantBlob is the only way to introduce or replace a variant blob.
func (s *Store) CommitVariantBlob(ctx context.Context, mediaType string, source io.Reader, guid string, res [2]int, bbox [4]int, regions [][4]int) (blob.BlobRef, error) {
	s.blobLifecycle.RLock()
	defer s.blobLifecycle.RUnlock()
	ref, err := s.blobs.Put(ctx, mediaType, source)
	if err != nil {
		return blob.BlobRef{}, err
	}
	if err := s.putVariant(guid, res, ref, bbox, regions); err != nil {
		return blob.BlobRef{}, err
	}
	return ref, nil
}

// ReadBlob reads and verifies an entire asset object.
func (s *Store) ReadBlob(ctx context.Context, ref blob.BlobRef) ([]byte, error) {
	return s.blobs.ReadRange(ctx, ref, 0, ref.Size)
}

// PutVariant 锁内读记录 → 按 Resolution upsert 单条 Variant → 写回。
func (s *Store) putVariant(guid string, res [2]int, blobRef blob.BlobRef, bbox [4]int, regions [][4]int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.recs[guid]
	if !ok {
		return fmt.Errorf("PutVariant: guid %q not found", guid)
	}
	v := Variant{
		Resolution: res,
		BBox:       bbox,
		Regions:    append([][4]int(nil), regions...),
		Blob:       blobRef,
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
	writeErr := s.writeRecord(rec)
	if writeErr != nil && !durablefs.Committed(writeErr) {
		return fmt.Errorf("PutVariant write: %w", writeErr)
	}
	s.recs[guid] = rec
	if writeErr != nil {
		return fmt.Errorf("PutVariant write committed without confirmed durability: %w", writeErr)
	}
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
	writeErr := s.writeRecord(rec)
	if writeErr != nil && !durablefs.Committed(writeErr) {
		return fmt.Errorf("RemoveVariant write: %w", writeErr)
	}
	s.recs[guid] = rec
	if writeErr != nil {
		return fmt.Errorf("RemoveVariant write committed without confirmed durability: %w", writeErr)
	}
	return nil
}

// atomicWriteFile writes through a unique sibling staging file.
func atomicWriteFile(path string, data []byte) error {
	return durablefs.WriteFile(path, data, 0o600)
}

func cloneRecord(source AssetRecord) AssetRecord {
	clone := source
	clone.Tags = append([]string(nil), source.Tags...)
	clone.Variants = make([]Variant, len(source.Variants))
	for i, variant := range source.Variants {
		clone.Variants[i] = variant
		clone.Variants[i].Regions = append([][4]int(nil), variant.Regions...)
	}
	if source.Blob != nil {
		ref := *source.Blob
		clone.Blob = &ref
	}
	return clone
}
