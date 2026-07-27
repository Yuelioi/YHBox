package migrate

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/storage"
	"github.com/yottaapp/yotta/internal/storage/catalog"
)

func TestInspectIsReadOnlyAndApplyPublishesVerifiedLayout(t *testing.T) {
	root := layoutOneFixture(t)
	ctx := context.Background()
	plan, err := Inspect(ctx, Options{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if plan.From != "1" || plan.To != "2" ||
		plan.MigrationID != layoutOneToTwoID || len(plan.Blocked) != 0 ||
		plan.RequiredFreeBytes <= plan.EstimatedBackupBytes {
		t.Fatalf("Inspect() = %#v", plan)
	}
	roots, _ := storage.Resolve(root)
	if _, err := os.Stat(filepath.Join(roots.Migrations, layoutOneToTwoID)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("dry-run created migration state: %v", err)
	}
	result, err := Apply(ctx, Options{Root: root, MaxRuns: 8})
	if err != nil {
		t.Fatal(err)
	}
	if result.Journal.State != StateCommitted {
		t.Fatalf("Apply() = %#v", result)
	}
	health, err := storage.Inspect(ctx, storage.InspectOptions{Root: root})
	if err != nil || !health.Supported || health.LayoutVersion != storage.LayoutVersion {
		t.Fatalf("storage health = %#v, %v", health, err)
	}
	databases, err := catalog.Inspect(ctx, roots)
	if err != nil || !databases.Healthy ||
		databases.Content.SchemaVersion != catalog.ContentSchemaVersion ||
		databases.Runs.SchemaVersion != catalog.RunSchemaVersion {
		t.Fatalf("catalog health = %#v, %v", databases, err)
	}
	if _, err := os.Stat(filepath.Join(
		roots.Migrations, layoutOneToTwoID, "snapshot", snapshotManifestFilename,
	)); err != nil {
		t.Fatalf("snapshot manifest: %v", err)
	}
	status, err := InspectRecovery(ctx, Options{Root: root})
	if err != nil || status.Journal == nil || status.Journal.State != StateCommitted ||
		status.Plan.From != storage.LayoutVersion || len(status.Quarantine) != 0 {
		t.Fatalf("InspectRecovery() = %#v, %v", status, err)
	}
}

func TestWriteJournalUpgradesLegacyDocumentVersion(t *testing.T) {
	migrations, err := registry()
	if err != nil {
		t.Fatal(err)
	}
	steps, err := migrations.Plan("1")
	if err != nil || len(steps) == 0 {
		t.Fatalf("migration plan = %#v, %v", steps, err)
	}
	now := time.Date(2026, 7, 27, 1, 2, 3, 0, time.UTC).Format(time.RFC3339Nano)
	journal := Journal{
		Format: JournalFormat, Version: legacyDocVersion,
		MigrationID: steps[0].ID, From: steps[0].From, To: steps[0].To,
		StepChecksum: steps[0].Checksum.String(), State: StateApplying,
		StartedAt: now, UpdatedAt: now, BackupManifest: "snapshot/" + snapshotManifestFilename,
	}
	dir := t.TempDir()
	if err := writeJournal(dir, journal); err != nil {
		t.Fatal(err)
	}
	loaded, found, err := readJournal(filepath.Join(dir, journalFilename))
	if err != nil || !found {
		t.Fatalf("read rewritten journal = %#v, %v, %v", loaded, found, err)
	}
	if loaded.Version != DocumentVersion {
		t.Fatalf("rewritten journal version = %d, want %d", loaded.Version, DocumentVersion)
	}
}

func TestMigrationResumesEveryDurabilityBoundary(t *testing.T) {
	for _, test := range []struct {
		name  string
		hooks faultHooks
	}{
		{name: "after prepared", hooks: faultHooks{afterPrepared: injectedMigrationFailure}},
		{name: "after Catalog", hooks: faultHooks{afterCatalog: injectedMigrationFailure}},
		{name: "before manifest commit", hooks: faultHooks{beforeCommit: injectedMigrationFailure}},
		{name: "after manifest commit", hooks: faultHooks{afterCommit: injectedMigrationFailure}},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := layoutOneFixture(t)
			_, err := apply(context.Background(), Options{Root: root}, applyOptions{
				now: func() time.Time {
					return time.Date(2026, 7, 25, 10, 0, 0, 0, time.UTC)
				},
				faults: test.hooks,
			})
			if !errors.Is(err, ErrRecoveryRequired) {
				t.Fatalf("injected Apply() = %v", err)
			}
			resumed, err := Resume(context.Background(), Options{Root: root})
			if err != nil || resumed.Journal.State != StateCommitted {
				t.Fatalf("Resume() = %#v, %v", resumed, err)
			}
			health, err := storage.Inspect(context.Background(), storage.InspectOptions{Root: root})
			if err != nil || !health.Supported {
				t.Fatalf("health after resume = %#v, %v", health, err)
			}
		})
	}
}

