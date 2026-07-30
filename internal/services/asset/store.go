package asset

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/storage/catalog"
)

// Store is the deep persistence module used by Asset Service. SQLite query,
// revision, and reference details stay in AssetRepository; immutable bytes,
// streaming validation, and runtime pins stay in Blob Store.
type Store struct {
	gcMu       sync.RWMutex
	recordMu   sync.Mutex
	repository *catalog.AssetRepository
	objects    *catalog.ObjectRepository
	blobs      *blob.Store
	gcGrace    time.Duration
	now        func() time.Time
}

const DefaultGCGracePeriod = 24 * time.Hour

type StoreOption func(*Store) error

func WithGCGracePeriod(grace time.Duration) StoreOption {
	return func(store *Store) error {
		if grace < 0 {
			return errors.New("asset GC grace period cannot be negative")
		}
		store.gcGrace = grace
		return nil
	}
}

func WithGCClock(now func() time.Time) StoreOption {
	return func(store *Store) error {
		if now == nil {
			return errors.New("asset GC clock is required")
		}
		store.now = now
		return nil
	}
}

func NewStore(
	repository *catalog.AssetRepository,
	objects *catalog.ObjectRepository,
	blobs *blob.Store,
	options ...StoreOption,
) (*Store, error) {
	if repository == nil {
		return nil, errors.New("asset store requires the Content Catalog repository")
	}
	if objects == nil {
		return nil, errors.New("asset store requires the Content Catalog object repository")
	}
	if blobs == nil {
		return nil, errors.New("asset store requires the shared content-addressed Blob Store")
	}
	store := &Store{
		repository: repository,
		objects:    objects,
		blobs:      blobs,
		gcGrace:    DefaultGCGracePeriod,
		now:        time.Now,
	}
	for _, option := range options {
		if option == nil {
			return nil, errors.New("asset store option is nil")
		}
		if err := option(store); err != nil {
			return nil, err
		}
	}
	return store, nil
}

func (s *Store) Get(guid string) (AssetRecord, bool) {
	record, found, err := s.repository.Get(context.Background(), guid)
	if err != nil {
		return AssetRecord{}, false
	}
	return assetRecordFromCatalog(record), found
}

func (s *Store) get(guid string) (AssetRecord, bool, error) {
	record, found, err := s.repository.Get(context.Background(), guid)
	return assetRecordFromCatalog(record), found, err
}

func (s *Store) ListWithRevision() ([]AssetRecord, uint64) {
	records, revision, err := s.repository.List(context.Background())
	if err != nil {
		return nil, 0
	}
	return assetRecordsFromCatalog(records), revision
}

func (s *Store) listWithRevision() ([]AssetRecord, uint64, error) {
	records, revision, err := s.repository.List(context.Background())
	return assetRecordsFromCatalog(records), revision, err
}

func (s *Store) List() []AssetRecord {
	records, _, _ := s.listWithRevision()
	return records
}

func (s *Store) Revision() uint64 {
	revision, _ := s.repository.Revision(context.Background())
	return revision
}

// PutRecord writes metadata-only records. Any Blob references must enter
// through CommitRecordBlob so bytes are durably published first.
func (s *Store) PutRecord(record AssetRecord) error {
	s.gcMu.RLock()
	defer s.gcMu.RUnlock()
	s.recordMu.Lock()
	defer s.recordMu.Unlock()
	if len(record.Variants) != 0 || record.Blob != nil {
		return errors.New("records with blob references require CommitRecordBlob")
	}
	return s.putRecordLocked(record)
}

func (s *Store) putRecord(record AssetRecord) error {
	s.recordMu.Lock()
	defer s.recordMu.Unlock()
	return s.putRecordLocked(record)
}

func (s *Store) putRecordLocked(record AssetRecord) error {
	_, err := s.repository.Put(context.Background(), assetRecordToCatalog(record))
	return err
}

func (s *Store) PutRecordMeta(guid, name, description, category string, tags []string) error {
	s.gcMu.RLock()
	defer s.gcMu.RUnlock()
	s.recordMu.Lock()
	defer s.recordMu.Unlock()
	record, found, err := s.get(guid)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("PutRecordMeta: guid %q not found", guid)
	}
	record.Name = name
	record.Description = description
	record.Category = category
	record.Tags = append([]string(nil), tags...)
	return s.putRecordLocked(record)
}

