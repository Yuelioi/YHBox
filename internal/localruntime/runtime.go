// Package localruntime opens the complete storage-backed Yotta runtime shared
// by desktop and command-line presentation adapters.
package localruntime

import (
	"context"
	"errors"
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
	"github.com/yottaapp/yotta/internal/scriptengine"
	"github.com/yottaapp/yotta/internal/services"
	"github.com/yottaapp/yotta/internal/storage"
	"github.com/yottaapp/yotta/internal/storage/catalog"
	"github.com/yottaapp/yotta/internal/wasmrunner"
)

type Config struct {
	StorageRoot       string
	Executable        string
	LogSink           *services.LogSink
	RootLog           zerolog.Logger
	ConfigureFileLogs bool
	AISecrets         *services.AISecrets
	WorkflowLog       noderuntime.LogEmitter
	Now               func() time.Time
	OnRunEvent        func(appcore.RunEvent)
	OnDebugEvent      func(appcore.DebugEvent)
}

// Runtime owns every non-presentation resource opened for one local profile.
// Callers use the projected repositories but close only Runtime.
type Runtime struct {
	Roots    storage.Roots
	Settings *services.App
	Workflow *appbootstrap.Runtime
	Assets   *catalog.AssetRepository
	Objects  *catalog.ObjectRepository

	profile    *storage.Profile
	foundation *catalog.Foundation
}

func Open(ctx context.Context, config Config) (_ *Runtime, resultErr error) {
	if ctx == nil {
		return nil, errors.New("open local runtime requires a context")
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	if config.Executable == "" || config.WorkflowLog == nil {
		return nil, errors.New("open local runtime requires executable and workflow log")
	}
	opened := &Runtime{}
	defer func() {
		if resultErr == nil {
			return
		}
		closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		resultErr = errors.Join(resultErr, opened.Close(closeCtx))
	}()

	profile, err := storage.Open(ctx, storage.OpenOptions{Root: config.StorageRoot})
	if err != nil {
		return nil, err
	}
	opened.profile = profile
	opened.Roots = profile.Roots

	foundation, err := catalog.Open(ctx, opened.Roots)
	if err != nil {
		return nil, err
	}
	opened.foundation = foundation
	opened.Assets = foundation.Assets()
	opened.Objects = foundation.Objects()

	if config.ConfigureFileLogs {
		opened.Settings, err = services.OpenConfiguredApp(
			opened.Roots.SettingsFile(), opened.Roots.Logs, config.LogSink, config.RootLog,
		)
	} else {
		opened.Settings, err = services.OpenApp(
			opened.Roots.SettingsFile(), opened.Roots.Logs, config.LogSink, config.RootLog,
		)
	}
	if err != nil {
		return nil, err
	}
	settings := opened.Settings.Settings()

	aiInstallations, err := ai.Install(settings.AI.InstallationDrafts(), config.AISecrets)
	if err != nil {
		return nil, err
	}
	httpInstallations, err := httpegress.Install(settings.Network.InstallationDrafts())
	if err != nil {
		return nil, err
	}
	applicationInstallations, err := appcontrol.Install(settings.Applications.InstallationDrafts())
	if err != nil {
		return nil, err
	}
	automationDrafts, err := settings.Automation.InstallationDrafts(
		settings.Applications,
		settings.ActiveMouseCounts360(),
	)
	if err != nil {
		return nil, err
	}
	automationInstallations, err := automationinstalled.Install(automationDrafts)
	if err != nil {
		return nil, err
	}
	automationOwned := false
	defer func() {
		if resultErr != nil && !automationOwned {
			resultErr = errors.Join(resultErr, automationInstallations.Close())
		}
	}()

	blobStore, err := blob.Open(
		opened.Roots.Objects,
		blob.Limits{MaxBlobBytes: 256 << 20, MaxTotalBytes: 4 << 30},
		opened.Objects,
	)
	if err != nil {
		return nil, err
	}
	scriptRuntime, err := scriptengine.NewRuntime(scriptengine.RuntimeOptions{
		Executable:         filepath.Join(filepath.Dir(config.Executable), scriptengine.WorkerExecutableName),
		ProcessMemoryBytes: scriptengine.DefaultMemoryBytes,
		JobMemoryBytes:     scriptengine.DefaultMemoryBytes,
	})
	if err != nil {
		return nil, err
	}
	packages, _, err := nodepackage.OpenStoreIfPresent(ctx, filepath.Join(opened.Roots.Packages, "node"))
	if err != nil {
		return nil, err
	}
	opened.Workflow, err = appbootstrap.Build(appbootstrap.Config{
		DataRoot:           opened.Roots.Data,
		ProgramCacheRoot:   filepath.Join(opened.Roots.Cache, "programs"),
		WorkflowRepository: foundation.Workflows(),
		RunRepository:      foundation.Runs(),
		BlobStore:          blobStore,
		Limits: appbootstrap.Limits{
			MaxSources: 4096, MaxPrograms: 16384, MaxRuns: 65536,
			MaxProgramCacheBytes:    2 << 30,
			MaxResourcePayloadBytes: 4 << 20,
			BlobChunkBytes:          64 << 10,
			BlobQueueCapacity:       8,
			StreamCapacity:          16,
			StreamChunkBytes:        64 << 10,
		},
		AIInstallations:          aiInstallations,
		HTTPInstallations:        httpInstallations,
		ApplicationInstallations: applicationInstallations,
		AutomationInstallations:  automationInstallations,
		ScriptRuntime:            scriptRuntime,
		NodePackageStore:         packages,
		WasmRunnerExecutable:     filepath.Join(filepath.Dir(config.Executable), wasmrunner.WorkerExecutableName),
		LogEmitter:               config.WorkflowLog,
		OwnerCloseTimeout:        10 * time.Second,
		Now:                      config.Now,
		OnRunEvent:               config.OnRunEvent,
		OnDebugEvent:             config.OnDebugEvent,
	})
	if err != nil {
		return nil, err
	}
	automationOwned = true
	return opened, nil
}

func (runtime *Runtime) Close(ctx context.Context) error {
	if runtime == nil {
		return nil
	}
	var result error
	if runtime.Workflow != nil {
		result = errors.Join(result, runtime.Workflow.Close(ctx))
		runtime.Workflow = nil
	}
	if runtime.Settings != nil {
		result = errors.Join(result, runtime.Settings.ShutdownContext(ctx))
		runtime.Settings = nil
	}
	if runtime.foundation != nil {
		result = errors.Join(result, runtime.foundation.Close())
		runtime.foundation = nil
	}
	if runtime.profile != nil {
		result = errors.Join(result, runtime.profile.Close())
		runtime.profile = nil
	}
	return result
}
