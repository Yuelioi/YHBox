package catalog

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/blob"
)

var (
	ErrRunLedgerConflict = errors.New("run Ledger generation conflict")
	ErrRunLedgerNotFound = errors.New("run Ledger record not found")
)

// RunSummaryRecord is the small mutable projection of one Run. The canonical
// domain Record is reconstructed by internal/run from this summary and the
// append-only event/value rows.
type RunSummaryRecord struct {
	RunID           string
	Generation      uint64
	Digest          artifact.Digest
	Status          string
	QueuedAt        time.Time
	StartedAt       *time.Time
	EndedAt         *time.Time
	SummaryArtifact []byte
	JournalCount    uint64
	ArchivedAt      *time.Time
	UpdatedAt       time.Time
}

type RunEventRecord struct {
	Sequence   uint64
	Kind       string
	OccurredAt time.Time
	Artifact   []byte
}

type RunValueRecord struct {
	Ordinal     int
	ValueID     string
	ValueDigest artifact.Digest
	Artifact    []byte
	Blob        *blob.BlobRef
}

type RunLedgerRecord struct {
	Summary RunSummaryRecord
	Events  []RunEventRecord
	Values  []RunValueRecord
}

type RunEventPage struct {
	Events []RunEventRecord
	Page   int
	Pages  int
	Total  int
}

// RunRepository owns the SQLite representation of the Run Ledger. It does not
// interpret Run state transitions or journal ordering.
type RunRepository struct {
	database *database
}

func (r *RunRepository) Count(ctx context.Context) (int, error) {
	if err := r.ready(); err != nil {
		return 0, err
	}
	var count int
	err := r.database.db.QueryRowContext(ctx, "SELECT count(*) FROM runs").Scan(&count)
	return count, err
}

func (r *RunRepository) Create(ctx context.Context, record RunLedgerRecord) error {
	if err := r.ready(); err != nil {
		return err
	}
	if err := validateRunLedgerRecord(record); err != nil {
		return err
	}
	tx, err := r.database.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := insertRunLedgerRecord(ctx, tx, record); err != nil {
		return errors.Join(err, tx.Rollback())
	}
	return tx.Commit()
}

// Import is idempotent for an exact already-imported Record and otherwise
// preserves the same collision semantics as Create.
func (r *RunRepository) Import(ctx context.Context, record RunLedgerRecord) error {
	if err := r.Create(ctx, record); err == nil {
		return nil
	} else if !errors.Is(err, ErrRunLedgerConflict) {
		return err
	}
	existing, err := r.Get(ctx, record.Summary.RunID)
	if err != nil {
		return err
	}
	if !equalRunLedgerRecord(existing, record) {
		return ErrRunLedgerConflict
	}
	return nil
}

func (r *RunRepository) Get(ctx context.Context, runID string) (RunLedgerRecord, error) {
	if err := r.ready(); err != nil {
		return RunLedgerRecord{}, err
	}
	if strings.TrimSpace(runID) == "" {
		return RunLedgerRecord{}, errors.New("run identity is required")
	}
	tx, err := r.database.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return RunLedgerRecord{}, err
	}
	rollback := func(cause error) (RunLedgerRecord, error) {
		return RunLedgerRecord{}, errors.Join(cause, tx.Rollback())
	}
	summary, err := scanRunSummary(tx.QueryRowContext(ctx, `
		SELECT run_id, generation, record_digest, status, queued_at, started_at,
			ended_at, summary_artifact, journal_count, archived_at, updated_at
		FROM runs WHERE run_id = ?
	`, runID))
	if errors.Is(err, sql.ErrNoRows) {
		return rollback(ErrRunLedgerNotFound)
	}
	if err != nil {
		return rollback(err)
	}
	events, err := queryRunEvents(ctx, tx, runID, false)
	if err != nil {
		return rollback(err)
	}
	values, err := queryRunValues(ctx, tx, runID)
	if err != nil {
		return rollback(err)
	}
	record := RunLedgerRecord{Summary: summary, Events: events, Values: values}
	if err := validateRunLedgerRecord(record); err != nil {
		return rollback(ErrSchemaDrift)
	}
	if err := tx.Commit(); err != nil {
		return RunLedgerRecord{}, err
	}
	return record, nil
}

