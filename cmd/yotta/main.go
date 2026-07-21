// Command yotta exposes the headless Workflow 3.1 Application command surface.
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
	"github.com/yottaapp/yotta/internal/wasmrunner"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
	"github.com/yottaapp/yotta/internal/workflow/schema"
)

type options struct {
	dataRoot   string
	settings   string
	principal  string
	timeout    time.Duration
	executable string
	command    string
	argument   string
}

type compileView struct {
	SourceHash  string              `json:"sourceHash,omitempty"`
	ProgramHash string              `json:"programHash,omitempty"`
	Diagnostics []schema.Diagnostic `json:"diagnostics"`
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
	runtime, err := buildRuntime(opt)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), opt.timeout)
	defer cancel()
	if err := runtime.Start(ctx); err != nil {
		return fmt.Errorf("start Application: %w", err)
	}
	defer func() {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer closeCancel()
		_ = runtime.Close(closeCtx)
	}()

	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
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
	set.StringVar(&opt.dataRoot, "data-root", "", "Yotta data directory")
	set.StringVar(&opt.settings, "settings", "", "settings JSON path")
	set.StringVar(&opt.principal, "principal", "local-cli", "Run principal")
	set.DurationVar(&opt.timeout, "timeout", 5*time.Minute, "command deadline")
	if err := set.Parse(arguments); err != nil {
		return options{}, err
	}
	if opt.timeout <= 0 || set.NArg() != 2 {
		return options{}, errors.New("usage: yotta [--data-root DIR] [--settings FILE] [--timeout 5m] validate <source.json> | compile|inspect|run <workflow-id>")
	}
	opt.command, opt.argument = set.Arg(0), set.Arg(1)
	if opt.command != "validate" && opt.command != "compile" && opt.command != "inspect" && opt.command != "run" {
		return options{}, fmt.Errorf("unknown command %q", opt.command)
	}
	executable, err := os.Executable()
	if err != nil {
		return options{}, err
	}
	opt.executable, err = filepath.Abs(executable)
	if err != nil {
		return options{}, err
	}
	if opt.dataRoot == "" {
		opt.dataRoot = filepath.Join(filepath.Dir(opt.executable), "data")
	}
	opt.dataRoot, err = filepath.Abs(opt.dataRoot)
	return opt, err
}

func buildRuntime(opt options) (*appbootstrap.Runtime, error) {
	logger := zerolog.New(io.Discard)
	settingsApp := services.NewApp(opt.settings, nil, logger)
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
	blobStore, err := blob.Open(filepath.Join(opt.dataRoot, "blobs"), blob.Limits{MaxBlobBytes: 256 << 20, MaxTotalBytes: 4 << 30})
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
	packages, _, err := nodepackage.OpenStoreIfPresent(context.Background(), filepath.Join(opt.dataRoot, "node-packages"))
	if err != nil {
		return nil, err
	}
	return appbootstrap.Build(appbootstrap.Config{
		DataRoot: opt.dataRoot, BlobStore: blobStore,
		Limits: appbootstrap.Limits{
			MaxSources: 4096, MaxPrograms: 16384, MaxRuns: 65536, MaxResourcePayloadBytes: 4 << 20,
			BlobChunkBytes: 64 << 10, BlobQueueCapacity: 8, StreamCapacity: 16, StreamChunkBytes: 64 << 10,
		},
		AIInstallations: aiInstallations, HTTPInstallations: httpInstallations,
		ApplicationInstallations: applicationInstallations, AutomationInstallations: automationInstallations,
		ScriptRuntime: scriptRuntime, NodePackageStore: packages,
		WasmRunnerExecutable: filepath.Join(filepath.Dir(opt.executable), wasmrunner.WorkerExecutableName),
		LogEmitter:           discardLog{}, GrantTTL: 5 * time.Minute, OwnerCloseTimeout: 10 * time.Second, Now: time.Now,
	})
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
