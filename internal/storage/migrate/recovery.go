package migrate

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yottaapp/yotta/internal/durablefs"
	"github.com/yottaapp/yotta/internal/storage"
	"github.com/yottaapp/yotta/internal/storage/catalog"
)

type RecoveryStatus struct {
	Plan       Plan               `json:"plan"`
	Journal    *Journal           `json:"journal,omitempty"`
	Quarantine []QuarantineRecord `json:"quarantine"`
}

// InspectRecovery returns the complete read-only state needed by CLI and GUI
// recovery adapters. It never creates migration directories or databases.
func InspectRecovery(ctx context.Context, options Options) (RecoveryStatus, error) {
	if ctx == nil {
		return RecoveryStatus{}, errors.New("inspect storage recovery requires a context")
	}
	plan, err := Inspect(ctx, options)
	if err != nil {
		return RecoveryStatus{}, err
	}
	roots, err := storage.Resolve(options.Root)
	if err != nil {
		return RecoveryStatus{}, err
	}
	journal, found, err := readJournal(
		filepath.Join(roots.Migrations, layoutOneToTwoID, journalFilename),
	)
	if err != nil {
		return RecoveryStatus{}, err
	}
	quarantine, err := ListQuarantine(options)
	if err != nil {
		return RecoveryStatus{}, err
	}
	result := RecoveryStatus{Plan: plan, Quarantine: quarantine}
	if found {
		result.Journal = &journal
	}
	return result, nil
}

func Rollback(ctx context.Context, options Options) (result Result, resultErr error) {
	if ctx == nil {
		return Result{}, errors.New("rollback storage migration requires a context")
	}
	roots, err := storage.Resolve(options.Root)
	if err != nil {
		return Result{}, err
	}
	migrationDir := filepath.Join(roots.Migrations, layoutOneToTwoID)
	journal, found, err := readJournal(filepath.Join(migrationDir, journalFilename))
	if err != nil {
		return Result{}, err
	}
	if !found {
		return Result{}, errors.New("storage migration journal does not exist")
	}
	if journal.State == StateCommitted {
		return Result{}, errors.New("committed storage migration requires an explicit future downgrade step")
	}
	manifest, err := readSnapshotManifest(filepath.Join(migrationDir, "snapshot", snapshotManifestFilename))
	if err != nil {
		return Result{}, err
	}
	if err := validateSnapshotFiles(roots, filepath.Join(migrationDir, "snapshot"), manifest); err != nil {
		return Result{}, err
	}
	profile, err := openRecoveryProfile(ctx, roots, journal.From)
	if err != nil {
		return Result{}, err
	}
	defer func() { resultErr = errors.Join(resultErr, profile.Close()) }()

	for _, database := range []string{
		filepath.Join(roots.Catalog, catalog.ContentFilename),
		filepath.Join(roots.State, catalog.RunFilename),
	} {
		for _, suffix := range []string{"-wal", "-shm"} {
			if err := removeIfPresent(database + suffix); err != nil {
				return Result{}, err
			}
		}
	}
	var rootEntry *SnapshotFile
	for index := range manifest.Files {
		entry := manifest.Files[index]
		target := filepath.Clean(filepath.Join(roots.Root, filepath.FromSlash(entry.Path)))
		if target == roots.ManifestFile() {
			copy := entry
			rootEntry = &copy
			continue
		}
		if err := restoreSnapshotEntry(roots, migrationDir, entry); err != nil {
			return Result{}, err
		}
	}
	if rootEntry == nil || !rootEntry.Present {
		return Result{}, errors.New("migration snapshot does not contain the old root manifest")
	}
	if err := restoreSnapshotEntry(roots, migrationDir, *rootEntry); err != nil {
		return Result{}, err
	}
	journal.State = StateRolledBack
	journal.LastError = ""
	journal.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if err := writeJournal(migrationDir, journal); err != nil {
		return Result{}, err
	}
	return Result{Journal: journal}, nil
}

