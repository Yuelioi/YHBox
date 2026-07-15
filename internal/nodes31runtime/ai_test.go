package nodes31runtime_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/admission"
	"github.com/yottaapp/yotta/internal/ai"
	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/nodes31"
	"github.com/yottaapp/yotta/internal/nodes31runtime"
	"github.com/yottaapp/yotta/internal/resource"
	run31 "github.com/yottaapp/yotta/internal/run"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
)

func TestExecutorRunsAIGenerateThroughInstallationSlotAndJournalsProviderFacts(t *testing.T) {
	builtins, err := nodes31.Build()
	if err != nil {
		t.Fatal(err)
	}
	started, ok := builtins.Definition(nodes31.RunStartedNodeID)
	if !ok {
		t.Fatal("RunStarted definition is missing")
	}
	generate := builtins.AIGenerateContract.NodeRef()
	source := []byte(fmt.Sprintf(`{
		"format":"yotta.workflow","version":"3.1","workflow":{"id":"wf-ai","name":"AI"},
		"revision":0,"entryGraph":"main","graphs":[{"id":"main","kind":"main","nodes":[
			{"id":"start","nodeRef":{"nodeTypeId":%q,"semanticDigest":%q},"position":{"x":0,"y":0},"config":{},"bindings":{}},
			{"id":"generate","nodeRef":{"nodeTypeId":%q,"semanticDigest":%q},"position":{"x":1,"y":0},
			 "config":{"slot":"default","instructions":"Be concise","maxOutputTokens":128},
			 "bindings":{"prompt":{"kind":"value","value":"hello"}}}
		],"edges":[{"channel":"exec","from":{"nodeId":"start","portId":"started"},"to":{"nodeId":"generate","portId":"in"}}],
		"inputs":[],"outputs":[]}],"variables":[],"secretRefs":[]
	}`, started.Contract.NodeRef().NodeTypeID, started.Contract.NodeRef().SemanticDigest, generate.NodeTypeID, generate.SemanticDigest))
	compiled, err := compiler.New(aiTestDigest(t, "compiler"), builtins.ConfigValidators).CompileDraft(context.Background(), compiler.CompileRequest{SourceJSON: source, Catalog: builtins.Catalog})
	if err != nil || len(compiled.Diagnostics) != 0 {
		t.Fatalf("compile = %v, diagnostics %#v", err, compiled.Diagnostics)
	}
	program, ok := compiled.Program()
	if !ok {
		t.Fatal("compiler did not produce an AI Program")
	}
	entries := program.CapabilityPlan().Entries()
	if len(entries) != 1 || entries[0].Requirement.TargetSlot != "default" || entries[0].Requirement.CredentialSlot != "default" {
		t.Fatalf("effective AI requirement = %#v", entries)
	}

	providerDigest := aiTestDigest(t, "provider")
	capabilityDefinition, _ := builtins.Catalog.LookupCapability(nodes31.AIGenerationCapabilityID)
	profileDraft := executionProfile(t, builtins)
	profileDraft.Providers = []admission.ProviderDescriptor{{
		ID: "ai-test", ArtifactDigest: providerDigest, ABI: ai.ProviderABI, PluginInstanceID: "builtin",
		OperatingSystems: []string{"windows"}, Architectures: []string{"amd64"}, HostAPIs: []string{"3.1"},
		Capabilities: []admission.ProviderCapability{{Capability: capabilityDefinition.Ref(), ResourceKind: ai.KindModelSession}},
	}}
	profileDraft.Targets = profileDraft.Targets[:1]
	profileDraft.Targets[0].ID, profileDraft.Targets[0].Kind, profileDraft.Targets[0].ProviderID = "model-test", "ai-model", "ai-test"
	profileDraft.Credentials = []admission.CredentialBinding{{ID: "credential-test", ProviderID: "ai-test", Capability: capabilityDefinition.Ref()}}
	profileDraft.TargetSlots = []admission.TargetSlotBinding{{Slot: "default", TargetID: "model-test"}}
	profileDraft.CredentialSlots = []admission.CredentialSlotBinding{{Slot: "default", CredentialID: "credential-test"}}
	profile, err := admission.SealHostProfile(profileDraft)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 7, 16, 1, 0, 0, 0, time.UTC)
	store, err := run31.OpenStore(t.TempDir(), builtins.Catalog, run31.StoreOptions{MaxRecords: 1})
	if err != nil {
		t.Fatal(err)
	}
	policy := admission.PolicyFunc(func(context.Context, admission.PolicyRequest) (admission.PolicyDecision, error) {
		return admission.PolicyDecision{
			Outcome: admission.PolicyApproved, Generation: "policy-ai", ExpiresAt: now.Add(time.Minute),
			ConsentLineage: []artifact.Digest{aiTestDigest(t, "consent")},
		}, nil
	})
	admitter, err := admission.New(builtins.Catalog, profile, store, policy, admission.Options{Now: func() time.Time { return now }, MaxGrantTTL: 5 * time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	admitted, err := admitter.Admit(context.Background(), admission.Request{Program: program, Principal: "user-1"})
	if err != nil {
		t.Fatal(err)
	}
	running, err := admitted.Record.Start(now)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Update(context.Background(), admitted.Record.Digest(), running); err != nil {
		t.Fatal(err)
	}
	journal, err := store.OpenJournal(admitted.Grant.RunID())
	if err != nil {
		t.Fatal(err)
	}
	modelProvider := &aiModelProvider{}
	owner, err := run31.NewOwner(context.Background(), admitted.Grant, map[string]run31.InstalledProvider{
		"ai-test": {ArtifactDigest: providerDigest, ABI: ai.ProviderABI, Provider: modelProvider},
	}, resource.Options{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = owner.Close(context.Background()) })
	adapters, err := nodes31runtime.Installed(builtins)
	if err != nil {
		t.Fatal(err)
	}
	executor := compiler.NewExecutor(builtins.Catalog, adapters, compiler.ExecutorOptions{Now: func() time.Time { return now.Add(time.Second) }})
	result, err := executor.Run(context.Background(), program, owner, journal)
	if err != nil {
		t.Fatal(err)
	}
	if value := result.NodeOutputs["generate"]["result"].InlineJSON(); string(value) != `"generated"` || modelProvider.open.CredentialBindingID != "credential-test" {
		t.Fatalf("AI result = %s, open = %#v", value, modelProvider.open)
	}
	var facts map[string]string
	for _, entry := range journal.Current().Journal() {
		if entry.Kind == run31.JournalAdapterAction && entry.NodeID == "generate" {
			facts = entry.Summary.Facts
		}
	}
	if facts["provider_request_id"] != "request-1" || facts["provider_response_id"] != "response-1" || facts["finish"] != "completed" {
		t.Fatalf("AI journal facts = %#v", facts)
	}
}

func TestCompilerRejectsAIExtractSchemaOutsidePinnedStrictProfile(t *testing.T) {
	builtins, err := nodes31.Build()
	if err != nil {
		t.Fatal(err)
	}
	started, ok := builtins.Definition(nodes31.RunStartedNodeID)
	if !ok {
		t.Fatal("RunStarted definition is missing")
	}
	extract := builtins.AIExtractContract.NodeRef()
	invalidSchema := `{"type":"object","properties":{},"required":[],"additionalProperties":true}`
	source := []byte(fmt.Sprintf(`{
		"format":"yotta.workflow","version":"3.1","workflow":{"id":"wf-ai-extract","name":"AI Extract"},
		"revision":0,"entryGraph":"main","graphs":[{"id":"main","kind":"main","nodes":[
			{"id":"start","nodeRef":{"nodeTypeId":%q,"semanticDigest":%q},"position":{"x":0,"y":0},"config":{},"bindings":{}},
			{"id":"extract","nodeRef":{"nodeTypeId":%q,"semanticDigest":%q},"position":{"x":1,"y":0},
			 "config":{"slot":"default","schema":%q},"bindings":{"prompt":{"kind":"value","value":"hello"}}}
		],"edges":[{"channel":"exec","from":{"nodeId":"start","portId":"started"},"to":{"nodeId":"extract","portId":"in"}}],
		"inputs":[],"outputs":[]}],"variables":[],"secretRefs":[]
	}`, started.Contract.NodeRef().NodeTypeID, started.Contract.NodeRef().SemanticDigest,
		extract.NodeTypeID, extract.SemanticDigest, invalidSchema))

	compiled, err := compiler.New(aiTestDigest(t, "compiler"), builtins.ConfigValidators).CompileDraft(context.Background(), compiler.CompileRequest{
		SourceJSON: source, Catalog: builtins.Catalog,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := compiled.Program(); ok {
		t.Fatal("compiler produced a Program for a schema outside the pinned strict profile")
	}
	for _, diagnostic := range compiled.Diagnostics {
		if diagnostic.Code == compiler.CodeInvalidConfig && diagnostic.NodeID == "extract" {
			return
		}
	}
	t.Fatalf("missing INVALID_CONFIG diagnostic: %#v", compiled.Diagnostics)
}

type aiModelProvider struct{ open resource.ProviderOpenRequest }

func (p *aiModelProvider) Open(_ context.Context, request resource.ProviderOpenRequest) (any, error) {
	p.open = request
	return struct{}{}, nil
}

func (p *aiModelProvider) Invoke(_ context.Context, _ any, operation string, _ []byte) ([]byte, error) {
	if operation != ai.OperationGenerate {
		return nil, fmt.Errorf("unexpected operation %q", operation)
	}
	input, output := int64(7), int64(2)
	return artifact.Marshal(ai.Outcome{
		Provider: ai.ProviderOpenAIResponses, RequestedModel: "model-1", ResolvedModel: "model-1",
		ProviderRequestID: "request-1", ProviderResponseID: "response-1",
		Items:  []ai.OutputItem{{Kind: ai.OutputText, Text: &ai.TextOutput{Text: "generated"}}},
		Finish: ai.Finish{Kind: ai.FinishCompleted}, Usage: ai.TokenUsage{InputTotal: &input, OutputTotal: &output},
	})
}

func (*aiModelProvider) Close(context.Context, any) error { return nil }

func aiTestDigest(t *testing.T, label string) artifact.Digest {
	t.Helper()
	digest, err := artifact.Sum("yotta/test/ai-runtime/v1", []byte(label))
	if err != nil {
		t.Fatal(err)
	}
	return digest
}