func (r *RunRepository) GetSummary(ctx context.Context, runID string) (RunSummaryRecord, error) {
	if err := r.ready(); err != nil {
		return RunSummaryRecord{}, err
	}
	if strings.TrimSpace(runID) == "" {
		return RunSummaryRecord{}, errors.New("run identity is required")
	}
	row := r.database.db.QueryRowContext(ctx, `
		SELECT run_id, generation, record_digest, status, queued_at, started_at,
			ended_at, summary_artifact, journal_count, archived_at, updated_at
		FROM runs WHERE run_id = ?
	`, runID)
	record, err := scanRunSummary(row)
	if errors.Is(err, sql.ErrNoRows) {
		return RunSummaryRecord{}, ErrRunLedgerNotFound
	}
	return record, err
}

func (r *RunRepository) List(ctx context.Context) ([]RunLedgerRecord, error) {
	if err := r.ready(); err != nil {
		return nil, err
	}
	rows, err := r.database.db.QueryContext(ctx, `
		SELECT run_id FROM runs ORDER BY run_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	result := make([]RunLedgerRecord, 0, len(ids))
	for _, id := range ids {
		record, err := r.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		result = append(result, record)
	}
	return result, nil
}

func (r *RunRepository) ListByStatus(ctx context.Context, status string) ([]RunSummaryRecord, error) {
	if err := r.ready(); err != nil {
		return nil, err
	}
	if !validRunStatus(status) {
		return nil, errors.New("run status is invalid")
	}
	rows, err := r.database.db.QueryContext(ctx, `
		SELECT run_id, generation, record_digest, status, queued_at, started_at,
			ended_at, summary_artifact, journal_count, archived_at, updated_at
		FROM runs WHERE status = ? ORDER BY queued_at, run_id
	`, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []RunSummaryRecord
	for rows.Next() {
		record, err := scanRunSummary(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, record)
	}
	return result, rows.Err()
}

// AppendEvent advances only the small Run head and inserts one immutable event
// in the same transaction. The summary artifact is deliberately not rewritten.
func (r *RunRepository) AppendEvent(
	ctx context.Context,
	runID string,
	previousGeneration uint64,
	previousDigest artifact.Digest,
	nextGeneration uint64,
	nextDigest artifact.Digest,
	event RunEventRecord,
	updatedAt time.Time,
) error {
	if err := r.ready(); err != nil {
		return err
	}
	if strings.TrimSpace(runID) == "" || previousGeneration == 0 ||
		nextGeneration != previousGeneration+1 || !previousDigest.Valid() ||
		!nextDigest.Valid() || validateRunEvent(event) != nil || updatedAt.IsZero() {
		return errors.New("run event append is invalid")
	}
	tx, err := r.database.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `
		UPDATE runs SET
			generation = ?, record_digest = ?, journal_count = journal_count + 1, updated_at = ?
		WHERE run_id = ? AND generation = ? AND record_digest = ?
			AND status = 'running' AND journal_count = ?
	`, nextGeneration, nextDigest.String(), formatRunTime(updatedAt), runID,
		previousGeneration, previousDigest.String(), event.Sequence-1)
	if err != nil {
		return errors.Join(err, tx.Rollback())
	}
	updated, err := result.RowsAffected()
	if err != nil || updated != 1 {
		if err == nil {
			err = ErrRunLedgerConflict
		}
		return errors.Join(err, tx.Rollback())
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO run_events(run_id, sequence, kind, occurred_at, artifact)
		VALUES (?, ?, ?, ?, ?)
	`, runID, event.Sequence, event.Kind, formatRunTime(event.OccurredAt), event.Artifact); err != nil {
		return errors.Join(err, tx.Rollback())
	}
	return tx.Commit()
}

