package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/blob"
)

const (
	AssetKindTemplate = "template"
	AssetKindClip     = "clip"
	AssetKindMacro    = "macro"
)

var catalogAssetIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

type AssetOrigin struct {
	Kind     string
	SourceID string
}

type AssetVariant struct {
	Resolution [2]int
	BBox       [4]int
	Regions    [][4]int
	Blob       blob.BlobRef
}

type AssetRecord struct {
	GUID        string
	Kind        string
	Name        string
	Description string
	Category    string
	Tags        []string
	Origin      AssetOrigin
	Variants    []AssetVariant
	Blob        *blob.BlobRef
	CreatedAt   time.Time
	Revision    uint64
}

type AssetQuery struct {
	Search      string
	Kind        string
	Category    string
	Tags        []string
	Sort        string
	Page        int
	PageSize    int
	RecentGUIDs []string
}

type AssetFacet struct {
	Value string
	Count int
}

type AssetPage struct {
	Records    []AssetRecord
	Total      int
	Page       int
	PageSize   int
	Revision   uint64
	Categories []AssetFacet
	Tags       []AssetFacet
}

type AssetBinding struct {
	GUID       string
	Kind       string
	Name       string
	Resolution [2]int
	Blob       blob.BlobRef
}

// AssetRepository owns Global Asset metadata, query indexes, revisions, and
// object references in the Content Catalog.
type AssetRepository struct {
	database *database
}

func (r *AssetRepository) Revision(ctx context.Context) (uint64, error) {
	if err := r.ready(); err != nil {
		return 0, err
	}
	return readAssetRevision(ctx, r.database.db)
}

func (r *AssetRepository) Get(ctx context.Context, guid string) (AssetRecord, bool, error) {
	if err := r.ready(); err != nil {
		return AssetRecord{}, false, err
	}
	records, err := r.loadRecords(ctx, `
		SELECT guid, kind, name, description, category, origin_kind,
			origin_source_id, created_at, record_revision,
			record_blob_media_type, record_blob_digest, record_blob_size
		FROM assets WHERE guid = ?
	`, []any{guid})
	if err != nil {
		return AssetRecord{}, false, err
	}
	if len(records) == 0 {
		return AssetRecord{}, false, nil
	}
	return records[0], true, nil
}

func (r *AssetRepository) List(ctx context.Context) ([]AssetRecord, uint64, error) {
	if err := r.ready(); err != nil {
		return nil, 0, err
	}
	records, err := r.loadRecords(ctx, `
		SELECT guid, kind, name, description, category, origin_kind,
			origin_source_id, created_at, record_revision,
			record_blob_media_type, record_blob_digest, record_blob_size
		FROM assets ORDER BY guid
	`, nil)
	if err != nil {
		return nil, 0, err
	}
	revision, err := readAssetRevision(ctx, r.database.db)
	return records, revision, err
}

func (r *AssetRepository) Query(ctx context.Context, query AssetQuery) (AssetPage, error) {
	if err := r.ready(); err != nil {
		return AssetPage{}, err
	}
	if err := validateAssetQuery(query); err != nil {
		return AssetPage{}, err
	}
	where, args := buildAssetWhere(query)
	var total int
	if err := r.database.db.QueryRowContext(ctx,
		"SELECT count(*) FROM assets a "+where, args...).Scan(&total); err != nil {
		return AssetPage{}, fmt.Errorf("count assets: %w", err)
	}
	order, orderArgs := assetOrder(query)
	offset := (query.Page - 1) * query.PageSize
	pageArgs := append(append([]any{}, args...), orderArgs...)
	pageArgs = append(pageArgs, query.PageSize, offset)
	records, err := r.loadRecords(ctx, `
		SELECT a.guid, a.kind, a.name, a.description, a.category, a.origin_kind,
			a.origin_source_id, a.created_at, a.record_revision,
			a.record_blob_media_type, a.record_blob_digest, a.record_blob_size
		FROM assets a `+where+` `+order+` LIMIT ? OFFSET ?
	`, pageArgs)
	if err != nil {
		return AssetPage{}, err
	}
	revision, err := readAssetRevision(ctx, r.database.db)
	if err != nil {
		return AssetPage{}, err
	}
	categories, err := r.facets(ctx, "category", query.Kind)
	if err != nil {
		return AssetPage{}, err
	}
	tags, err := r.tagFacets(ctx, query.Kind)
	if err != nil {
		return AssetPage{}, err
	}
	return AssetPage{
		Records: records, Total: total, Page: query.Page, PageSize: query.PageSize,
		Revision: revision, Categories: categories, Tags: tags,
	}, nil
}