func ExportDiagnostics(ctx context.Context, options Options, destination string) error {
	if ctx == nil {
		return errors.New("export migration diagnostics requires a context")
	}
	roots, err := storage.Resolve(options.Root)
	if err != nil {
		return err
	}
	plan, planErr := Inspect(ctx, options)
	reportErrors := make([]string, 0, 4)
	if planErr != nil {
		plan = Plan{
			Format: PlanFormat, Version: DocumentVersion,
			Blocked: []string{redactRoot(planErr.Error(), roots.Root)},
		}
		reportErrors = append(reportErrors, "plan: "+redactRoot(planErr.Error(), roots.Root))
	}
	journal, found, journalErr := readJournal(
		filepath.Join(roots.Migrations, layoutOneToTwoID, journalFilename),
	)
	if journalErr != nil {
		reportErrors = append(reportErrors, "journal: "+redactRoot(journalErr.Error(), roots.Root))
	}
	if found {
		journal.LastError = redactRoot(journal.LastError, roots.Root)
		journal.BackupManifest = filepath.Base(journal.BackupManifest)
	}
	storageHealth, storageErr := storage.Inspect(ctx, storage.InspectOptions{Root: roots.Root})
	if storageErr != nil {
		reportErrors = append(reportErrors, "storage: "+redactRoot(storageErr.Error(), roots.Root))
	}
	catalogHealth, catalogErr := catalog.Inspect(ctx, roots)
	if catalogErr != nil {
		reportErrors = append(reportErrors, "catalog: "+redactRoot(catalogErr.Error(), roots.Root))
	}
	report := Diagnostics{
		Format: DiagnosticsFormat, Version: DocumentVersion,
		CreatedAt: time.Now().UTC().Format(time.RFC3339Nano),
		Plan:      plan, Storage: storageHealth, Catalog: catalogHealth,
		Errors: reportErrors,
	}
	if found {
		report.Journal = &journal
	}
	return writeJSON(destination, report)
}

func ExportDiagnosticsToProfile(ctx context.Context, options Options) (string, error) {
	roots, err := storage.Resolve(options.Root)
	if err != nil {
		return "", err
	}
	destinationRoot := filepath.Join(roots.Migrations, layoutOneToTwoID, "diagnostics")
	if err := ensureTrustedSubdirectory(roots.Root, destinationRoot); err != nil {
		return "", err
	}
	destination := filepath.Join(
		destinationRoot,
		"storage-migration-"+time.Now().UTC().Format("20060102T150405.000000000Z")+".json",
	)
	if err := ExportDiagnostics(ctx, options, destination); err != nil {
		return "", err
	}
	return destination, nil
}

func openRecoveryProfile(ctx context.Context, roots storage.Roots, from string) (*storage.Profile, error) {
	health, err := storage.Inspect(ctx, storage.InspectOptions{Root: roots.Root})
	if err != nil {
		return nil, err
	}
	if health.LayoutVersion == storage.LayoutVersion {
		return storage.Open(ctx, storage.OpenOptions{Root: roots.Root})
	}
	return storage.OpenForMigration(ctx, storage.OpenOptions{Root: roots.Root}, from)
}

func restoreSnapshotEntry(roots storage.Roots, migrationDir string, entry SnapshotFile) error {
	relative := filepath.FromSlash(entry.Path)
	if filepath.IsAbs(relative) || strings.HasPrefix(relative, "..") {
		return errors.New("migration snapshot restore path is invalid")
	}
	target := filepath.Clean(filepath.Join(roots.Root, relative))
	rootPrefix := filepath.Clean(roots.Root) + string(filepath.Separator)
	if target != filepath.Clean(roots.Root) && !strings.HasPrefix(target, rootPrefix) {
		return errors.New("migration snapshot restore escaped storage root")
	}
	if !entry.Present {
		return removeIfPresent(target)
	}
	source := filepath.Join(migrationDir, "snapshot", "files", relative)
	_, _, err := copyFileDurable(source, target)
	return err
}

func removeIfPresent(path string) error {
	err := durablefs.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func redactRoot(value, root string) string {
	if value == "" {
		return ""
	}
	return strings.ReplaceAll(value, root, storage.RedactedRoot)
}
