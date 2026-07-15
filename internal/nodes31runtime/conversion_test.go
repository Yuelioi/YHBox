package nodes31runtime_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/admission"
	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/nodecatalog"
	"github.com/yottaapp/yotta/internal/nodes31"
	"github.com/yottaapp/yotta/internal/nodes31runtime"
	"github.com/yottaapp/yotta/internal/resource"
	run31 "github.com/yottaapp/yotta/internal/run"
	"github.com/yottaapp/yotta/internal/scriptengine"
	"github.com/yottaapp/yotta/internal/stream"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
	"github.com/yottaapp/yotta/internal/workspacefs"
)

func TestInstalledAdaptersRejectCatalogThatSelfAssertsAnotherImplementation(t *testing.T) {
	builtins, err := nodes31.Build()
	if err != nil {
		t.Fatal(err)
	}
	bindings := make([]nodecatalog.Binding, 0, len(builtins.Contracts))
	for _, contract := range builtins.Contracts {
		entry, ok := builtins.Catalog.Lookup(contract.NodeRef().NodeTypeID)
		if !ok {
			t.Fatal("built-in contract is missing from its Catalog")
		}
		lock := entry.Implementation
		if contract.NodeRef().NodeTypeID == nodes31.BlobToStreamNodeID {
			lock.ArtifactDigest, err = artifact.Sum("yotta/test/forged-implementation/v1", []byte("different adapter"))
			if err != nil {
				t.Fatal(err)
			}
		}
		bindings = append(bindings, nodecatalog.Binding{Contract: contract, Implementation: lock})
	}
	forged, err := nodecatalog.Seal(builtins.Types, builtins.Capabilities, bindings, "v1")
	if err != nil {
		t.Fatal(err)
	}
	builtins.Catalog = forged
	if _, err := nodes31runtime.Installed(builtins, testDependencies()); err == nil {
		t.Fatal("installed adapters trusted implementation identity asserted by the supplied Catalog")
	}
}

func TestExecutorRunsPureProgramWithoutResourceProviders(t *testing.T) {
	ctx := context.Background()
	graphID := strings.Repeat("g", 128)
	nodeID := strings.Repeat("n", 128)
	builtins, err := nodes31.Build()
	if err != nil {
		t.Fatal(err)
	}
	build, err := artifact.Sum("yotta/test/compiler-build/v1", []byte("nodes31runtime pure test"))
	if err != nil {
		t.Fatal(err)
	}
	ref := builtins.ConcatContract.NodeRef()
	source := []byte(fmt.Sprintf(`{
		"format":"yotta.workflow","version":"3.1","workflow":{"id":"wf-pure","name":"Pure"},
		"revision":0,"entryGraph":%q,"graphs":[{"id":%q,"kind":"main","nodes":[{
			"id":%q,"nodeRef":{"nodeTypeId":%q,"semanticDigest":%q},"position":{"x":0,"y":0},
			"config":{},"bindings":{"a":{"kind":"value","value":"Yotta "},"b":{"kind":"value","value":"3.1"}}
		}],"edges":[],"inputs":[],"outputs":[]}],"variables":[],"secretRefs":[]
	}`, graphID, graphID, nodeID, ref.NodeTypeID, ref.SemanticDigest))
	compiled, err := compiler.New(build, builtins.ConfigValidators).CompileDraft(ctx, compiler.CompileRequest{SourceJSON: source, Catalog: builtins.Catalog})
	if err != nil || len(compiled.Diagnostics) != 0 {
		t.Fatalf("compile = %v, diagnostics %#v", err, compiled.Diagnostics)
	}
	program, ok := compiled.Program()
	if !ok {
		t.Fatal("compiler did not produce a Program")
	}
	now := time.Date(2026, 7, 15, 3, 0, 0, 0, time.UTC)
	_, owner, journal := admittedExecution(t, builtins, program, nil, now)
	t.Cleanup(func() { _ = owner.Close(context.Background()) })
	adapters, err := nodes31runtime.Installed(builtins, testDependencies())
	if err != nil {
		t.Fatal(err)
	}
	executor := compiler.NewExecutor(builtins.Catalog, adapters, compiler.ExecutorOptions{Now: func() time.Time { return now.Add(2 * time.Second) }})
	execution, err := executor.Run(ctx, program, owner, journal)
	if err != nil {
		t.Fatal(err)
	}
	var got string
	if err := json.Unmarshal(execution.NodeOutputs[nodeID]["result"].InlineJSON(), &got); err != nil {
		t.Fatal(err)
	}
	if got != "Yotta 3.1" {
		t.Fatalf("concat result = %q", got)
	}
	if journal.Current().Status() != run31.StatusSucceeded {
		t.Fatalf("pure Run status = %s", journal.Current().Status())
	}
	if err := owner.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	_, deniedOwner, deniedJournal := admittedExecution(t, builtins, program, nil, now)
	if err := deniedOwner.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := executor.Run(ctx, program, deniedOwner, deniedJournal); !errors.Is(err, run31.ErrGrantDenied) {
		t.Fatalf("execution after Run Owner close = %v", err)
	}
}

