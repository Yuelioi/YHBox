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

var contentSchemaObjects = append([]schemaObject{}, append(foundationSchemaObjects,
	[]schemaObject{
		{kind: "table", name: "assets"},
		{kind: "table", name: "asset_variants"},
		{kind: "table", name: "asset_tags"},
		{kind: "table", name: "object_refs"},
		{kind: "table", name: "gc_objects"},
		{kind: "table", name: "object_leases"},
		{kind: "table", name: "workflow_sources"},
		{kind: "table", name: "workflow_refs"},
		{kind: "table", name: "workflow_quarantine"},
		{kind: "table", name: "workflow_releases"},
		{kind: "table", name: "workflow_installations"},
		{kind: "table", name: "workflow_installation_configurations"},
		{kind: "index", name: "idx_assets_kind_name"},
		{kind: "index", name: "idx_assets_kind_created"},
		{kind: "index", name: "idx_asset_tags_normalized"},
		{kind: "index", name: "idx_object_refs_digest"},
		{kind: "index", name: "idx_gc_objects_state_unreachable"},
		{kind: "index", name: "idx_object_leases_expiry"},
		{kind: "index", name: "idx_workflow_sources_name"},
		{kind: "index", name: "idx_workflow_refs_digest"},
		{kind: "index", name: "idx_workflow_quarantine_created"},
		{kind: "index", name: "idx_workflow_releases_source"},
		{kind: "index", name: "idx_workflow_installations_release"},
		{kind: "index", name: "idx_workflow_installations_lifecycle"},
	}...)...)

var runSchemaObjects = append([]schemaObject{}, append(foundationSchemaObjects,
	[]schemaObject{
		{kind: "table", name: "runs"},
		{kind: "table", name: "run_events"},
		{kind: "table", name: "run_values"},
		{kind: "index", name: "idx_runs_status_queued"},
		{kind: "index", name: "idx_runs_archived_ended"},
		{kind: "index", name: "idx_run_events_occurred"},
		{kind: "index", name: "idx_run_values_blob"},
	}...)...)

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

var contentMigrations = []migration{
	{id: "content.foundation.1", from: 0, to: 1, statements: foundationStatements},
	{id: "content.assets-and-objects.2", from: 1, to: 2, statements: assetCatalogStatements},
	{id: "content.workflow-sources.3", from: 2, to: 3, statements: workflowCatalogStatements},
	{id: "content.workflow-installations.4", from: 3, to: 4, statements: workflowInstallationStatements},
	{id: "content.workflow-installation-configurations.5", from: 4, to: 5, statements: workflowInstallationConfigurationStatements},
}

var runMigrations = []migration{
	{id: "runs.foundation.1", from: 0, to: 1, statements: foundationStatements},
	{id: "runs.ledger.2", from: 1, to: 2, statements: runLedgerStatements},
}

var runLedgerStatements = []string{
	`CREATE TABLE runs (
		run_id TEXT PRIMARY KEY NOT NULL,
		generation INTEGER NOT NULL CHECK (generation > 0),
		record_digest TEXT NOT NULL CHECK (length(record_digest) = 71),
		status TEXT NOT NULL CHECK (status IN (
			'queued', 'running', 'succeeded', 'failed', 'cancelled', 'interrupted'
		)),
		queued_at TEXT NOT NULL,
		started_at TEXT,
		ended_at TEXT,
		summary_artifact BLOB NOT NULL CHECK (length(summary_artifact) > 0),
		journal_count INTEGER NOT NULL CHECK (journal_count >= 0),
		archived_at TEXT,
		updated_at TEXT NOT NULL
	) STRICT`,
	`CREATE TABLE run_events (
		run_id TEXT NOT NULL REFERENCES runs(run_id) ON DELETE CASCADE,
		sequence INTEGER NOT NULL CHECK (sequence > 0),
		kind TEXT NOT NULL,
		occurred_at TEXT NOT NULL,
		artifact BLOB NOT NULL CHECK (length(artifact) > 0),
		PRIMARY KEY (run_id, sequence)
	) STRICT`,
	`CREATE TABLE run_values (
		run_id TEXT NOT NULL REFERENCES runs(run_id) ON DELETE CASCADE,
		ordinal INTEGER NOT NULL CHECK (ordinal >= 0),
		value_id TEXT NOT NULL,
		value_digest TEXT NOT NULL CHECK (length(value_digest) = 71),
		artifact BLOB NOT NULL CHECK (length(artifact) > 0),
		blob_media_type TEXT,
		blob_digest TEXT,
		blob_size INTEGER,
		PRIMARY KEY (run_id, value_id),
		UNIQUE (run_id, ordinal),
		CHECK (
			(blob_media_type IS NULL AND blob_digest IS NULL AND blob_size IS NULL)
			OR
			(blob_media_type IS NOT NULL AND length(blob_digest) = 71 AND blob_size >= 0)
		)
	) STRICT`,
	`CREATE INDEX idx_runs_status_queued ON runs(status, queued_at, run_id)`,
	`CREATE INDEX idx_runs_archived_ended ON runs(archived_at, ended_at, run_id)`,
	`CREATE INDEX idx_run_events_occurred ON run_events(run_id, occurred_at, sequence)`,
	`CREATE INDEX idx_run_values_blob ON run_values(blob_digest, run_id) WHERE blob_digest IS NOT NULL`,
}