// Transition replaces the small Run summary and terminal value set while
// keeping the event history immutable.
func (r *RunRepository) Transition(
	ctx context.Context,
	previousGeneration uint64,
	previousDigest artifact.Digest,
	next RunSummaryRecord,
	values []RunValueRecord,
) error {
	if err := r.ready(); err != nil {
		return err
	}
	if previousGeneration == 0 || next.Generation != previousGeneration+1 ||
		!previousDigest.Valid() || validateRunSummary(next) != nil ||
		validateRunValues(values) != nil {
		return errors.New("run transition is invalid")
	}
	tx, err := r.database.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `
		UPDATE runs SET
			generation = ?, record_digest = ?, status = ?, queued_at = ?,
			started_at = ?, ended_at = ?, summary_artifact = ?, archived_at = ?,
			updated_at = ?
		WHERE run_id = ? AND generation = ? AND record_digest = ?
			AND journal_count = ?
	`, next.Generation, next.Digest.String(), next.Status, formatRunTime(next.QueuedAt),
		formatOptionalRunTime(next.StartedAt), formatOptionalRunTime(next.EndedAt),
		next.SummaryArtifact, formatOptionalRunTime(next.ArchivedAt), formatRunTime(next.UpdatedAt),
		next.RunID, previousGeneration, previousDigest.String(), next.JournalCount)
	if err != nil {
		return errors.Join(err, tx.Rollback())
	}
	updated, err := result.RowsAffected()
	if err != nil || updated != 1 {
		if err == nil {
			err = ErrRunLedgerConflict
		}
		return errors.Join(err, tx.Rollback())
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM run_values WHERE run_id = ?", next.RunID); err != nil {
		return errors.Join(err, tx.Rollback())
	}
	if err := insertRunValues(ctx, tx, next.RunID, values); err != nil {
		return errors.Join(err, tx.Rollback())
	}
	return tx.Commit()
}

// TimelinePage reads the bounded summary and one event page from the same
// SQLite snapshot so an in-flight append cannot produce a mixed generation.
func (r *RunRepository) TimelinePage(
	ctx context.Context,
	runID string,
	page, pageSize int,
) (RunSummaryRecord, RunEventPage, error) {
	if err := r.ready(); err != nil {
		return RunSummaryRecord{}, RunEventPage{}, err
	}
	if strings.TrimSpace(runID) == "" {
		return RunSummaryRecord{}, RunEventPage{}, errors.New("run identity is required")
	}
	if pageSize <= 0 {
		pageSize = 200
	}
	if pageSize > 500 {
		pageSize = 500
	}
	tx, err := r.database.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return RunSummaryRecord{}, RunEventPage{}, err
	}
	rollback := func(cause error) (RunSummaryRecord, RunEventPage, error) {
		return RunSummaryRecord{}, RunEventPage{}, errors.Join(cause, tx.Rollback())
	}
	summary, err := scanRunSummary(tx.QueryRowContext(ctx, `
		SELECT run_id, generation, record_digest, status, queued_at, started_at,
			ended_at, summary_artifact, journal_count, archived_at, updated_at
		FROM runs WHERE run_id = ?
	`, runID))
	if errors.Is(err, sql.ErrNoRows) {
		return rollback(ErrRunLedgerNotFound)
	}
	if err != nil {
		return rollback(err)
	}
	total := int(summary.JournalCount)
	pages := (total + pageSize - 1) / pageSize
	if pages == 0 {
		pages = 1
	}
	if page < 1 {
		page = 1
	}
	if page > pages {
		page = pages
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT sequence, kind, occurred_at, artifact
		FROM run_events WHERE run_id = ?
		ORDER BY sequence DESC LIMIT ? OFFSET ?
	`, runID, pageSize, (page-1)*pageSize)
	if err != nil {
		return rollback(err)
	}
	var descending []RunEventRecord
	for rows.Next() {
		event, err := scanRunEvent(rows)
		if err != nil {
			_ = rows.Close()
			return rollback(err)
		}
		descending = append(descending, event)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return rollback(err)
	}
	if err := rows.Close(); err != nil {
		return rollback(err)
	}
	if err := tx.Commit(); err != nil {
		return RunSummaryRecord{}, RunEventPage{}, err
	}
	events := make([]RunEventRecord, len(descending))
	for index := range descending {
		events[len(descending)-1-index] = descending[index]
	}
	return summary, RunEventPage{
		Events: events, Page: page, Pages: pages, Total: total,
	}, nil
}

func (r *RunRepository) BlobReferences(ctx context.Context) ([]blob.BlobRef, error) {
	if err := r.ready(); err != nil {
		return nil, err
	}
	rows, err := r.database.db.QueryContext(ctx, `
		SELECT blob_media_type, blob_digest, blob_size
		FROM run_values
		WHERE blob_digest IS NOT NULL
		GROUP BY blob_media_type, blob_digest, blob_size
		ORDER BY blob_digest, blob_media_type
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []blob.BlobRef
	for rows.Next() {
		var ref blob.BlobRef
		var rawDigest string
		if err := rows.Scan(&ref.MediaType, &rawDigest, &ref.Size); err != nil {
			return nil, err
		}
		ref.Digest = artifact.Digest(rawDigest)
		if err := ref.Validate(); err != nil {
			return nil, ErrSchemaDrift
		}
		result = append(result, ref)
	}
	return result, rows.Err()
}