func TestExecutorClosesSuccessfulAttemptWhenCallerCancelsAsAdapterReturns(t *testing.T) {
	builtins, err := nodes31.Build()
	if err != nil {
		t.Fatal(err)
	}
	build, err := artifact.Sum("yotta/test/compiler-build/v1", []byte("nodes31runtime cancellation race"))
	if err != nil {
		t.Fatal(err)
	}
	ref := builtins.ConcatContract.NodeRef()
	source := []byte(fmt.Sprintf(`{
		"format":"yotta.workflow","version":"3.1","workflow":{"id":"wf-cancel-race","name":"Cancel race"},
		"revision":0,"entryGraph":"main","graphs":[{"id":"main","kind":"main","nodes":[{
			"id":"concat","nodeRef":{"nodeTypeId":%q,"semanticDigest":%q},"position":{"x":0,"y":0},
			"config":{},"bindings":{"a":{"kind":"value","value":"Yotta "},"b":{"kind":"value","value":"3.1"}}
		}],"edges":[],"inputs":[],"outputs":[]}],"variables":[],"secretRefs":[]
	}`, ref.NodeTypeID, ref.SemanticDigest))
	compiled, err := compiler.New(build, builtins.ConfigValidators).CompileDraft(context.Background(), compiler.CompileRequest{SourceJSON: source, Catalog: builtins.Catalog})
	if err != nil || len(compiled.Diagnostics) != 0 {
		t.Fatalf("compile = %v, diagnostics %#v", err, compiled.Diagnostics)
	}
	program, ok := compiled.Program()
	if !ok {
		t.Fatal("compiler did not produce a Program")
	}
	now := time.Date(2026, 7, 15, 3, 30, 0, 0, time.UTC)
	_, owner, journal := admittedExecution(t, builtins, program, nil, now)
	t.Cleanup(func() { _ = owner.Close(context.Background()) })
	adapters, err := nodes31runtime.Installed(builtins, testDependencies())
	if err != nil {
		t.Fatal(err)
	}
	entry, ok := builtins.Catalog.Lookup(nodes31.ConcatNodeID)
	if !ok {
		t.Fatal("concat Catalog entry is missing")
	}
	original := adapters[entry.Implementation.Entrypoint]
	runCtx, cancel := context.WithCancel(context.Background())
	originalRun := original.Run
	original.Run = func(ctx context.Context, invocation compiler.Invocation) (compiler.AdapterResult, error) {
		outputs, err := originalRun(ctx, invocation)
		cancel()
		return outputs, err
	}
	adapters[entry.Implementation.Entrypoint] = original
	executor := compiler.NewExecutor(builtins.Catalog, adapters, compiler.ExecutorOptions{Now: func() time.Time { return now.Add(2 * time.Second) }})
	if _, err := executor.Run(runCtx, program, owner, journal); err != nil {
		t.Fatalf("Run error = %v", err)
	}
	if journal.Current().Status() != run31.StatusSucceeded {
		t.Fatalf("Run status = %s", journal.Current().Status())
	}
	facts := journal.Current().Journal()
	if len(facts) != 2 || facts[1].AttemptOutcome != run31.AttemptSucceeded {
		t.Fatalf("journal = %#v", facts)
	}
}

