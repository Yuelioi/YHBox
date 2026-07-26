// Command yotta exposes the headless Workflow Application command surface.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
	"github.com/yottaapp/yotta/internal/ai"
	"github.com/yottaapp/yotta/internal/appbootstrap"
	"github.com/yottaapp/yotta/internal/appcontrol"
	appcore "github.com/yottaapp/yotta/internal/application"
	automationinstalled "github.com/yottaapp/yotta/internal/automation/installed"
	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/httpegress"
	"github.com/yottaapp/yotta/internal/nodepackage"
	"github.com/yottaapp/yotta/internal/noderuntime"
	runrecord "github.com/yottaapp/yotta/internal/run"
	"github.com/yottaapp/yotta/internal/scriptengine"
	"github.com/yottaapp/yotta/internal/securestore"
	"github.com/yottaapp/yotta/internal/services"
	"github.com/yottaapp/yotta/internal/storage"
	"github.com/yottaapp/yotta/internal/storage/catalog"
	storagemigrate "github.com/yottaapp/yotta/internal/storage/migrate"
	"github.com/yottaapp/yotta/internal/wasmrunner"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
	"github.com/yottaapp/yotta/internal/workflow/schema"
)

type options struct {
	dataRoot    string
	principal   string
	timeout     time.Duration
	executable  string
	showPath    bool
	command     string
	argument    string
	destination string
}

type compileView struct {
	SourceHash  string              `json:"sourceHash,omitempty"`
	ProgramHash string              `json:"programHash,omitempty"`
	Diagnostics []schema.Diagnostic `json:"diagnostics"`
}

type healthView struct {
	storage.HealthReport
	Databases catalog.HealthReport `json:"databases"`
}

type commandRuntime struct {
	*appbootstrap.Runtime
	profile           *storage.Profile
	catalogFoundation *catalog.Foundation
	settingsApp       *services.App
}

