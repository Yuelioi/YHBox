package appbootstrap_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/admission"
	"github.com/yottaapp/yotta/internal/ai"
	"github.com/yottaapp/yotta/internal/appbootstrap"
	app31 "github.com/yottaapp/yotta/internal/application"
	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/nodes31"
	run31 "github.com/yottaapp/yotta/internal/run"
	"github.com/yottaapp/yotta/internal/scriptengine"
	"github.com/yottaapp/yotta/internal/services/workflow31"
	"github.com/yottaapp/yotta/internal/workflow/authoring"
	"github.com/yottaapp/yotta/internal/workflow/schema"
)

func TestBuildComposesWorkflowServiceThroughProductionProgramChain(t *testing.T) {
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	events := make(chan app31.RunEvent, 16)
	runtime, err := appbootstrap.Build(appbootstrap.Config{
		DataRoot: t.TempDir(), Limits: testLimits(), AIInstallations: emptyAIInstallations(t), ScriptRuntime: bootstrapScriptRuntime(t), GrantTTL: 5 * time.Minute,
		OwnerCloseTimeout: time.Second, Now: func() time.Time { return now },
		OnRunEvent: func(event app31.RunEvent) { events <- event },
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := runtime.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := runtime.Close(ctx); err != nil {
			t.Errorf("Close = %v", err)
		}
	})
	service, err := workflow31.NewService(runtime.Application)
	if err != nil {
		t.Fatal(err)
	}
	createdSource, err := service.CreateSource("Bootstrap")
	if err != nil {
		t.Fatal(err)
	}
	patched, err := service.ApplyPatch(createdSource.WorkflowID, createdSource.Revision, []authoring.Command{
		{Kind: authoring.CommandAddNode, AddNode: &authoring.AddNodeCommand{
			GraphID: "main", NodeTypeID: nodes31.ConcatNodeID, Handle: "concat", Position: schema.Position{},
		}},
		{Kind: authoring.CommandBindValue, BindValue: &authoring.BindValueCommand{GraphID: "main", NodeID: "$concat", PortID: "a", Value: "a"}},
		{Kind: authoring.CommandBindValue, BindValue: &authoring.BindValueCommand{GraphID: "main", NodeID: "$concat", PortID: "b", Value: "b"}},
	})
	if err != nil || patched.Source.SourceJSON == "" || patched.Source.Revision != 1 || len(patched.GeneratedNodes) != 1 {
		t.Fatalf("ApplyPatch = %#v, %v", patched, err)
	}
	saved := patched.Source
	compiled, err := service.CompileSource(saved.WorkflowID)
	if err != nil || !compiled.ProgramHash.Valid() || len(compiled.Diagnostics) != 0 || saved.SourceHash != compiled.SourceHash {
		t.Fatalf("CompileSource = %#v, %v", compiled, err)
	}
	listed, err := service.ListSources()
	if err != nil || len(listed) != 1 || listed[0].Name != "Bootstrap" || listed[0].SourceJSON != "" || listed[0].SourceHash != saved.SourceHash {
		t.Fatalf("ListSources = %#v, %v", listed, err)
	}
	if authoring := service.GetAuthoringProjection(); !strings.Contains(authoring, `"format":"yotta.node-authoring-projection"`) {
		t.Fatalf("GetAuthoringProjection = %s", authoring)
	}
	created, err := service.CreateSource(" Empty ")
	if err != nil || created.Name != "Empty" || created.Revision != 0 || !strings.Contains(created.SourceJSON, `"version":"3.1"`) {
		t.Fatalf("CreateSource = %#v, %v", created, err)
	}
	started, err := service.StartRun(saved.WorkflowID)
	if err != nil || started.Run == nil || started.Run.Status != string(run31.StatusQueued) || started.ProgramHash != compiled.ProgramHash {
		t.Fatalf("StartRun = %#v, %v", started, err)
	}
	deadline := time.After(5 * time.Second)
	for {
		select {
		case event := <-events:
			if event.RunID == started.Run.RunID && event.Status == run31.StatusSucceeded {
				timeline, err := service.GetRunTimeline(event.RunID)
				if err != nil || timeline.Status != string(run31.StatusSucceeded) || len(timeline.Timeline) != 2 || timeline.Failure != nil {
					t.Fatalf("GetRunTimeline = %#v, %v", timeline, err)
				}
				if catalog := service.GetCatalog(); !strings.Contains(catalog, `"version":"3.1"`) {
					t.Fatalf("GetCatalog = %s", catalog)
				}
				return
			}
		case <-deadline:
			t.Fatal("production Run did not succeed")
		}
	}
}

func TestBuiltinPolicyRejectsUninstalledProviderIdentity(t *testing.T) {
	now := time.Date(2026, 7, 15, 12, 30, 0, 0, time.UTC)
	policy, err := appbootstrap.NewBuiltinPolicy(func() time.Time { return now }, time.Minute, emptyAIInstallations(t))
	if err != nil {
		t.Fatal(err)
	}
	decision, err := policy.Authorize(context.Background(), admission.PolicyRequest{Bindings: []capability.Binding{{
		ProviderID: "third-party", ProviderArtifactDigest: testDigest(t, "forged"), ProviderABI: "https://example.test/abi/v1",
		TargetID: "remote", TargetKind: "remote", PluginInstanceID: "plugin",
	}}})
	if err != nil || decision.Outcome != admission.PolicyDenied {
		t.Fatalf("Authorize = %#v, %v", decision, err)
	}
}

