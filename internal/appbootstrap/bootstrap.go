// Package appbootstrap constructs the production Yotta 3.1 application with
// immutable contracts, durable stores, installed providers, admission, and a
// single Program worker.
package appbootstrap

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"time"

	"github.com/yottaapp/yotta/internal/admission"
	"github.com/yottaapp/yotta/internal/ai"
	"github.com/yottaapp/yotta/internal/appcontrol"
	appcore "github.com/yottaapp/yotta/internal/application"
	"github.com/yottaapp/yotta/internal/artifact"
	automationinstalled "github.com/yottaapp/yotta/internal/automation/installed"
	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/httpegress"
	"github.com/yottaapp/yotta/internal/nodeauthoring"
	"github.com/yottaapp/yotta/internal/nodecontract"
	"github.com/yottaapp/yotta/internal/nodepackage"
	"github.com/yottaapp/yotta/internal/noderuntime"
	"github.com/yottaapp/yotta/internal/nodes"
	"github.com/yottaapp/yotta/internal/pluginhost"
	"github.com/yottaapp/yotta/internal/resource"
	run "github.com/yottaapp/yotta/internal/run"
	"github.com/yottaapp/yotta/internal/scriptengine"
	"github.com/yottaapp/yotta/internal/stream"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
	"github.com/yottaapp/yotta/internal/workflowstore"
	"github.com/yottaapp/yotta/internal/workspacefs"
)

type Limits struct {
	MaxSources              int
	MaxPrograms             int
	MaxRuns                 int
	MaxResourcePayloadBytes int
	BlobChunkBytes          int
	BlobQueueCapacity       int
	StreamCapacity          int
	StreamChunkBytes        int
}

type Config struct {
	DataRoot                 string
	BlobStore                *blob.Store
	Limits                   Limits
	AIInstallations          ai.Installations
	HTTPInstallations        httpegress.Installations
	ApplicationInstallations appcontrol.Installations
	AutomationInstallations  automationinstalled.Installations
	ScriptRuntime            *scriptengine.Runtime
	NodePackageStore         *nodepackage.Store
	WasmRunnerExecutable     string
	PluginExecution          pluginhost.ProcessHostOptions
	LogEmitter               noderuntime.LogEmitter
	GrantTTL                 time.Duration
	OwnerCloseTimeout        time.Duration
	Now                      func() time.Time
	OnRunEvent               func(appcore.RunEvent)
	OnDebugEvent             func(appcore.DebugEvent)
}

type Runtime struct {
	Application  *appcore.Application
	Builtins     nodes.Builtins
	BlobStore    *blob.Store
	ai           ai.Installations
	http         httpegress.Installations
	applications appcontrol.Installations
	automation   automationinstalled.Installations
}

