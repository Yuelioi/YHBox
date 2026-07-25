package catalog

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/yottaapp/yotta/internal/storage"
)

func TestOpenCreatesAndValidatesDistinctDatabases(t *testing.T) {
	roots := testRoots(t)
	ctx := context.Background()

	foundation, err := Open(ctx, roots)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	report, err := foundation.Check(ctx)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if !report.Healthy || !report.Content.Healthy || !report.Runs.Healthy {
		t.Fatalf("Check() = %#v, want healthy", report)
	}
	if report.Content.ApplicationID == report.Runs.ApplicationID {
		t.Fatalf("application IDs are both %q, want distinct identities", report.Content.ApplicationID)
	}
	if report.Content.JournalMode != "wal" || report.Runs.JournalMode != "wal" {
		t.Fatalf("journal modes = %q/%q, want wal", report.Content.JournalMode, report.Runs.JournalMode)
	}
	if report.Content.Synchronous != 2 || report.Runs.Synchronous != 2 {
		t.Fatalf("synchronous = %d/%d, want FULL(2)", report.Content.Synchronous, report.Runs.Synchronous)
	}
	if err := foundation.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	for _, path := range []string{
		filepath.Join(roots.Catalog, ContentFilename),
		filepath.Join(roots.State, RunFilename),
	} {
		if info, err := os.Stat(path); err != nil || info.Size() == 0 {
			t.Fatalf("database %q was not durably created: info=%v err=%v", path, info, err)
		}
	}

	reopened, err := Open(ctx, roots)
	if err != nil {
		t.Fatalf("reopen error = %v", err)
	}
	defer reopened.Close()
	if _, err := reopened.Check(ctx); err != nil {
		t.Fatalf("reopened Check() error = %v", err)
	}
}

func TestInspectIsReadOnlyAndReportsMissingFoundation(t *testing.T) {
	roots := testRoots(t)
	report, err := Inspect(context.Background(), roots)
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	if report.Healthy || report.Content.Present || report.Runs.Present {
		t.Fatalf("missing Inspect() = %#v", report)
	}
	if _, err := os.Lstat(roots.Catalog); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Inspect() created catalog directory: %v", err)
	}

	foundation, err := Open(context.Background(), roots)
	if err != nil {
		t.Fatal(err)
	}
	if err := foundation.Close(); err != nil {
		t.Fatal(err)
	}
	report, err = Inspect(context.Background(), roots)
	if err != nil {
		t.Fatalf("Inspect(existing) error = %v", err)
	}
	if !report.Healthy || !report.Content.Present || !report.Runs.Present {
		t.Fatalf("existing Inspect() = %#v", report)
	}
}

func TestMigrationFaultRollsBackAndCanResume(t *testing.T) {
	roots := testRoots(t)
	injected := errors.New("injected before migration commit")
	_, err := open(context.Background(), roots, openOptions{faults: faultHooks{
		beforeMigrationCommit: func(kind databaseKind, version int) error {
			if kind == contentKind && version == ContentSchemaVersion {
				return injected
			}
			return nil
		},
	}})
	if !errors.Is(err, injected) {
		t.Fatalf("open() error = %v, want injected fault", err)
	}

	db := openRaw(t, filepath.Join(roots.Catalog, ContentFilename), false)
	identity, identityErr := readIdentity(context.Background(), db)
	if identityErr != nil {
		t.Fatalf("readIdentity() error = %v", identityErr)
	}
	var objects int
	if err := db.QueryRow(`
		SELECT count(*) FROM sqlite_schema WHERE name NOT LIKE 'sqlite_%'
	`).Scan(&objects); err != nil {
		t.Fatalf("count schema objects: %v", err)
	}
	_ = db.Close()
	if identity.applicationID != 0 || identity.userVersion != 0 || objects != 0 {
		t.Fatalf("rolled-back identity = %#v, objects=%d; want empty database", identity, objects)
	}

	foundation, err := Open(context.Background(), roots)
	if err != nil {
		t.Fatalf("Open() after rollback error = %v", err)
	}
	defer foundation.Close()
}

func TestOpenRejectsWrongUnclaimedAndFutureDatabases(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(*testing.T, storage.Roots)
		want    error
	}{
		{
			name: "wrong application",
			prepare: func(t *testing.T, roots storage.Roots) {
				db := openRaw(t, filepath.Join(roots.Catalog, ContentFilename), false)
				if _, err := db.Exec("PRAGMA application_id = 123456"); err != nil {
					t.Fatal(err)
				}
				_ = db.Close()
			},
			want: ErrWrongDatabase,
		},
		{
			name: "unclaimed content",
			prepare: func(t *testing.T, roots storage.Roots) {
				db := openRaw(t, filepath.Join(roots.Catalog, ContentFilename), false)
				if _, err := db.Exec("CREATE TABLE foreign_data(value TEXT)"); err != nil {
					t.Fatal(err)
				}
				_ = db.Close()
			},
			want: ErrUnclaimedDatabase,
		},
		{
			name: "future schema",
			prepare: func(t *testing.T, roots storage.Roots) {
				db := openRaw(t, filepath.Join(roots.Catalog, ContentFilename), false)
				if _, err := db.Exec("PRAGMA application_id = " + intString(ContentApplicationID)); err != nil {
					t.Fatal(err)
				}
				if _, err := db.Exec("PRAGMA user_version = 99"); err != nil {
					t.Fatal(err)
				}
				_ = db.Close()
			},
			want: ErrFutureSchema,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			roots := testRoots(t)
			if err := os.MkdirAll(roots.Catalog, 0o700); err != nil {
				t.Fatal(err)
			}
			test.prepare(t, roots)
			_, err := Open(context.Background(), roots)
			if !errors.Is(err, test.want) {
				t.Fatalf("Open() error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestOpenRejectsMigrationAndSchemaDrift(t *testing.T) {
	tests := []struct {
		name   string
		mutate string
	}{
		{name: "migration checksum", mutate: `
			UPDATE schema_migrations SET checksum = printf('%064d', 0)
		`},
		{name: "required index", mutate: `DROP INDEX idx_schema_migrations_to_version`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			roots := testRoots(t)
			foundation, err := Open(context.Background(), roots)
			if err != nil {
				t.Fatal(err)
			}
			if err := foundation.Close(); err != nil {
				t.Fatal(err)
			}
			db := openRaw(t, filepath.Join(roots.Catalog, ContentFilename), false)
			if _, err := db.Exec(test.mutate); err != nil {
				t.Fatal(err)
			}
			_ = db.Close()

			_, err = Open(context.Background(), roots)
			if !errors.Is(err, ErrSchemaDrift) {
				t.Fatalf("Open() error = %v, want ErrSchemaDrift", err)
			}
		})
	}
}

func TestOpenRejectsCorruptDatabase(t *testing.T) {
	roots := testRoots(t)
	if err := os.MkdirAll(roots.Catalog, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(roots.Catalog, ContentFilename), []byte("not a sqlite database"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(context.Background(), roots); err == nil {
		t.Fatal("Open() error = nil, want corrupt database failure")
	}
}

func testRoots(t *testing.T) storage.Roots {
	t.Helper()
	roots, err := storage.Resolve(filepath.Join(t.TempDir(), "profile"))
	if err != nil {
		t.Fatal(err)
	}
	return roots
}

func openRaw(t *testing.T, path string, readOnly bool) *sql.DB {
	t.Helper()
	dsn, err := sqliteURI(path, readOnly)
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open(driverName, dsn)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		t.Fatal(err)
	}
	return db
}

func intString(value int64) string {
	return fmt.Sprint(value)
}
