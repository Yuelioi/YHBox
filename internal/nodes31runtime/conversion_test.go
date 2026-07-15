package nodes31runtime_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/nodecatalog"
	"github.com/yottaapp/yotta/internal/nodes31"
	"github.com/yottaapp/yotta/internal/nodes31runtime"
	"github.com/yottaapp/yotta/internal/resource"
	run31 "github.com/yottaapp/yotta/internal/run"
	"github.com/yottaapp/yotta/internal/runid"
	"github.com/yottaapp/yotta/internal/stream"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
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
	capabilities := make([]capability.Definition, 0, 3)
	for _, id := range []string{nodes31.BlobReadCapabilityID, nodes31.BlobWriteCapabilityID, nodes31.StreamCapabilityID} {
		definition, ok := builtins.Catalog.LookupCapability(id)
		if !ok {
			t.Fatalf("missing capability %q", id)
		}
		capabilities = append(capabilities, definition)
	}
	forged, err := nodecatalog.Seal(builtins.Types, capabilities, bindings, "v1")
	if err != nil {
		t.Fatal(err)
	}
	builtins.Catalog = forged
	if _, err := nodes31runtime.Installed(builtins); err == nil {
		t.Fatal("installed adapters trusted implementation identity asserted by the supplied Catalog")
	}
}

func TestExecutorRunsPureProgramWithoutResourceProviders(t *testing.T) {
	ctx := context.Background()
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
		"revision":0,"entryGraph":"main","graphs":[{"id":"main","kind":"main","nodes":[{
			"id":"concat","nodeRef":{"nodeTypeId":%q,"semanticDigest":%q},"position":{"x":0,"y":0},
			"config":{},"bindings":{"a":{"kind":"value","value":"Yotta "},"b":{"kind":"value","value":"3.1"}}
		}],"edges":[],"inputs":[],"outputs":[]}],"variables":[],"secretRefs":[]
	}`, ref.NodeTypeID, ref.SemanticDigest))
	compiled, err := compiler.New(build).CompileDraft(ctx, compiler.CompileRequest{SourceJSON: source, Catalog: builtins.Catalog})
	if err != nil || len(compiled.Diagnostics) != 0 {
		t.Fatalf("compile = %v, diagnostics %#v", err, compiled.Diagnostics)
	}
	program, ok := compiled.Program()
	if !ok {
		t.Fatal("compiler did not produce a Program")
	}
	now := time.Date(2026, 7, 15, 3, 0, 0, 0, time.UTC)
	runID, err := runid.New()
	if err != nil {
		t.Fatal(err)
	}
	grant, err := capability.SealRunGrant(capability.GrantRequest{
		ProgramHash: program.Hash(), Plan: program.CapabilityPlan(), RunID: runID,
		Principal: "user-1", PolicyGeneration: "policy-1", IssuedAt: now, ExpiresAt: now.Add(time.Minute),
		Bindings: []capability.Binding{},
	}, builtins.Catalog)
	if err != nil {
		t.Fatal(err)
	}
	owner, err := run31.NewOwner(ctx, grant, nil, resource.Options{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = owner.Close(context.Background()) })
	adapters, err := nodes31runtime.Installed(builtins)
	if err != nil {
		t.Fatal(err)
	}
	execution, err := compiler.NewExecutor(builtins.Catalog, adapters).Run(ctx, program, owner)
	if err != nil {
		t.Fatal(err)
	}
	var got string
	if err := json.Unmarshal(execution.NodeOutputs["concat"]["result"].InlineJSON(), &got); err != nil {
		t.Fatal(err)
	}
	if got != "Yotta 3.1" {
		t.Fatalf("concat result = %q", got)
	}
	if err := owner.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := compiler.NewExecutor(builtins.Catalog, adapters).Run(ctx, program, owner); !errors.Is(err, run31.ErrGrantDenied) {
		t.Fatalf("execution after Run Owner close = %v", err)
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
	compileResult, err := compiler.New(build).CompileDraft(ctx, compiler.CompileRequest{
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
	runID, err := runid.New()
	if err != nil {
		t.Fatal(err)
	}
	bindings := exactBindings(t, program.CapabilityPlan())
	grant, err := capability.SealRunGrant(capability.GrantRequest{
		ProgramHash: program.Hash(), Plan: program.CapabilityPlan(), RunID: runID,
		Principal: "user-1", PolicyGeneration: "policy-1", IssuedAt: now, ExpiresAt: now.Add(time.Minute),
		Bindings: bindings,
	}, builtins.Catalog)
	if err != nil {
		t.Fatal(err)
	}
	blobProvider, err := blob.NewProvider(store, blob.ProviderLimits{MaxChunkBytes: 64 << 10, QueueCapacity: 4})
	if err != nil {
		t.Fatal(err)
	}
	streamProvider, err := stream.NewProvider(stream.Limits{MaxCapacity: 4, MaxChunkBytes: 64 << 10})
	if err != nil {
		t.Fatal(err)
	}
	owner, err := run31.NewOwner(ctx, grant, map[string]resource.Provider{
		blob.ProviderID: blobProvider, stream.ProviderID: streamProvider,
	}, resource.Options{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := owner.Close(context.Background()); err != nil {
			t.Errorf("close Run Owner: %v", err)
		}
	})
	adapters, err := nodes31runtime.Installed(builtins)
	if err != nil {
		t.Fatal(err)
	}
	execution, err := compiler.NewExecutor(builtins.Catalog, adapters).Run(ctx, program, owner)
	if err != nil {
		t.Fatal(err)
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

func exactBindings(t *testing.T, plan capability.Plan) []capability.Binding {
	t.Helper()
	entries := plan.Entries()
	bindings := make([]capability.Binding, 0, len(entries))
	for _, entry := range entries {
		binding := capability.Binding{
			GraphID: entry.GraphID, NodeID: entry.NodeID, RequirementID: entry.Requirement.ID,
			PluginInstanceID: "builtin", SessionID: "conversion-1",
		}
		switch entry.Requirement.ID {
		case "blob-read":
			binding.ProviderID = blob.ProviderID
			binding.TargetID = "workspace"
			binding.TargetKind = "blob-store"
			binding.ResourceKind = blob.KindReader
		case "blob-write":
			binding.ProviderID = blob.ProviderID
			binding.TargetID = "workspace"
			binding.TargetKind = "blob-store"
			binding.ResourceKind = blob.KindWriter
		case "stream":
			binding.ProviderID = stream.ProviderID
			binding.TargetID = "memory"
			binding.TargetKind = "stream-session"
			binding.ResourceKind = stream.Kind
		default:
			t.Fatalf("unexpected conversion requirement %q", entry.Requirement.ID)
		}
		bindings = append(bindings, binding)
	}
	return bindings
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