func Build(config Config) (*Runtime, error) {
	if config.Now == nil {
		config.Now = time.Now
	}
	if !config.AIInstallations.Valid() || !config.HTTPInstallations.Valid() || !config.ApplicationInstallations.Valid() || !config.AutomationInstallations.Valid() || config.BlobStore == nil || config.ScriptRuntime == nil || config.LogEmitter == nil || config.GrantTTL <= 0 || config.GrantTTL > 24*time.Hour || config.OwnerCloseTimeout <= 0 {
		return nil, errors.New("app bootstrap requires trusted installations, isolated effect runtimes, and bounded Run lifetimes")
	}
	if err := validateLimits(config.Limits); err != nil {
		return nil, err
	}
	root, err := filepath.Abs(config.DataRoot)
	if err != nil || config.DataRoot == "" {
		return nil, errors.New("app bootstrap requires a data root")
	}
	builtins, err := nodes.Build()
	if err != nil {
		return nil, fmt.Errorf("build Catalog 3.1: %w", err)
	}
	catalog := builtins.Catalog
	var runtimePackages []nodepackage.RuntimePackage
	var processPlugins *pluginhost.ProcessHost
	var wasmPlugins *pluginhost.WasmHost
	pluginFeatures := []string{}
	if config.NodePackageStore != nil {
		runtimePackages, err = config.NodePackageStore.RuntimePackages(context.Background(), nodepackage.RuntimeHost{
			APIGeneration: "3.1", OperatingSystem: runtime.GOOS, Architecture: runtime.GOARCH,
		})
		if err != nil {
			return nil, fmt.Errorf("project enabled node packages: %w", err)
		}
		if len(runtimePackages) != 0 {
			projection, err := pluginhost.MergeCatalog(catalog, runtimePackages, nodes.GeneratorVersion)
			if err != nil {
				return nil, err
			}
			catalog = projection.Catalog
			hasProcess, hasWasm := false, false
			for _, runtimePackage := range runtimePackages {
				for _, node := range runtimePackage.Nodes {
					hasProcess = hasProcess || node.Implementation.ABI.Kind == nodecontract.ABIProcess
					hasWasm = hasWasm || node.Implementation.ABI.Kind == nodecontract.ABIWIT
				}
			}
			if hasProcess {
				processPlugins, err = pluginhost.NewProcessHost(catalog, config.PluginExecution)
				if err != nil {
					return nil, err
				}
				pluginFeatures = append(pluginFeatures, processPlugins.HostFeatures()...)
			}
			if hasWasm {
				wasmPlugins, err = pluginhost.NewWasmHost(catalog, pluginhost.WasmHostOptions{
					RunnerExecutable: config.WasmRunnerExecutable, Execution: config.PluginExecution,
				})
				if err != nil {
					return nil, err
				}
				pluginFeatures = append(pluginFeatures, wasmPlugins.HostFeatures()...)
			}
		}
	}
	config.AIInstallations, err = config.AIInstallations.ForEvaluationArtifacts(builtins.AIEvaluationArtifacts())
	if err != nil {
		return nil, err
	}
	bindings := catalog.Bindings()
	contracts := make([]nodecontract.Contract, 0, len(bindings))
	for _, binding := range bindings {
		contracts = append(contracts, binding.Contract)
	}
	authoringProjection, err := nodeauthoring.Project(nodeauthoring.Input{
		Catalog: catalog, Types: catalog.Types(), Capabilities: catalog.Capabilities(),
		Contracts: contracts, GeneratorVersion: nodes.GeneratorVersion,
	})
	if err != nil {
		return nil, fmt.Errorf("build Authoring Projection 3.1: %w", err)
	}
	build, err := compiler.BuildDigest()
	if err != nil {
		return nil, fmt.Errorf("build compiler identity: %w", err)
	}
	workspace := filepath.Join(root, "workspace-3.1")
	sources, err := workflowstore.OpenSourceStore(filepath.Join(workspace, "workflows"), workflowstore.SourceStoreOptions{MaxSources: config.Limits.MaxSources})
	if err != nil {
		return nil, err
	}
	programs, err := workflowstore.OpenProgramStore(filepath.Join(workspace, "programs"), catalog, builtins.ConfigValidators, build, workflowstore.ProgramStoreOptions{MaxPrograms: config.Limits.MaxPrograms})
	if err != nil {
		return nil, err
	}
	runs, err := run.OpenStore(filepath.Join(workspace, "runs"), catalog, run.StoreOptions{MaxRecords: config.Limits.MaxRuns})
	if err != nil {
		return nil, err
	}
	blobStore := config.BlobStore
	blobProvider, err := blob.NewProvider(blobStore, blob.ProviderLimits{
		MaxChunkBytes: config.Limits.BlobChunkBytes, QueueCapacity: config.Limits.BlobQueueCapacity,
	})
	if err != nil {
		return nil, err
	}
	streamProvider, err := stream.NewProvider(stream.Limits{
		MaxCapacity: config.Limits.StreamCapacity, MaxChunkBytes: config.Limits.StreamChunkBytes,
	})
	if err != nil {
		return nil, err
	}
	workspaceFileProvider, err := workspacefs.NewProvider(filepath.Join(workspace, "files"), workspacefs.Limits{MaxReadBytes: nodes.DefaultFileReadBytes})
	if err != nil {
		return nil, err
	}
	blobDigest, err := blob.ProviderArtifactDigest()
	if err != nil {
		return nil, err
	}
	streamDigest, err := stream.ProviderArtifactDigest()
	if err != nil {
		return nil, err
	}
	workspaceFileDigest, err := workspacefs.ProviderArtifactDigest()
	if err != nil {
		return nil, err
	}
	profile, err := builtinHostProfile(builtins, blobDigest, streamDigest, workspaceFileDigest, config.ScriptRuntime, pluginFeatures, config.AIInstallations, config.HTTPInstallations, config.ApplicationInstallations, config.AutomationInstallations)
	if err != nil {
		return nil, err
	}
	policy, err := NewBuiltinPolicy(config.Now, config.GrantTTL, config.AIInstallations, config.HTTPInstallations, config.ApplicationInstallations, config.AutomationInstallations)
	if err != nil {
		return nil, err
	}
	admitter, err := admission.New(catalog, profile, runs, policy, admission.Options{
		Now: config.Now, MaxGrantTTL: config.GrantTTL,
	})
	if err != nil {
		return nil, err
	}
	adapters, err := noderuntime.Installed(builtins, noderuntime.Dependencies{Script: config.ScriptRuntime, Log: config.LogEmitter})
	if err != nil {
		return nil, err
	}
	mergePluginAdapters := func(installed map[string]compiler.InstalledAdapter, err error) error {
		if err != nil {
			return err
		}
		for entrypoint, adapter := range installed {
			if _, conflict := adapters[entrypoint]; conflict {
				return fmt.Errorf("plugin adapter entrypoint %q conflicts with an installed adapter", entrypoint)
			}
			adapters[entrypoint] = adapter
		}
		return nil
	}
	if processPlugins != nil {
		if err := mergePluginAdapters(processPlugins.Adapters(runtimePackages)); err != nil {
			return nil, err
		}
	}
	if wasmPlugins != nil {
		if err := mergePluginAdapters(wasmPlugins.Adapters(runtimePackages)); err != nil {
			return nil, err
		}
	}
	executor := compiler.NewExecutor(catalog, adapters, compiler.ExecutorOptions{Now: config.Now})
	providers := map[string]run.InstalledProvider{
		blob.ProviderID:        {ArtifactDigest: blobDigest, ABI: blob.ProviderABI, Provider: blobProvider},
		stream.ProviderID:      {ArtifactDigest: streamDigest, ABI: stream.ProviderABI, Provider: streamProvider},
		workspacefs.ProviderID: {ArtifactDigest: workspaceFileDigest, ABI: workspacefs.ProviderABI, Provider: workspaceFileProvider},
	}
	for _, installed := range config.AIInstallations.Entries() {
		if existing, ok := providers[installed.ProviderID]; ok {
			if existing.ArtifactDigest != installed.ProviderArtifact || existing.ABI != ai.ProviderABI || existing.Provider != installed.Provider {
				return nil, errors.New("conflicting AI provider installation")
			}
			continue
		}
		providers[installed.ProviderID] = run.InstalledProvider{
			ArtifactDigest: installed.ProviderArtifact, ABI: ai.ProviderABI, Provider: installed.Provider,
		}
	}
	for _, installed := range config.HTTPInstallations.Entries() {
		if existing, ok := providers[installed.ProviderID]; ok {
			if existing.ArtifactDigest != installed.ProviderArtifact || existing.ABI != httpegress.ProviderABI || existing.Provider != installed.Provider {
				return nil, errors.New("conflicting HTTP provider installation")
			}
			continue
		}
		providers[installed.ProviderID] = run.InstalledProvider{ArtifactDigest: installed.ProviderArtifact, ABI: httpegress.ProviderABI, Provider: installed.Provider}
	}
	for _, installed := range config.ApplicationInstallations.Entries() {
		if existing, ok := providers[installed.ProviderID]; ok {
			if existing.ArtifactDigest != installed.ProviderArtifact || existing.ABI != appcontrol.ProviderABI || existing.Provider != installed.Provider {
				return nil, errors.New("conflicting application provider installation")
			}
			continue
		}
		providers[installed.ProviderID] = run.InstalledProvider{ArtifactDigest: installed.ProviderArtifact, ABI: appcontrol.ProviderABI, Provider: installed.Provider}
	}
	for _, installed := range config.AutomationInstallations.Entries() {
		if existing, ok := providers[installed.ProviderID]; ok {
			if existing.ArtifactDigest != installed.ProviderArtifact || existing.ABI != automationinstalled.ProviderABI || existing.Provider != installed.Provider {
				return nil, errors.New("conflicting installed automation provider")
			}
			continue
		}
		providers[installed.ProviderID] = run.InstalledProvider{ArtifactDigest: installed.ProviderArtifact, ABI: automationinstalled.ProviderABI, Provider: installed.Provider}
	}
	application, err := appcore.New(appcore.Config{
		Catalog: catalog, Authoring: authoringProjection, CompilerBuild: build, ConfigValidators: builtins.ConfigValidators,
		Sources: sources, Programs: programs, Runs: runs,
		Admitter: admitter, Executor: executor,
		Providers: providers,
		ResourceOptions: resource.Options{
			Now: config.Now, MaxPayloadBytes: config.Limits.MaxResourcePayloadBytes,
		},
		OwnerCloseTimeout: config.OwnerCloseTimeout, Now: config.Now,
		OnRunEvent: config.OnRunEvent, OnDebugEvent: config.OnDebugEvent,
	})
	if err != nil {
		return nil, err
	}
	return &Runtime{Application: application, Builtins: builtins, BlobStore: blobStore, ai: config.AIInstallations, http: config.HTTPInstallations, applications: config.ApplicationInstallations, automation: config.AutomationInstallations}, nil
}