// ArchiveTerminal marks a bounded set of old terminal Runs without deleting
// their queryable history or payload roots.
func (r *RunRepository) ArchiveTerminal(ctx context.Context, endedBefore, archivedAt time.Time, limit int) (int, error) {
	if err := r.ready(); err != nil {
		return 0, err
	}
	if endedBefore.IsZero() || archivedAt.IsZero() || limit < 1 || limit > 10000 {
		return 0, errors.New("run archive request is invalid")
	}
	result, err := r.database.db.ExecContext(ctx, `
		UPDATE runs SET archived_at = ?, updated_at = ?
		WHERE run_id IN (
			SELECT run_id FROM runs
			WHERE archived_at IS NULL AND ended_at IS NOT NULL AND ended_at < ?
			ORDER BY ended_at, run_id LIMIT ?
		)
	`, formatRunTime(archivedAt), formatRunTime(archivedAt), formatRunTime(endedBefore), limit)
	if err != nil {
		return 0, err
	}
	count, err := result.RowsAffected()
	return int(count), err
}

// PurgeArchived is the destructive half of retention and only removes Runs
// that were already archived before the supplied cutoff.
func (r *RunRepository) PurgeArchived(ctx context.Context, archivedBefore time.Time, limit int) (int, error) {
	if err := r.ready(); err != nil {
		return 0, err
	}
	if archivedBefore.IsZero() || limit < 1 || limit > 10000 {
		return 0, errors.New("run purge request is invalid")
	}
	result, err := r.database.db.ExecContext(ctx, `
		DELETE FROM runs WHERE run_id IN (
			SELECT run_id FROM runs
			WHERE archived_at IS NOT NULL AND archived_at < ?
			ORDER BY archived_at, run_id LIMIT ?
		)
	`, formatRunTime(archivedBefore), limit)
	if err != nil {
		return 0, err
	}
	count, err := result.RowsAffected()
	return int(count), err
}