func TestExecutorConvertsBlobToStreamAndBackThroughAdmittedCapabilities(t *testing.T) {
	ctx := context.Background()
	builtins, err := nodes31.Build()
	if err != nil {
		t.Fatal(err)
	}
	store, err := blob.Open(t.TempDir(), blob.Limits{MaxBlobBytes: 2 << 20, MaxTotalBytes: 4 << 20})
	if err != nil {
		t.Fatal(err)
	}
	want := bytes.Repeat([]byte("Yotta-3.1-conversion\n"), 20_000)
	inputRef, err := store.Put(ctx, "application/octet-stream", bytes.NewReader(want))
	if err != nil {
		t.Fatal(err)
	}
	build, err := artifact.Sum("yotta/test/compiler-build/v1", []byte("nodes31runtime conversion test"))
	if err != nil {
		t.Fatal(err)
	}
	compileResult, err := compiler.New(build, builtins.ConfigValidators).CompileDraft(ctx, compiler.CompileRequest{
		SourceJSON: conversionSource(builtins, inputRef), Catalog: builtins.Catalog,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(compileResult.Diagnostics) != 0 {
		t.Fatalf("compile diagnostics = %#v", compileResult.Diagnostics)
	}
	program, ok := compileResult.Program()
	if !ok {
		t.Fatal("compiler did not produce a Program")
	}

	now := time.Date(2026, 7, 15, 3, 0, 0, 0, time.UTC)
	blobProvider, err := blob.NewProvider(store, blob.ProviderLimits{MaxChunkBytes: 64 << 10, QueueCapacity: 4})
	if err != nil {
		t.Fatal(err)
	}
	streamProvider, err := stream.NewProvider(stream.Limits{MaxCapacity: 4, MaxChunkBytes: 64 << 10})
	if err != nil {
		t.Fatal(err)
	}
	_, owner, journal := admittedExecution(t, builtins, program, map[string]run31.InstalledProvider{
		blob.ProviderID:   {ArtifactDigest: blobProviderDigest(t), ABI: blob.ProviderABI, Provider: blobProvider},
		stream.ProviderID: {ArtifactDigest: streamProviderDigest(t), ABI: stream.ProviderABI, Provider: streamProvider},
	}, now)
	t.Cleanup(func() {
		if err := owner.Close(context.Background()); err != nil {
			t.Errorf("close Run Owner: %v", err)
		}
	})
	adapters, err := nodes31runtime.Installed(builtins, testDependencies())
	if err != nil {
		t.Fatal(err)
	}
	executor := compiler.NewExecutor(builtins.Catalog, adapters, compiler.ExecutorOptions{Now: func() time.Time { return now.Add(2 * time.Second) }})
	execution, err := executor.Run(ctx, program, owner, journal)
	if err != nil {
		t.Fatal(err)
	}
	facts := journal.Current().Journal()
	if len(facts) != 6 || facts[1].Kind != run31.JournalAdapterAction || facts[1].Action != "conversion.stream-opened" ||
		facts[1].ActionOutcome != run31.ActionSucceeded || facts[1].Summary.Counters["bytes"] != int64(len(want)) ||
		facts[4].Kind != run31.JournalAdapterAction || facts[4].Action != "conversion.blob-committed" {
		t.Fatalf("conversion journal = %#v", facts)
	}
	if journal.Current().Status() != run31.StatusSucceeded {
		t.Fatalf("conversion Run status = %s", journal.Current().Status())
	}
	if _, exposed := execution.NodeOutputs["to-stream"]["stream"]; exposed {
		t.Fatal("ExecutionResult exposed a reclaimed runtime authority")
	}
	output, ok := execution.NodeOutputs["to-blob"]["blob"].BlobRef()
	if !ok {
		t.Fatal("conversion did not produce a BlobRef")
	}
	if reclaimed, err := store.Sweep(nil); err != nil || reclaimed != 0 {
		t.Fatalf("Sweep reclaimed uncommitted Run output: %d, %v", reclaimed, err)
	}
	got, err := store.ReadRange(ctx, output, 0, output.Size)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("converted bytes differ: got %d bytes, want %d", len(got), len(want))
	}
	if err := owner.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if reclaimed, err := store.Sweep(nil); err != nil || reclaimed != 1 {
		t.Fatalf("Sweep after Run close = %d, %v; want released output", reclaimed, err)
	}
}

func TestExecutorFailsClosedWhenEffectAdapterJournalIsMissingOrCancelled(t *testing.T) {
	ctx := context.Background()
	builtins, err := nodes31.Build()
	if err != nil {
		t.Fatal(err)
	}
	store, err := blob.Open(t.TempDir(), blob.Limits{MaxBlobBytes: 1 << 20, MaxTotalBytes: 2 << 20})
	if err != nil {
		t.Fatal(err)
	}
	inputRef, err := store.Put(ctx, "application/octet-stream", bytes.NewReader([]byte("journal")))
	if err != nil {
		t.Fatal(err)
	}
	build, err := artifact.Sum("yotta/test/compiler-build/v1", []byte("nodes31runtime journal failures"))
	if err != nil {
		t.Fatal(err)
	}
	compiled, err := compiler.New(build, builtins.ConfigValidators).CompileDraft(ctx, compiler.CompileRequest{
		SourceJSON: conversionSource(builtins, inputRef), Catalog: builtins.Catalog,
	})
	if err != nil || len(compiled.Diagnostics) != 0 {
		t.Fatalf("compile = %v, diagnostics %#v", err, compiled.Diagnostics)
	}
	program, ok := compiled.Program()
	if !ok {
		t.Fatal("compiler did not produce a Program")
	}
	blobProvider, err := blob.NewProvider(store, blob.ProviderLimits{MaxChunkBytes: 64 << 10, QueueCapacity: 4})
	if err != nil {
		t.Fatal(err)
	}
	streamProvider, err := stream.NewProvider(stream.Limits{MaxCapacity: 4, MaxChunkBytes: 64 << 10})
	if err != nil {
		t.Fatal(err)
	}
	providers := map[string]run31.InstalledProvider{
		blob.ProviderID:   {ArtifactDigest: blobProviderDigest(t), ABI: blob.ProviderABI, Provider: blobProvider},
		stream.ProviderID: {ArtifactDigest: streamProviderDigest(t), ABI: stream.ProviderABI, Provider: streamProvider},
	}
	installed, err := nodes31runtime.Installed(builtins, testDependencies())
	if err != nil {
		t.Fatal(err)
	}
	entry, ok := builtins.Catalog.Lookup(nodes31.BlobToStreamNodeID)
	if !ok {
		t.Fatal("blob-to-stream Catalog entry is missing")
	}
	now := time.Date(2026, 7, 15, 4, 0, 0, 0, time.UTC)
	original := installed[entry.Implementation.Entrypoint].Run
	tests := []struct {
		name       string
		adapter    compiler.Adapter
		wantStatus run31.Status
		wantError  error
		wantText   string
	}{
		{name: "missing action", adapter: func(context.Context, compiler.Invocation) (compiler.AdapterResult, error) {
			return compiler.AdapterResult{}, errors.New("adapter omitted its action")
		}, wantStatus: run31.StatusFailed},
		{name: "cancelled action", adapter: func(ctx context.Context, invocation compiler.Invocation) (compiler.AdapterResult, error) {
			err := invocation.RecordAction(context.WithoutCancel(ctx), compiler.AdapterAction{
				EffectID: nodes31.BlobToStreamEffectID, Action: "conversion.test-action", Outcome: run31.ActionCancelled,
				SummaryCode: "conversion.test",
			})
			return compiler.AdapterResult{}, errors.Join(context.Canceled, err)
		}, wantStatus: run31.StatusCancelled, wantError: context.Canceled},
		{name: "duplicate action", adapter: func(ctx context.Context, invocation compiler.Invocation) (compiler.AdapterResult, error) {
			outputs, err := original(ctx, invocation)
			if err != nil {
				return compiler.AdapterResult{}, err
			}
			_ = invocation.RecordAction(context.WithoutCancel(ctx), compiler.AdapterAction{
				EffectID: nodes31.BlobToStreamEffectID, Action: "conversion.duplicate", Outcome: run31.ActionSucceeded,
				SummaryCode: "conversion.test",
			})
			return outputs, nil
		}, wantStatus: run31.StatusFailed, wantText: "more than once"},
		{name: "failed action with successful return", adapter: func(ctx context.Context, invocation compiler.Invocation) (compiler.AdapterResult, error) {
			err := invocation.RecordAction(context.WithoutCancel(ctx), compiler.AdapterAction{
				EffectID: nodes31.BlobToStreamEffectID, Action: "conversion.failed", Outcome: run31.ActionFailed,
				ErrorCode: "conversion.blob_to_stream_failed", SummaryCode: "conversion.test",
			})
			return compiler.AdapterResult{}, err
		}, wantStatus: run31.StatusFailed},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			grant, owner, journal := admittedExecution(t, builtins, program, providers, now)
			t.Cleanup(func() { _ = owner.Close(context.Background()) })
			adapters := make(map[string]compiler.InstalledAdapter, len(installed))
			for id, adapter := range installed {
				adapters[id] = adapter
			}
			adapters[entry.Implementation.Entrypoint] = compiler.InstalledAdapter{Implementation: entry.Implementation, Run: test.adapter}
			executor := compiler.NewExecutor(builtins.Catalog, adapters, compiler.ExecutorOptions{Now: func() time.Time { return now.Add(2 * time.Second) }})
			_, runErr := executor.Run(ctx, program, owner, journal)
			if runErr == nil || test.wantError != nil && !errors.Is(runErr, test.wantError) || test.wantText != "" && !strings.Contains(runErr.Error(), test.wantText) {
				t.Fatalf("Run error = %v", runErr)
			}
			if journal.Current().Status() != test.wantStatus {
				t.Fatalf("Run %s status = %s", grant.RunID(), journal.Current().Status())
			}
		})
	}
}