func (r *AssetRepository) Put(ctx context.Context, record AssetRecord) (AssetRecord, error) {
	if err := r.ready(); err != nil {
		return AssetRecord{}, err
	}
	if err := validateCatalogAsset(record); err != nil {
		return AssetRecord{}, err
	}
	tx, err := r.database.db.BeginTx(ctx, nil)
	if err != nil {
		return AssetRecord{}, err
	}
	rollback := func(cause error) (AssetRecord, error) {
		return AssetRecord{}, errors.Join(cause, tx.Rollback())
	}
	var current uint64
	err = tx.QueryRowContext(ctx, "SELECT record_revision FROM assets WHERE guid = ?", record.GUID).Scan(&current)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return rollback(err)
	}
	record.Revision = current + 1
	var recordMedia, recordDigest any
	var recordSize any
	if record.Blob != nil {
		recordMedia, recordDigest, recordSize = record.Blob.MediaType, record.Blob.Digest.String(), record.Blob.Size
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO assets(
			guid, kind, name, description, category, origin_kind, origin_source_id,
			created_at, record_revision, record_blob_media_type, record_blob_digest, record_blob_size
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(guid) DO UPDATE SET
			kind=excluded.kind, name=excluded.name, description=excluded.description,
			category=excluded.category, origin_kind=excluded.origin_kind,
			origin_source_id=excluded.origin_source_id, created_at=excluded.created_at,
			record_revision=excluded.record_revision,
			record_blob_media_type=excluded.record_blob_media_type,
			record_blob_digest=excluded.record_blob_digest, record_blob_size=excluded.record_blob_size
	`, record.GUID, record.Kind, record.Name, record.Description, record.Category,
		record.Origin.Kind, record.Origin.SourceID, record.CreatedAt.UTC().Format(time.RFC3339Nano),
		record.Revision, recordMedia, recordDigest, recordSize); err != nil {
		return rollback(fmt.Errorf("write asset %q: %w", record.GUID, err))
	}
	for _, table := range []string{"asset_tags", "asset_variants"} {
		if _, err := tx.ExecContext(ctx, "DELETE FROM "+table+" WHERE asset_guid = ?", record.GUID); err != nil {
			return rollback(err)
		}
	}
	for ordinal, tag := range cleanCatalogTags(record.Tags) {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO asset_tags(asset_guid, ordinal, tag, normalized_tag) VALUES (?, ?, ?, ?)
		`, record.GUID, ordinal, tag, strings.ToLower(tag)); err != nil {
			return rollback(err)
		}
	}
	for ordinal, variant := range record.Variants {
		regions, err := json.Marshal(variant.Regions)
		if err != nil {
			return rollback(err)
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO asset_variants(
				asset_guid, ordinal, width, height, bbox_x1, bbox_y1, bbox_x2, bbox_y2,
				regions_json, blob_media_type, blob_digest, blob_size
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, record.GUID, ordinal, variant.Resolution[0], variant.Resolution[1],
			variant.BBox[0], variant.BBox[1], variant.BBox[2], variant.BBox[3],
			string(regions), variant.Blob.MediaType, variant.Blob.Digest.String(), variant.Blob.Size); err != nil {
			return rollback(err)
		}
	}
	if err := replaceAssetObjectReferences(ctx, tx, record); err != nil {
		return rollback(err)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE meta SET value = CAST(CAST(value AS INTEGER) + 1 AS TEXT)
		WHERE key = 'asset_revision'
	`); err != nil {
		return rollback(err)
	}
	if err := tx.Commit(); err != nil {
		return AssetRecord{}, err
	}
	record.Tags = cleanCatalogTags(record.Tags)
	return record, nil
}

func (r *AssetRepository) Delete(ctx context.Context, guid string) (bool, error) {
	if err := r.ready(); err != nil {
		return false, err
	}
	tx, err := r.database.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	result, err := tx.ExecContext(ctx, "DELETE FROM assets WHERE guid = ?", guid)
	if err != nil {
		_ = tx.Rollback()
		return false, err
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		_ = tx.Rollback()
		return false, err
	}
	if deleted != 0 {
		if _, err := tx.ExecContext(ctx,
			"DELETE FROM object_refs WHERE owner_kind = 'asset' AND owner_id = ?", guid); err != nil {
			_ = tx.Rollback()
			return false, err
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE meta SET value = CAST(CAST(value AS INTEGER) + 1 AS TEXT)
			WHERE key = 'asset_revision'
		`); err != nil {
			_ = tx.Rollback()
			return false, err
		}
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return deleted != 0, nil
}

