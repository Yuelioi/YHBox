package catalog

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/blob"
)

// ObjectRepository owns persistent object observations, references, leases,
// and GC state. Physical byte operations remain in the Blob Store.
type ObjectRepository struct {
	database *database
}

type ObjectGCPlan struct {
	Objects        []blob.Object
	Candidates     []blob.Object
	LiveCount      int
	CandidateBytes int64
}

func (r *ObjectRepository) Observe(ctx context.Context, object blob.Object) error {
	if err := r.ready(); err != nil {
		return err
	}
	if !object.Digest.Valid() || object.Size < 0 {
		return errors.New("observed object identity is invalid")
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	tx, err := r.database.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO gc_objects(
			digest, size, physical_generation, state, observed_at, unreachable_since, last_error
		) VALUES (?, ?, 1, 'active', ?, NULL, NULL)
		ON CONFLICT(digest) DO UPDATE SET
			observed_at=excluded.observed_at,
			state='active',
			last_error=NULL
		WHERE gc_objects.size = excluded.size
	`, object.Digest.String(), object.Size, now); err != nil {
		_ = tx.Rollback()
		return err
	}
	var size int64
	if err := tx.QueryRowContext(ctx,
		"SELECT size FROM gc_objects WHERE digest = ?", object.Digest.String()).Scan(&size); err != nil {
		_ = tx.Rollback()
		return err
	}
	if size != object.Size {
		_ = tx.Rollback()
		return fmt.Errorf("object %s has conflicting observed sizes", object.Digest)
	}
	return tx.Commit()
}

func (r *ObjectRepository) Forget(ctx context.Context, digest artifact.Digest) error {
	if err := r.ready(); err != nil {
		return err
	}
	if !digest.Valid() {
		return errors.New("forgotten object digest is invalid")
	}
	tx, err := r.database.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM object_leases WHERE digest = ? AND expires_at <= ?
	`, digest.String(), now); err != nil {
		_ = tx.Rollback()
		return err
	}
	result, err := tx.ExecContext(ctx, `
		DELETE FROM gc_objects
		WHERE digest = ?
		  AND NOT EXISTS (SELECT 1 FROM object_refs WHERE object_refs.digest = gc_objects.digest)
		  AND NOT EXISTS (
			SELECT 1 FROM object_leases
			WHERE object_leases.digest = gc_objects.digest AND expires_at > ?
		  )
	`, digest.String(), now)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	if deleted == 0 {
		_ = tx.Rollback()
		return errors.New("object became referenced before physical deletion completed")
	}
	return tx.Commit()
}

func (r *ObjectRepository) Objects(ctx context.Context) ([]blob.Object, error) {
	if err := r.ready(); err != nil {
		return nil, err
	}
	rows, err := r.database.db.QueryContext(ctx,
		"SELECT digest, size FROM gc_objects ORDER BY digest")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []blob.Object
	for rows.Next() {
		var raw string
		var object blob.Object
		if err := rows.Scan(&raw, &object.Size); err != nil {
			return nil, err
		}
		object.Digest = artifact.Digest(raw)
		if !object.Digest.Valid() || object.Size < 0 {
			return nil, ErrSchemaDrift
		}
		result = append(result, object)
	}
	return result, rows.Err()
}

