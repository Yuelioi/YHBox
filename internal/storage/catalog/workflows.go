package catalog

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/blob"
)

const workflowObjectOwnerKind = "workflow-source"

var ErrWorkflowSourceConflict = errors.New("workflow source revision conflict")

// WorkflowSourceRecord is the local Catalog projection of one canonical,
// portable Workflow Source artifact.
type WorkflowSourceRecord struct {
	WorkflowID string
	Name       string
	Revision   int64
	Hash       artifact.Digest
	Format     string
	Version    string
	Artifact   []byte
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type WorkflowReference struct {
	Role string
	Blob blob.BlobRef
}

type WorkflowQuarantineRecord struct {
	ID           artifact.Digest
	OriginalName string
	Reason       string
	Artifact     []byte
	CreatedAt    time.Time
}

// WorkflowRepository owns Workflow Source metadata, canonical bytes, CAS
// references, and isolated invalid artifacts in the Content Catalog.
type WorkflowRepository struct {
	database *database
}

func (r *WorkflowRepository) Count(ctx context.Context) (int, error) {
	if err := r.ready(); err != nil {
		return 0, err
	}
	var count int
	err := r.database.db.QueryRowContext(ctx, "SELECT count(*) FROM workflow_sources").Scan(&count)
	return count, err
}

func (r *WorkflowRepository) Get(ctx context.Context, workflowID string) (WorkflowSourceRecord, bool, error) {
	if err := r.ready(); err != nil {
		return WorkflowSourceRecord{}, false, err
	}
	if strings.TrimSpace(workflowID) == "" {
		return WorkflowSourceRecord{}, false, errors.New("workflow identity is required")
	}
	row := r.database.db.QueryRowContext(ctx, `
		SELECT workflow_id, name, revision, source_hash, format, version, artifact, created_at, updated_at
		FROM workflow_sources WHERE workflow_id = ?
	`, workflowID)
	record, err := scanWorkflowSource(row)
	if errors.Is(err, sql.ErrNoRows) {
		return WorkflowSourceRecord{}, false, nil
	}
	return record, err == nil, err
}

func (r *WorkflowRepository) List(ctx context.Context) ([]WorkflowSourceRecord, error) {
	if err := r.ready(); err != nil {
		return nil, err
	}
	rows, err := r.database.db.QueryContext(ctx, `
		SELECT workflow_id, name, revision, source_hash, format, version, artifact, created_at, updated_at
		FROM workflow_sources ORDER BY workflow_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []WorkflowSourceRecord
	for rows.Next() {
		record, err := scanWorkflowSource(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, record)
	}
	return result, rows.Err()
}

// Commit applies one exact revision transition and replaces the complete CAS
// reference set in the same Content Catalog transaction.
func (r *WorkflowRepository) Commit(
	ctx context.Context,
	baseRevision int64,
	record WorkflowSourceRecord,
	references []WorkflowReference,
) error {
	if err := r.ready(); err != nil {
		return err
	}
	if err := validateWorkflowSourceRecord(record); err != nil {
		return err
	}
	if err := validateWorkflowReferences(references); err != nil {
		return err
	}
	tx, err := r.database.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := commitWorkflowSource(ctx, tx, baseRevision, record, references); err != nil {
		return errors.Join(err, tx.Rollback())
	}
	return tx.Commit()
}

// PublishMigration atomically replaces one exact Workflow Source artifact
// without inventing a user edit revision. The previous hash is the migration
// CAS; the workflow identity and revision in record remain unchanged.
func (r *WorkflowRepository) PublishMigration(
	ctx context.Context,
	previousHash artifact.Digest,
	record WorkflowSourceRecord,
	references []WorkflowReference,
) error {
	if err := r.ready(); err != nil {
		return err
	}
	if !previousHash.Valid() {
		return errors.New("workflow migration requires the previous source hash")
	}
	if err := validateWorkflowSourceRecord(record); err != nil {
		return err
	}
	if err := validateWorkflowReferences(references); err != nil {
		return err
	}
	tx, err := r.database.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `
		UPDATE workflow_sources SET
			name = ?, source_hash = ?, format = ?, version = ?, artifact = ?, updated_at = ?
		WHERE workflow_id = ? AND revision = ? AND source_hash = ?
	`, record.Name, record.Hash.String(), record.Format, record.Version, record.Artifact,
		record.UpdatedAt.UTC().Format(time.RFC3339Nano), record.WorkflowID, record.Revision, previousHash.String())
	if err != nil {
		return errors.Join(err, tx.Rollback())
	}
	updated, err := result.RowsAffected()
	if err != nil || updated != 1 {
		if err == nil {
			err = ErrWorkflowSourceConflict
		}
		return errors.Join(err, tx.Rollback())
	}
	if err := replaceWorkflowReferences(ctx, tx, record.WorkflowID, references); err != nil {
		return errors.Join(err, tx.Rollback())
	}
	return tx.Commit()
}

func (r *WorkflowRepository) Delete(
	ctx context.Context,
	workflowID string,
	revision int64,
	hash artifact.Digest,
) (bool, error) {
	if err := r.ready(); err != nil {
		return false, err
	}
	if strings.TrimSpace(workflowID) == "" || revision < 0 || !hash.Valid() {
		return false, errors.New("workflow delete requires exact identity")
	}
	tx, err := r.database.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	if _, err := tx.ExecContext(ctx,
		"DELETE FROM object_refs WHERE owner_kind = ? AND owner_id = ?",
		workflowObjectOwnerKind, workflowID); err != nil {
		return false, errors.Join(err, tx.Rollback())
	}
	result, err := tx.ExecContext(ctx, `
		DELETE FROM workflow_sources
		WHERE workflow_id = ? AND revision = ? AND source_hash = ?
	`, workflowID, revision, hash.String())
	if err != nil {
		return false, errors.Join(err, tx.Rollback())
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return false, errors.Join(err, tx.Rollback())
	}
	if deleted == 0 {
		return false, errors.Join(ErrWorkflowSourceConflict, tx.Rollback())
	}
	return true, tx.Commit()
}

func (r *WorkflowRepository) PutQuarantine(ctx context.Context, record WorkflowQuarantineRecord) error {
	if err := r.ready(); err != nil {
		return err
	}
	if err := validateWorkflowQuarantine(record); err != nil {
		return err
	}
	result, err := r.database.db.ExecContext(ctx, `
		INSERT INTO workflow_quarantine(recovery_id, original_name, reason, artifact, created_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(recovery_id) DO NOTHING
	`, record.ID.String(), record.OriginalName, record.Reason, record.Artifact,
		record.CreatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return err
	}
	inserted, err := result.RowsAffected()
	if err != nil || inserted == 1 {
		return err
	}
	var originalName, reason string
	var artifactBytes []byte
	if err := r.database.db.QueryRowContext(ctx, `
		SELECT original_name, reason, artifact FROM workflow_quarantine WHERE recovery_id = ?
	`, record.ID.String()).Scan(&originalName, &reason, &artifactBytes); err != nil {
		return err
	}
	if originalName != record.OriginalName || reason != record.Reason || !bytes.Equal(artifactBytes, record.Artifact) {
		return errors.New("workflow quarantine identity collision")
	}
	return nil
}

func (r *WorkflowRepository) ListQuarantine(ctx context.Context) ([]WorkflowQuarantineRecord, error) {
	if err := r.ready(); err != nil {
		return nil, err
	}
	rows, err := r.database.db.QueryContext(ctx, `
		SELECT recovery_id, original_name, reason, artifact, created_at
		FROM workflow_quarantine ORDER BY recovery_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []WorkflowQuarantineRecord
	for rows.Next() {
		var rawID, rawCreated string
		var record WorkflowQuarantineRecord
		if err := rows.Scan(&rawID, &record.OriginalName, &record.Reason, &record.Artifact, &rawCreated); err != nil {
			return nil, err
		}
		record.ID = artifact.Digest(rawID)
		record.CreatedAt, err = time.Parse(time.RFC3339Nano, rawCreated)
		if err != nil || validateWorkflowQuarantine(record) != nil {
			return nil, ErrSchemaDrift
		}
		record.Artifact = append([]byte(nil), record.Artifact...)
		result = append(result, record)
	}
	return result, rows.Err()
}

// Repair atomically publishes a replacement Source and removes one exact
// quarantine record.
func (r *WorkflowRepository) Repair(
	ctx context.Context,
	recoveryID artifact.Digest,
	record WorkflowSourceRecord,
	references []WorkflowReference,
) error {
	if err := r.ready(); err != nil {
		return err
	}
	if !recoveryID.Valid() {
		return errors.New("workflow recovery identity is invalid")
	}
	if err := validateWorkflowSourceRecord(record); err != nil {
		return err
	}
	if record.Revision != 0 {
		return errors.New("repaired workflow source must begin at revision zero")
	}
	if err := validateWorkflowReferences(references); err != nil {
		return err
	}
	tx, err := r.database.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	var exists int
	if err := tx.QueryRowContext(ctx,
		"SELECT count(*) FROM workflow_quarantine WHERE recovery_id = ?",
		recoveryID.String()).Scan(&exists); err != nil || exists != 1 {
		if err == nil {
			err = errors.New("workflow source recovery not found")
		}
		return errors.Join(err, tx.Rollback())
	}
	if err := commitWorkflowSource(ctx, tx, -1, record, references); err != nil {
		return errors.Join(err, tx.Rollback())
	}
	if _, err := tx.ExecContext(ctx,
		"DELETE FROM workflow_quarantine WHERE recovery_id = ?", recoveryID.String()); err != nil {
		return errors.Join(err, tx.Rollback())
	}
	return tx.Commit()
}

func (r *WorkflowRepository) DeleteQuarantine(ctx context.Context, recoveryID artifact.Digest) (bool, error) {
	if err := r.ready(); err != nil {
		return false, err
	}
	if !recoveryID.Valid() {
		return false, errors.New("workflow recovery identity is invalid")
	}
	result, err := r.database.db.ExecContext(ctx,
		"DELETE FROM workflow_quarantine WHERE recovery_id = ?", recoveryID.String())
	if err != nil {
		return false, err
	}
	deleted, err := result.RowsAffected()
	return deleted != 0, err
}

func commitWorkflowSource(
	ctx context.Context,
	tx *sql.Tx,
	baseRevision int64,
	record WorkflowSourceRecord,
	references []WorkflowReference,
) error {
	now := record.UpdatedAt.UTC().Format(time.RFC3339Nano)
	if baseRevision == -1 {
		if record.Revision != 0 {
			return errors.New("new workflow source must begin at revision zero")
		}
		created := record.CreatedAt
		if created.IsZero() {
			created = record.UpdatedAt
		}
		result, err := tx.ExecContext(ctx, `
			INSERT INTO workflow_sources(
				workflow_id, name, revision, source_hash, format, version, artifact, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(workflow_id) DO NOTHING
		`, record.WorkflowID, record.Name, record.Revision, record.Hash.String(),
			record.Format, record.Version, record.Artifact,
			created.UTC().Format(time.RFC3339Nano), now)
		if err != nil {
			return err
		}
		inserted, err := result.RowsAffected()
		if err != nil || inserted != 1 {
			if err == nil {
				err = ErrWorkflowSourceConflict
			}
			return err
		}
	} else {
		if record.Revision != baseRevision+1 {
			return ErrWorkflowSourceConflict
		}
		result, err := tx.ExecContext(ctx, `
			UPDATE workflow_sources SET
				name = ?, revision = ?, source_hash = ?, format = ?, version = ?,
				artifact = ?, updated_at = ?
			WHERE workflow_id = ? AND revision = ?
		`, record.Name, record.Revision, record.Hash.String(), record.Format, record.Version,
			record.Artifact, now, record.WorkflowID, baseRevision)
		if err != nil {
			return err
		}
		updated, err := result.RowsAffected()
		if err != nil || updated != 1 {
			if err == nil {
				err = ErrWorkflowSourceConflict
			}
			return err
		}
	}
	return replaceWorkflowReferences(ctx, tx, record.WorkflowID, references)
}

func replaceWorkflowReferences(
	ctx context.Context,
	tx *sql.Tx,
	workflowID string,
	references []WorkflowReference,
) error {
	if _, err := tx.ExecContext(ctx,
		"DELETE FROM object_refs WHERE owner_kind = ? AND owner_id = ?",
		workflowObjectOwnerKind, workflowID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		"DELETE FROM workflow_refs WHERE workflow_id = ?", workflowID); err != nil {
		return err
	}
	for _, reference := range references {
		result, err := tx.ExecContext(ctx, `
			INSERT INTO workflow_refs(workflow_id, role, digest, media_type, size)
			SELECT ?, ?, digest, ?, ? FROM gc_objects
			WHERE digest = ? AND size = ? AND state = 'active'
		`, workflowID, reference.Role, reference.Blob.MediaType, reference.Blob.Size,
			reference.Blob.Digest.String(), reference.Blob.Size)
		if err != nil {
			return err
		}
		inserted, err := result.RowsAffected()
		if err != nil || inserted != 1 {
			if err == nil {
				err = fmt.Errorf("workflow object %s is not active in the CAS inventory", reference.Blob.Digest)
			}
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO object_refs(owner_kind, owner_id, role, digest, media_type, size)
			VALUES (?, ?, ?, ?, ?, ?)
		`, workflowObjectOwnerKind, workflowID, reference.Role,
			reference.Blob.Digest.String(), reference.Blob.MediaType, reference.Blob.Size); err != nil {
			return err
		}
	}
	return nil
}

type workflowSourceScanner interface {
	Scan(...any) error
}

func scanWorkflowSource(scanner workflowSourceScanner) (WorkflowSourceRecord, error) {
	var record WorkflowSourceRecord
	var rawHash, rawCreated, rawUpdated string
	if err := scanner.Scan(
		&record.WorkflowID, &record.Name, &record.Revision, &rawHash,
		&record.Format, &record.Version, &record.Artifact, &rawCreated, &rawUpdated,
	); err != nil {
		return WorkflowSourceRecord{}, err
	}
	record.Hash = artifact.Digest(rawHash)
	var err error
	record.CreatedAt, err = time.Parse(time.RFC3339Nano, rawCreated)
	if err != nil {
		return WorkflowSourceRecord{}, ErrSchemaDrift
	}
	record.UpdatedAt, err = time.Parse(time.RFC3339Nano, rawUpdated)
	if err != nil || validateWorkflowSourceRecord(record) != nil {
		return WorkflowSourceRecord{}, ErrSchemaDrift
	}
	record.Artifact = append([]byte(nil), record.Artifact...)
	return record, nil
}

func validateWorkflowSourceRecord(record WorkflowSourceRecord) error {
	if strings.TrimSpace(record.WorkflowID) == "" || strings.TrimSpace(record.Name) == "" ||
		record.Revision < 0 || !record.Hash.Valid() || strings.TrimSpace(record.Format) == "" ||
		strings.TrimSpace(record.Version) == "" || len(record.Artifact) == 0 ||
		record.UpdatedAt.IsZero() {
		return errors.New("workflow source record is invalid")
	}
	return nil
}

func validateWorkflowReferences(references []WorkflowReference) error {
	roles := make(map[string]struct{}, len(references))
	for _, reference := range references {
		if strings.TrimSpace(reference.Role) == "" || reference.Blob.Validate() != nil {
			return errors.New("workflow source reference is invalid")
		}
		if _, duplicate := roles[reference.Role]; duplicate {
			return errors.New("workflow source reference role is duplicated")
		}
		roles[reference.Role] = struct{}{}
	}
	return nil
}

func validateWorkflowQuarantine(record WorkflowQuarantineRecord) error {
	if !record.ID.Valid() || strings.TrimSpace(record.OriginalName) == "" ||
		record.OriginalName != filepath.Base(record.OriginalName) ||
		strings.TrimSpace(record.Reason) == "" || len(record.Artifact) == 0 ||
		record.CreatedAt.IsZero() {
		return errors.New("workflow quarantine record is invalid")
	}
	return nil
}

func (r *WorkflowRepository) ready() error {
	if r == nil || r.database == nil || r.database.db == nil {
		return errors.New("workflow repository is not open")
	}
	return nil
}
