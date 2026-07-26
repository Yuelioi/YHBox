// Package migrate owns root-layout upgrades as an explicit lifecycle. It is
// the only Module allowed to coordinate storage.Profile, Catalog migration,
// legacy domain import, snapshot, journal, rollback, and layout publication.
package migrate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/durablefs"
	"github.com/yottaapp/yotta/internal/nodes"
	"github.com/yottaapp/yotta/internal/run"
	"github.com/yottaapp/yotta/internal/storage"
	"github.com/yottaapp/yotta/internal/storage/catalog"
)

const (
	PlanFormat         = "yotta.storage-migration-plan"
	JournalFormat      = "yotta.storage-migration-journal"
	DiagnosticsFormat  = "yotta.storage-migration-diagnostics"
	DocumentVersion    = 2
	legacyDocVersion   = 1
	layoutOneToTwoID   = "layout-1-to-2"
	layoutTwoToThreeID = "layout-2-to-3"
	journalFilename    = "journal.json"
	planFilename       = "plan.json"
)

var (
	ErrRecoveryRequired = errors.New("storage root migration recovery is required")
	ErrPreflightBlocked = errors.New("storage root migration preflight is blocked")
)

type Options struct {
	Root    string
	MaxRuns int
}

type Plan struct {
	Format               string   `json:"format"`
	Version              int      `json:"version"`
	MigrationID          string   `json:"migrationId,omitempty"`
	From                 string   `json:"from"`
	To                   string   `json:"to"`
	StepChecksum         string   `json:"stepChecksum,omitempty"`
	EstimatedBackupBytes uint64   `json:"estimatedBackupBytes"`
	RequiredFreeBytes    uint64   `json:"requiredFreeBytes"`
	AvailableBytes       uint64   `json:"availableBytes"`
	LegacyRunRecords     int      `json:"legacyRunRecords"`
	LegacyRunBytes       uint64   `json:"legacyRunBytes"`
	UnknownRootEntries   uint64   `json:"unknownRootEntries"`
	BlobLayoutFrom       string   `json:"blobLayoutFrom,omitempty"`
	Actions              []string `json:"actions"`
	Blocked              []string `json:"blocked"`
}

type JournalState string

const (
	StatePrepared         JournalState = "prepared"
	StateApplying         JournalState = "applying"
	StateVerifying        JournalState = "verifying"
	StateRecoveryRequired JournalState = "recovery-required"
	StateCommitted        JournalState = "committed"
	StateRolledBack       JournalState = "rolled-back"
)

type Journal struct {
	Format         string       `json:"format"`
	Version        int          `json:"version"`
	MigrationID    string       `json:"migrationId"`
	From           string       `json:"from"`
	To             string       `json:"to"`
	StepChecksum   string       `json:"stepChecksum"`
	State          JournalState `json:"state"`
	StartedAt      string       `json:"startedAt"`
	UpdatedAt      string       `json:"updatedAt"`
	BackupManifest string       `json:"backupManifest"`
	BlockedEntry   string       `json:"blockedEntry,omitempty"`
	BlobLayoutFrom string       `json:"blobLayoutFrom,omitempty"`
	LastError      string       `json:"lastError,omitempty"`
}

type Result struct {
	Plan    Plan    `json:"plan"`
	Journal Journal `json:"journal"`
}

type Diagnostics struct {
	Format    string               `json:"format"`
	Version   int                  `json:"version"`
	CreatedAt string               `json:"createdAt"`
	Plan      Plan                 `json:"plan"`
	Journal   *Journal             `json:"journal,omitempty"`
	Storage   storage.HealthReport `json:"storage"`
	Catalog   catalog.HealthReport `json:"catalog"`
	Errors    []string             `json:"errors"`
}

type faultHooks struct {
	afterPrepared func() error
	afterCatalog  func() error
	beforeCommit  func() error
	afterCommit   func() error
}

type applyOptions struct {
	now    func() time.Time
	faults faultHooks
}

