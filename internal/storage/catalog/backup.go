package catalog

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/yottaapp/yotta/internal/durablefs"
	sqlitedriver "modernc.org/sqlite"
)

const (
	BackupFormat  = "yotta.sqlite-backup-set"
	BackupVersion = 1
	manifestName  = "manifest.json"
)

type BackupDatabase struct {
	Kind          string `json:"kind"`
	Filename      string `json:"filename"`
	ApplicationID string `json:"applicationId"`
	SchemaVersion int    `json:"schemaVersion"`
	Bytes         int64  `json:"bytes"`
	SHA256        string `json:"sha256"`
}

type BackupSet struct {
	Format    string           `json:"format"`
	Version   int              `json:"version"`
	CreatedAt string           `json:"createdAt"`
	Databases []BackupDatabase `json:"databases"`
}

type onlineBackuper interface {
	NewBackup(string) (*sqlitedriver.Backup, error)
}

// Backup creates two independent consistent SQLite snapshots. destination must
// not exist; manifest.json is published only after both snapshots validate.
func (f *Foundation) Backup(ctx context.Context, destination string) (BackupSet, error) {
	return f.backup(ctx, destination, faultHooks{})
}

func (f *Foundation) backup(ctx context.Context, destination string, faults faultHooks) (BackupSet, error) {
	if f == nil || f.content == nil || f.runs == nil {
		return BackupSet{}, errors.New("catalog foundation is not open")
	}
	if ctx == nil {
		return BackupSet{}, errors.New("catalog backup requires a context")
	}
	absolute, err := filepath.Abs(destination)
	if err != nil {
		return BackupSet{}, err
	}
	if _, err := os.Lstat(absolute); !errors.Is(err, os.ErrNotExist) {
		if err == nil {
			return BackupSet{}, errors.New("catalog backup destination already exists")
		}
		return BackupSet{}, fmt.Errorf("inspect catalog backup destination: %w", err)
	}
	if err := os.MkdirAll(absolute, 0o700); err != nil {
		return BackupSet{}, fmt.Errorf("create catalog backup destination: %w", err)
	}
	complete := false
	defer func() {
		if !complete {
			_ = os.RemoveAll(absolute)
		}
	}()

	set := BackupSet{
		Format: BackupFormat, Version: BackupVersion,
		CreatedAt: time.Now().UTC().Format(time.RFC3339Nano),
		Databases: make([]BackupDatabase, 0, 2),
	}
	for _, source := range []*database{f.content, f.runs} {
		entry, err := backupDatabase(ctx, source, filepath.Join(absolute, filenameForKind(source.spec.kind)))
		if err != nil {
			return BackupSet{}, err
		}
		set.Databases = append(set.Databases, entry)
	}
	if faults.beforeBackupManifest != nil {
		if err := faults.beforeBackupManifest(); err != nil {
			return BackupSet{}, err
		}
	}
	raw, err := json.MarshalIndent(set, "", "  ")
	if err != nil {
		return BackupSet{}, err
	}
	raw = append(raw, '\n')
	if err := durablefs.WriteFile(filepath.Join(absolute, manifestName), raw, 0o600); err != nil {
		return BackupSet{}, fmt.Errorf("publish catalog backup manifest: %w", err)
	}
	complete = true
	return set, nil
}

func backupDatabase(ctx context.Context, source *database, destination string) (BackupDatabase, error) {
	connection, err := source.db.Conn(ctx)
	if err != nil {
		return BackupDatabase{}, fmt.Errorf("reserve %s backup connection: %w", source.spec.kind, err)
	}
	defer connection.Close()
	if err := connection.Raw(func(driverConnection any) error {
		backuper, ok := driverConnection.(onlineBackuper)
		if !ok {
			return errors.New("configured SQLite driver does not expose Online Backup API")
		}
		backup, err := backuper.NewBackup(destination)
		if err != nil {
			return err
		}
		finished := false
		defer func() {
			if !finished {
				_ = backup.Finish()
			}
		}()
		for {
			if err := ctx.Err(); err != nil {
				return err
			}
			more, err := backup.Step(256)
			if err != nil {
				return err
			}
			if !more {
				break
			}
		}
		if err := backup.Finish(); err != nil {
			return err
		}
		finished = true
		return nil
	}); err != nil {
		return BackupDatabase{}, fmt.Errorf("online backup %s: %w", source.spec.kind, err)
	}
	if err := validateSnapshot(ctx, source.spec, destination); err != nil {
		return BackupDatabase{}, err
	}
	info, err := os.Stat(destination)
	if err != nil {
		return BackupDatabase{}, err
	}
	file, err := os.Open(destination)
	if err != nil {
		return BackupDatabase{}, err
	}
	defer file.Close()
	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		return BackupDatabase{}, err
	}
	return BackupDatabase{
		Kind: string(source.spec.kind), Filename: filepath.Base(destination),
		ApplicationID: fmt.Sprintf("0x%08X", source.spec.applicationID),
		SchemaVersion: source.spec.currentVersion, Bytes: info.Size(),
		SHA256: hex.EncodeToString(digest.Sum(nil)),
	}, nil
}

func validateSnapshot(ctx context.Context, spec databaseSpec, path string) error {
	dsn, err := sqliteURI(path, true)
	if err != nil {
		return err
	}
	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	identity, err := readIdentity(ctx, db)
	if err != nil {
		return fmt.Errorf("read %s backup identity: %w", spec.kind, err)
	}
	if identity.applicationID != spec.applicationID || identity.userVersion != spec.currentVersion {
		return fmt.Errorf("%w: %s backup identity does not match source", ErrSchemaDrift, spec.kind)
	}
	results, err := queryStrings(ctx, db, "PRAGMA quick_check")
	if err != nil {
		return fmt.Errorf("quick-check %s backup: %w", spec.kind, err)
	}
	if len(results) != 1 || results[0] != "ok" {
		return fmt.Errorf("%w: %s backup quick-check failed", ErrSchemaDrift, spec.kind)
	}
	return nil
}
