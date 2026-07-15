// Package appbootstrap constructs the production Yotta 3.1 application with
// immutable contracts, durable stores, installed providers, admission, and a
// single Program worker. It contains no legacy Container runtime branch.
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
	app31 "github.com/yottaapp/yotta/internal/application"
	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/httpegress"
	"github.com/yottaapp/yotta/internal/nodeauthoring"
	"github.com/yottaapp/yotta/internal/nodes31"
	"github.com/yottaapp/yotta/internal/nodes31runtime"
	"github.com/yottaapp/yotta/internal/resource"
	run31 "github.com/yottaapp/yotta/internal/run"
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
	MaxBlobBytes            int64
	MaxTotalBlobBytes       int64
	MaxResourcePayloadBytes int
	BlobChunkBytes          int
	BlobQueueCapacity       int
	StreamCapacity          int
	StreamChunkBytes        int
}

type Config struct {
	DataRoot          string
	Limits            Limits
	AIInstallations   ai.Installations
	HTTPInstallations httpegress.Installations
	ScriptRuntime     *scriptengine.Runtime
	LogEmitter        nodes31runtime.LogEmitter
	GrantTTL          time.Duration
	OwnerCloseTimeout time.Duration
	Now               func() time.Time
	OnRunEvent        func(app31.RunEvent)
}

type Runtime struct {
	Application *app31.Application
	Builtins    nodes31.Builtins
	BlobStore   *blob.Store
	ai          ai.Installations
	http        httpegress.Installations
}