func (r *AssetRepository) ResolveBinding(ctx context.Context, ref blob.BlobRef) ([]AssetBinding, error) {
	if err := r.ready(); err != nil {
		return nil, err
	}
	if err := ref.Validate(); err != nil {
		return nil, err
	}
	rows, err := r.database.db.QueryContext(ctx, `
		SELECT a.guid, a.kind, a.name, refs.role
		FROM object_refs refs
		JOIN assets a ON a.guid = refs.owner_id
		WHERE refs.owner_kind = 'asset' AND refs.digest = ? AND refs.media_type = ? AND refs.size = ?
		ORDER BY a.guid, refs.role
	`, ref.Digest.String(), ref.MediaType, ref.Size)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []AssetBinding
	for rows.Next() {
		var item AssetBinding
		var role string
		if err := rows.Scan(&item.GUID, &item.Kind, &item.Name, &role); err != nil {
			return nil, err
		}
		item.Blob = ref
		if strings.HasPrefix(role, "variant:") {
			parts := strings.Split(strings.TrimPrefix(role, "variant:"), "x")
			if len(parts) == 2 {
				item.Resolution[0], _ = strconv.Atoi(parts[0])
				item.Resolution[1], _ = strconv.Atoi(parts[1])
			}
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (r *AssetRepository) ready() error {
	if r == nil || r.database == nil || r.database.db == nil {
		return errors.New("asset repository is not open")
	}
	return nil
}

func validateCatalogAsset(record AssetRecord) error {
	if !catalogAssetIDPattern.MatchString(record.GUID) {
		return fmt.Errorf("invalid asset GUID %q", record.GUID)
	}
	if record.Kind != AssetKindTemplate && record.Kind != AssetKindClip && record.Kind != AssetKindMacro {
		return fmt.Errorf("invalid asset kind %q", record.Kind)
	}
	if record.CreatedAt.IsZero() {
		return errors.New("asset created time is required")
	}
	if record.Blob != nil {
		if err := record.Blob.Validate(); err != nil {
			return err
		}
	}
	seenResolution := make(map[[2]int]struct{}, len(record.Variants))
	for _, variant := range record.Variants {
		if variant.Resolution[0] <= 0 || variant.Resolution[1] <= 0 {
			return errors.New("asset variant resolution must be positive")
		}
		if _, exists := seenResolution[variant.Resolution]; exists {
			return errors.New("asset variant resolution is duplicated")
		}
		seenResolution[variant.Resolution] = struct{}{}
		if err := variant.Blob.Validate(); err != nil {
			return err
		}
	}
	if record.Kind == AssetKindTemplate && record.Blob != nil {
		return errors.New("template asset cannot have a record blob")
	}
	if record.Kind != AssetKindTemplate && len(record.Variants) != 0 {
		return errors.New("clip and macro assets cannot have variants")
	}
	return nil
}

func replaceAssetObjectReferences(ctx context.Context, tx *sql.Tx, record AssetRecord) error {
	if _, err := tx.ExecContext(ctx,
		"DELETE FROM object_refs WHERE owner_kind = 'asset' AND owner_id = ?", record.GUID); err != nil {
		return err
	}
	type reference struct {
		role string
		ref  blob.BlobRef
	}
	references := make([]reference, 0, len(record.Variants)+1)
	if record.Blob != nil {
		references = append(references, reference{role: "record", ref: *record.Blob})
	}
	for _, variant := range record.Variants {
		references = append(references, reference{
			role: fmt.Sprintf("variant:%dx%d", variant.Resolution[0], variant.Resolution[1]),
			ref:  variant.Blob,
		})
	}
	for _, item := range references {
		var observedSize int64
		var state string
		if err := tx.QueryRowContext(ctx, `
			SELECT size, state FROM gc_objects WHERE digest = ?
		`, item.ref.Digest.String()).Scan(&observedSize, &state); errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("blob %s is not present in the CAS inventory", item.ref.Digest)
		} else if err != nil {
			return err
		}
		if observedSize != item.ref.Size {
			return fmt.Errorf("blob %s has conflicting observed sizes", item.ref.Digest)
		}
		if state != "active" {
			return fmt.Errorf("blob %s is not active in the CAS inventory", item.ref.Digest)
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE gc_objects SET unreachable_since = NULL, last_error = NULL
			WHERE digest = ?
		`, item.ref.Digest.String()); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO object_refs(owner_kind, owner_id, role, digest, media_type, size)
			VALUES ('asset', ?, ?, ?, ?, ?)
		`, record.GUID, item.role, item.ref.Digest.String(), item.ref.MediaType, item.ref.Size); err != nil {
			return err
		}
	}
	return nil
}

func (r *AssetRepository) loadRecords(ctx context.Context, query string, args []any) ([]AssetRecord, error) {
	rows, err := r.database.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []AssetRecord
	for rows.Next() {
		var record AssetRecord
		var created string
		var media, digest sql.NullString
		var size sql.NullInt64
		if err := rows.Scan(
			&record.GUID, &record.Kind, &record.Name, &record.Description, &record.Category,
			&record.Origin.Kind, &record.Origin.SourceID, &created, &record.Revision,
			&media, &digest, &size,
		); err != nil {
			return nil, err
		}
		record.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
		if err != nil {
			return nil, err
		}
		if media.Valid || digest.Valid || size.Valid {
			if !media.Valid || !digest.Valid || !size.Valid {
				return nil, ErrSchemaDrift
			}
			ref := blob.BlobRef{MediaType: media.String, Digest: artifact.Digest(digest.String), Size: size.Int64}
			if err := ref.Validate(); err != nil {
				return nil, err
			}
			record.Blob = &ref
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := r.loadRecordDetails(ctx, records); err != nil {
		return nil, err
	}
	return records, nil
}

func (r *AssetRepository) loadRecordDetails(ctx context.Context, records []AssetRecord) error {
	if len(records) == 0 {
		return nil
	}
	index := make(map[string]int, len(records))
	placeholders := make([]string, len(records))
	args := make([]any, len(records))
	for position, record := range records {
		index[record.GUID] = position
		placeholders[position] = "?"
		args[position] = record.GUID
	}
	tags, err := r.database.db.QueryContext(ctx, `
		SELECT asset_guid, tag FROM asset_tags
		WHERE asset_guid IN (`+strings.Join(placeholders, ",")+`)
		ORDER BY asset_guid, ordinal
	`, args...)
	if err != nil {
		return err
	}
	for tags.Next() {
		var guid, tag string
		if err := tags.Scan(&guid, &tag); err != nil {
			tags.Close()
			return err
		}
		records[index[guid]].Tags = append(records[index[guid]].Tags, tag)
	}
	if err := tags.Close(); err != nil {
		return err
	}
	variants, err := r.database.db.QueryContext(ctx, `
		SELECT asset_guid, width, height, bbox_x1, bbox_y1, bbox_x2, bbox_y2,
			regions_json, blob_media_type, blob_digest, blob_size
		FROM asset_variants
		WHERE asset_guid IN (`+strings.Join(placeholders, ",")+`)
		ORDER BY asset_guid, ordinal
	`, args...)
	if err != nil {
		return err
	}
	defer variants.Close()
	for variants.Next() {
		var guid, regions, mediaType, digest string
		var variant AssetVariant
		if err := variants.Scan(&guid, &variant.Resolution[0], &variant.Resolution[1],
			&variant.BBox[0], &variant.BBox[1], &variant.BBox[2], &variant.BBox[3],
			&regions, &mediaType, &digest, &variant.Blob.Size); err != nil {
			return err
		}
		variant.Blob.MediaType = mediaType
		variant.Blob.Digest = artifact.Digest(digest)
		if err := variant.Blob.Validate(); err != nil {
			return err
		}
		if err := json.Unmarshal([]byte(regions), &variant.Regions); err != nil {
			return err
		}
		records[index[guid]].Variants = append(records[index[guid]].Variants, variant)
	}
	return variants.Err()
}

func validateAssetQuery(query AssetQuery) error {
	if query.Page <= 0 || query.PageSize <= 0 || query.PageSize > 100 {
		return errors.New("asset query pagination is invalid")
	}
	if query.Kind != "" && query.Kind != AssetKindTemplate && query.Kind != AssetKindClip && query.Kind != AssetKindMacro {
		return errors.New("asset query kind is invalid")
	}
	if len([]rune(query.Search)) > 200 || len([]rune(query.Category)) > 100 ||
		len(query.Tags) > 16 || len(query.RecentGUIDs) > 64 {
		return errors.New("asset query filter budget exceeded")
	}
	switch query.Sort {
	case "", "name_asc", "name_desc", "created_desc", "recent_desc":
		return nil
	default:
		return errors.New("asset query sort is invalid")
	}
}

func buildAssetWhere(query AssetQuery) (string, []any) {
	var clauses []string
	var args []any
	if query.Kind != "" {
		clauses = append(clauses, "a.kind = ?")
		args = append(args, query.Kind)
	}
	if category := strings.ToLower(strings.TrimSpace(query.Category)); category != "" {
		clauses = append(clauses, "lower(a.category) = ?")
		args = append(args, category)
	}
	if search := strings.ToLower(strings.TrimSpace(query.Search)); search != "" {
		clauses = append(clauses, `(
			lower(a.name) LIKE ? ESCAPE '\' OR lower(a.description) LIKE ? ESCAPE '\'
			OR lower(a.category) LIKE ? ESCAPE '\' OR lower(a.guid) LIKE ? ESCAPE '\'
			OR EXISTS (
				SELECT 1 FROM asset_tags search_tags
				WHERE search_tags.asset_guid = a.guid AND search_tags.normalized_tag LIKE ? ESCAPE '\'
			)
		)`)
		pattern := "%" + escapeLike(search) + "%"
		for range 5 {
			args = append(args, pattern)
		}
	}
	tags := normalizedCatalogTags(query.Tags)
	for _, tag := range tags {
		clauses = append(clauses, `EXISTS (
			SELECT 1 FROM asset_tags wanted_tag
			WHERE wanted_tag.asset_guid = a.guid AND wanted_tag.normalized_tag = ?
		)`)
		args = append(args, tag)
	}
	if len(clauses) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}

func assetOrder(query AssetQuery) (string, []any) {
	switch query.Sort {
	case "name_desc":
		return "ORDER BY a.name COLLATE NOCASE DESC, a.guid DESC", nil
	case "created_desc":
		return "ORDER BY a.created_at DESC, a.guid", nil
	case "recent_desc":
		if len(query.RecentGUIDs) == 0 {
			return "ORDER BY a.name COLLATE NOCASE, a.guid", nil
		}
		var builder strings.Builder
		builder.WriteString("ORDER BY CASE a.guid")
		args := make([]any, 0, len(query.RecentGUIDs))
		for index, guid := range query.RecentGUIDs {
			builder.WriteString(" WHEN ? THEN ")
			builder.WriteString(strconv.Itoa(index))
			args = append(args, guid)
		}
		builder.WriteString(" ELSE ")
		builder.WriteString(strconv.Itoa(len(query.RecentGUIDs)))
		builder.WriteString(" END, a.name COLLATE NOCASE, a.guid")
		return builder.String(), args
	default:
		return "ORDER BY a.name COLLATE NOCASE, a.guid", nil
	}
}

func (r *AssetRepository) facets(ctx context.Context, column, kind string) ([]AssetFacet, error) {
	where := "WHERE " + column + " <> ''"
	args := []any{}
	if kind != "" {
		where += " AND kind = ?"
		args = append(args, kind)
	}
	rows, err := r.database.db.QueryContext(ctx, `
		SELECT min(`+column+`), count(*) FROM assets `+where+`
		GROUP BY lower(`+column+`) ORDER BY lower(`+column+`)
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []AssetFacet
	for rows.Next() {
		var facet AssetFacet
		if err := rows.Scan(&facet.Value, &facet.Count); err != nil {
			return nil, err
		}
		result = append(result, facet)
	}
	return result, rows.Err()
}

func (r *AssetRepository) tagFacets(ctx context.Context, kind string) ([]AssetFacet, error) {
	where := ""
	args := []any{}
	if kind != "" {
		where = "WHERE a.kind = ?"
		args = append(args, kind)
	}
	rows, err := r.database.db.QueryContext(ctx, `
		SELECT min(t.tag), count(*) FROM asset_tags t
		JOIN assets a ON a.guid = t.asset_guid `+where+`
		GROUP BY t.normalized_tag ORDER BY t.normalized_tag
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []AssetFacet
	for rows.Next() {
		var facet AssetFacet
		if err := rows.Scan(&facet.Value, &facet.Count); err != nil {
			return nil, err
		}
		result = append(result, facet)
	}
	return result, rows.Err()
}

func readAssetRevision(ctx context.Context, query interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}) (uint64, error) {
	var raw string
	if err := query.QueryRowContext(ctx,
		"SELECT value FROM meta WHERE key = 'asset_revision'").Scan(&raw); err != nil {
		return 0, err
	}
	return strconv.ParseUint(raw, 10, 64)
}

func cleanCatalogTags(tags []string) []string {
	seen := make(map[string]struct{}, len(tags))
	result := make([]string, 0, len(tags))
	for _, raw := range tags {
		tag := strings.TrimSpace(raw)
		key := strings.ToLower(tag)
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, tag)
	}
	return result
}

func normalizedCatalogTags(tags []string) []string {
	clean := cleanCatalogTags(tags)
	result := make([]string, len(clean))
	for index, tag := range clean {
		result[index] = strings.ToLower(tag)
	}
	sort.Strings(result)
	return result
}

func escapeLike(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(value)
}