func insertRunLedgerRecord(ctx context.Context, tx *sql.Tx, record RunLedgerRecord) error {
	summary := record.Summary
	result, err := tx.ExecContext(ctx, `
		INSERT INTO runs(
			run_id, generation, record_digest, status, queued_at, started_at,
			ended_at, summary_artifact, journal_count, archived_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(run_id) DO NOTHING
	`, summary.RunID, summary.Generation, summary.Digest.String(), summary.Status,
		formatRunTime(summary.QueuedAt), formatOptionalRunTime(summary.StartedAt),
		formatOptionalRunTime(summary.EndedAt), summary.SummaryArtifact, summary.JournalCount,
		formatOptionalRunTime(summary.ArchivedAt), formatRunTime(summary.UpdatedAt))
	if err != nil {
		return err
	}
	inserted, err := result.RowsAffected()
	if err != nil || inserted != 1 {
		if err == nil {
			err = ErrRunLedgerConflict
		}
		return err
	}
	for _, event := range record.Events {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO run_events(run_id, sequence, kind, occurred_at, artifact)
			VALUES (?, ?, ?, ?, ?)
		`, summary.RunID, event.Sequence, event.Kind, formatRunTime(event.OccurredAt), event.Artifact); err != nil {
			return err
		}
	}
	return insertRunValues(ctx, tx, summary.RunID, record.Values)
}

func insertRunValues(ctx context.Context, tx *sql.Tx, runID string, values []RunValueRecord) error {
	for _, value := range values {
		var mediaType, digest any
		var size any
		if value.Blob != nil {
			mediaType = value.Blob.MediaType
			digest = value.Blob.Digest.String()
			size = value.Blob.Size
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO run_values(
				run_id, ordinal, value_id, value_digest, artifact,
				blob_media_type, blob_digest, blob_size
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, runID, value.Ordinal, value.ValueID, value.ValueDigest.String(),
			value.Artifact, mediaType, digest, size); err != nil {
			return err
		}
	}
	return nil
}

type runQueryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func queryRunEvents(ctx context.Context, queryer runQueryer, runID string, descending bool) ([]RunEventRecord, error) {
	order := "ASC"
	if descending {
		order = "DESC"
	}
	rows, err := queryer.QueryContext(ctx, `
		SELECT sequence, kind, occurred_at, artifact
		FROM run_events WHERE run_id = ?
	ORDER BY sequence `+order, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []RunEventRecord
	for rows.Next() {
		event, err := scanRunEvent(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, event)
	}
	return result, rows.Err()
}

func queryRunValues(ctx context.Context, queryer runQueryer, runID string) ([]RunValueRecord, error) {
	rows, err := queryer.QueryContext(ctx, `
		SELECT ordinal, value_id, value_digest, artifact,
			blob_media_type, blob_digest, blob_size
		FROM run_values WHERE run_id = ? ORDER BY ordinal
	`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []RunValueRecord
	for rows.Next() {
		var value RunValueRecord
		var rawDigest string
		var mediaType, blobDigest sql.NullString
		var blobSize sql.NullInt64
		if err := rows.Scan(&value.Ordinal, &value.ValueID, &rawDigest, &value.Artifact,
			&mediaType, &blobDigest, &blobSize); err != nil {
			return nil, err
		}
		value.ValueDigest = artifact.Digest(rawDigest)
		if mediaType.Valid || blobDigest.Valid || blobSize.Valid {
			if !mediaType.Valid || !blobDigest.Valid || !blobSize.Valid {
				return nil, ErrSchemaDrift
			}
			value.Blob = &blob.BlobRef{
				MediaType: mediaType.String,
				Digest:    artifact.Digest(blobDigest.String),
				Size:      blobSize.Int64,
			}
		}
		value.Artifact = append([]byte(nil), value.Artifact...)
		if err := validateRunValue(value); err != nil {
			return nil, ErrSchemaDrift
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

type runScanner interface {
	Scan(...any) error
}

func scanRunSummary(scanner runScanner) (RunSummaryRecord, error) {
	var record RunSummaryRecord
	var rawDigest, queuedAt, updatedAt string
	var startedAt, endedAt, archivedAt sql.NullString
	if err := scanner.Scan(
		&record.RunID, &record.Generation, &rawDigest, &record.Status,
		&queuedAt, &startedAt, &endedAt, &record.SummaryArtifact,
		&record.JournalCount, &archivedAt, &updatedAt,
	); err != nil {
		return RunSummaryRecord{}, err
	}
	record.Digest = artifact.Digest(rawDigest)
	var err error
	record.QueuedAt, err = time.Parse(time.RFC3339Nano, queuedAt)
	if err != nil {
		return RunSummaryRecord{}, ErrSchemaDrift
	}
	record.StartedAt, err = parseOptionalRunTime(startedAt)
	if err != nil {
		return RunSummaryRecord{}, ErrSchemaDrift
	}
	record.EndedAt, err = parseOptionalRunTime(endedAt)
	if err != nil {
		return RunSummaryRecord{}, ErrSchemaDrift
	}
	record.ArchivedAt, err = parseOptionalRunTime(archivedAt)
	if err != nil {
		return RunSummaryRecord{}, ErrSchemaDrift
	}
	record.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt)
	if err != nil || validateRunSummary(record) != nil {
		return RunSummaryRecord{}, ErrSchemaDrift
	}
	record.SummaryArtifact = append([]byte(nil), record.SummaryArtifact...)
	return record, nil
}

func scanRunEvent(scanner runScanner) (RunEventRecord, error) {
	var event RunEventRecord
	var occurredAt string
	if err := scanner.Scan(&event.Sequence, &event.Kind, &occurredAt, &event.Artifact); err != nil {
		return RunEventRecord{}, err
	}
	var err error
	event.OccurredAt, err = time.Parse(time.RFC3339Nano, occurredAt)
	event.Artifact = append([]byte(nil), event.Artifact...)
	if err != nil || validateRunEvent(event) != nil {
		return RunEventRecord{}, ErrSchemaDrift
	}
	return event, nil
}

func validateRunLedgerRecord(record RunLedgerRecord) error {
	if err := validateRunSummary(record.Summary); err != nil {
		return err
	}
	if uint64(len(record.Events)) != record.Summary.JournalCount {
		return errors.New("run event count does not match its summary")
	}
	for index, event := range record.Events {
		if err := validateRunEvent(event); err != nil || event.Sequence != uint64(index+1) {
			return errors.New("run events are not a contiguous append log")
		}
	}
	return validateRunValues(record.Values)
}

func validateRunSummary(record RunSummaryRecord) error {
	if strings.TrimSpace(record.RunID) == "" || record.Generation == 0 ||
		!record.Digest.Valid() || !validRunStatus(record.Status) ||
		record.QueuedAt.IsZero() || record.QueuedAt.Location() != time.UTC ||
		len(record.SummaryArtifact) == 0 || record.UpdatedAt.IsZero() ||
		record.UpdatedAt.Location() != time.UTC {
		return errors.New("run summary is invalid")
	}
	for _, value := range []*time.Time{record.StartedAt, record.EndedAt, record.ArchivedAt} {
		if value != nil && (value.IsZero() || value.Location() != time.UTC) {
			return errors.New("run summary time is invalid")
		}
	}
	return nil
}

func validateRunEvent(event RunEventRecord) error {
	if event.Sequence == 0 || strings.TrimSpace(event.Kind) == "" ||
		event.OccurredAt.IsZero() || event.OccurredAt.Location() != time.UTC ||
		len(event.Artifact) == 0 {
		return errors.New("run event is invalid")
	}
	return nil
}

func validateRunValues(values []RunValueRecord) error {
	ids := make(map[string]struct{}, len(values))
	for index, value := range values {
		if err := validateRunValue(value); err != nil || value.Ordinal != index {
			return errors.New("run values are invalid")
		}
		if _, duplicate := ids[value.ValueID]; duplicate {
			return errors.New("run value identity is duplicated")
		}
		ids[value.ValueID] = struct{}{}
	}
	return nil
}

func validateRunValue(value RunValueRecord) error {
	if value.Ordinal < 0 || strings.TrimSpace(value.ValueID) == "" ||
		!value.ValueDigest.Valid() || len(value.Artifact) == 0 {
		return errors.New("run value is invalid")
	}
	if value.Blob != nil && value.Blob.Validate() != nil {
		return errors.New("run value Blob reference is invalid")
	}
	return nil
}

func validRunStatus(status string) bool {
	switch status {
	case "queued", "running", "succeeded", "failed", "cancelled", "interrupted":
		return true
	default:
		return false
	}
}

func formatRunTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

func formatOptionalRunTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return formatRunTime(*value)
}

func parseOptionalRunTime(value sql.NullString) (*time.Time, error) {
	if !value.Valid {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, value.String)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func equalRunLedgerRecord(left, right RunLedgerRecord) bool {
	if left.Summary.RunID != right.Summary.RunID ||
		left.Summary.Generation != right.Summary.Generation ||
		left.Summary.Digest != right.Summary.Digest ||
		left.Summary.Status != right.Summary.Status ||
		!left.Summary.QueuedAt.Equal(right.Summary.QueuedAt) ||
		!equalOptionalTime(left.Summary.StartedAt, right.Summary.StartedAt) ||
		!equalOptionalTime(left.Summary.EndedAt, right.Summary.EndedAt) ||
		left.Summary.JournalCount != right.Summary.JournalCount ||
		!bytes.Equal(left.Summary.SummaryArtifact, right.Summary.SummaryArtifact) ||
		len(left.Events) != len(right.Events) || len(left.Values) != len(right.Values) {
		return false
	}
	for index := range left.Events {
		a, b := left.Events[index], right.Events[index]
		if a.Sequence != b.Sequence || a.Kind != b.Kind ||
			!a.OccurredAt.Equal(b.OccurredAt) || !bytes.Equal(a.Artifact, b.Artifact) {
			return false
		}
	}
	for index := range left.Values {
		a, b := left.Values[index], right.Values[index]
		if a.Ordinal != b.Ordinal || a.ValueID != b.ValueID ||
			a.ValueDigest != b.ValueDigest || !bytes.Equal(a.Artifact, b.Artifact) ||
			!equalOptionalBlob(a.Blob, b.Blob) {
			return false
		}
	}
	return true
}

func equalOptionalTime(left, right *time.Time) bool {
	if left == nil || right == nil {
		return left == right
	}
	return left.Equal(*right)
}

func equalOptionalBlob(left, right *blob.BlobRef) bool {
	if left == nil || right == nil {
		return left == right
	}
	return *left == *right
}

func (r *RunRepository) ready() error {
	if r == nil || r.database == nil || r.database.db == nil {
		return errors.New("run repository is not open")
	}
	return nil
}
