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
	app31 "github.com/yottaapp/yotta/internal/application"
	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/nodes31"
	"github.com/yottaapp/yotta/internal/nodes31runtime"
	"github.com/yottaapp/yotta/internal/resource"
	run31 "github.com/yottaapp/yotta/internal/run"
	"github.com/yottaapp/yotta/internal/stream"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
	"github.com/yottaapp/yotta/internal/workflowstore"
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
	Policy            admission.Policy
	GrantTTL          time.Duration
	OwnerCloseTimeout time.Duration
	Now               func() time.Time
	OnRunEvent        func(app31.RunEvent)
}

type Runtime struct {
	Application *app31.Application
	Builtins    nodes31.Builtins
	BlobStore   *blob.Store
}

func Build(config Config) (*Runtime, error) {
	if config.Now == nil {
		config.Now = time.Now
	}
	if config.Policy == nil || config.GrantTTL <= 0 || config.GrantTTL > 24*time.Hour || config.OwnerCloseTimeout <= 0 {
		return nil, errors.New("app bootstrap requires policy and bounded Run lifetimes")
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
	build, err := compiler.BuildDigest()
	if err != nil {
		return nil, fmt.Errorf("build compiler identity: %w", err)
	}
	workspace := filepath.Join(root, "workspace-3.1")
	sources, err := workflowstore.OpenSourceStore(filepath.Join(workspace, "workflows"), workflowstore.SourceStoreOptions{MaxSources: config.Limits.MaxSources})
	if err != nil {
		return nil, err
	}
	programs, err := workflowstore.OpenProgramStore(filepath.Join(workspace, "programs"), builtins.Catalog, build, workflowstore.ProgramStoreOptions{MaxPrograms: config.Limits.MaxPrograms})
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
	blobDigest, err := blob.ProviderArtifactDigest()
	if err != nil {
		return nil, err
	}
	streamDigest, err := stream.ProviderArtifactDigest()
	if err != nil {
		return nil, err
	}
	profile, err := builtinHostProfile(builtins, blobDigest, streamDigest)
	if err != nil {
		return nil, err
	}
	admitter, err := admission.New(builtins.Catalog, profile, runs, config.Policy, admission.Options{
		Now: config.Now, MaxGrantTTL: config.GrantTTL,
	})
	if err != nil {
		return nil, err
	}
	adapters, err := nodes31runtime.Installed(builtins)
	if err != nil {
		return nil, err
	}
	executor := compiler.NewExecutor(builtins.Catalog, adapters, compiler.ExecutorOptions{Now: config.Now})
	application, err := app31.New(app31.Config{
		Catalog: builtins.Catalog, CompilerBuild: build, Sources: sources, Programs: programs, Runs: runs,
		Admitter: admitter, Executor: executor,
		Providers: map[string]run31.InstalledProvider{
			blob.ProviderID:   {ArtifactDigest: blobDigest, ABI: blob.ProviderABI, Provider: blobProvider},
			stream.ProviderID: {ArtifactDigest: streamDigest, ABI: stream.ProviderABI, Provider: streamProvider},
		},
		ResourceOptions: resource.Options{
			Now: config.Now, MaxPayloadBytes: config.Limits.MaxResourcePayloadBytes,
		},
		OwnerCloseTimeout: config.OwnerCloseTimeout, Now: config.Now, OnRunEvent: config.OnRunEvent,
	})
	if err != nil {
		return nil, err
	}
	return &Runtime{Application: application, Builtins: builtins, BlobStore: blobStore}, nil
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
	return r.Application.Close(ctx)
}

func validateLimits(limits Limits) error {
	if limits.MaxSources <= 0 || limits.MaxPrograms <= 0 || limits.MaxRuns <= 0 ||
		limits.MaxBlobBytes <= 0 || limits.MaxTotalBlobBytes < limits.MaxBlobBytes ||
		limits.MaxResourcePayloadBytes <= 0 || limits.BlobChunkBytes <= 0 || limits.BlobChunkBytes > limits.MaxResourcePayloadBytes ||
		limits.BlobQueueCapacity <= 0 || limits.StreamCapacity <= 0 || limits.StreamChunkBytes <= 0 ||
		limits.StreamChunkBytes > limits.MaxResourcePayloadBytes {
		return errors.New("app bootstrap limits are invalid")
	}
	return nil
}

func builtinHostProfile(builtins nodes31.Builtins, blobDigest, streamDigest artifact.Digest) (admission.HostProfile, error) {
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
	return admission.SealHostProfile(admission.HostProfileDraft{
		OS: runtime.GOOS, Architecture: runtime.GOARCH, HostAPIGeneration: "3.1",
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
		},
		Targets: []admission.AutomationTarget{
			{ID: "workspace", Kind: "blob-store", ProviderID: blob.ProviderID},
			{ID: "memory", Kind: "stream-session", ProviderID: stream.ProviderID},
		},
	})
}