func TestBuiltinPolicyRequiresExactAIInstallationConsent(t *testing.T) {
	now := time.Date(2026, 7, 15, 13, 0, 0, 0, time.UTC)
	profileDraft := ai.ModelProfileDraft{
		Provider: ai.ProviderOpenAIResponses, Model: "gpt-test", MaxOutputTokens: 4096,
		Capabilities: ai.ProfileCapabilities{StructuredOutput: true}, Evaluation: ai.EvaluationUnverified,
	}
	withoutConsent, err := ai.Install([]ai.InstallationDraft{{Slot: "primary", Profile: profileDraft}}, testAICredentials{})
	if err != nil {
		t.Fatal(err)
	}
	entry := withoutConsent.Entries()[0]
	binding := capability.Binding{
		ProviderID: entry.ProviderID, ProviderArtifactDigest: entry.ProviderArtifact, ProviderABI: ai.ProviderABI,
		TargetID: entry.TargetID, TargetKind: "ai-model", ResourceKind: ai.KindModelSession,
		PluginInstanceID: "builtin", CredentialBindingID: entry.CredentialBindingID,
	}
	policy, err := appbootstrap.NewBuiltinPolicy(func() time.Time { return now }, time.Minute, withoutConsent)
	if err != nil {
		t.Fatal(err)
	}
	decision, err := policy.Authorize(context.Background(), admission.PolicyRequest{Bindings: []capability.Binding{binding}})
	if err != nil || decision.Outcome != admission.PolicyConsentRequired {
		t.Fatalf("unconsented decision = %#v, %v", decision, err)
	}
	profile, err := ai.SealModelProfile(profileDraft)
	if err != nil {
		t.Fatal(err)
	}
	consent, err := ai.WorkflowConsentDigest("primary", profile)
	if err != nil {
		t.Fatal(err)
	}
	installed, err := ai.Install([]ai.InstallationDraft{{Slot: "primary", Profile: profileDraft, Consent: consent}}, testAICredentials{})
	if err != nil {
		t.Fatal(err)
	}
	entry = installed.Entries()[0]
	binding.ProviderID, binding.ProviderArtifactDigest = entry.ProviderID, entry.ProviderArtifact
	binding.TargetID, binding.CredentialBindingID = entry.TargetID, entry.CredentialBindingID
	policy, err = appbootstrap.NewBuiltinPolicy(func() time.Time { return now }, time.Minute, installed)
	if err != nil {
		t.Fatal(err)
	}
	decision, err = policy.Authorize(context.Background(), admission.PolicyRequest{Bindings: []capability.Binding{binding}})
	if err != nil || decision.Outcome != admission.PolicyApproved || len(decision.ConsentLineage) != 1 || decision.ConsentLineage[0] != consent {
		t.Fatalf("consented decision = %#v, %v", decision, err)
	}
	binding.CredentialBindingID = "ai-credential/forged"
	decision, err = policy.Authorize(context.Background(), admission.PolicyRequest{Bindings: []capability.Binding{binding}})
	if err != nil || decision.Outcome != admission.PolicyDenied {
		t.Fatalf("forged decision = %#v, %v", decision, err)
	}
}

type testAICredentials struct{}

func (testAICredentials) Get(string) (string, error) { return "secret", nil }

func bootstrapScriptRuntime(t *testing.T) *scriptengine.Runtime {
	t.Helper()
	runtime, err := scriptengine.NewRuntime(scriptengine.RuntimeOptions{
		Executable:         filepath.Join(t.TempDir(), scriptengine.WorkerExecutableName),
		ProcessMemoryBytes: scriptengine.DefaultMemoryBytes, JobMemoryBytes: scriptengine.DefaultMemoryBytes,
	})
	if err != nil {
		t.Fatal(err)
	}
	return runtime
}

func emptyAIInstallations(t *testing.T) ai.Installations {
	t.Helper()
	installations, err := ai.Install(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	return installations
}

func testLimits() appbootstrap.Limits {
	return appbootstrap.Limits{
		MaxSources: 8, MaxPrograms: 8, MaxRuns: 8,
		MaxBlobBytes: 1 << 20, MaxTotalBlobBytes: 8 << 20, MaxResourcePayloadBytes: 1 << 20,
		BlobChunkBytes: 64 << 10, BlobQueueCapacity: 2, StreamCapacity: 4, StreamChunkBytes: 64 << 10,
	}
}

func testDigest(t *testing.T, label string) artifact.Digest {
	t.Helper()
	digest, err := artifact.Sum("yotta/test/appbootstrap/v1", []byte(label))
	if err != nil {
		t.Fatal(err)
	}
	return digest
}
