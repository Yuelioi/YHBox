package catalog

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

type schemaObject struct {
	kind string
	name string
}

var foundationSchemaObjects = []schemaObject{
	{kind: "table", name: "meta"},
	{kind: "table", name: "schema_migrations"},
	{kind: "index", name: "idx_schema_migrations_to_version"},
}

type migration struct {
	id         string
	from       int
	to         int
	statements []string
}

func (m migration) checksum() string {
	sum := sha256.Sum256([]byte(strings.Join(m.statements, "\n-- statement boundary --\n")))
	return hex.EncodeToString(sum[:])
}

var migrationIDPattern = regexp.MustCompile(`^[a-z][a-z0-9._-]{2,127}$`)

var foundationStatements = []string{
	`CREATE TABLE meta (
		key TEXT PRIMARY KEY NOT NULL,
		value TEXT NOT NULL
	) STRICT`,
	`CREATE TABLE schema_migrations (
		id TEXT PRIMARY KEY NOT NULL,
		ordinal INTEGER NOT NULL UNIQUE CHECK (ordinal > 0),
		from_version INTEGER NOT NULL CHECK (from_version >= 0),
		to_version INTEGER NOT NULL UNIQUE CHECK (to_version > from_version),
		checksum TEXT NOT NULL CHECK (length(checksum) = 64),
		applied_at TEXT NOT NULL
	) STRICT`,
	`CREATE INDEX idx_schema_migrations_to_version ON schema_migrations(to_version)`,
}

var contentMigrations = []migration{{
	id: "content.foundation.1", from: 0, to: 1, statements: foundationStatements,
}}

var runMigrations = []migration{{
	id: "runs.foundation.1", from: 0, to: 1, statements: foundationStatements,
}}

func (d *database) prepare(ctx context.Context, faults faultHooks) error {
	if err := validateRegistry(d.spec); err != nil {
		return err
	}
	identity, err := readIdentity(ctx, d.db)
	if err != nil {
		return fmt.Errorf("read %s identity: %w", d.spec.kind, err)
	}
	if identity.applicationID != 0 && identity.applicationID != d.spec.applicationID {
		return fmt.Errorf("%w: %s has 0x%08X, require 0x%08X",
			ErrWrongDatabase, d.spec.kind, identity.applicationID, d.spec.applicationID)
	}
	if identity.userVersion > d.spec.currentVersion {
		return fmt.Errorf("%w: %s has %d, current %d",
			ErrFutureSchema, d.spec.kind, identity.userVersion, d.spec.currentVersion)
	}
	if identity.applicationID == 0 {
		if identity.userVersion != 0 {
			return fmt.Errorf("%w: %s has user_version %d without an application ID",
				ErrUnclaimedDatabase, d.spec.kind, identity.userVersion)
		}
		objects, err := userObjectCount(ctx, d.db)
		if err != nil {
			return fmt.Errorf("inspect unclaimed %s: %w", d.spec.kind, err)
		}
		if objects != 0 {
			return fmt.Errorf("%w: %s contains %d user schema objects",
				ErrUnclaimedDatabase, d.spec.kind, objects)
		}
	}
	if err := d.applyMigrations(ctx, identity.userVersion, faults); err != nil {
		return err
	}
	return d.validate(ctx)
}

type databaseIdentity struct {
	applicationID int64
	userVersion   int
}

func readIdentity(ctx context.Context, query interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}) (databaseIdentity, error) {
	var identity databaseIdentity
	if err := query.QueryRowContext(ctx, "PRAGMA application_id").Scan(&identity.applicationID); err != nil {
		return databaseIdentity{}, err
	}
	if err := query.QueryRowContext(ctx, "PRAGMA user_version").Scan(&identity.userVersion); err != nil {
		return databaseIdentity{}, err
	}
	return identity, nil
}

func userObjectCount(ctx context.Context, db *sql.DB) (int, error) {
	var count int
	err := db.QueryRowContext(ctx, `
		SELECT count(*)
		FROM sqlite_schema
		WHERE name NOT LIKE 'sqlite_%'
	`).Scan(&count)
	return count, err
}

func validateRegistry(spec databaseSpec) error {
	if spec.currentVersion < 1 || len(spec.migrations) == 0 {
		return fmt.Errorf("%s migration registry is empty", spec.kind)
	}
	expected := 0
	ids := make(map[string]struct{}, len(spec.migrations))
	for ordinal, item := range spec.migrations {
		if !migrationIDPattern.MatchString(item.id) {
			return fmt.Errorf("%s migration %d has invalid ID %q", spec.kind, ordinal+1, item.id)
		}
		if _, exists := ids[item.id]; exists {
			return fmt.Errorf("%s migration ID %q is duplicated", spec.kind, item.id)
		}
		ids[item.id] = struct{}{}
		if item.from != expected || item.to != item.from+1 {
			return fmt.Errorf("%s migration %q does not form a contiguous chain", spec.kind, item.id)
		}
		if len(item.statements) == 0 || len(item.checksum()) != 64 {
			return fmt.Errorf("%s migration %q has no valid implementation checksum", spec.kind, item.id)
		}
		expected = item.to
	}
	if expected != spec.currentVersion {
		return fmt.Errorf("%s migration registry ends at %d, require %d", spec.kind, expected, spec.currentVersion)
	}
	return nil
}