func Inspect(ctx context.Context, options Options) (Plan, error) {
	if ctx == nil {
		return Plan{}, errors.New("inspect storage migration requires a context")
	}
	roots, err := storage.Resolve(options.Root)
	if err != nil {
		return Plan{}, err
	}
	info, err := os.Lstat(roots.Root)
	if errors.Is(err, os.ErrNotExist) {
		return currentPlan(), nil
	}
	if err != nil {
		return Plan{}, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return Plan{}, errors.New("storage migration root is not a trusted directory")
	}
	entries, err := os.ReadDir(roots.Root)
	if err != nil {
		return Plan{}, err
	}
	if len(entries) == 0 {
		return currentPlan(), nil
	}
	health, err := storage.Inspect(ctx, storage.InspectOptions{Root: roots.Root})
	if err != nil {
		return Plan{}, err
	}
	if health.LayoutVersion == storage.LayoutVersion {
		return Plan{
			Format: PlanFormat, Version: DocumentVersion,
			From: storage.LayoutVersion, To: storage.LayoutVersion,
			Actions: []string{},
		}, nil
	}
	registry, err := registry()
	if err != nil {
		return Plan{}, err
	}
	steps, err := registry.Plan(health.LayoutVersion)
	if err != nil {
		return Plan{}, err
	}
	if len(steps) == 0 {
		return Plan{}, errors.New("root migration has no reviewed step")
	}
	step := steps[0]
	backupBytes, err := estimateSnapshotBytes(roots)
	if err != nil {
		return Plan{}, err
	}
	var legacyCount int
	var legacyBytes uint64
	var blobLayoutFrom string
	var actions []string
	switch step.ID {
	case layoutOneToTwoID:
		legacyCount, legacyBytes, err = inspectLegacyRuns(roots)
		actions = []string{
			"checkpoint closed SQLite databases",
			"snapshot root manifest, settings, Content Catalog, and Run Ledger",
			"migrate Catalog schemas and import legacy Run records",
			"verify both databases and publish root layout 2",
		}
	case layoutTwoToThreeID:
		var inspection blob.LayoutInspection
		inspection, err = blob.InspectLayout(ctx, roots.Objects)
		if err == nil {
			blobLayoutFrom = inspection.Version
		}
		actions = []string{
			"checkpoint closed SQLite databases",
			"snapshot root manifest, settings, Content Catalog, and Run Ledger",
			"migrate legacy Blob Store objects to verified sharded storage",
			"reconcile the durable object inventory and publish root layout 3",
		}
	default:
		return Plan{}, fmt.Errorf("root migration %s has no implementation", step.ID)
	}
	if err != nil {
		return Plan{}, err
	}
	available, err := diskFreeBytes(roots.Root)
	if err != nil {
		return Plan{}, err
	}
	required := backupBytes + backupBytes/4 + 16<<20
	plan := Plan{
		Format: PlanFormat, Version: DocumentVersion,
		MigrationID: step.ID, From: step.From, To: step.To,
		StepChecksum:         step.Checksum.String(),
		EstimatedBackupBytes: backupBytes, RequiredFreeBytes: required,
		AvailableBytes: available, LegacyRunRecords: legacyCount,
		LegacyRunBytes: legacyBytes, UnknownRootEntries: health.UnknownEntries,
		BlobLayoutFrom: blobLayoutFrom, Actions: actions,
	}
	if health.UnknownEntries != 0 {
		plan.Blocked = append(plan.Blocked, "unknown root entries must be moved or reviewed")
	}
	if available < required {
		plan.Blocked = append(plan.Blocked, "insufficient free space for migration snapshot")
	}
	return plan, nil
}

// Ensure leaves new/current roots untouched and applies every registered
// released upgrade before callers open domain stores.
func Ensure(ctx context.Context, options Options) (Result, error) {
	roots, err := storage.Resolve(options.Root)
	if err != nil {
		return Result{}, err
	}
	if _, found, err := activeJournal(roots); err != nil {
		return Result{}, err
	} else if found {
		return Resume(ctx, options)
	}
	return Apply(ctx, options)
}

func Apply(ctx context.Context, options Options) (Result, error) {
	return apply(ctx, options, applyOptions{now: time.Now})
}

func apply(ctx context.Context, options Options, internal applyOptions) (result Result, resultErr error) {
	for {
		next, err := applyNext(ctx, options, internal)
		if err != nil {
			return next, err
		}
		result = next
		if next.Plan.From == next.Plan.To || next.Plan.To == storage.LayoutVersion {
			return result, nil
		}
	}
}