var assetCatalogStatements = []string{
	`CREATE TABLE assets (
		guid TEXT PRIMARY KEY NOT NULL,
		kind TEXT NOT NULL CHECK (kind IN ('template', 'clip', 'macro')),
		name TEXT NOT NULL,
		description TEXT NOT NULL,
		category TEXT NOT NULL,
		origin_kind TEXT NOT NULL,
		origin_source_id TEXT NOT NULL,
		created_at TEXT NOT NULL,
		record_revision INTEGER NOT NULL CHECK (record_revision > 0),
		record_blob_media_type TEXT,
		record_blob_digest TEXT,
		record_blob_size INTEGER,
		CHECK (
			(record_blob_media_type IS NULL AND record_blob_digest IS NULL AND record_blob_size IS NULL)
			OR
			(record_blob_media_type IS NOT NULL AND record_blob_digest IS NOT NULL AND record_blob_size >= 0)
		)
	) STRICT`,
	`CREATE TABLE asset_variants (
		asset_guid TEXT NOT NULL REFERENCES assets(guid) ON DELETE CASCADE,
		ordinal INTEGER NOT NULL CHECK (ordinal >= 0),
		width INTEGER NOT NULL CHECK (width > 0),
		height INTEGER NOT NULL CHECK (height > 0),
		bbox_x1 INTEGER NOT NULL,
		bbox_y1 INTEGER NOT NULL,
		bbox_x2 INTEGER NOT NULL,
		bbox_y2 INTEGER NOT NULL,
		regions_json TEXT NOT NULL,
		blob_media_type TEXT NOT NULL,
		blob_digest TEXT NOT NULL,
		blob_size INTEGER NOT NULL CHECK (blob_size >= 0),
		PRIMARY KEY (asset_guid, width, height),
		UNIQUE (asset_guid, ordinal)
	) STRICT`,
	`CREATE TABLE asset_tags (
		asset_guid TEXT NOT NULL REFERENCES assets(guid) ON DELETE CASCADE,
		ordinal INTEGER NOT NULL CHECK (ordinal >= 0),
		tag TEXT NOT NULL,
		normalized_tag TEXT NOT NULL,
		PRIMARY KEY (asset_guid, normalized_tag),
		UNIQUE (asset_guid, ordinal)
	) STRICT`,
	`CREATE TABLE object_refs (
		owner_kind TEXT NOT NULL,
		owner_id TEXT NOT NULL,
		role TEXT NOT NULL,
		digest TEXT NOT NULL,
		media_type TEXT NOT NULL,
		size INTEGER NOT NULL CHECK (size >= 0),
		PRIMARY KEY (owner_kind, owner_id, role),
		FOREIGN KEY (digest) REFERENCES gc_objects(digest) ON DELETE RESTRICT
	) STRICT`,
	`CREATE TABLE gc_objects (
		digest TEXT PRIMARY KEY NOT NULL,
		size INTEGER NOT NULL CHECK (size >= 0),
		physical_generation INTEGER NOT NULL CHECK (physical_generation > 0),
		state TEXT NOT NULL CHECK (state IN ('active', 'deleting')),
		observed_at TEXT NOT NULL,
		unreachable_since TEXT,
		last_error TEXT
	) STRICT`,
	`CREATE TABLE object_leases (
		token TEXT PRIMARY KEY NOT NULL,
		digest TEXT NOT NULL,
		owner_kind TEXT NOT NULL,
		owner_id TEXT NOT NULL,
		expires_at TEXT NOT NULL,
		FOREIGN KEY (digest) REFERENCES gc_objects(digest) ON DELETE RESTRICT
	) STRICT`,
	`CREATE INDEX idx_assets_kind_name ON assets(kind, name COLLATE NOCASE, guid)`,
	`CREATE INDEX idx_assets_kind_created ON assets(kind, created_at DESC, guid)`,
	`CREATE INDEX idx_asset_tags_normalized ON asset_tags(normalized_tag, asset_guid)`,
	`CREATE INDEX idx_object_refs_digest ON object_refs(digest)`,
	`CREATE INDEX idx_gc_objects_state_unreachable ON gc_objects(state, unreachable_since, digest)`,
	`CREATE INDEX idx_object_leases_expiry ON object_leases(expires_at, digest)`,
	`INSERT INTO meta(key, value) VALUES ('asset_revision', '0')`,
}