func (d *database) applyMigrations(ctx context.Context, from int, faults faultHooks) error {
	for ordinal, item := range d.spec.migrations {
		if item.to <= from {
			continue
		}
		if item.from != from {
			return fmt.Errorf("%w: %s has no migration from %d", ErrSchemaDrift, d.spec.kind, from)
		}
		tx, err := d.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin %s migration %q: %w", d.spec.kind, item.id, err)
		}
		rollback := func(cause error) error {
			return errors.Join(cause, tx.Rollback())
		}
		for _, statement := range item.statements {
			if _, err := tx.ExecContext(ctx, statement); err != nil {
				return rollback(fmt.Errorf("apply %s migration %q: %w", d.spec.kind, item.id, err))
			}
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO meta(key, value) VALUES ('database_kind', ?), ('schema_version', ?)
			ON CONFLICT(key) DO UPDATE SET value = excluded.value
		`, string(d.spec.kind), fmt.Sprint(item.to)); err != nil {
			return rollback(fmt.Errorf("record %s metadata: %w", d.spec.kind, err))
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO schema_migrations(
				id, ordinal, from_version, to_version, checksum, applied_at
			) VALUES (?, ?, ?, ?, ?, ?)
		`, item.id, ordinal+1, item.from, item.to, item.checksum(), time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
			return rollback(fmt.Errorf("record %s migration %q: %w", d.spec.kind, item.id, err))
		}
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("PRAGMA application_id = %d", d.spec.applicationID)); err != nil {
			return rollback(fmt.Errorf("claim %s application identity: %w", d.spec.kind, err))
		}
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", item.to)); err != nil {
			return rollback(fmt.Errorf("advance %s schema version: %w", d.spec.kind, err))
		}
		if faults.beforeMigrationCommit != nil {
			if err := faults.beforeMigrationCommit(d.spec.kind, item.to); err != nil {
				return rollback(err)
			}
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit %s migration %q: %w", d.spec.kind, item.id, err)
		}
		from = item.to
	}
	return nil
}

func (d *database) validate(ctx context.Context) error {
	identity, err := readIdentity(ctx, d.db)
	if err != nil {
		return fmt.Errorf("validate %s identity: %w", d.spec.kind, err)
	}
	if identity.applicationID != d.spec.applicationID {
		return fmt.Errorf("%w: %s application ID is 0x%08X", ErrSchemaDrift, d.spec.kind, identity.applicationID)
	}
	if identity.userVersion != d.spec.currentVersion {
		return fmt.Errorf("%w: %s schema is %d, require %d",
			ErrSchemaDrift, d.spec.kind, identity.userVersion, d.spec.currentVersion)
	}
	for _, object := range d.spec.requiredObjects {
		var count int
		if err := d.db.QueryRowContext(ctx,
			`SELECT count(*) FROM sqlite_schema WHERE type = ? AND name = ?`,
			object.kind, object.name).Scan(&count); err != nil {
			return fmt.Errorf("validate %s object %s: %w", d.spec.kind, object.name, err)
		}
		if count != 1 {
			return fmt.Errorf("%w: %s is missing required %s %q",
				ErrSchemaDrift, d.spec.kind, object.kind, object.name)
		}
	}
	return d.validateMigrationLedger(ctx)
}

func (d *database) validateMigrationLedger(ctx context.Context) error {
	rows, err := d.db.QueryContext(ctx, `
		SELECT id, ordinal, from_version, to_version, checksum
		FROM schema_migrations
		ORDER BY ordinal
	`)
	if err != nil {
		return fmt.Errorf("read %s migration ledger: %w", d.spec.kind, err)
	}
	defer rows.Close()
	type applied struct {
		id                string
		ordinal, from, to int
		checksum          string
	}
	var actual []applied
	for rows.Next() {
		var item applied
		if err := rows.Scan(&item.id, &item.ordinal, &item.from, &item.to, &item.checksum); err != nil {
			return err
		}
		actual = append(actual, item)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(actual) != len(d.spec.migrations) {
		return fmt.Errorf("%w: %s has %d migration rows, require %d",
			ErrSchemaDrift, d.spec.kind, len(actual), len(d.spec.migrations))
	}
	for index, expected := range d.spec.migrations {
		item := actual[index]
		if item.id != expected.id || item.ordinal != index+1 || item.from != expected.from ||
			item.to != expected.to || item.checksum != expected.checksum() {
			return fmt.Errorf("%w: %s migration %d does not match compiled registry",
				ErrSchemaDrift, d.spec.kind, index+1)
		}
	}
	return nil
}