func applyNext(ctx context.Context, options Options, internal applyOptions) (result Result, resultErr error) {
	if internal.now == nil {
		internal.now = time.Now
	}
	plan, err := Inspect(ctx, options)
	if err != nil {
		return Result{}, err
	}
	if plan.From == plan.To {
		return Result{Plan: plan}, nil
	}
	if len(plan.Blocked) != 0 {
		return Result{Plan: plan}, fmt.Errorf("%w: %s", ErrPreflightBlocked, strings.Join(plan.Blocked, "; "))
	}
	roots, err := storage.Resolve(options.Root)
	if err != nil {
		return Result{}, err
	}
	profile, err := storage.OpenForMigration(ctx, storage.OpenOptions{Root: roots.Root}, plan.From)
	if err != nil {
		return Result{}, err
	}
	defer func() { resultErr = errors.Join(resultErr, profile.Close()) }()

	migrationDir := filepath.Join(roots.Migrations, plan.MigrationID)
	if err := ensureTrustedSubdirectory(roots.Root, migrationDir); err != nil {
		return Result{}, err
	}
	if err := writeJSON(filepath.Join(migrationDir, planFilename), plan); err != nil {
		return Result{}, err
	}
	now := internal.now().UTC()
	journal, found, err := readJournal(filepath.Join(migrationDir, journalFilename))
	if err != nil {
		return Result{}, err
	}
	if !found || journal.State == StateRolledBack {
		manifestPath, err := createSnapshot(ctx, roots, migrationDir, now)
		if err != nil {
			return Result{}, err
		}
		journal = Journal{
			Format: JournalFormat, Version: DocumentVersion,
			MigrationID: plan.MigrationID, From: plan.From, To: plan.To,
			StepChecksum: plan.StepChecksum, State: StatePrepared,
			StartedAt: now.Format(time.RFC3339Nano), UpdatedAt: now.Format(time.RFC3339Nano),
			BackupManifest: filepath.ToSlash(manifestPath),
			BlobLayoutFrom: plan.BlobLayoutFrom,
		}
		if err := writeJournal(migrationDir, journal); err != nil {
			return Result{}, err
		}
	} else {
		snapshotDir := filepath.Join(migrationDir, "snapshot")
		manifest, err := readSnapshotManifest(filepath.Join(snapshotDir, snapshotManifestFilename))
		if err != nil {
			return Result{}, err
		}
		if err := validateSnapshotFiles(roots, snapshotDir, manifest); err != nil {
			return Result{}, err
		}
	}
	result = Result{Plan: plan, Journal: journal}
	fail := func(cause error) (Result, error) {
		journal.State = StateRecoveryRequired
		journal.UpdatedAt = internal.now().UTC().Format(time.RFC3339Nano)
		journal.LastError = cause.Error()
		var legacyError *run.LegacyImportError
		if errors.As(cause, &legacyError) {
			journal.BlockedEntry = legacyError.Name
		}
		_ = writeJournal(migrationDir, journal)
		result.Journal = journal
		return result, errors.Join(ErrRecoveryRequired, cause)
	}
	if internal.faults.afterPrepared != nil {
		if err := internal.faults.afterPrepared(); err != nil {
			return fail(err)
		}
	}
	journal.State = StateApplying
	journal.LastError = ""
	journal.BlockedEntry = ""
	journal.UpdatedAt = internal.now().UTC().Format(time.RFC3339Nano)
	if err := writeJournal(migrationDir, journal); err != nil {
		return fail(err)
	}
	foundation, err := catalog.Open(ctx, roots)
	if err != nil {
		return fail(err)
	}
	closeFoundation := true
	defer func() {
		if closeFoundation {
			resultErr = errors.Join(resultErr, foundation.Close())
		}
	}()
	switch plan.MigrationID {
	case layoutOneToTwoID:
		builtins, err := nodes.Build()
		if err != nil {
			return fail(err)
		}
		maxRuns := options.MaxRuns
		if maxRuns <= 0 {
			maxRuns = 65536
		}
		if err := run.ImportLegacyStore(
			ctx, foundation.Runs(), builtins.Catalog,
			filepath.Join(roots.Data, "workspace", "runs"), maxRuns,
		); err != nil {
			return fail(err)
		}
	case layoutTwoToThreeID:
		if _, err := blob.MigrateLayoutOneToTwo(
			ctx, roots.Objects, foundation.Objects(),
		); err != nil {
			return fail(err)
		}
	default:
		return fail(fmt.Errorf("root migration %s has no implementation", plan.MigrationID))
	}
	if internal.faults.afterCatalog != nil {
		if err := internal.faults.afterCatalog(); err != nil {
			return fail(err)
		}
	}
	journal.State = StateVerifying
	journal.UpdatedAt = internal.now().UTC().Format(time.RFC3339Nano)
	if err := writeJournal(migrationDir, journal); err != nil {
		return fail(err)
	}
	report, err := foundation.Check(ctx)
	if err != nil || !report.Healthy {
		return fail(errors.Join(err, errors.New("migrated Catalog foundation is unhealthy")))
	}
	if err := foundation.Close(); err != nil {
		closeFoundation = false
		return fail(err)
	}
	closeFoundation = false
	if plan.MigrationID == layoutTwoToThreeID {
		verificationFoundation, err := catalog.Open(ctx, roots)
		if err != nil {
			return fail(err)
		}
		if _, err := blob.Open(
			roots.Objects,
			blob.Limits{MaxBlobBytes: 256 << 20, MaxTotalBytes: 4 << 30},
			verificationFoundation.Objects(),
		); err != nil {
			_ = verificationFoundation.Close()
			return fail(fmt.Errorf("verify migrated Blob Store: %w", err))
		}
		if err := verificationFoundation.Close(); err != nil {
			return fail(err)
		}
	}
	if internal.faults.beforeCommit != nil {
		if err := internal.faults.beforeCommit(); err != nil {
			return fail(err)
		}
	}
	if err := profile.PublishLayout(plan.From, plan.To); err != nil {
		return fail(err)
	}
	if internal.faults.afterCommit != nil {
		if err := internal.faults.afterCommit(); err != nil {
			return fail(err)
		}
	}
	journal.State = StateCommitted
	journal.UpdatedAt = internal.now().UTC().Format(time.RFC3339Nano)
	journal.LastError = ""
	journal.BlockedEntry = ""
	if err := writeJournal(migrationDir, journal); err != nil {
		return Result{Plan: plan, Journal: journal}, err
	}
	return Result{Plan: plan, Journal: journal}, nil
}