func (r *Runtime) Start(ctx context.Context) error {
	if r == nil || r.Application == nil {
		return errors.New("app runtime is unavailable")
	}
	return r.Application.Start(ctx)
}

func (r *Runtime) Close(ctx context.Context) error {
	if r == nil || r.Application == nil {
		return nil
	}
	err := errors.Join(r.Application.Close(ctx), r.automation.Close())
	r.ai.CloseIdleConnections()
	r.http.CloseIdleConnections()
	return err
}

func validateLimits(limits Limits) error {
	if limits.MaxSources <= 0 || limits.MaxPrograms <= 0 || limits.MaxRuns <= 0 ||
		limits.MaxResourcePayloadBytes < 2*nodes.DefaultFileReadBytes || limits.BlobChunkBytes <= 0 || limits.BlobChunkBytes > limits.MaxResourcePayloadBytes ||
		limits.BlobQueueCapacity <= 0 || limits.StreamCapacity <= 0 || limits.StreamChunkBytes <= 0 ||
		limits.StreamChunkBytes > limits.MaxResourcePayloadBytes {
		return errors.New("app bootstrap limits are invalid")
	}
	return nil
}

func builtinHostProfile(builtins nodes.Builtins, blobDigest, streamDigest, workspaceFileDigest artifact.Digest, scriptRuntime *scriptengine.Runtime, pluginFeatures []string, aiInstallations ai.Installations, httpInstallations httpegress.Installations, applicationInstallations appcontrol.Installations, automationInstallations automationinstalled.Installations) (admission.HostProfile, error) {
	lookup := func(id string) (capability.Ref, error) {
		definition, ok := builtins.Catalog.LookupCapability(id)
		if !ok {
			return capability.Ref{}, fmt.Errorf("built-in capability %q is missing", id)
		}
		return definition.Ref(), nil
	}
	blobRead, err := lookup(nodes.BlobReadCapabilityID)
	if err != nil {
		return admission.HostProfile{}, err
	}
	blobWrite, err := lookup(nodes.BlobWriteCapabilityID)
	if err != nil {
		return admission.HostProfile{}, err
	}
	streamSession, err := lookup(nodes.StreamCapabilityID)
	if err != nil {
		return admission.HostProfile{}, err
	}
	aiGeneration, err := lookup(nodes.AIGenerationCapabilityID)
	if err != nil {
		return admission.HostProfile{}, err
	}
	filesystemRead, err := lookup(nodes.FilesystemReadCapabilityID)
	if err != nil {
		return admission.HostProfile{}, err
	}
	httpGet, err := lookup(nodes.HTTPGetCapabilityID)
	if err != nil {
		return admission.HostProfile{}, err
	}
	applicationLifecycle, err := lookup(nodes.ApplicationLifecycleCapabilityID)
	if err != nil {
		return admission.HostProfile{}, err
	}
	automationInput, err := lookup(nodes.AutomationInputCapabilityID)
	if err != nil {
		return admission.HostProfile{}, err
	}
	automationWindow, err := lookup(nodes.AutomationWindowCapabilityID)
	if err != nil {
		return admission.HostProfile{}, err
	}
	automationCapture, err := lookup(nodes.AutomationCaptureCapabilityID)
	if err != nil {
		return admission.HostProfile{}, err
	}
	automationPlayback, err := lookup(nodes.AutomationPlaybackCapabilityID)
	if err != nil {
		return admission.HostProfile{}, err
	}
	draft := admission.HostProfileDraft{
		OS: runtime.GOOS, Architecture: runtime.GOARCH, HostAPIGeneration: "3.1",
		Features: append(scriptRuntime.HostFeatures(), pluginFeatures...),
		Providers: []admission.ProviderDescriptor{
			{ID: blob.ProviderID, ArtifactDigest: blobDigest, ABI: blob.ProviderABI, PluginInstanceID: "builtin",
				OperatingSystems: []string{runtime.GOOS}, Architectures: []string{runtime.GOARCH}, HostAPIs: []string{"3.1"},
				Capabilities: []admission.ProviderCapability{
					{Capability: blobRead, ResourceKind: blob.KindReader},
					{Capability: blobWrite, ResourceKind: blob.KindWriter},
				}},
			{ID: stream.ProviderID, ArtifactDigest: streamDigest, ABI: stream.ProviderABI, PluginInstanceID: "builtin",
				OperatingSystems: []string{runtime.GOOS}, Architectures: []string{runtime.GOARCH}, HostAPIs: []string{"3.1"},
				Capabilities: []admission.ProviderCapability{{Capability: streamSession, ResourceKind: stream.Kind}}},
			{ID: workspacefs.ProviderID, ArtifactDigest: workspaceFileDigest, ABI: workspacefs.ProviderABI, PluginInstanceID: "builtin",
				OperatingSystems: []string{runtime.GOOS}, Architectures: []string{runtime.GOARCH}, HostAPIs: []string{"3.1"},
				Capabilities: []admission.ProviderCapability{{Capability: filesystemRead, ResourceKind: workspacefs.Kind}}},
		},
		Targets: []admission.AutomationTarget{
			{ID: "workspace", Kind: "blob-store", ProviderID: blob.ProviderID},
			{ID: "memory", Kind: "stream-session", ProviderID: stream.ProviderID},
			{ID: workspacefs.TargetID, Kind: workspacefs.TargetKind, ProviderID: workspacefs.ProviderID},
		},
		TargetSlots: []admission.TargetSlotBinding{{Slot: "workspace-files", TargetID: workspacefs.TargetID}},
	}
	providerIDs := map[string]struct{}{blob.ProviderID: {}, stream.ProviderID: {}, workspacefs.ProviderID: {}}
	for _, installed := range aiInstallations.Entries() {
		if _, exists := providerIDs[installed.ProviderID]; !exists {
			draft.Providers = append(draft.Providers, admission.ProviderDescriptor{
				ID: installed.ProviderID, ArtifactDigest: installed.ProviderArtifact, ABI: ai.ProviderABI, PluginInstanceID: "builtin",
				OperatingSystems: []string{runtime.GOOS}, Architectures: []string{runtime.GOARCH}, HostAPIs: []string{"3.1"},
				Capabilities: []admission.ProviderCapability{{Capability: aiGeneration, ResourceKind: ai.KindModelSession}},
			})
			providerIDs[installed.ProviderID] = struct{}{}
		}
		draft.Targets = append(draft.Targets, admission.AutomationTarget{
			ID: installed.TargetID, Kind: "ai-model", ProviderID: installed.ProviderID,
		})
		draft.Credentials = append(draft.Credentials, admission.CredentialBinding{
			ID: installed.CredentialBindingID, ProviderID: installed.ProviderID, Capability: aiGeneration,
		})
		draft.TargetSlots = append(draft.TargetSlots, admission.TargetSlotBinding{Slot: installed.Slot, TargetID: installed.TargetID})
		draft.CredentialSlots = append(draft.CredentialSlots, admission.CredentialSlotBinding{
			Slot: installed.Slot, CredentialID: installed.CredentialBindingID,
		})
	}
	for _, installed := range httpInstallations.Entries() {
		if _, exists := providerIDs[installed.ProviderID]; !exists {
			draft.Providers = append(draft.Providers, admission.ProviderDescriptor{
				ID: installed.ProviderID, ArtifactDigest: installed.ProviderArtifact, ABI: httpegress.ProviderABI, PluginInstanceID: "builtin",
				OperatingSystems: []string{runtime.GOOS}, Architectures: []string{runtime.GOARCH}, HostAPIs: []string{"3.1"},
				Capabilities: []admission.ProviderCapability{{Capability: httpGet, ResourceKind: httpegress.KindHTTPSession}},
			})
			providerIDs[installed.ProviderID] = struct{}{}
		}
		draft.Targets = append(draft.Targets, admission.AutomationTarget{ID: installed.TargetID, Kind: httpegress.TargetKind, ProviderID: installed.ProviderID})
		draft.TargetSlots = append(draft.TargetSlots, admission.TargetSlotBinding{Slot: installed.Slot, TargetID: installed.TargetID})
	}
	for _, installed := range applicationInstallations.Entries() {
		if _, exists := providerIDs[installed.ProviderID]; !exists {
			draft.Providers = append(draft.Providers, admission.ProviderDescriptor{
				ID: installed.ProviderID, ArtifactDigest: installed.ProviderArtifact, ABI: appcontrol.ProviderABI, PluginInstanceID: "builtin",
				OperatingSystems: []string{runtime.GOOS}, Architectures: []string{runtime.GOARCH}, HostAPIs: []string{"3.1"},
				Capabilities: []admission.ProviderCapability{{Capability: applicationLifecycle, ResourceKind: appcontrol.KindApplication}},
			})
			providerIDs[installed.ProviderID] = struct{}{}
		}
		draft.Targets = append(draft.Targets, admission.AutomationTarget{ID: installed.TargetID, Kind: appcontrol.TargetKind, ProviderID: installed.ProviderID})
		draft.TargetSlots = append(draft.TargetSlots, admission.TargetSlotBinding{Slot: installed.Slot, TargetID: installed.TargetID})
	}
	for _, installed := range automationInstallations.Entries() {
		if _, exists := providerIDs[installed.ProviderID]; !exists {
			draft.Providers = append(draft.Providers, admission.ProviderDescriptor{
				ID: installed.ProviderID, ArtifactDigest: installed.ProviderArtifact, ABI: automationinstalled.ProviderABI, PluginInstanceID: "builtin",
				OperatingSystems: []string{runtime.GOOS}, Architectures: []string{runtime.GOARCH}, HostAPIs: []string{"3.1"},
				Capabilities: []admission.ProviderCapability{
					{Capability: automationInput, ResourceKind: automationinstalled.KindInput},
					{Capability: automationWindow, ResourceKind: automationinstalled.KindWindow},
					{Capability: automationCapture, ResourceKind: automationinstalled.KindCapture},
					{Capability: automationPlayback, ResourceKind: automationinstalled.KindPlayback},
				},
			})
			providerIDs[installed.ProviderID] = struct{}{}
		}
		draft.Targets = append(draft.Targets, admission.AutomationTarget{ID: installed.TargetID, Kind: automationinstalled.TargetKind, ProviderID: installed.ProviderID})
		draft.TargetSlots = append(draft.TargetSlots, admission.TargetSlotBinding{Slot: installed.Slot, TargetID: installed.TargetID})
	}
	return admission.SealHostProfile(draft)
}