func (r *ObjectRepository) TotalBytes(ctx context.Context) (int64, error) {
	if err := r.ready(); err != nil {
		return 0, err
	}
	var total int64
	if err := r.database.db.QueryRowContext(ctx,
		"SELECT coalesce(sum(size), 0) FROM gc_objects").Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func (r *ObjectRepository) Object(
	ctx context.Context,
	digest artifact.Digest,
) (blob.Object, bool, error) {
	if err := r.ready(); err != nil {
		return blob.Object{}, false, err
	}
	if !digest.Valid() {
		return blob.Object{}, false, errors.New("object digest is invalid")
	}
	var object blob.Object
	err := r.database.db.QueryRowContext(ctx,
		"SELECT size FROM gc_objects WHERE digest = ?", digest.String()).Scan(&object.Size)
	if errors.Is(err, sql.ErrNoRows) {
		return blob.Object{}, false, nil
	}
	if err != nil {
		return blob.Object{}, false, err
	}
	object.Digest = digest
	return object, true, nil
}

func (r *ObjectRepository) Lease(
	ctx context.Context,
	token string,
	digest artifact.Digest,
	ownerKind string,
	ownerID string,
	expiresAt time.Time,
) error {
	if err := r.ready(); err != nil {
		return err
	}
	if strings.TrimSpace(token) == "" || !digest.Valid() ||
		strings.TrimSpace(ownerKind) == "" || strings.TrimSpace(ownerID) == "" ||
		!expiresAt.After(time.Now().UTC()) {
		return errors.New("object lease is invalid")
	}
	result, err := r.database.db.ExecContext(ctx, `
		INSERT INTO object_leases(token, digest, owner_kind, owner_id, expires_at)
		SELECT ?, digest, ?, ?, ? FROM gc_objects
		WHERE digest = ? AND state = 'active'
		ON CONFLICT(token) DO UPDATE SET
			digest=excluded.digest,
			owner_kind=excluded.owner_kind,
			owner_id=excluded.owner_id,
			expires_at=excluded.expires_at
	`, token, ownerKind, ownerID, expiresAt.UTC().Format(time.RFC3339Nano), digest.String())
	if err != nil {
		return err
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if updated == 0 {
		return errors.New("leased object is not active in the CAS inventory")
	}
	return nil
}

func (r *ObjectRepository) ReleaseLease(ctx context.Context, token string) error {
	if err := r.ready(); err != nil {
		return err
	}
	if strings.TrimSpace(token) == "" {
		return errors.New("object lease token is required")
	}
	_, err := r.database.db.ExecContext(ctx,
		"DELETE FROM object_leases WHERE token = ?", token)
	return err
}

// PlanGC atomically refreshes reachability timestamps and returns only objects
// that have remained unreachable for the complete grace period. Catalog
// references, unexpired durable leases, and caller-supplied roots are all live.
func (r *ObjectRepository) PlanGC(
	ctx context.Context,
	external []blob.BlobRef,
	now time.Time,
	grace time.Duration,
) (ObjectGCPlan, error) {
	if err := r.ready(); err != nil {
		return ObjectGCPlan{}, err
	}
	if now.IsZero() || grace < 0 {
		return ObjectGCPlan{}, errors.New("object GC policy is invalid")
	}
	byDigest := make(map[artifact.Digest]int64, len(external))
	for _, ref := range external {
		if err := ref.Validate(); err != nil {
			return ObjectGCPlan{}, err
		}
		if size, exists := byDigest[ref.Digest]; exists && size != ref.Size {
			return ObjectGCPlan{}, fmt.Errorf("object %s has conflicting live sizes", ref.Digest)
		}
		byDigest[ref.Digest] = ref.Size
	}
	tx, err := r.database.db.BeginTx(ctx, nil)
	if err != nil {
		return ObjectGCPlan{}, err
	}
	rollback := func(cause error) (ObjectGCPlan, error) {
		return ObjectGCPlan{}, errors.Join(cause, tx.Rollback())
	}
	if _, err := tx.ExecContext(ctx, `
		CREATE TEMP TABLE IF NOT EXISTS gc_external_live(
			digest TEXT PRIMARY KEY NOT NULL,
			size INTEGER NOT NULL
		) STRICT
	`); err != nil {
		return rollback(err)
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM gc_external_live"); err != nil {
		return rollback(err)
	}
	for digest, size := range byDigest {
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO gc_external_live(digest, size) VALUES (?, ?)",
			digest.String(), size); err != nil {
			return rollback(err)
		}
	}
	nowText := now.UTC().Format(time.RFC3339Nano)
	if _, err := tx.ExecContext(ctx,
		"DELETE FROM object_leases WHERE expires_at <= ?", nowText); err != nil {
		return rollback(err)
	}
	livePredicate := `
		EXISTS (SELECT 1 FROM object_refs refs WHERE refs.digest = gc_objects.digest)
		OR EXISTS (SELECT 1 FROM object_leases leases WHERE leases.digest = gc_objects.digest)
		OR EXISTS (SELECT 1 FROM gc_external_live external WHERE external.digest = gc_objects.digest)
	`
	if _, err := tx.ExecContext(ctx, `
		UPDATE gc_objects SET unreachable_since = NULL
		WHERE `+livePredicate); err != nil {
		return rollback(err)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE gc_objects SET unreachable_since = ?
		WHERE unreachable_since IS NULL AND NOT (`+livePredicate+`)
	`, nowText); err != nil {
		return rollback(err)
	}
	plan, err := readObjectGCPlan(ctx, tx, now.Add(-grace))
	if err != nil {
		return rollback(err)
	}
	if err := tx.Commit(); err != nil {
		return ObjectGCPlan{}, err
	}
	return plan, nil
}

func readObjectGCPlan(ctx context.Context, tx *sql.Tx, cutoff time.Time) (ObjectGCPlan, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT digest, size,
			CASE WHEN unreachable_since IS NOT NULL AND unreachable_since <= ? THEN 1 ELSE 0 END
		FROM gc_objects
		ORDER BY digest
	`, cutoff.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return ObjectGCPlan{}, err
	}
	defer rows.Close()
	var result ObjectGCPlan
	for rows.Next() {
		var raw string
		var object blob.Object
		var candidate int
		if err := rows.Scan(&raw, &object.Size, &candidate); err != nil {
			return ObjectGCPlan{}, err
		}
		object.Digest = artifact.Digest(raw)
		if !object.Digest.Valid() || object.Size < 0 {
			return ObjectGCPlan{}, ErrSchemaDrift
		}
		result.Objects = append(result.Objects, object)
		if candidate != 0 {
			result.Candidates = append(result.Candidates, object)
			result.CandidateBytes += object.Size
		} else {
			result.LiveCount++
		}
	}
	return result, rows.Err()
}

func (r *ObjectRepository) ready() error {
	if r == nil || r.database == nil || r.database.db == nil {
		return errors.New("object repository is not open")
	}
	return nil
}