func Build(config Config) (*Runtime, error) {
	if config.Now == nil {
		config.Now = time.Now
	}
	if !config.AIInstallations.Valid() || !config.HTTPInstallations.Valid() || config.ScriptRuntime == nil || config.LogEmitter == nil || config.GrantTTL <= 0 || config.GrantTTL > 24*time.Hour || config.OwnerCloseTimeout <= 0 {
		return nil, errors.New("app bootstrap requires trusted installations, isolated effect runtimes, and bounded Run lifetimes")
	}
	if err := validateLimits(config.Limits); err != nil {
		return nil, err
	}
	root, err := filepath.Abs(config.DataRoot)
	if err != nil || config.DataRoot == "" {
		return nil, errors.New("app bootstrap requires a data root")
	}
	builtins, err := nodes31.Build()
	if err != nil {
		return nil, fmt.Errorf("build Catalog 3.1: %w", err)
	}
	authoringProjection, err := nodeauthoring.Project(nodeauthoring.Input{
		Catalog: builtins.Catalog, Types: builtins.Types, Capabilities: builtins.Capabilities,
		Contracts: builtins.Contracts, GeneratorVersion: nodes31.GeneratorVersion,
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
	programs, err := workflowstore.OpenProgramStore(filepath.Join(workspace, "programs"), builtins.Catalog, builtins.ConfigValidators, build, workflowstore.ProgramStoreOptions{MaxPrograms: config.Limits.MaxPrograms})
	if err != nil {
		return nil, err
	}
	runs, err := run31.OpenStore(filepath.Join(workspace, "runs"), builtins.Catalog, run31.StoreOptions{MaxRecords: config.Limits.MaxRuns})
	if err != nil {
		return nil, err
	}
	blobStore, err := blob.Open(filepath.Join(workspace, "blobs"), blob.Limits{
		MaxBlobBytes: config.Limits.MaxBlobBytes, MaxTotalBytes: config.Limits.MaxTotalBlobBytes,
	})
	if err != nil {
		return nil, err
	}
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
	workspaceFileProvider, err := workspacefs.NewProvider(filepath.Join(workspace, "files"), workspacefs.Limits{MaxReadBytes: nodes31.DefaultFileReadBytes})
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
	profile, err := builtinHostProfile(builtins, blobDigest, streamDigest, workspaceFileDigest, config.ScriptRuntime, config.AIInstallations, config.HTTPInstallations)
	if err != nil {
		return nil, err
	}
	policy, err := NewBuiltinPolicy(config.Now, config.GrantTTL, config.AIInstallations, config.HTTPInstallations)
	if err != nil {
		return nil, err
	}
	admitter, err := admission.New(builtins.Catalog, profile, runs, policy, admission.Options{
		Now: config.Now, MaxGrantTTL: config.GrantTTL,
	})
	if err != nil {
		return nil, err
	}
	adapters, err := nodes31runtime.Installed(builtins, nodes31runtime.Dependencies{Script: config.ScriptRuntime, Log: config.LogEmitter})
	if err != nil {
		return nil, err
	}
	executor := compiler.NewExecutor(builtins.Catalog, adapters, compiler.ExecutorOptions{Now: config.Now})
	providers := map[string]run31.InstalledProvider{
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
		providers[installed.ProviderID] = run31.InstalledProvider{
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
		providers[installed.ProviderID] = run31.InstalledProvider{ArtifactDigest: installed.ProviderArtifact, ABI: httpegress.ProviderABI, Provider: installed.Provider}
	}
	application, err := app31.New(app31.Config{
		Catalog: builtins.Catalog, Authoring: authoringProjection, CompilerBuild: build, ConfigValidators: builtins.ConfigValidators,
		Sources: sources, Programs: programs, Runs: runs,
		Admitter: admitter, Executor: executor,
		Providers: providers,
		ResourceOptions: resource.Options{
			Now: config.Now, MaxPayloadBytes: config.Limits.MaxResourcePayloadBytes,
		},
		OwnerCloseTimeout: config.OwnerCloseTimeout, Now: config.Now, OnRunEvent: config.OnRunEvent,
	})
	if err != nil {
		return nil, err
	}
	return &Runtime{Application: application, Builtins: builtins, BlobStore: blobStore, ai: config.AIInstallations, http: config.HTTPInstallations}, nil
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
	err := r.Application.Close(ctx)
	r.ai.CloseIdleConnections()
	r.http.CloseIdleConnections()
	return err
}

func validateLimits(limits Limits) error {
	if limits.MaxSources <= 0 || limits.MaxPrograms <= 0 || limits.MaxRuns <= 0 ||
		limits.MaxBlobBytes <= 0 || limits.MaxTotalBlobBytes < limits.MaxBlobBytes ||
		limits.MaxResourcePayloadBytes < 2*nodes31.DefaultFileReadBytes || limits.BlobChunkBytes <= 0 || limits.BlobChunkBytes > limits.MaxResourcePayloadBytes ||
		limits.BlobQueueCapacity <= 0 || limits.StreamCapacity <= 0 || limits.StreamChunkBytes <= 0 ||
		limits.StreamChunkBytes > limits.MaxResourcePayloadBytes {
		return errors.New("app bootstrap limits are invalid")
	}
	return nil
}

func builtinHostProfile(builtins nodes31.Builtins, blobDigest, streamDigest, workspaceFileDigest artifact.Digest, scriptRuntime *scriptengine.Runtime, aiInstallations ai.Installations, httpInstallations httpegress.Installations) (admission.HostProfile, error) {
	lookup := func(id string) (capability.Ref, error) {
		definition, ok := builtins.Catalog.LookupCapability(id)
		if !ok {
			return capability.Ref{}, fmt.Errorf("built-in capability %q is missing", id)
		}
		return definition.Ref(), nil
	}
	blobRead, err := lookup(nodes31.BlobReadCapabilityID)
	if err != nil {
		return admission.HostProfile{}, err
	}
	blobWrite, err := lookup(nodes31.BlobWriteCapabilityID)
	if err != nil {
		return admission.HostProfile{}, err
	}
	streamSession, err := lookup(nodes31.StreamCapabilityID)
	if err != nil {
		return admission.HostProfile{}, err
	}
	aiGeneration, err := lookup(nodes31.AIGenerationCapabilityID)
	if err != nil {
		return admission.HostProfile{}, err
	}
	filesystemRead, err := lookup(nodes31.FilesystemReadCapabilityID)
	if err != nil {
		return admission.HostProfile{}, err
	}
	httpGet, err := lookup(nodes31.HTTPGetCapabilityID)
	if err != nil {
		return admission.HostProfile{}, err
	}
	draft := admission.HostProfileDraft{
		OS: runtime.GOOS, Architecture: runtime.GOARCH, HostAPIGeneration: "3.1",
		Features: scriptRuntime.HostFeatures(),
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
	return admission.SealHostProfile(draft)
}