func (s *Store) DeleteRecord(guid string) error {
	s.gcMu.RLock()
	defer s.gcMu.RUnlock()
	s.recordMu.Lock()
	defer s.recordMu.Unlock()
	_, err := s.repository.Delete(context.Background(), guid)
	return err
}

// CommitRecordBlob publishes immutable bytes before the Catalog transaction.
// A Catalog failure can only leave an unreachable object for grace-period GC.
func (s *Store) CommitRecordBlob(
	ctx context.Context,
	mediaType string,
	source io.Reader,
	build func(blob.BlobRef) AssetRecord,
) (blob.BlobRef, error) {
	s.gcMu.RLock()
	defer s.gcMu.RUnlock()
	if build == nil {
		return blob.BlobRef{}, errors.New("blob record builder is required")
	}
	ref, err := s.blobs.Put(ctx, mediaType, source)
	if err != nil {
		return blob.BlobRef{}, err
	}
	if err := s.putRecord(build(ref)); err != nil {
		return blob.BlobRef{}, err
	}
	return ref, nil
}

// PublishExistingRecord atomically publishes Global Asset metadata for BlobRefs
// that are already present in the shared CAS. It never overwrites an existing
// Global Asset identity.
func (s *Store) PublishExistingRecord(ctx context.Context, record AssetRecord) error {
	if ctx == nil {
		return errors.New("publish existing asset context is required")
	}
	s.gcMu.RLock()
	defer s.gcMu.RUnlock()
	for index, variant := range record.Variants {
		if err := s.blobs.Verify(ctx, variant.Blob); err != nil {
			return fmt.Errorf("verify existing asset variant %d: %w", index, err)
		}
	}
	if record.Blob != nil {
		if err := s.blobs.Verify(ctx, *record.Blob); err != nil {
			return fmt.Errorf("verify existing asset blob: %w", err)
		}
	}
	s.recordMu.Lock()
	defer s.recordMu.Unlock()
	if _, found, err := s.get(record.GUID); err != nil {
		return err
	} else if found {
		return fmt.Errorf("asset %q already exists", record.GUID)
	}
	return s.putRecordLocked(record)
}

func (s *Store) CommitVariantBlob(
	ctx context.Context,
	mediaType string,
	source io.Reader,
	guid string,
	resolution [2]int,
	bbox [4]int,
	regions [][4]int,
) (blob.BlobRef, error) {
	s.gcMu.RLock()
	defer s.gcMu.RUnlock()
	ref, err := s.blobs.Put(ctx, mediaType, source)
	if err != nil {
		return blob.BlobRef{}, err
	}
	if err := s.putVariant(guid, resolution, ref, bbox, regions); err != nil {
		return blob.BlobRef{}, err
	}
	return ref, nil
}

func (s *Store) ReadBlob(ctx context.Context, ref blob.BlobRef) ([]byte, error) {
	return s.blobs.ReadRange(ctx, ref, 0, ref.Size)
}

func (s *Store) putVariant(
	guid string,
	resolution [2]int,
	blobRef blob.BlobRef,
	bbox [4]int,
	regions [][4]int,
) error {
	s.recordMu.Lock()
	defer s.recordMu.Unlock()
	record, found, err := s.get(guid)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("PutVariant: guid %q not found", guid)
	}
	variants := make([]Variant, len(record.Variants), len(record.Variants)+1)
	copy(variants, record.Variants)
	replacement := Variant{
		Resolution: resolution, BBox: bbox,
		Regions: append([][4]int(nil), regions...), Blob: blobRef,
	}
	found = false
	for index, existing := range variants {
		if existing.Resolution == resolution {
			variants[index] = replacement
			found = true
			break
		}
	}
	if !found {
		variants = append(variants, replacement)
	}
	record.Variants = variants
	return s.putRecordLocked(record)
}

func (s *Store) RemoveVariant(guid string, resolution [2]int) error {
	s.gcMu.RLock()
	defer s.gcMu.RUnlock()
	s.recordMu.Lock()
	defer s.recordMu.Unlock()
	record, found, err := s.get(guid)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("RemoveVariant: guid %q not found", guid)
	}
	filtered := make([]Variant, 0, len(record.Variants))
	for _, variant := range record.Variants {
		if variant.Resolution != resolution {
			filtered = append(filtered, variant)
		}
	}
	record.Variants = filtered
	return s.putRecordLocked(record)
}

