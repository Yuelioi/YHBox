package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestBackupCreatesVerifiedManifestLastSnapshots(t *testing.T) {
	roots := testRoots(t)
	ctx := context.Background()
	foundation, err := Open(ctx, roots)
	if err != nil {
		t.Fatal(err)
	}
	defer foundation.Close()

	if _, err := foundation.content.db.ExecContext(ctx,
		"INSERT INTO meta(key, value) VALUES ('test_marker', 'before-backup')"); err != nil {
		t.Fatal(err)
	}
	walInfo, err := os.Stat(filepath.Join(roots.Catalog, ContentFilename) + "-wal")
	if err != nil || walInfo.Size() == 0 {
		t.Fatalf("active WAL missing before backup: info=%v err=%v", walInfo, err)
	}
	destination := filepath.Join(t.TempDir(), "backup-set")
	set, err := foundation.Backup(ctx, destination)
	if err != nil {
		t.Fatalf("Backup() error = %v", err)
	}
	if set.Format != BackupFormat || set.Version != BackupVersion || len(set.Databases) != 2 {
		t.Fatalf("Backup() = %#v", set)
	}
	raw, err := os.ReadFile(filepath.Join(destination, manifestName))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var persisted BackupSet
	if err := json.Unmarshal(raw, &persisted); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	if len(persisted.Databases) != 2 {
		t.Fatalf("persisted databases = %d, want 2", len(persisted.Databases))
	}
	for _, item := range persisted.Databases {
		if item.Bytes <= 0 || len(item.SHA256) != 64 {
			t.Fatalf("backup entry = %#v", item)
		}
	}

	if _, err := foundation.content.db.ExecContext(ctx,
		"UPDATE meta SET value = 'after-backup' WHERE key = 'test_marker'"); err != nil {
		t.Fatal(err)
	}
	backupDB := openRaw(t, filepath.Join(destination, ContentFilename), true)
	defer backupDB.Close()
	var marker string
	if err := backupDB.QueryRow("SELECT value FROM meta WHERE key = 'test_marker'").Scan(&marker); err != nil {
		t.Fatal(err)
	}
	if marker != "before-backup" {
		t.Fatalf("backup marker = %q, want consistent pre-mutation value", marker)
	}
}

func TestBackupFailureDoesNotPublishPartialSet(t *testing.T) {
	roots := testRoots(t)
	foundation, err := Open(context.Background(), roots)
	if err != nil {
		t.Fatal(err)
	}
	defer foundation.Close()

	injected := errors.New("injected before manifest")
	destination := filepath.Join(t.TempDir(), "failed-backup")
	_, err = foundation.backup(context.Background(), destination, faultHooks{
		beforeBackupManifest: func() error { return injected },
	})
	if !errors.Is(err, injected) {
		t.Fatalf("backup() error = %v, want injected", err)
	}
	if _, err := os.Lstat(destination); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("failed backup destination remains: %v", err)
	}
}

func TestValidateSnapshotRejectsWrongIdentity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wrong.db")
	db := openRaw(t, path, false)
	if _, err := db.Exec("PRAGMA application_id = 7"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("PRAGMA user_version = 1"); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()

	err := validateSnapshot(context.Background(), newDatabaseSpec(contentKind, path), path)
	if !errors.Is(err, ErrSchemaDrift) {
		t.Fatalf("validateSnapshot() error = %v, want ErrSchemaDrift", err)
	}
}