func admittedExecution(t *testing.T, builtins nodes31.Builtins, program compiler.ProgramSnapshot, providers map[string]run31.InstalledProvider, now time.Time) (capability.RunGrant, *run31.Owner, *run31.JournalWriter) {
	t.Helper()
	return admittedExecutionWithProfile(t, builtins, program, providers, now, executionProfile(t, builtins))
}

func admittedExecutionWithProfile(t *testing.T, builtins nodes31.Builtins, program compiler.ProgramSnapshot, providers map[string]run31.InstalledProvider, now time.Time, profileDraft admission.HostProfileDraft) (capability.RunGrant, *run31.Owner, *run31.JournalWriter) {
	t.Helper()
	return admittedExecutionWithConsent(t, builtins, program, providers, now, profileDraft, nil)
}

func admittedExecutionWithConsent(t *testing.T, builtins nodes31.Builtins, program compiler.ProgramSnapshot, providers map[string]run31.InstalledProvider, now time.Time, profileDraft admission.HostProfileDraft, consentLineage []artifact.Digest) (capability.RunGrant, *run31.Owner, *run31.JournalWriter) {
	t.Helper()
	profile, err := admission.SealHostProfile(profileDraft)
	if err != nil {
		t.Fatal(err)
	}
	store, err := run31.OpenStore(t.TempDir(), builtins.Catalog, run31.StoreOptions{MaxRecords: 1})
	if err != nil {
		t.Fatal(err)
	}
	policy := admission.PolicyFunc(func(context.Context, admission.PolicyRequest) (admission.PolicyDecision, error) {
		return admission.PolicyDecision{Outcome: admission.PolicyApproved, Generation: "policy-1", ExpiresAt: now.Add(time.Minute), ConsentLineage: consentLineage}, nil
	})
	admitter, err := admission.New(builtins.Catalog, profile, store, policy, admission.Options{Now: func() time.Time { return now }, MaxGrantTTL: 5 * time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	result, err := admitter.Admit(context.Background(), admission.Request{Program: program, Principal: "user-1"})
	if err != nil {
		t.Fatal(err)
	}
	running, err := result.Record.Start(now)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Update(context.Background(), result.Record.Digest(), running); err != nil {
		t.Fatal(err)
	}
	journal, err := store.OpenJournal(result.Grant.RunID())
	if err != nil {
		t.Fatal(err)
	}
	owner, err := run31.NewOwner(context.Background(), result.Grant, providers, resource.Options{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	return result.Grant, owner, journal
}

func executionProfile(t *testing.T, builtins nodes31.Builtins) admission.HostProfileDraft {
	t.Helper()
	capabilityRef := func(id string) capability.Ref {
		definition, ok := builtins.Catalog.LookupCapability(id)
		if !ok {
			t.Fatalf("missing capability %q", id)
		}
		return definition.Ref()
	}
	return admission.HostProfileDraft{
		OS: "windows", Architecture: "amd64", HostAPIGeneration: "3.1",
		Features: []string{scriptengine.IsolationHostFeatureID},
		Providers: []admission.ProviderDescriptor{
			{ID: blob.ProviderID, ArtifactDigest: blobProviderDigest(t), ABI: blob.ProviderABI, PluginInstanceID: "builtin",
				OperatingSystems: []string{"windows"}, Architectures: []string{"amd64"}, HostAPIs: []string{"3.1"}, Capabilities: []admission.ProviderCapability{
					{Capability: capabilityRef(nodes31.BlobReadCapabilityID), ResourceKind: blob.KindReader},
					{Capability: capabilityRef(nodes31.BlobWriteCapabilityID), ResourceKind: blob.KindWriter},
				}},
			{ID: stream.ProviderID, ArtifactDigest: streamProviderDigest(t), ABI: stream.ProviderABI, PluginInstanceID: "builtin",
				OperatingSystems: []string{"windows"}, Architectures: []string{"amd64"}, HostAPIs: []string{"3.1"}, Capabilities: []admission.ProviderCapability{
					{Capability: capabilityRef(nodes31.StreamCapabilityID), ResourceKind: stream.Kind},
				}},
			{ID: workspacefs.ProviderID, ArtifactDigest: workspaceFSProviderDigest(t), ABI: workspacefs.ProviderABI, PluginInstanceID: "builtin",
				OperatingSystems: []string{"windows"}, Architectures: []string{"amd64"}, HostAPIs: []string{"3.1"}, Capabilities: []admission.ProviderCapability{
					{Capability: capabilityRef(nodes31.FilesystemReadCapabilityID), ResourceKind: workspacefs.Kind},
				}},
		},
		Targets: []admission.AutomationTarget{
			{ID: "workspace", Kind: "blob-store", ProviderID: blob.ProviderID},
			{ID: "memory", Kind: "stream-session", ProviderID: stream.ProviderID},
			{ID: workspacefs.TargetID, Kind: workspacefs.TargetKind, ProviderID: workspacefs.ProviderID},
		},
		TargetSlots: []admission.TargetSlotBinding{{Slot: "workspace-files", TargetID: workspacefs.TargetID}},
	}
}

func blobProviderDigest(t *testing.T) artifact.Digest {
	t.Helper()
	digest, err := blob.ProviderArtifactDigest()
	if err != nil {
		t.Fatal(err)
	}
	return digest
}

func streamProviderDigest(t *testing.T) artifact.Digest {
	t.Helper()
	digest, err := stream.ProviderArtifactDigest()
	if err != nil {
		t.Fatal(err)
	}
	return digest
}

func workspaceFSProviderDigest(t *testing.T) artifact.Digest {
	t.Helper()
	digest, err := workspacefs.ProviderArtifactDigest()
	if err != nil {
		t.Fatal(err)
	}
	return digest
}

func conversionSource(builtins nodes31.Builtins, ref blob.BlobRef) []byte {
	toStream := builtins.BlobToStreamContract.NodeRef()
	toBlob := builtins.StreamToBlobContract.NodeRef()
	return []byte(fmt.Sprintf(`{
		"format":"yotta.workflow","version":"3.1","workflow":{"id":"wf-convert","name":"Convert"},
		"revision":0,"entryGraph":"main","graphs":[{"id":"main","kind":"main","nodes":[
			{"id":"to-stream","nodeRef":{"nodeTypeId":%q,"semanticDigest":%q},"position":{"x":0,"y":0},"config":{},
			 "bindings":{"blob":{"kind":"blob","blob":{"mediaType":%q,"digest":%q,"size":%d}}}},
			{"id":"to-blob","nodeRef":{"nodeTypeId":%q,"semanticDigest":%q},"position":{"x":1,"y":0},
			 "config":{"mediaType":"application/octet-stream"},"bindings":{}}
		],"edges":[{"channel":"data","from":{"nodeId":"to-stream","portId":"stream"},
		"to":{"nodeId":"to-blob","portId":"stream"}}],"inputs":[],"outputs":[]}],"variables":[],"secretRefs":[]
	}`, toStream.NodeTypeID, toStream.SemanticDigest, ref.MediaType, ref.Digest, ref.Size, toBlob.NodeTypeID, toBlob.SemanticDigest))
}