func (runtime *commandRuntime) Close(ctx context.Context) error {
	if runtime == nil {
		return nil
	}
	var errs []error
	if runtime.Runtime != nil {
		errs = append(errs, runtime.Runtime.Close(ctx))
	}
	if runtime.settingsApp != nil {
		errs = append(errs, runtime.settingsApp.ShutdownContext(ctx))
	}
	if runtime.catalogFoundation != nil {
		errs = append(errs, runtime.catalogFoundation.Close())
	}
	if runtime.profile != nil {
		errs = append(errs, runtime.profile.Close())
	}
	return errors.Join(errs...)
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(arguments []string, output io.Writer) error {
	opt, err := parseOptions(arguments)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	if opt.command == "health" {
		ctx := context.Background()
		report, err := storage.Inspect(ctx, storage.InspectOptions{
			Root: opt.dataRoot, ShowPhysicalPath: opt.showPath,
		})
		if err != nil {
			return err
		}
		roots, err := storage.Resolve(opt.dataRoot)
		if err != nil {
			return err
		}
		databases, err := catalog.Inspect(ctx, roots)
		if err != nil {
			return err
		}
		return encoder.Encode(healthView{HealthReport: report, Databases: databases})
	}
	if opt.command == "migrate" {
		return runMigrationCommand(context.Background(), encoder, opt)
	}
	runtime, err := buildRuntime(opt)
	if err != nil {
		return err
	}
	defer func() {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer closeCancel()
		_ = runtime.Close(closeCtx)
	}()
	ctx, cancel := context.WithTimeout(context.Background(), opt.timeout)
	defer cancel()
	if err := runtime.Start(ctx); err != nil {
		return fmt.Errorf("start Application: %w", err)
	}

	switch opt.command {
	case "validate":
		raw, readErr := os.ReadFile(opt.argument)
		if readErr != nil {
			return readErr
		}
		result, compileErr := runtime.Application.CompileDraft(ctx, raw)
		view := compileResultView(result)
		if err := encoder.Encode(view); err != nil {
			return err
		}
		if compileErr != nil || schema.HasErrors(result.Diagnostics) {
			return errors.Join(errors.New("workflow source validation failed"), compileErr)
		}
		return nil
	case "compile":
		result, compileErr := runtime.Application.CompileSource(ctx, opt.argument)
		view := compileResultView(result)
		if err := encoder.Encode(view); err != nil {
			return err
		}
		if compileErr != nil || schema.HasErrors(result.Diagnostics) {
			return errors.Join(errors.New("workflow compilation failed"), compileErr)
		}
		return nil
	case "inspect":
		snapshot, loadErr := runtime.Application.GetSource(opt.argument)
		if loadErr != nil {
			return loadErr
		}
		return encoder.Encode(struct {
			WorkflowID string          `json:"workflowId"`
			Revision   int64           `json:"revision"`
			SourceHash string          `json:"sourceHash"`
			Source     json.RawMessage `json:"source"`
		}{snapshot.WorkflowID(), snapshot.Revision(), snapshot.Hash().String(), snapshot.Artifact()})
	case "run":
		started, startErr := runtime.Application.StartRun(ctx, appcore.StartRunRequest{WorkflowID: opt.argument, Principal: opt.principal})
		if startErr != nil {
			_ = encoder.Encode(compileResultView(compiler.CompileResult{SourceHash: started.SourceHash, Diagnostics: started.Diagnostics}))
			return startErr
		}
		return waitRun(ctx, encoder, runtime.Application, started.Record.Admission().RunID)
	default:
		return fmt.Errorf("unsupported command %q", opt.command)
	}
}

func parseOptions(arguments []string) (options, error) {
	set := flag.NewFlagSet("yotta", flag.ContinueOnError)
	set.SetOutput(io.Discard)
	var opt options
	set.StringVar(&opt.dataRoot, "data-root", "", "Yotta storage profile root")
	set.StringVar(&opt.principal, "principal", "local-cli", "Run principal")
	set.DurationVar(&opt.timeout, "timeout", 5*time.Minute, "command deadline")
	set.BoolVar(&opt.showPath, "show-path", false, "include the physical storage path in health output")
	if err := set.Parse(arguments); err != nil {
		return options{}, err
	}
	if opt.timeout <= 0 || set.NArg() == 0 {
		return options{}, errors.New("usage: yotta [--data-root DIR] health | migrate plan|apply|resume|rollback|list|quarantine|restore|export [argument] | [--timeout 5m] validate <source.json> | compile|inspect|run <workflow-id>")
	}
	opt.command = set.Arg(0)
	if opt.command == "health" {
		if set.NArg() != 1 {
			return options{}, errors.New("usage: yotta [--data-root DIR] [--show-path] health")
		}
	} else if opt.command == "migrate" {
		if set.NArg() < 2 {
			return options{}, errors.New("storage migration requires plan, apply, resume, rollback, list, quarantine, restore, or export")
		}
		opt.argument = set.Arg(1)
		switch opt.argument {
		case "plan", "apply", "resume", "rollback", "list":
			if set.NArg() != 2 {
				return options{}, errors.New("storage migration action has unexpected arguments")
			}
		case "export", "quarantine", "restore":
			if set.NArg() != 3 {
				return options{}, errors.New("storage migration action requires one path or record name")
			}
			opt.destination = set.Arg(2)
		default:
			return options{}, fmt.Errorf("unknown storage migration action %q", opt.argument)
		}
	} else if set.NArg() != 2 {
		return options{}, errors.New("workflow commands require an argument")
	} else {
		opt.argument = set.Arg(1)
	}
	if opt.command != "health" && opt.command != "migrate" && opt.command != "validate" && opt.command != "compile" && opt.command != "inspect" && opt.command != "run" {
		return options{}, fmt.Errorf("unknown command %q", opt.command)
	}
	if opt.dataRoot != "" {
		absoluteRoot, err := filepath.Abs(opt.dataRoot)
		if err != nil {
			return options{}, err
		}
		opt.dataRoot = absoluteRoot
	}
	if opt.command == "health" || opt.command == "migrate" {
		return opt, nil
	}
	executable, err := os.Executable()
	if err != nil {
		return options{}, err
	}
	opt.executable, err = filepath.Abs(executable)
	if err != nil {
		return options{}, err
	}
	return opt, nil
}

func runMigrationCommand(ctx context.Context, encoder *json.Encoder, opt options) error {
	migrationOptions := storagemigrate.Options{Root: opt.dataRoot, MaxRuns: 65536}
	switch opt.argument {
	case "plan":
		plan, err := storagemigrate.Inspect(ctx, migrationOptions)
		if err != nil {
			return err
		}
		return encoder.Encode(plan)
	case "apply":
		result, err := storagemigrate.Apply(ctx, migrationOptions)
		if encodeErr := encoder.Encode(result); encodeErr != nil {
			return encodeErr
		}
		return err
	case "resume":
		result, err := storagemigrate.Resume(ctx, migrationOptions)
		if encodeErr := encoder.Encode(result); encodeErr != nil {
			return encodeErr
		}
		return err
	case "rollback":
		result, err := storagemigrate.Rollback(ctx, migrationOptions)
		if encodeErr := encoder.Encode(result); encodeErr != nil {
			return encodeErr
		}
		return err
	case "list":
		records, err := storagemigrate.ListQuarantine(migrationOptions)
		if err != nil {
			return err
		}
		return encoder.Encode(records)
	case "quarantine":
		record, err := storagemigrate.QuarantineLegacyRun(ctx, migrationOptions, opt.destination)
		if encodeErr := encoder.Encode(record); encodeErr != nil {
			return encodeErr
		}
		return err
	case "restore":
		record, err := storagemigrate.RestoreLegacyRun(ctx, migrationOptions, opt.destination)
		if encodeErr := encoder.Encode(record); encodeErr != nil {
			return encodeErr
		}
		return err
	case "export":
		return storagemigrate.ExportDiagnostics(ctx, migrationOptions, opt.destination)
	default:
		return fmt.Errorf("unsupported storage migration action %q", opt.argument)
	}
}

func buildRuntime(opt options) (_ *commandRuntime, resultErr error) {
	logger := zerolog.New(io.Discard)
	if _, err := storagemigrate.Ensure(context.Background(), storagemigrate.Options{
		Root: opt.dataRoot, MaxRuns: 65536,
	}); err != nil {
		return nil, fmt.Errorf("prepare storage profile migration: %w", err)
	}
	profile, err := storage.Open(context.Background(), storage.OpenOptions{Root: opt.dataRoot})
	if err != nil {
		return nil, fmt.Errorf("open storage profile: %w", err)
	}
	defer func() {
		if resultErr != nil {
			resultErr = errors.Join(resultErr, profile.Close())
		}
	}()
	roots := profile.Roots
	catalogFoundation, err := catalog.Open(context.Background(), roots)
	if err != nil {
		return nil, fmt.Errorf("open catalog foundation: %w", err)
	}
	defer func() {
		if resultErr != nil {
			resultErr = errors.Join(resultErr, catalogFoundation.Close())
		}
	}()
	settingsApp, err := services.OpenApp(roots.SettingsFile(), roots.Logs, nil, logger)
	if err != nil {
		return nil, fmt.Errorf("open application settings: %w", err)
	}
	defer func() {
		if resultErr != nil {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			resultErr = errors.Join(resultErr, settingsApp.ShutdownContext(shutdownCtx))
		}
	}()
	secrets := services.NewAISecrets(securestore.New())
	aiInstallations, err := ai.Install(settingsApp.Settings().AI.InstallationDrafts(), secrets)
	if err != nil {
		return nil, err
	}
	httpInstallations, err := httpegress.Install(settingsApp.Settings().Network.InstallationDrafts())
	if err != nil {
		return nil, err
	}
	applicationInstallations, err := appcontrol.Install(settingsApp.Settings().Applications.InstallationDrafts())
	if err != nil {
		return nil, err
	}
	automationDrafts, err := settingsApp.Settings().Automation.InstallationDrafts(settingsApp.Settings().Applications, settingsApp.Settings().ActiveMouseCounts360())
	if err != nil {
		return nil, err
	}
	automationInstallations, err := automationinstalled.Install(automationDrafts)
	if err != nil {
		return nil, err
	}
	blobStore, err := blob.Open(
		roots.Objects,
		blob.Limits{MaxBlobBytes: 256 << 20, MaxTotalBytes: 4 << 30},
		catalogFoundation.Objects(),
	)
	if err != nil {
		return nil, err
	}
	scriptRuntime, err := scriptengine.NewRuntime(scriptengine.RuntimeOptions{
		Executable:         filepath.Join(filepath.Dir(opt.executable), scriptengine.WorkerExecutableName),
		ProcessMemoryBytes: scriptengine.DefaultMemoryBytes, JobMemoryBytes: scriptengine.DefaultMemoryBytes,
	})
	if err != nil {
		return nil, err
	}
	packages, _, err := nodepackage.OpenStoreIfPresent(context.Background(), filepath.Join(roots.Packages, "node"))
	if err != nil {
		return nil, err
	}
	applicationRuntime, err := appbootstrap.Build(appbootstrap.Config{
		DataRoot: roots.Data, ProgramCacheRoot: filepath.Join(roots.Cache, "programs"),
		WorkflowRepository: catalogFoundation.Workflows(),
		RunRepository:      catalogFoundation.Runs(),
		BlobStore:          blobStore,
		Limits: appbootstrap.Limits{
			MaxSources: 4096, MaxPrograms: 16384, MaxRuns: 65536, MaxResourcePayloadBytes: 4 << 20,
			MaxProgramCacheBytes: 2 << 30,
			BlobChunkBytes:       64 << 10, BlobQueueCapacity: 8, StreamCapacity: 16, StreamChunkBytes: 64 << 10,
		},
		AIInstallations: aiInstallations, HTTPInstallations: httpInstallations,
		ApplicationInstallations: applicationInstallations, AutomationInstallations: automationInstallations,
		ScriptRuntime: scriptRuntime, NodePackageStore: packages,
		WasmRunnerExecutable: filepath.Join(filepath.Dir(opt.executable), wasmrunner.WorkerExecutableName),
		LogEmitter:           discardLog{}, GrantTTL: 5 * time.Minute, OwnerCloseTimeout: 10 * time.Second, Now: time.Now,
	})
	if err != nil {
		return nil, err
	}
	return &commandRuntime{
		Runtime: applicationRuntime, profile: profile,
		catalogFoundation: catalogFoundation, settingsApp: settingsApp,
	}, nil
}

func compileResultView(result compiler.CompileResult) compileView {
	view := compileView{SourceHash: result.SourceHash.String(), Diagnostics: append([]schema.Diagnostic(nil), result.Diagnostics...)}
	if program, ok := result.Program(); ok {
		view.ProgramHash = program.Hash().String()
	}
	return view
}

func waitRun(ctx context.Context, encoder *json.Encoder, application *appcore.Application, runID string) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		record, err := application.GetRun(runID)
		if err != nil {
			return err
		}
		switch record.Status() {
		case runrecord.StatusSucceeded:
			return encoder.Encode(json.RawMessage(record.Bytes()))
		case runrecord.StatusFailed, runrecord.StatusCancelled, runrecord.StatusInterrupted:
			if err := encoder.Encode(json.RawMessage(record.Bytes())); err != nil {
				return err
			}
			return fmt.Errorf("run %s ended as %s", runID, record.Status())
		}
		select {
		case <-ctx.Done():
			_, _ = application.CancelRun(context.Background(), runID)
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

type discardLog struct{}

func (discardLog) EmitWorkflowLog(context.Context, noderuntime.LogEntry) error { return nil }
