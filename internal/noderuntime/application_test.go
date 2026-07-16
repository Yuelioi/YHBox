package noderuntime_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/admission"
	"github.com/yottaapp/yotta/internal/appcontrol"
	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/noderuntime"
	"github.com/yottaapp/yotta/internal/nodes"
	"github.com/yottaapp/yotta/internal/resource"
	run "github.com/yottaapp/yotta/internal/run"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
)

type applicationProvider struct{ operations []string }

func (p *applicationProvider) Open(_ context.Context, request resource.ProviderOpenRequest) (any, error) {
	if request.Kind != appcontrol.KindApplication || len(request.Operations) != 1 || len(request.Config) != 2 {
		return nil, fmt.Errorf("unexpected application open request: %#v", request)
	}
	p.operations = append(p.operations, "open:"+request.Operations[0])
	return request.Operations[0], nil
}
func (p *applicationProvider) Invoke(_ context.Context, object any, operation string, payload []byte) ([]byte, error) {
	if object != operation || string(payload) != `{}` {
		return nil, fmt.Errorf("unexpected application invocation")
	}
	p.operations = append(p.operations, "invoke:"+operation)
	if operation == appcontrol.OperationLaunch {
		return artifact.Marshal(appcontrol.LaunchResponse{ProcessID: 4242})
	}
	return artifact.Marshal(appcontrol.TerminateResponse{TerminatedCount: 2})
}
func (*applicationProvider) Close(context.Context, any) error { return nil }

func TestApplicationLifecycleUsesInstalledTargetAndRedactsJournal(t *testing.T) {
	builtins, err := nodes.Build()
	if err != nil {
		t.Fatal(err)
	}
	providerDigest, err := artifact.Sum("yotta/test/application-provider/v1", []byte("exact-installed-app"))
	if err != nil {
		t.Fatal(err)
	}
	provider := &applicationProvider{}
	const providerID, targetID, slot = "application-test", "installed-application/test", "application-test"
	capabilityDefinition, ok := builtins.Catalog.LookupCapability(nodes.ApplicationLifecycleCapabilityID)
	if !ok {
		t.Fatal("application lifecycle capability is missing")
	}
	profileDraft := executionProfile(t, builtins)
	profileDraft.Providers = append(profileDraft.Providers, admission.ProviderDescriptor{
		ID: providerID, ArtifactDigest: providerDigest, ABI: appcontrol.ProviderABI, PluginInstanceID: "builtin",
		OperatingSystems: []string{"windows"}, Architectures: []string{"amd64"}, HostAPIs: []string{"3.1"},
		Capabilities: []admission.ProviderCapability{{Capability: capabilityDefinition.Ref(), ResourceKind: appcontrol.KindApplication}},
	})
	profileDraft.Targets = append(profileDraft.Targets, admission.AutomationTarget{ID: targetID, Kind: appcontrol.TargetKind, ProviderID: providerID})
	profileDraft.TargetSlots = append(profileDraft.TargetSlots, admission.TargetSlotBinding{Slot: slot, TargetID: targetID})
	program := compilePrimitiveProgram(t, builtins, applicationSource(t, builtins, slot))
	now := time.Date(2026, 7, 16, 18, 0, 0, 0, time.UTC)
	consent, err := artifact.Sum("yotta/test/application-consent/v1", []byte(slot))
	if err != nil {
		t.Fatal(err)
	}
	_, owner, journal := admittedExecutionWithConsent(t, builtins, program, map[string]run.InstalledProvider{
		providerID: {ArtifactDigest: providerDigest, ABI: appcontrol.ProviderABI, Provider: provider},
	}, now, profileDraft, []artifact.Digest{consent})
	t.Cleanup(func() { _ = owner.Close(context.Background()) })
	adapters, err := noderuntime.Installed(builtins, testDependencies())
	if err != nil {
		t.Fatal(err)
	}
	result, err := compiler.NewExecutor(builtins.Catalog, adapters, compiler.ExecutorOptions{Now: func() time.Time { return now }}).Run(context.Background(), program, owner, journal)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(result.NodeOutputs["stop"]["terminated-count"].InlineJSON()); got != "2" {
		t.Fatalf("terminated-count = %s", got)
	}
	if fmt.Sprint(provider.operations) != "[open:launch invoke:launch open:terminate invoke:terminate]" {
		t.Fatalf("operations = %v", provider.operations)
	}
	actions := 0
	for _, entry := range journal.Current().Journal() {
		if entry.Kind != run.JournalAdapterAction {
			continue
		}
		actions++
		raw, _ := json.Marshal(entry)
		if bytes.Contains(raw, []byte("C:\\Program Files")) || bytes.Contains(raw, []byte("--secret")) || bytes.Contains(raw, []byte("4242")) {
			t.Fatalf("journal leaked application identity: %s", raw)
		}
	}
	if actions != 2 {
		t.Fatalf("application actions = %d", actions)
	}
}

func applicationSource(t *testing.T, builtins nodes.Builtins, slot string) []byte {
	t.Helper()
	started, _ := builtins.Definition(nodes.RunStartedNodeID)
	launch, _ := builtins.Definition(nodes.LaunchApplicationNodeID)
	terminate, _ := builtins.Definition(nodes.TerminateApplicationNodeID)
	return []byte(fmt.Sprintf(`{
		"format":"yotta.workflow","version":"3.1","workflow":{"id":"wf-application","name":"Application"},"revision":0,"entryGraph":"main",
		"graphs":[{"id":"main","kind":"main","nodes":[
			{"id":"start","nodeRef":{"nodeTypeId":%q,"version":"1.0.0","semanticDigest":%q},"position":{"x":0,"y":0},"config":{},"bindings":{}},
			{"id":"launch","nodeRef":{"nodeTypeId":%q,"version":"1.0.0","semanticDigest":%q},"position":{"x":1,"y":0},"config":{"slot":%q},"bindings":{}},
			{"id":"stop","nodeRef":{"nodeTypeId":%q,"version":"1.0.0","semanticDigest":%q},"position":{"x":2,"y":0},"config":{"slot":%q},"bindings":{}}
		],"edges":[
			{"channel":"exec","from":{"nodeId":"start","portId":"started"},"to":{"nodeId":"launch","portId":"in"}},
			{"channel":"exec","from":{"nodeId":"launch","portId":"completed"},"to":{"nodeId":"stop","portId":"in"}}
		],"inputs":[],"outputs":[]}],"variables":[],"secretRefs":[]
	}`, started.Contract.NodeRef().NodeTypeID, started.Contract.NodeRef().SemanticDigest,
		launch.Contract.NodeRef().NodeTypeID, launch.Contract.NodeRef().SemanticDigest, slot,
		terminate.Contract.NodeRef().NodeTypeID, terminate.Contract.NodeRef().SemanticDigest, slot))
}
