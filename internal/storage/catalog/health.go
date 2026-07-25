package catalog

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yottaapp/yotta/internal/storage"
)

type DatabaseHealth struct {
	Kind            string   `json:"kind"`
	Filename        string   `json:"filename"`
	Present         bool     `json:"present"`
	ApplicationID   string   `json:"applicationId"`
	SchemaVersion   int      `json:"schemaVersion"`
	SQLiteVersion   string   `json:"sqliteVersion"`
	JournalMode     string   `json:"journalMode"`
	Synchronous     int      `json:"synchronous"`
	PageCount       int64    `json:"pageCount"`
	FreelistCount   int64    `json:"freelistCount"`
	QuickCheck      []string `json:"quickCheck"`
	ForeignKeyCheck []string `json:"foreignKeyCheck"`
	Healthy         bool     `json:"healthy"`
}

type HealthReport struct {
	Content DatabaseHealth `json:"content"`
	Runs    DatabaseHealth `json:"runs"`
	Healthy bool           `json:"healthy"`
}

// Check performs bounded SQLite checks on both already-open databases.
// quick_check is intentional; full integrity_check is reserved for explicit
// diagnostics and recovery.
func (f *Foundation) Check(ctx context.Context) (HealthReport, error) {
	if f == nil || f.content == nil || f.runs == nil {
		return HealthReport{}, errors.New("catalog foundation is not open")
	}
	content, contentErr := f.content.check(ctx)
	runs, runsErr := f.runs.check(ctx)
	report := HealthReport{
		Content: content,
		Runs:    runs,
		Healthy: content.Healthy && runs.Healthy && contentErr == nil && runsErr == nil,
	}
	return report, errors.Join(contentErr, runsErr)
}

// Inspect checks existing database files without creating them or acquiring a
// writer lease. Missing foundation databases are reported as not present.
func Inspect(ctx context.Context, roots storage.Roots) (HealthReport, error) {
	if ctx == nil {
		return HealthReport{}, errors.New("inspect catalog foundation requires a context")
	}
	content, contentErr := inspectDatabase(ctx, newDatabaseSpec(
		contentKind, filepath.Join(roots.Catalog, ContentFilename),
	))
	runs, runsErr := inspectDatabase(ctx, newDatabaseSpec(
		runKind, filepath.Join(roots.State, RunFilename),
	))
	report := HealthReport{
		Content: content,
		Runs:    runs,
		Healthy: content.Healthy && runs.Healthy && contentErr == nil && runsErr == nil,
	}
	return report, errors.Join(contentErr, runsErr)
}

func inspectDatabase(ctx context.Context, spec databaseSpec) (DatabaseHealth, error) {
	report := DatabaseHealth{
		Kind: string(spec.kind), Filename: filenameForKind(spec.kind),
		ApplicationID: fmt.Sprintf("0x%08X", spec.applicationID),
	}
	info, err := os.Lstat(spec.path)
	if errors.Is(err, os.ErrNotExist) {
		return report, nil
	}
	if err != nil {
		return report, fmt.Errorf("inspect %s database file: %w", spec.kind, err)
	}
	if !info.Mode().IsRegular() {
		return report, fmt.Errorf("%s database path is not a regular file", spec.kind)
	}
	dsn, err := sqliteURI(spec.path, true)
	if err != nil {
		return report, err
	}
	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return report, err
	}
	defer db.Close()
	database := &database{spec: spec, db: db}
	if err := database.validate(ctx); err != nil {
		return report, err
	}
	return database.check(ctx)
}

func (d *database) check(ctx context.Context) (DatabaseHealth, error) {
	if d == nil || d.db == nil {
		return DatabaseHealth{}, errors.New("catalog database is closed")
	}
	report := DatabaseHealth{
		Kind:          string(d.spec.kind),
		Filename:      filenameForKind(d.spec.kind),
		Present:       true,
		ApplicationID: fmt.Sprintf("0x%08X", d.spec.applicationID),
	}
	var identity databaseIdentity
	var err error
	if identity, err = readIdentity(ctx, d.db); err != nil {
		return report, fmt.Errorf("read %s health identity: %w", d.spec.kind, err)
	}
	report.SchemaVersion = identity.userVersion
	if identity.applicationID != d.spec.applicationID {
		return report, fmt.Errorf("%w: %s application ID is 0x%08X",
			ErrSchemaDrift, d.spec.kind, identity.applicationID)
	}
	if err := scanSingle(ctx, d.db, "SELECT sqlite_version()", &report.SQLiteVersion); err != nil {
		return report, fmt.Errorf("read %s SQLite version: %w", d.spec.kind, err)
	}
	if err := scanSingle(ctx, d.db, "PRAGMA journal_mode", &report.JournalMode); err != nil {
		return report, fmt.Errorf("read %s journal mode: %w", d.spec.kind, err)
	}
	if err := scanSingle(ctx, d.db, "PRAGMA synchronous", &report.Synchronous); err != nil {
		return report, fmt.Errorf("read %s synchronous mode: %w", d.spec.kind, err)
	}
	if err := scanSingle(ctx, d.db, "PRAGMA page_count", &report.PageCount); err != nil {
		return report, fmt.Errorf("read %s page count: %w", d.spec.kind, err)
	}
	if err := scanSingle(ctx, d.db, "PRAGMA freelist_count", &report.FreelistCount); err != nil {
		return report, fmt.Errorf("read %s freelist count: %w", d.spec.kind, err)
	}
	report.QuickCheck, err = queryStrings(ctx, d.db, "PRAGMA quick_check")
	if err != nil {
		return report, fmt.Errorf("quick-check %s: %w", d.spec.kind, err)
	}
	report.ForeignKeyCheck, err = foreignKeyViolations(ctx, d.db)
	if err != nil {
		return report, fmt.Errorf("foreign-key check %s: %w", d.spec.kind, err)
	}
	report.JournalMode = strings.ToLower(report.JournalMode)
	report.Healthy = len(report.QuickCheck) == 1 && report.QuickCheck[0] == "ok" &&
		len(report.ForeignKeyCheck) == 0 && report.SchemaVersion == d.spec.currentVersion &&
		report.JournalMode == "wal"
	if !report.Healthy {
		return report, fmt.Errorf("%w: %s health checks failed", ErrSchemaDrift, d.spec.kind)
	}
	return report, nil
}

func filenameForKind(kind databaseKind) string {
	if kind == contentKind {
		return ContentFilename
	}
	return RunFilename
}

func scanSingle(ctx context.Context, db *sql.DB, query string, destination any) error {
	return db.QueryRowContext(ctx, query).Scan(destination)
}

func queryStrings(ctx context.Context, db *sql.DB, query string) ([]string, error) {
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var values []string
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func foreignKeyViolations(ctx context.Context, db *sql.DB) ([]string, error) {
	rows, err := db.QueryContext(ctx, "PRAGMA foreign_key_check")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []string
	for rows.Next() {
		var table, parent string
		var rowID sql.NullInt64
		var constraint int
		if err := rows.Scan(&table, &rowID, &parent, &constraint); err != nil {
			return nil, err
		}
		result = append(result, fmt.Sprintf("%s:%d:%s:%d", table, rowID.Int64, parent, constraint))
	}
	return result, rows.Err()
}