func TestEnsureMigratesLegacyBlobStoreBeforeDesktopOpen(t *testing.T) {
	root, ref := layoutTwoLegacyBlobFixture(t)

	result, err := Ensure(context.Background(), Options{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if result.Journal.MigrationID != layoutTwoToThreeID ||
		result.Journal.State != StateCommitted {
		t.Fatalf("Ensure() = %#v", result)
	}
	assertMigratedBlobStore(t, root, ref)
}

func TestBlobLayoutMigrationResumesEveryDurabilityBoundary(t *testing.T) {
	for _, test := range []struct {
		name  string
		hooks faultHooks
	}{
		{name: "after Blob migration", hooks: faultHooks{afterCatalog: injectedMigrationFailure}},
		{name: "before manifest commit", hooks: faultHooks{beforeCommit: injectedMigrationFailure}},
		{name: "after manifest commit", hooks: faultHooks{afterCommit: injectedMigrationFailure}},
	} {
		t.Run(test.name, func(t *testing.T) {
			root, ref := layoutTwoLegacyBlobFixture(t)
			_, err := apply(context.Background(), Options{Root: root}, applyOptions{
				now:    time.Now,
				faults: test.hooks,
			})
			if !errors.Is(err, ErrRecoveryRequired) {
				t.Fatalf("injected Apply() = %v", err)
			}
			result, err := Resume(context.Background(), Options{Root: root})
			if err != nil || result.Journal.State != StateCommitted {
				t.Fatalf("Resume() = %#v, %v", result, err)
			}
			assertMigratedBlobStore(t, root, ref)
		})
	}
}

func TestBlobLayoutMigrationRollbackRestoresFlatAuthority(t *testing.T) {
	root, ref := layoutTwoLegacyBlobFixture(t)
	_, err := apply(context.Background(), Options{Root: root}, applyOptions{
		now:    time.Now,
		faults: faultHooks{afterCatalog: injectedMigrationFailure},
	})
	if !errors.Is(err, ErrRecoveryRequired) {
		t.Fatalf("Apply() = %v", err)
	}

	result, err := Rollback(context.Background(), Options{Root: root})
	if err != nil || result.Journal.State != StateRolledBack {
		t.Fatalf("Rollback() = %#v, %v", result, err)
	}
	roots, _ := storage.Resolve(root)
	name := strings.TrimPrefix(ref.Digest.String(), "sha256:")
	if raw, err := os.ReadFile(filepath.Join(roots.Objects, name)); err != nil ||
		string(raw) != "legacy-blob" {
		t.Fatalf("rolled-back Blob = %q, %v", raw, err)
	}
	if marker, err := os.ReadFile(filepath.Join(roots.Objects, ".yotta-blob-store")); err != nil ||
		string(marker) != "yotta/blob-store/1\n" {
		t.Fatalf("rolled-back marker = %q, %v", marker, err)
	}
	health, err := storage.Inspect(context.Background(), storage.InspectOptions{Root: root})
	if err != nil || health.LayoutVersion != "2" || health.Supported {
		t.Fatalf("rolled-back health = %#v, %v", health, err)
	}

	if _, err := Resume(context.Background(), Options{Root: root}); err != nil {
		t.Fatal(err)
	}
	assertMigratedBlobStore(t, root, ref)
}

func TestEnsureReconcilesPublishedManifestWithInterruptedJournal(t *testing.T) {
	root := layoutOneFixture(t)
	_, err := apply(context.Background(), Options{Root: root}, applyOptions{
		now:    time.Now,
		faults: faultHooks{afterCommit: injectedMigrationFailure},
	})
	if !errors.Is(err, ErrRecoveryRequired) {
		t.Fatalf("Apply() = %v", err)
	}
	result, err := Ensure(context.Background(), Options{Root: root})
	if err != nil || result.Journal.State != StateCommitted {
		t.Fatalf("Ensure() = %#v, %v", result, err)
	}
}

func TestMigrationRejectsTamperedRecoveryAuthority(t *testing.T) {
	tests := []struct {
		name   string
		tamper func(*testing.T, string)
	}{
		{
			name: "journal checksum",
			tamper: func(t *testing.T, migrationDir string) {
				t.Helper()
				path := filepath.Join(migrationDir, journalFilename)
				raw, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				var journal map[string]any
				if err := json.Unmarshal(raw, &journal); err != nil {
					t.Fatal(err)
				}
				journal["stepChecksum"] = "sha256:" + strings.Repeat("0", 64)
				writeTestJSON(t, path, journal)
			},
		},
		{
			name: "snapshot authority set",
			tamper: func(t *testing.T, migrationDir string) {
				t.Helper()
				path := filepath.Join(migrationDir, "snapshot", snapshotManifestFilename)
				raw, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				var manifest SnapshotManifest
				if err := json.Unmarshal(raw, &manifest); err != nil {
					t.Fatal(err)
				}
				manifest.Files = manifest.Files[:len(manifest.Files)-1]
				writeTestJSON(t, path, manifest)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := layoutOneFixture(t)
			_, err := apply(context.Background(), Options{Root: root}, applyOptions{
				now:    time.Now,
				faults: faultHooks{afterPrepared: injectedMigrationFailure},
			})
			if !errors.Is(err, ErrRecoveryRequired) {
				t.Fatalf("Apply() = %v", err)
			}
			roots, err := storage.Resolve(root)
			if err != nil {
				t.Fatal(err)
			}
			test.tamper(t, filepath.Join(roots.Migrations, layoutOneToTwoID))
			if _, err := Resume(context.Background(), Options{Root: root}); err == nil {
				t.Fatal("Resume() accepted tampered recovery authority")
			}
		})
	}
}

func TestMigrationRollbackRestoresOldAuthorityAndCanReapply(t *testing.T) {
	root := layoutOneFixture(t)
	roots, _ := storage.Resolve(root)
	settings := filepath.Join(roots.Config, "settings.json")
	before, err := os.ReadFile(settings)
	if err != nil {
		t.Fatal(err)
	}
	_, err = apply(context.Background(), Options{Root: root}, applyOptions{
		now: func() time.Time {
			return time.Date(2026, 7, 25, 11, 0, 0, 0, time.UTC)
		},
		faults: faultHooks{afterCatalog: injectedMigrationFailure},
	})
	if !errors.Is(err, ErrRecoveryRequired) {
		t.Fatalf("Apply() = %v", err)
	}
	if err := os.WriteFile(settings, []byte("changed after snapshot"), 0o600); err != nil {
		t.Fatal(err)
	}
	rolledBack, err := Rollback(context.Background(), Options{Root: root})
	if err != nil || rolledBack.Journal.State != StateRolledBack {
		t.Fatalf("Rollback() = %#v, %v", rolledBack, err)
	}
	after, err := os.ReadFile(settings)
	if err != nil || string(after) != string(before) {
		t.Fatalf("restored settings = %q, %v", after, err)
	}
	health, err := storage.Inspect(context.Background(), storage.InspectOptions{Root: root})
	if err != nil || health.LayoutVersion != "1" || health.Supported {
		t.Fatalf("rolled-back health = %#v, %v", health, err)
	}
	reapplied, err := Resume(context.Background(), Options{Root: root})
	if err != nil || reapplied.Journal.State != StateCommitted {
		t.Fatalf("reapply = %#v, %v", reapplied, err)
	}
}

func TestMigrationPreflightBlocksUnknownRootEntries(t *testing.T) {
	root := layoutOneFixture(t)
	if err := os.WriteFile(filepath.Join(root, "foreign.txt"), []byte("unowned"), 0o600); err != nil {
		t.Fatal(err)
	}
	plan, err := Inspect(context.Background(), Options{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Blocked) != 1 || plan.UnknownRootEntries != 1 {
		t.Fatalf("blocked plan = %#v", plan)
	}
	if _, err := Apply(context.Background(), Options{Root: root}); !errors.Is(err, ErrPreflightBlocked) {
		t.Fatalf("Apply() = %v", err)
	}
}

func TestMigrationQuarantinesAndRestoresOneInvalidLegacyRun(t *testing.T) {
	root := layoutOneFixture(t)
	roots, _ := storage.Resolve(root)
	legacyRoot := filepath.Join(roots.Data, "workspace", "runs")
	name := "0190c7d4-1e40-7cc5-a783-57b16d5c8e3a.json"
	if err := os.CopyFS(legacyRoot, os.DirFS("testdata/invalid-legacy-run")); err != nil {
		t.Fatal(err)
	}
	result, err := Apply(context.Background(), Options{Root: root})
	if !errors.Is(err, ErrRecoveryRequired) || result.Journal.BlockedEntry != name {
		t.Fatalf("Apply() = %#v, %v", result, err)
	}
	diagnosticsPath, err := ExportDiagnosticsToProfile(context.Background(), Options{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	rawDiagnostics, err := os.ReadFile(diagnosticsPath)
	if err != nil {
		t.Fatal(err)
	}
	var diagnostics Diagnostics
	if err := json.Unmarshal(rawDiagnostics, &diagnostics); err != nil {
		t.Fatal(err)
	}
	if diagnostics.Journal == nil || diagnostics.Journal.BlockedEntry != name ||
		filepath.IsAbs(diagnostics.Journal.BackupManifest) ||
		string(rawDiagnostics) == "" || strings.Contains(string(rawDiagnostics), root) {
		t.Fatalf("diagnostics are incomplete or unredacted: %s", rawDiagnostics)
	}
	record, err := QuarantineLegacyRun(context.Background(), Options{Root: root}, name)
	if err != nil || record.Name != name || record.Bytes == 0 {
		t.Fatalf("QuarantineLegacyRun() = %#v, %v", record, err)
	}
	listed, err := ListQuarantine(Options{Root: root})
	if err != nil || len(listed) != 1 || listed[0] != record {
		t.Fatalf("ListQuarantine() = %#v, %v", listed, err)
	}
	if _, err := RestoreLegacyRun(context.Background(), Options{Root: root}, name); err != nil {
		t.Fatal(err)
	}
	if listed, err := ListQuarantine(Options{Root: root}); err != nil || len(listed) != 0 {
		t.Fatalf("quarantine after restore = %#v, %v", listed, err)
	}
	if _, err := Resume(context.Background(), Options{Root: root}); !errors.Is(err, ErrRecoveryRequired) {
		t.Fatalf("Resume(restored invalid) = %v", err)
	}
	if _, err := QuarantineLegacyRun(context.Background(), Options{Root: root}, name); err != nil {
		t.Fatal(err)
	}
	resumed, err := Resume(context.Background(), Options{Root: root})
	if err != nil || resumed.Journal.State != StateCommitted {
		t.Fatalf("Resume(quarantined) = %#v, %v", resumed, err)
	}
}

func layoutOneFixture(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "layout-1")
	if err := os.CopyFS(root, os.DirFS("testdata/layout-1")); err != nil {
		t.Fatal(err)
	}
	return root
}

func layoutTwoLegacyBlobFixture(t *testing.T) (string, blob.BlobRef) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "layout-2")
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	writeTestJSON(t, filepath.Join(root, "root.json"), storage.RootManifest{
		Format: storage.RootFormat, Version: "2",
	})
	profile, err := storage.OpenForMigration(
		context.Background(), storage.OpenOptions{Root: root}, "2",
	)
	if err != nil {
		t.Fatal(err)
	}
	roots := profile.Roots
	if err := profile.Close(); err != nil {
		t.Fatal(err)
	}
	foundation, err := catalog.Open(context.Background(), roots)
	if err != nil {
		t.Fatal(err)
	}
	if err := foundation.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(roots.Objects, ".yotta-blob-store"),
		[]byte("yotta/blob-store/1\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	const digest = "db0d377203aa1a21d9da03aa22b4e07abaeb07023c28080107bbe8ed6d03da2b"
	if err := os.WriteFile(
		filepath.Join(roots.Objects, digest), []byte("legacy-blob"), 0o600,
	); err != nil {
		t.Fatal(err)
	}
	return root, blob.BlobRef{
		MediaType: "application/octet-stream",
		Digest:    artifact.Digest("sha256:" + digest),
		Size:      int64(len("legacy-blob")),
	}
}

func assertMigratedBlobStore(t *testing.T, root string, ref blob.BlobRef) {
	t.Helper()
	health, err := storage.Inspect(context.Background(), storage.InspectOptions{Root: root})
	if err != nil || !health.Supported || health.LayoutVersion != storage.LayoutVersion {
		t.Fatalf("storage health = %#v, %v", health, err)
	}
	profile, err := storage.Open(context.Background(), storage.OpenOptions{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	defer profile.Close()
	foundation, err := catalog.Open(context.Background(), profile.Roots)
	if err != nil {
		t.Fatal(err)
	}
	defer foundation.Close()
	store, err := blob.Open(
		profile.Roots.Objects,
		blob.Limits{MaxBlobBytes: 1024, MaxTotalBytes: 4096},
		foundation.Objects(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Verify(context.Background(), ref); err != nil {
		t.Fatalf("migrated Blob = %v", err)
	}
	if _, found, err := foundation.Objects().Object(context.Background(), ref.Digest); err != nil || !found {
		t.Fatalf("migrated Blob inventory = %v, %v", found, err)
	}
}

func injectedMigrationFailure() error { return errors.New("injected migration kill point") }

func writeTestJSON(t *testing.T, path string, value any) {
	t.Helper()
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
}