func Resume(ctx context.Context, options Options) (Result, error) {
	roots, err := storage.Resolve(options.Root)
	if err != nil {
		return Result{}, err
	}
	journal, found, err := activeJournal(roots)
	if err != nil {
		return Result{}, err
	}
	if !found {
		journal, found, err = latestJournal(roots)
		if err != nil {
			return Result{}, err
		}
		if !found {
			return Result{}, errors.New("storage migration journal does not exist")
		}
	}
	if journal.State == StateCommitted {
		return Result{Journal: journal}, nil
	}
	if journal.State == StateRolledBack {
		return Apply(ctx, options)
	}
	raw, readErr := os.ReadFile(roots.ManifestFile())
	if readErr == nil {
		var manifest storage.RootManifest
		if json.Unmarshal(raw, &manifest) == nil && manifest.Version == journal.To {
			var profile *storage.Profile
			if journal.To == storage.LayoutVersion {
				profile, err = storage.Open(ctx, storage.OpenOptions{Root: roots.Root})
			} else {
				profile, err = storage.OpenForMigration(
					ctx, storage.OpenOptions{Root: roots.Root}, journal.To,
				)
			}
			if err != nil {
				return Result{}, err
			}
			migrationDir := filepath.Join(roots.Migrations, journal.MigrationID)
			snapshotDir := filepath.Join(migrationDir, "snapshot")
			snapshot, err := readSnapshotManifest(
				filepath.Join(snapshotDir, snapshotManifestFilename),
			)
			if err == nil {
				err = validateSnapshotFiles(roots, snapshotDir, snapshot)
			}
			if err == nil {
				var databases catalog.HealthReport
				databases, err = catalog.Inspect(ctx, roots)
				if err == nil && !databases.Healthy {
					err = errors.New("published storage migration is not healthy")
				}
			}
			closeErr := profile.Close()
			if err != nil || closeErr != nil {
				return Result{}, errors.Join(err, closeErr)
			}
			journal.State = StateCommitted
			journal.LastError = ""
			journal.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
			if err := writeJournal(migrationDir, journal); err != nil {
				return Result{}, err
			}
			if journal.To != storage.LayoutVersion {
				return Apply(ctx, options)
			}
			return Result{Journal: journal}, nil
		}
	}
	return Apply(ctx, options)
}

func registry() (storage.MigrationRegistry, error) {
	oneToTwoChecksum, err := artifact.Sum(
		"yotta/storage-layout-migration/v1",
		[]byte("layout-1-to-2:checkpoint,snapshot,catalog-schema,legacy-runs,verify,publish"),
	)
	if err != nil {
		return storage.MigrationRegistry{}, err
	}
	twoToThreeChecksum, err := artifact.Sum(
		"yotta/storage-layout-migration/v1",
		[]byte("layout-2-to-3:checkpoint,snapshot,blob-v1-to-v2,inventory,verify,publish"),
	)
	if err != nil {
		return storage.MigrationRegistry{}, err
	}
	return storage.NewMigrationRegistry(storage.LayoutVersion, []storage.MigrationStep{
		{ID: layoutOneToTwoID, From: "1", To: "2", Checksum: oneToTwoChecksum},
		{ID: layoutTwoToThreeID, From: "2", To: "3", Checksum: twoToThreeChecksum},
	})
}

