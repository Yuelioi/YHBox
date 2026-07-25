// Package catalog owns the SQLite lifecycle shared by the Content Catalog and
// Run Ledger. It deliberately does not expose *sql.DB: later domain
// repositories live behind this boundary instead of leaking SQL into services.
package catalog

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/yottaapp/yotta/internal/storage"
	_ "modernc.org/sqlite"
)

const (
	ContentFilename      = "content.db"
	RunFilename          = "runs.db"
	ContentSchemaVersion = 1
	RunSchemaVersion     = 1
	ContentApplicationID = 0x594F5443 // YOTC; provisional, not SQLite-registered.
	RunApplicationID     = 0x594F5452 // YOTR; provisional, not SQLite-registered.

	driverName = "sqlite"
)

var (
	ErrWrongDatabase     = errors.New("SQLite database belongs to another application")
	ErrUnclaimedDatabase = errors.New("SQLite database has content but no application identity")
	ErrFutureSchema      = errors.New("SQLite database schema is newer than this application")
	ErrSchemaDrift       = errors.New("SQLite database schema or migration history does not match this application")
)

type databaseKind string

const (
	contentKind databaseKind = "content-catalog"
	runKind     databaseKind = "run-ledger"
)

type databaseSpec struct {
	kind            databaseKind
	path            string
	applicationID   int64
	currentVersion  int
	migrations      []migration
	requiredObjects []schemaObject
}

type database struct {
	spec databaseSpec
	db   *sql.DB
}

// Foundation owns both SQLite consistency domains for one already-open
// storage profile.
type Foundation struct {
	content *database
	runs    *database
	once    sync.Once
}

type faultHooks struct {
	beforeMigrationCommit func(databaseKind, int) error
	beforeBackupManifest  func() error
}

type openOptions struct {
	faults faultHooks
}

// Open creates or validates both databases. The caller must already own the
// storage.Profile writer lease and must Close the returned Foundation before
// releasing that profile.
func Open(ctx context.Context, roots storage.Roots) (*Foundation, error) {
	return open(ctx, roots, openOptions{})
}

func open(ctx context.Context, roots storage.Roots, options openOptions) (*Foundation, error) {
	if ctx == nil {
		return nil, errors.New("open catalog foundation requires a context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(roots.Catalog) == "" || strings.TrimSpace(roots.State) == "" {
		return nil, errors.New("open catalog foundation requires projected storage roots")
	}
	if err := os.MkdirAll(roots.Catalog, 0o700); err != nil {
		return nil, fmt.Errorf("create Content Catalog directory: %w", err)
	}
	if err := os.MkdirAll(roots.State, 0o700); err != nil {
		return nil, fmt.Errorf("create Run Ledger directory: %w", err)
	}

	contentSpec := newDatabaseSpec(contentKind, filepath.Join(roots.Catalog, ContentFilename))
	runSpec := newDatabaseSpec(runKind, filepath.Join(roots.State, RunFilename))
	content, err := openDatabase(ctx, contentSpec, options.faults)
	if err != nil {
		return nil, err
	}
	runs, err := openDatabase(ctx, runSpec, options.faults)
	if err != nil {
		return nil, errors.Join(err, content.close())
	}
	return &Foundation{content: content, runs: runs}, nil
}

func newDatabaseSpec(kind databaseKind, path string) databaseSpec {
	switch kind {
	case contentKind:
		return databaseSpec{
			kind: contentKind, path: path, applicationID: ContentApplicationID,
			currentVersion: ContentSchemaVersion, migrations: contentMigrations,
			requiredObjects: foundationSchemaObjects,
		}
	case runKind:
		return databaseSpec{
			kind: runKind, path: path, applicationID: RunApplicationID,
			currentVersion: RunSchemaVersion, migrations: runMigrations,
			requiredObjects: foundationSchemaObjects,
		}
	default:
		panic("unknown catalog database kind")
	}
}

func openDatabase(ctx context.Context, spec databaseSpec, faults faultHooks) (*database, error) {
	dsn, err := sqliteURI(spec.path, false)
	if err != nil {
		return nil, fmt.Errorf("project %s SQLite URI: %w", spec.kind, err)
	}
	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", spec.kind, err)
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	result := &database{spec: spec, db: db}
	if err := db.PingContext(ctx); err != nil {
		return nil, errors.Join(fmt.Errorf("connect %s: %w", spec.kind, err), result.close())
	}
	if err := result.prepare(ctx, faults); err != nil {
		return nil, errors.Join(err, result.close())
	}
	return result, nil
}

func sqliteURI(path string, readOnly bool) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	slash := filepath.ToSlash(absolute)
	if volume := filepath.VolumeName(absolute); volume != "" && !strings.HasPrefix(slash, "/") {
		slash = "/" + slash
	}
	uri := url.URL{Scheme: "file", Path: slash}
	query := uri.Query()
	if readOnly {
		query.Set("mode", "ro")
	} else {
		query.Add("_pragma", "journal_mode(WAL)")
		query.Add("_txlock", "immediate")
	}
	query.Add("_pragma", "foreign_keys(1)")
	query.Add("_pragma", "busy_timeout(5000)")
	query.Add("_pragma", "trusted_schema(0)")
	query.Add("_pragma", "synchronous(FULL)")
	uri.RawQuery = query.Encode()
	return uri.String(), nil
}

func (f *Foundation) Close() error {
	if f == nil {
		return nil
	}
	var result error
	f.once.Do(func() {
		result = errors.Join(f.runs.close(), f.content.close())
	})
	return result
}

func (d *database) close() error {
	if d == nil || d.db == nil {
		return nil
	}
	db := d.db
	d.db = nil
	return db.Close()
}