func (s *Store) query(query AssetQuery) (AssetPage, error) {
	page, err := s.repository.Query(context.Background(), catalog.AssetQuery{
		Search: query.Search, Kind: query.Kind, Category: query.Category,
		Tags: append([]string(nil), query.Tags...), Sort: query.Sort,
		Page: query.Page, PageSize: query.PageSize,
		RecentGUIDs:  append([]string(nil), query.RecentGUIDs...),
		CreatedSince: parseAssetCreatedSince(query.CreatedSince),
	})
	if err != nil {
		return AssetPage{}, err
	}
	items := make([]AssetSummary, len(page.Records))
	for index, record := range page.Records {
		items[index] = assetSummary(assetRecordFromCatalog(record))
	}
	categories := make([]FacetValue, len(page.Categories))
	for index, facet := range page.Categories {
		categories[index] = FacetValue{Value: facet.Value, Count: facet.Count}
	}
	tags := make([]FacetValue, len(page.Tags))
	for index, facet := range page.Tags {
		tags[index] = FacetValue{Value: facet.Value, Count: facet.Count}
	}
	return AssetPage{
		Items: items, Total: page.Total, Page: page.Page, PageSize: page.PageSize,
		Revision: page.Revision, Categories: categories, Tags: tags,
	}, nil
}

func parseAssetCreatedSince(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	parsed, _ := time.Parse(time.RFC3339, value)
	return parsed
}

func (s *Store) resolveBinding(ref blob.BlobRef) ([]AssetBinding, error) {
	matches, err := s.repository.ResolveBinding(context.Background(), ref)
	if err != nil {
		return nil, err
	}
	result := make([]AssetBinding, len(matches))
	for index, item := range matches {
		result[index] = AssetBinding{
			Found: true, GUID: item.GUID, Kind: item.Kind, Name: item.Name,
			Resolution: item.Resolution, Blob: item.Blob,
		}
	}
	return result, nil
}

func assetRecordToCatalog(record AssetRecord) catalog.AssetRecord {
	result := catalog.AssetRecord{
		GUID: record.GUID, Kind: record.Kind, Name: record.Name,
		Description: record.Description, Category: record.Category,
		Tags:      append([]string(nil), record.Tags...),
		Origin:    catalog.AssetOrigin{Kind: record.Origin.Kind, SourceID: record.Origin.SourceID},
		CreatedAt: record.CreatedAt,
	}
	if record.Blob != nil {
		ref := *record.Blob
		result.Blob = &ref
	}
	result.Variants = make([]catalog.AssetVariant, len(record.Variants))
	for index, variant := range record.Variants {
		result.Variants[index] = catalog.AssetVariant{
			Resolution: variant.Resolution, BBox: variant.BBox,
			Regions: append([][4]int(nil), variant.Regions...), Blob: variant.Blob,
		}
	}
	return result
}

func assetRecordFromCatalog(record catalog.AssetRecord) AssetRecord {
	result := AssetRecord{
		SchemaVersion: RecordSchemaVersion,
		GUID:          record.GUID, Kind: record.Kind, Name: record.Name,
		Description: record.Description, Category: record.Category,
		Tags:      append([]string(nil), record.Tags...),
		Origin:    Origin{Kind: record.Origin.Kind, SourceID: record.Origin.SourceID},
		CreatedAt: record.CreatedAt,
	}
	if record.Blob != nil {
		ref := *record.Blob
		result.Blob = &ref
	}
	result.Variants = make([]Variant, len(record.Variants))
	for index, variant := range record.Variants {
		result.Variants[index] = Variant{
			Resolution: variant.Resolution, BBox: variant.BBox,
			Regions: append([][4]int(nil), variant.Regions...), Blob: variant.Blob,
		}
	}
	return result
}

func assetRecordsFromCatalog(records []catalog.AssetRecord) []AssetRecord {
	result := make([]AssetRecord, len(records))
	for index, record := range records {
		result[index] = assetRecordFromCatalog(record)
	}
	return result
}