func currentPlan() Plan {
	return Plan{
		Format: PlanFormat, Version: DocumentVersion,
		From: storage.LayoutVersion, To: storage.LayoutVersion, Actions: []string{},
	}
}

func writeJournal(migrationDir string, journal Journal) error {
	return writeJSON(filepath.Join(migrationDir, journalFilename), journal)
}

func readJournal(path string) (Journal, bool, error) {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return Journal{}, false, nil
	}
	if err != nil {
		return Journal{}, false, err
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, 64<<10+1))
	if err != nil {
		return Journal{}, false, err
	}
	if len(raw) > 64<<10 {
		return Journal{}, false, errors.New("storage migration journal exceeds byte budget")
	}
	var journal Journal
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&journal); err != nil {
		return Journal{}, false, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return Journal{}, false, errors.New("storage migration journal must contain one JSON value")
	}
	if err := validateJournal(journal); err != nil {
		return Journal{}, false, err
	}
	return journal, true, nil
}

func validateJournal(journal Journal) error {
	registry, err := registry()
	if err != nil {
		return err
	}
	steps, err := registry.Plan(journal.From)
	if err != nil || len(steps) == 0 {
		return errors.Join(err, errors.New("storage migration registry is invalid"))
	}
	expected := steps[0]
	if journal.Format != JournalFormat ||
		(journal.Version != legacyDocVersion && journal.Version != DocumentVersion) ||
		journal.MigrationID != expected.ID || journal.From != expected.From ||
		journal.To != expected.To || journal.StepChecksum != expected.Checksum.String() ||
		journal.BackupManifest != "snapshot/"+snapshotManifestFilename {
		return errors.New("storage migration journal is invalid")
	}
	if journal.Version == legacyDocVersion &&
		(journal.MigrationID != layoutOneToTwoID || journal.BlobLayoutFrom != "") {
		return errors.New("legacy storage migration journal is invalid")
	}
	if journal.MigrationID == layoutOneToTwoID && journal.BlobLayoutFrom != "" {
		return errors.New("Catalog migration journal contains Blob layout state")
	}
	if journal.MigrationID == layoutTwoToThreeID &&
		journal.BlobLayoutFrom != "" &&
		journal.BlobLayoutFrom != "1" &&
		journal.BlobLayoutFrom != blob.LayoutVersion {
		return errors.New("Blob migration journal has an invalid source layout")
	}
	switch journal.State {
	case StatePrepared, StateApplying, StateVerifying, StateRecoveryRequired,
		StateCommitted, StateRolledBack:
	default:
		return errors.New("storage migration journal has an invalid state")
	}
	if _, err := time.Parse(time.RFC3339Nano, journal.StartedAt); err != nil {
		return errors.New("storage migration journal has an invalid start time")
	}
	if _, err := time.Parse(time.RFC3339Nano, journal.UpdatedAt); err != nil {
		return errors.New("storage migration journal has an invalid update time")
	}
	if len(journal.LastError) > 16<<10 {
		return errors.New("storage migration journal error exceeds byte budget")
	}
	if journal.BlockedEntry != "" {
		if err := validateLegacyRunName(journal.BlockedEntry); err != nil {
			return errors.New("storage migration journal has an invalid blocker")
		}
	}
	return nil
}

func activeJournal(roots storage.Roots) (Journal, bool, error) {
	registry, err := registry()
	if err != nil {
		return Journal{}, false, err
	}
	steps, err := registry.Plan("1")
	if err != nil {
		return Journal{}, false, err
	}
	var active Journal
	foundActive := false
	for _, step := range steps {
		journal, found, err := readJournal(
			filepath.Join(roots.Migrations, step.ID, journalFilename),
		)
		if err != nil {
			return Journal{}, false, err
		}
		if !found || journal.State == StateCommitted || journal.State == StateRolledBack {
			continue
		}
		if foundActive {
			return Journal{}, false, errors.New("multiple storage migrations require recovery")
		}
		active, foundActive = journal, true
	}
	return active, foundActive, nil
}

func latestJournal(roots storage.Roots) (Journal, bool, error) {
	registry, err := registry()
	if err != nil {
		return Journal{}, false, err
	}
	steps, err := registry.Plan("1")
	if err != nil {
		return Journal{}, false, err
	}
	for index := len(steps) - 1; index >= 0; index-- {
		journal, found, err := readJournal(
			filepath.Join(roots.Migrations, steps[index].ID, journalFilename),
		)
		if err != nil {
			return Journal{}, false, err
		}
		if found {
			return journal, true, nil
		}
	}
	return Journal{}, false, nil
}

func writeJSON(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return durablefs.WriteFile(path, append(raw, '\n'), 0o600)
}