var workflowCatalogStatements = []string{
	`CREATE TABLE workflow_sources (
		workflow_id TEXT PRIMARY KEY NOT NULL,
		name TEXT NOT NULL,
		revision INTEGER NOT NULL CHECK (revision >= 0),
		source_hash TEXT NOT NULL CHECK (length(source_hash) = 71),
		format TEXT NOT NULL,
		version TEXT NOT NULL,
		artifact BLOB NOT NULL CHECK (length(artifact) > 0),
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	) STRICT`,
	`CREATE TABLE workflow_refs (
		workflow_id TEXT NOT NULL REFERENCES workflow_sources(workflow_id) ON DELETE CASCADE,
		role TEXT NOT NULL,
		digest TEXT NOT NULL,
		media_type TEXT NOT NULL,
		size INTEGER NOT NULL CHECK (size >= 0),
		PRIMARY KEY (workflow_id, role),
		FOREIGN KEY (digest) REFERENCES gc_objects(digest) ON DELETE RESTRICT
	) STRICT`,
	`CREATE TABLE workflow_quarantine (
		recovery_id TEXT PRIMARY KEY NOT NULL CHECK (length(recovery_id) = 71),
		original_name TEXT NOT NULL,
		reason TEXT NOT NULL,
		artifact BLOB NOT NULL CHECK (length(artifact) > 0),
		created_at TEXT NOT NULL
	) STRICT`,
	`CREATE INDEX idx_workflow_sources_name ON workflow_sources(name COLLATE NOCASE, workflow_id)`,
	`CREATE INDEX idx_workflow_refs_digest ON workflow_refs(digest, workflow_id)`,
	`CREATE INDEX idx_workflow_quarantine_created ON workflow_quarantine(created_at, recovery_id)`,
}

var workflowInstallationStatements = []string{
	`CREATE TABLE workflow_releases (
		release_id TEXT PRIMARY KEY NOT NULL CHECK (length(release_id) = 71),
		source_hash TEXT NOT NULL CHECK (length(source_hash) = 71),
		workflow_id TEXT NOT NULL,
		workflow_name TEXT NOT NULL,
		publisher_namespace TEXT NOT NULL,
		release_version TEXT NOT NULL,
		attestation_digest TEXT NOT NULL CHECK (length(attestation_digest) = 71),
		source_artifact BLOB NOT NULL CHECK (length(source_artifact) > 0),
		verified_at TEXT NOT NULL
	) STRICT`,
	`CREATE TABLE workflow_installations (
		installation_id TEXT PRIMARY KEY NOT NULL,
		release_id TEXT NOT NULL REFERENCES workflow_releases(release_id) ON DELETE RESTRICT,
		name TEXT NOT NULL,
		lifecycle TEXT NOT NULL CHECK (lifecycle IN ('active', 'archived')),
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	) STRICT`,
	`CREATE INDEX idx_workflow_releases_source ON workflow_releases(source_hash, release_id)`,
	`CREATE INDEX idx_workflow_installations_release ON workflow_installations(release_id, installation_id)`,
	`CREATE INDEX idx_workflow_installations_lifecycle ON workflow_installations(lifecycle, updated_at DESC, installation_id)`,
}

var workflowInstallationConfigurationStatements = []string{
	`CREATE TABLE workflow_installation_configurations (
		installation_id TEXT PRIMARY KEY NOT NULL REFERENCES workflow_installations(installation_id) ON DELETE CASCADE,
		generation INTEGER NOT NULL CHECK (generation > 0),
		target_bindings BLOB NOT NULL CHECK (length(target_bindings) > 0),
		credential_bindings BLOB NOT NULL CHECK (length(credential_bindings) > 0),
		run_consent_release TEXT CHECK (run_consent_release IS NULL OR length(run_consent_release) = 71),
		schedule_consent_release TEXT CHECK (schedule_consent_release IS NULL OR length(schedule_consent_release) = 71),
		updated_at TEXT NOT NULL
	) STRICT`,
	`INSERT INTO workflow_installation_configurations(
		installation_id, generation, target_bindings, credential_bindings, updated_at
	)
	SELECT installation_id, 1, x'7b7d', x'7b7d', created_at
	FROM workflow_installations`,
}

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
