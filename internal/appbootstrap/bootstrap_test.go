package appbootstrap_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/admission"
	"github.com/yottaapp/yotta/internal/ai"
	"github.com/yottaapp/yotta/internal/appbootstrap"
	"github.com/yottaapp/yotta/internal/appcontrol"
	appcore "github.com/yottaapp/yotta/internal/application"
	"github.com/yottaapp/yotta/internal/artifact"
	automationinstalled "github.com/yottaapp/yotta/internal/automation/installed"
	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/httpegress"
	"github.com/yottaapp/yotta/internal/noderuntime"
	"github.com/yottaapp/yotta/internal/nodes"
	run "github.com/yottaapp/yotta/internal/run"
	"github.com/yottaapp/yotta/internal/scriptengine"
	"github.com/yottaapp/yotta/internal/services/workflow"
	"github.com/yottaapp/yotta/internal/storage"
	"github.com/yottaapp/yotta/internal/storage/catalog"
	"github.com/yottaapp/yotta/internal/workflow/authoring"
	"github.com/yottaapp/yotta/internal/workflow/schema"
	"github.com/yottaapp/yotta/internal/workflowinstallation"
	"github.com/yottaapp/yotta/internal/workspacefs"
)

func TestBuildComposesWorkflowServiceThroughProductionProgramChain(t *testing.T) {
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	events := make(chan appcore.RunEvent, 16)
	stores := newTestWorkflowStorage(t)
	runtime, err := appbootstrap.Build(appbootstrap.Config{
		DataRoot: stores.roots.Data, ProgramCacheRoot: filepath.Join(stores.roots.Cache, "programs"),
		WorkflowRepository: stores.foundation.Workflows(), InstallationRepository: stores.foundation.WorkflowInstallations(),
		RunRepository: stores.foundation.Runs(),
		BlobStore:     stores.blobs,
		Limits:        testLimits(), AIInstallations: emptyAIInstallations(t), HTTPInstallations: emptyHTTPInstallations(t), ApplicationInstallations: emptyApplicationInstallations(t), AutomationInstallations: emptyAutomationInstallations(t), ScriptRuntime: bootstrapScriptRuntime(t), GrantTTL: 5 * time.Minute,
		LogEmitter:        discardWorkflowLog{},
		OwnerCloseTimeout: time.Second, Now: func() time.Time { return now },
		OnRunEvent: func(event appcore.RunEvent) { events <- event },
	})
	if err != nil {
		t.Fatal(err)
	}
	if runtime.Installations == nil {
		t.Fatal("Build() did not compose the Workflow Installation module")
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
	service, err := workflow.NewService(runtime.Application, workflow.WithInstallationRuntime(runtime))
	if err != nil {
		t.Fatal(err)
	}
	createdSource, err := service.CreateSource("Bootstrap")
	if err != nil {
		t.Fatal(err)
	}
	patched, err := service.ApplyPatch(createdSource.WorkflowID, createdSource.Revision, []authoring.Command{
		{Kind: authoring.CommandAddNode, AddNode: &authoring.AddNodeCommand{
			GraphID: "main", NodeTypeID: nodes.ConcatNodeID, Handle: "concat", Position: schema.Position{},
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
	if record, found, err := stores.foundation.Workflows().Get(context.Background(), saved.WorkflowID); err != nil ||
		!found || record.Revision != saved.Revision || record.Hash != saved.SourceHash {
		t.Fatalf("Catalog Workflow Source = %#v, found=%v, %v", record, found, err)
	}
	if _, err := os.Stat(filepath.Join(stores.roots.Cache, "programs")); err != nil {
		t.Fatalf("Program cache RootSet projection: %v", err)
	}
	for _, retired := range []string{
		filepath.Join(stores.roots.Data, "workspace", "workflows"),
		filepath.Join(stores.roots.Data, "workspace", "programs"),
	} {
		if _, err := os.Stat(retired); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("retired workspace store was created at %s: %v", retired, err)
		}
	}
	listed, err := service.ListSources()
	if err != nil || len(listed) != 1 || listed[0].Name != "Bootstrap" || listed[0].SourceJSON != "" || listed[0].SourceHash != saved.SourceHash {
		t.Fatalf("ListSources = %#v, %v", listed, err)
	}
	if authoring := service.GetAuthoringProjection(); !strings.Contains(authoring, `"format":"yotta.node-authoring-projection"`) {
		t.Fatalf("GetAuthoringProjection = %s", authoring)
	}
	created, err := service.CreateSource(" Empty ")
	if err != nil || created.Name != "Empty" || created.Revision != 0 || !strings.Contains(created.SourceJSON, `"version":"1"`) {
		t.Fatalf("CreateSource = %#v, %v", created, err)
	}
	started, err := service.StartRun(saved.WorkflowID)
	if err != nil || started.Run == nil || started.Run.Status != string(run.StatusQueued) || started.ProgramHash != compiled.ProgramHash {
		t.Fatalf("StartRun = %#v, %v", started, err)
	}
	deadline := time.After(5 * time.Second)
	for {
		select {
		case event := <-events:
			if event.RunID == started.Run.RunID && event.Status == run.StatusSucceeded {
				timeline, err := service.GetRunTimeline(event.RunID)
				if err != nil || timeline.Status != string(run.StatusSucceeded) || len(timeline.Timeline) != 4 || timeline.Failure != nil {
					t.Fatalf("GetRunTimeline = %#v, %v", timeline, err)
				}
				if catalog := service.GetCatalog(); !strings.Contains(catalog, `"version":"1"`) {
					t.Fatalf("GetCatalog = %s", catalog)
				}
				return
			}
		case <-deadline:
			t.Fatal("production Run did not succeed")
		}
	}
}

func TestRuntimeStartsOnlyReadyWorkflowInstallationThroughSharedApplication(t *testing.T) {
	now := time.Date(2026, 7, 26, 6, 0, 0, 0, time.UTC)
	stores := newTestWorkflowStorage(t)
	runtime, err := appbootstrap.Build(appbootstrap.Config{
		DataRoot: stores.roots.Data, ProgramCacheRoot: filepath.Join(stores.roots.Cache, "programs"),
		WorkflowRepository: stores.foundation.Workflows(), InstallationRepository: stores.foundation.WorkflowInstallations(),
		RunRepository: stores.foundation.Runs(), BlobStore: stores.blobs,
		Limits: testLimits(), AIInstallations: emptyAIInstallations(t), HTTPInstallations: emptyHTTPInstallations(t),
		ApplicationInstallations: emptyApplicationInstallations(t), AutomationInstallations: emptyAutomationInstallations(t),
		ScriptRuntime: bootstrapScriptRuntime(t), LogEmitter: discardWorkflowLog{},
		GrantTTL: 5 * time.Minute, OwnerCloseTimeout: time.Second, Now: func() time.Time { return now },
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
	service, err := workflow.NewService(runtime.Application, workflow.WithInstallationRuntime(runtime))
	if err != nil {
		t.Fatal(err)
	}
	source, err := service.CreateSource("Installed release")
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := runtime.Application.GetSource(source.WorkflowID)
	if err != nil {
		t.Fatal(err)
	}
	release, err := workflowinstallation.NewVerifiedRelease(snapshot.Artifact(), workflowinstallation.VerificationReceipt{
		ReleaseDigest:      testDigest(t, "workflow-release"),
		AttestationDigest:  testDigest(t, "publisher-attestation"),
		PublisherNamespace: "https://example.test/publishers/acme", ReleaseVersion: "1.0.0", VerifiedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	installation, err := runtime.Installations.InstallVerified(context.Background(), release, "")
	if err != nil {
		t.Fatal(err)
	}
	listed, err := service.ListInstallations()
	if err != nil || len(listed) != 1 || listed[0].InstallationID != installation.ID {
		t.Fatalf("ListInstallations() = %#v, %v", listed, err)
	}
	derived, err := service.DeriveInstallationSource(installation.ID, "Editable release copy")
	if err != nil || derived.WorkflowID == release.WorkflowID || derived.Name != "Editable release copy" ||
		derived.Revision != 0 {
		t.Fatalf("DeriveInstallationSource() = %#v, %v", derived, err)
	}
	derivedSnapshot, err := runtime.Application.GetSource(derived.WorkflowID)
	if err != nil {
		t.Fatal(err)
	}
	derivedDocument, diagnostics := schema.ParseSource(derivedSnapshot.Artifact())
	if len(diagnostics) != 0 || derivedDocument.DerivedFrom == nil ||
		derivedDocument.DerivedFrom.ReleaseDigest != release.ID ||
		derivedDocument.DerivedFrom.SourceHash != release.SourceHash ||
		derivedDocument.DerivedFrom.AttestationDigest != release.AttestationDigest {
		t.Fatalf("derived Source = %#v, diagnostics = %#v", derivedDocument.DerivedFrom, diagnostics)
	}
	afterDerive, err := runtime.Installations.Get(context.Background(), installation.ID)
	if err != nil || afterDerive.ReleaseID != release.ID {
		t.Fatalf("Installation after derivation = %#v, %v", afterDerive, err)
	}
	readiness, err := service.GetInstallationReadiness(installation.ID)
	if err != nil || readiness.RunAllowed || readiness.ScheduleAllowed || len(readiness.Blockers) != 2 {
		t.Fatalf("GetInstallationReadiness() = %#v, %v", readiness, err)
	}
	if _, err := service.StartInstallationRun(installation.ID); err == nil {
		t.Fatal("run accepted Installation without manual execution consent")
	}
	afterConsent, err := service.GrantInstallationConsent(installation.ID, string(workflowinstallation.ScopeRun))
	if err != nil || !afterConsent.RunAllowed || afterConsent.ScheduleAllowed {
		t.Fatalf("GrantInstallationConsent() = %#v, %v", afterConsent, err)
	}
	if _, err := service.GrantInstallationConsent(installation.ID, "invalid"); err == nil {
		t.Fatal("GrantInstallationConsent accepted invalid scope")
	}
	started, err := service.StartInstallationRun(installation.ID)
	if err != nil || started.Run == nil || started.SourceHash != release.SourceHash {
		t.Fatalf("StartInstallationRun() = %#v, %v", started, err)
	}
	candidateSource, diagnostics := schema.ParseSource(release.SourceArtifact)
	if schema.HasErrors(diagnostics) {
		t.Fatalf("candidate Source diagnostics = %#v", diagnostics)
	}
	candidateSource.Workflow.Name = "Installed release v2"
	candidateArtifact, err := artifact.Marshal(candidateSource)
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := workflowinstallation.NewVerifiedRelease(
		candidateArtifact,
		workflowinstallation.VerificationReceipt{
			ReleaseDigest: testDigest(t, "workflow-release-v2"), AttestationDigest: testDigest(t, "publisher-attestation-v2"),
			PublisherNamespace: release.PublisherNamespace, ReleaseVersion: "2.0.0", VerifiedAt: now.Add(time.Minute),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	pausedInstallationID := ""
	if err := runtime.SetWorkflowInstallationSchedulePauser(func(installationID string) ([]string, error) {
		pausedInstallationID = installationID
		return []string{"schedule-a"}, nil
	}); err != nil {
		t.Fatal(err)
	}
	preparedUpdate, err := runtime.PrepareWorkflowInstallationUpdate(
		context.Background(), installation.ID, candidate,
	)
	if err != nil || !preparedUpdate.Valid() || len(preparedUpdate.Conflicts()) != 0 {
		t.Fatalf("PrepareWorkflowInstallationUpdate() = %#v, %v", preparedUpdate, err)
	}
	updatedInstallation, err := runtime.ApplyWorkflowInstallationUpdate(context.Background(), preparedUpdate)
	if err != nil || updatedInstallation.ReleaseID != candidate.ID ||
		updatedInstallation.PreviousReleaseID != release.ID ||
		pausedInstallationID != installation.ID {
		t.Fatalf("ApplyWorkflowInstallationUpdate() = %#v, %v", updatedInstallation, err)
	}
	afterUpdate, err := service.GetInstallationReadiness(installation.ID)
	if err != nil || afterUpdate.ReleaseID != candidate.ID || afterUpdate.RunAllowed ||
		afterUpdate.ScheduleAllowed || len(afterUpdate.Blockers) != 2 {
		t.Fatalf("readiness after update = %#v, %v", afterUpdate, err)
	}
	if _, err := service.StartInstallationRun(installation.ID); err == nil {
		t.Fatal("updated Installation retained execution consent from the previous Release")
	}
	if _, err := runtime.StartInstallationRun(
		context.Background(), installation.ID, workflowinstallation.ScopeSchedule,
	); err == nil {
		t.Fatal("schedule accepted Installation without schedule execution consent")
	}
}

func TestBuildStartsWithOneCorruptWorkflowSourceIsolatedAndRepairable(t *testing.T) {
	stores := newTestWorkflowStorage(t)
	build := func() *appbootstrap.Runtime {
		runtime, err := appbootstrap.Build(appbootstrap.Config{
			DataRoot: stores.roots.Data, ProgramCacheRoot: filepath.Join(stores.roots.Cache, "programs"),
			WorkflowRepository: stores.foundation.Workflows(), InstallationRepository: stores.foundation.WorkflowInstallations(),
			RunRepository: stores.foundation.Runs(),
			BlobStore:     stores.blobs, Limits: testLimits(),
			AIInstallations: emptyAIInstallations(t), HTTPInstallations: emptyHTTPInstallations(t),
			ApplicationInstallations: emptyApplicationInstallations(t), AutomationInstallations: emptyAutomationInstallations(t),
			ScriptRuntime: bootstrapScriptRuntime(t), LogEmitter: discardWorkflowLog{},
			GrantTTL: 5 * time.Minute, OwnerCloseTimeout: time.Second, Now: time.Now,
		})
		if err != nil {
			t.Fatalf("Build = %v", err)
		}
		if err := runtime.Start(context.Background()); err != nil {
			t.Fatalf("Start = %v", err)
		}
		return runtime
	}
	closeRuntime := func(runtime *appbootstrap.Runtime) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := runtime.Close(ctx); err != nil {
			t.Fatalf("Close = %v", err)
		}
	}

	first := build()
	service, err := workflow.NewService(first.Application)
	if err != nil {
		t.Fatal(err)
	}
	created, err := service.CreateSource("Recoverable")
	if err != nil {
		t.Fatal(err)
	}
	closeRuntime(first)
	if _, err := stores.foundation.Workflows().Delete(
		context.Background(), created.WorkflowID, created.Revision, created.SourceHash,
	); err != nil {
		t.Fatal(err)
	}
	corrupt := []byte(`{"format":"yotta.workflow","version":"1",`)
	recoveryID, err := artifact.Sum("yotta/test/workflow-quarantine/v1", corrupt)
	if err != nil {
		t.Fatal(err)
	}
	if err := stores.foundation.Workflows().PutQuarantine(context.Background(), catalog.WorkflowQuarantineRecord{
		ID: recoveryID, OriginalName: created.WorkflowID + ".json",
		Reason: "synthetic invalid JSON", Artifact: corrupt, CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}

	second := build()
	t.Cleanup(func() { closeRuntime(second) })
	recoveryService, err := workflow.NewService(second.Application)
	if err != nil {
		t.Fatal(err)
	}
	if listed, err := recoveryService.ListSources(); err != nil || len(listed) != 0 {
		t.Fatalf("healthy Sources after isolation = %#v, %v", listed, err)
	}
	recoveries := recoveryService.ListSourceRecoveries()
	if len(recoveries) != 1 || recoveries[0].OriginalName != created.WorkflowID+".json" {
		t.Fatalf("recoveries = %#v", recoveries)
	}
	repaired, err := recoveryService.RepairSourceRecovery(recoveries[0].RecoveryID, created.SourceJSON)
	if err != nil || repaired.WorkflowID != created.WorkflowID || len(recoveryService.ListSourceRecoveries()) != 0 {
		t.Fatalf("RepairSourceRecovery = %#v, %v", repaired, err)
	}
}

func TestRuntimeHotReplacesApplicationAutomationAndAuthoringGeneration(t *testing.T) {
	if !automationinstalled.PlatformSupported() {
		t.Skip("installed automation targets are intentionally unavailable")
	}
	stores := newTestWorkflowStorage(t)
	runtime, err := appbootstrap.Build(appbootstrap.Config{
		DataRoot: stores.roots.Data, ProgramCacheRoot: filepath.Join(stores.roots.Cache, "programs"),
		WorkflowRepository: stores.foundation.Workflows(), InstallationRepository: stores.foundation.WorkflowInstallations(),
		RunRepository: stores.foundation.Runs(),
		BlobStore:     stores.blobs, Limits: testLimits(),
		AIInstallations: emptyAIInstallations(t), HTTPInstallations: emptyHTTPInstallations(t),
		ApplicationInstallations: emptyApplicationInstallations(t), AutomationInstallations: emptyAutomationInstallations(t),
		ScriptRuntime: bootstrapScriptRuntime(t), GrantTTL: 5 * time.Minute, LogEmitter: discardWorkflowLog{},
		OwnerCloseTimeout: time.Second, Now: time.Now,
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
	path := filepath.Join(t.TempDir(), "editor.exe")
	if err := os.WriteFile(path, []byte("live-automation-target"), 0o700); err != nil {
		t.Fatal(err)
	}
	inspection, err := appcontrol.InspectExecutable(path)
	if err != nil {
		t.Fatal(err)
	}
	applicationDraft := appcontrol.ProfileDraft{Executable: inspection.Executable, Arguments: []string{}}
	automationProfile := automationinstalled.NewDesktopProfileDraft(automationinstalled.DesktopProfilePayload{
		Application: applicationDraft, WindowTitle: "Editor.*", WindowTitleMatch: "regex", WindowSelection: "topmost",
		WindowClass: "EditorWindow", InputBackend: "postmessage", CaptureBackend: "gdi", ResolveTimeoutMilliseconds: 500,
	})
	sealedAutomation, err := automationinstalled.SealProfile(automationProfile)
	if err != nil {
		t.Fatal(err)
	}
	automationConsent, err := automationinstalled.WorkflowConsentDigest("editor-window", sealedAutomation)
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := runtime.PrepareAutomation([]appcontrol.InstallationDraft{{Slot: "editor", Profile: applicationDraft}}, []automationinstalled.InstallationDraft{{
		Slot: "editor-window", Profile: automationProfile, Consent: automationConsent,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if err := prepared.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := prepared.Commit(); err == nil {
		t.Fatal("prepared automation generation committed twice")
	}
	aborted, err := runtime.PrepareAutomation(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	aborted.Abort()
	if err := aborted.Commit(); err == nil {
		t.Fatal("aborted automation generation committed")
	}
	if backend, err := runtime.AuthoringTargets().CaptureBackend("editor-window"); err != nil || backend != "gdi" {
		t.Fatalf("live authoring target = %q, %v", backend, err)
	}
	service, err := workflow.NewService(runtime.Application)
	if err != nil {
		t.Fatal(err)
	}
	source, err := service.CreateSource("Live target admission")
	if err != nil {
		t.Fatal(err)
	}
	patched, err := service.ApplyPatch(source.WorkflowID, source.Revision, []authoring.Command{
		{Kind: authoring.CommandAddNode, AddNode: &authoring.AddNodeCommand{GraphID: "main", NodeTypeID: nodes.PressKeysNodeID, Handle: "keys", Position: schema.Position{X: 400, Y: 160}}},
		{Kind: authoring.CommandSetConfig, SetConfig: &authoring.SetConfigCommand{GraphID: "main", NodeID: "$keys", FieldID: "slot", Value: "editor-window"}},
		{Kind: authoring.CommandBindValue, BindValue: &authoring.BindValueCommand{GraphID: "main", NodeID: "$keys", PortID: "keys", Value: []string{"F9"}}},
		{Kind: authoring.CommandConnect, Connect: &authoring.EdgeCommand{GraphID: "main", Edge: schema.Edge{Channel: schema.EdgeExec, From: schema.Endpoint{NodeID: "run-started", PortID: "started"}, To: schema.Endpoint{NodeID: "$keys", PortID: "in"}}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	started, err := service.StartRun(patched.Source.WorkflowID)
	if err != nil || started.Run == nil || started.Run.RunID == "" {
		t.Fatalf("same-process target admission = %#v, %v", started, err)
	}
	if err := runtime.ReplaceAutomation(emptyApplicationInstallations(t), emptyAutomationInstallations(t)); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.AuthoringTargets().CaptureBackend("editor-window"); err == nil {
		t.Fatal("removed target remained visible through the live authoring handle")
	}
	rollback, err := runtime.PrepareAutomation([]appcontrol.InstallationDraft{{Slot: "editor", Profile: applicationDraft}}, []automationinstalled.InstallationDraft{{
		Slot: "editor-window", Profile: automationProfile, Consent: automationConsent,
	}})
	if err != nil {
		t.Fatal(err)
	}
	closeCtx, cancelClose := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelClose()
	if err := runtime.Application.Close(closeCtx); err != nil {
		t.Fatal(err)
	}
	if err := rollback.Commit(); err == nil {
		t.Fatal("published automation generation after Application closed")
	}
	if _, err := runtime.AuthoringTargets().CaptureBackend("editor-window"); err == nil {
		t.Fatal("failed publication changed the authoring generation")
	}
}

func TestCompositionRootDoesNotReinferAutomationCapabilities(t *testing.T) {
	source, err := os.ReadFile("bootstrap.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	for _, forbidden := range []string{
		"switch resourceKind",
		"automationinstalled.OperationPressKeys",
		"nodes.AutomationKeyInputCapabilityID",
		"nodes.AutomationDesktopInputCapabilityID",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("composition root contains a second automation capability fact source: %s", forbidden)
		}
	}
	if !strings.Contains(text, "manifest.Capabilities") {
		t.Fatal("composition root does not project the sealed automation manifest")
	}
}

func TestBuiltinPolicyRejectsUninstalledProviderIdentity(t *testing.T) {
	now := time.Date(2026, 7, 15, 12, 30, 0, 0, time.UTC)
	policy, err := appbootstrap.NewBuiltinPolicy(func() time.Time { return now }, time.Minute, emptyAIInstallations(t), emptyHTTPInstallations(t), emptyApplicationInstallations(t), emptyAutomationInstallations(t))
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

func TestBuiltinPolicyPinsWorkspaceFilesystemProvider(t *testing.T) {
	now := time.Date(2026, 7, 16, 11, 0, 0, 0, time.UTC)
	policy, err := appbootstrap.NewBuiltinPolicy(func() time.Time { return now }, time.Minute, emptyAIInstallations(t), emptyHTTPInstallations(t), emptyApplicationInstallations(t), emptyAutomationInstallations(t))
	if err != nil {
		t.Fatal(err)
	}
	digest, err := workspacefs.ProviderArtifactDigest()
	if err != nil {
		t.Fatal(err)
	}
	binding := capability.Binding{
		ProviderID: workspacefs.ProviderID, ProviderArtifactDigest: digest, ProviderABI: workspacefs.ProviderABI,
		TargetID: workspacefs.TargetID, TargetKind: workspacefs.TargetKind, ResourceKind: workspacefs.Kind,
		PluginInstanceID: "builtin",
	}
	decision, err := policy.Authorize(context.Background(), admission.PolicyRequest{Bindings: []capability.Binding{binding}})
	if err != nil || decision.Outcome != admission.PolicyApproved {
		t.Fatalf("workspace filesystem decision = %#v, %v", decision, err)
	}
	binding.TargetID = "host-root"
	decision, err = policy.Authorize(context.Background(), admission.PolicyRequest{Bindings: []capability.Binding{binding}})
	if err != nil || decision.Outcome != admission.PolicyDenied {
		t.Fatalf("forged workspace filesystem decision = %#v, %v", decision, err)
	}
}

func TestBuiltinPolicyRequiresExactAIInstallationConsent(t *testing.T) {
	now := time.Date(2026, 7, 15, 13, 0, 0, 0, time.UTC)
	profileDraft := ai.ModelProfileDraft{
		Provider: ai.ProviderOpenAIResponses, Model: "gpt-test", MaxOutputTokens: 4096,
		Capabilities: ai.ProfileCapabilities{StructuredOutput: true},
	}
	profileDraft, evaluation := approvedAIProfile(t, profileDraft)
	withoutConsent, err := ai.Install([]ai.InstallationDraft{{Slot: "primary", Profile: profileDraft, Evaluation: evaluation}}, testAICredentials{})
	if err != nil {
		t.Fatal(err)
	}
	entry := withoutConsent.Entries()[0]
	binding := capability.Binding{
		ProviderID: entry.ProviderID, ProviderArtifactDigest: entry.ProviderArtifact, ProviderABI: ai.ProviderABI,
		TargetID: entry.TargetID, TargetKind: "ai-model", ResourceKind: ai.KindModelSession,
		PluginInstanceID: "builtin", CredentialBindingID: entry.CredentialBindingID,
	}
	policy, err := appbootstrap.NewBuiltinPolicy(func() time.Time { return now }, time.Minute, withoutConsent, emptyHTTPInstallations(t), emptyApplicationInstallations(t), emptyAutomationInstallations(t))
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
	installed, err := ai.Install([]ai.InstallationDraft{{Slot: "primary", Profile: profileDraft, Evaluation: evaluation, Consent: consent}}, testAICredentials{})
	if err != nil {
		t.Fatal(err)
	}
	entry = installed.Entries()[0]
	binding.ProviderID, binding.ProviderArtifactDigest = entry.ProviderID, entry.ProviderArtifact
	binding.TargetID, binding.CredentialBindingID = entry.TargetID, entry.CredentialBindingID
	policy, err = appbootstrap.NewBuiltinPolicy(func() time.Time { return now }, time.Minute, installed, emptyHTTPInstallations(t), emptyApplicationInstallations(t), emptyAutomationInstallations(t))
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

func TestAICredentialStatesExposeOnlyNonSecretAvailability(t *testing.T) {
	profileDraft, evaluation := approvedAIProfile(t, ai.ModelProfileDraft{
		Provider: ai.ProviderOpenAIResponses, Model: "gpt-test", MaxOutputTokens: 4096,
		Capabilities: ai.ProfileCapabilities{StructuredOutput: true},
	})
	installations, err := ai.Install([]ai.InstallationDraft{
		{Slot: "missing", Profile: profileDraft, Evaluation: evaluation},
		{Slot: "primary", Profile: profileDraft, Evaluation: evaluation},
	}, testAICredentials{})
	if err != nil {
		t.Fatal(err)
	}
	states, err := appbootstrap.AICredentialStates(
		context.Background(), installations,
		testCredentialAvailability{"primary": true},
	)
	if err != nil || len(states) != 2 ||
		states[0].CredentialBindingID != ai.CredentialBindingID("missing") ||
		states[0].Available ||
		states[1].CredentialBindingID != ai.CredentialBindingID("primary") ||
		states[1].Kind != ai.CredentialKindAPIKey || !states[1].Available {
		t.Fatalf("AICredentialStates() = %#v, %v", states, err)
	}
}

func TestBuiltinPolicyRequiresExactHTTPInstallationConsent(t *testing.T) {
	now := time.Date(2026, 7, 16, 17, 0, 0, 0, time.UTC)
	profileDraft := httpegress.ProfileDraft{Origin: "https://example.com", ResponseByteLimit: 4096, TimeoutMilliseconds: 5000}
	withoutConsent, err := httpegress.Install([]httpegress.InstallationDraft{{Slot: "http", Profile: profileDraft}})
	if err != nil {
		t.Fatal(err)
	}
	entry := withoutConsent.Entries()[0]
	binding := capability.Binding{ProviderID: entry.ProviderID, ProviderArtifactDigest: entry.ProviderArtifact, ProviderABI: httpegress.ProviderABI, TargetID: entry.TargetID, TargetKind: httpegress.TargetKind, ResourceKind: httpegress.KindHTTPSession, PluginInstanceID: "builtin"}
	policy, err := appbootstrap.NewBuiltinPolicy(func() time.Time { return now }, time.Minute, emptyAIInstallations(t), withoutConsent, emptyApplicationInstallations(t), emptyAutomationInstallations(t))
	if err != nil {
		t.Fatal(err)
	}
	decision, err := policy.Authorize(context.Background(), admission.PolicyRequest{Bindings: []capability.Binding{binding}})
	if err != nil || decision.Outcome != admission.PolicyConsentRequired {
		t.Fatalf("unconsented HTTP decision = %#v, %v", decision, err)
	}
	profile, err := httpegress.SealProfile(profileDraft)
	if err != nil {
		t.Fatal(err)
	}
	consent, err := httpegress.WorkflowConsentDigest("http", profile)
	if err != nil {
		t.Fatal(err)
	}
	installed, err := httpegress.Install([]httpegress.InstallationDraft{{Slot: "http", Profile: profileDraft, Consent: consent}})
	if err != nil {
		t.Fatal(err)
	}
	entry = installed.Entries()[0]
	binding.ProviderID, binding.ProviderArtifactDigest, binding.TargetID = entry.ProviderID, entry.ProviderArtifact, entry.TargetID
	policy, err = appbootstrap.NewBuiltinPolicy(func() time.Time { return now }, time.Minute, emptyAIInstallations(t), installed, emptyApplicationInstallations(t), emptyAutomationInstallations(t))
	if err != nil {
		t.Fatal(err)
	}
	decision, err = policy.Authorize(context.Background(), admission.PolicyRequest{Bindings: []capability.Binding{binding}})
	if err != nil || decision.Outcome != admission.PolicyApproved || len(decision.ConsentLineage) != 1 || decision.ConsentLineage[0] != consent {
		t.Fatalf("consented HTTP decision = %#v, %v", decision, err)
	}
	binding.TargetID = "http-origin/forged"
	decision, err = policy.Authorize(context.Background(), admission.PolicyRequest{Bindings: []capability.Binding{binding}})
	if err != nil || decision.Outcome != admission.PolicyDenied {
		t.Fatalf("forged HTTP decision = %#v, %v", decision, err)
	}
}

func TestBuiltinPolicyRequiresExactApplicationInstallationConsent(t *testing.T) {
	if !appcontrol.PlatformSupported() {
		t.Skip("application lifecycle is intentionally unavailable")
	}
	path := filepath.Join(t.TempDir(), "tool.exe")
	if err := os.WriteFile(path, []byte("installed-application"), 0o700); err != nil {
		t.Fatal(err)
	}
	inspection, err := appcontrol.InspectExecutable(path)
	if err != nil {
		t.Fatal(err)
	}
	profileDraft := appcontrol.ProfileDraft{Executable: inspection.Executable, Arguments: []string{"--fixed"}}
	withoutConsent, err := appcontrol.Install([]appcontrol.InstallationDraft{{Slot: "tool", Profile: profileDraft}})
	if err != nil {
		t.Fatal(err)
	}
	entry := withoutConsent.Entries()[0]
	binding := capability.Binding{ProviderID: entry.ProviderID, ProviderArtifactDigest: entry.ProviderArtifact, ProviderABI: appcontrol.ProviderABI, TargetID: entry.TargetID, TargetKind: appcontrol.TargetKind, ResourceKind: appcontrol.KindApplication, PluginInstanceID: "builtin"}
	policy, err := appbootstrap.NewBuiltinPolicy(func() time.Time { return time.Now().UTC() }, time.Minute, emptyAIInstallations(t), emptyHTTPInstallations(t), withoutConsent, emptyAutomationInstallations(t))
	if err != nil {
		t.Fatal(err)
	}
	decision, err := policy.Authorize(context.Background(), admission.PolicyRequest{Bindings: []capability.Binding{binding}})
	if err != nil || decision.Outcome != admission.PolicyConsentRequired {
		t.Fatalf("unconsented decision = %#v, %v", decision, err)
	}
	profile, err := appcontrol.SealProfile(profileDraft)
	if err != nil {
		t.Fatal(err)
	}
	consent, err := appcontrol.WorkflowConsentDigest("tool", profile)
	if err != nil {
		t.Fatal(err)
	}
	installed, err := appcontrol.Install([]appcontrol.InstallationDraft{{Slot: "tool", Profile: profileDraft, Consent: consent}})
	if err != nil {
		t.Fatal(err)
	}
	entry = installed.Entries()[0]
	binding.ProviderID, binding.ProviderArtifactDigest, binding.TargetID = entry.ProviderID, entry.ProviderArtifact, entry.TargetID
	policy, err = appbootstrap.NewBuiltinPolicy(func() time.Time { return time.Now().UTC() }, time.Minute, emptyAIInstallations(t), emptyHTTPInstallations(t), installed, emptyAutomationInstallations(t))
	if err != nil {
		t.Fatal(err)
	}
	decision, err = policy.Authorize(context.Background(), admission.PolicyRequest{Bindings: []capability.Binding{binding}})
	if err != nil || decision.Outcome != admission.PolicyApproved || len(decision.ConsentLineage) != 1 || decision.ConsentLineage[0] != consent {
		t.Fatalf("consented decision = %#v, %v", decision, err)
	}
	binding.ProviderArtifactDigest = testDigest(t, "forged-application-provider")
	decision, err = policy.Authorize(context.Background(), admission.PolicyRequest{Bindings: []capability.Binding{binding}})
	if err != nil || decision.Outcome != admission.PolicyDenied {
		t.Fatalf("forged decision = %#v, %v", decision, err)
	}
}

func TestBuiltinPolicyRequiresExactAutomationInputConsent(t *testing.T) {
	if !automationinstalled.PlatformSupported() {
		t.Skip("installed automation targets are intentionally unavailable")
	}
	path := filepath.Join(t.TempDir(), "editor.exe")
	if err := os.WriteFile(path, []byte("installed-automation-target"), 0o700); err != nil {
		t.Fatal(err)
	}
	inspection, err := appcontrol.InspectExecutable(path)
	if err != nil {
		t.Fatal(err)
	}
	profileDraft := automationinstalled.NewDesktopProfileDraft(automationinstalled.DesktopProfilePayload{
		Application: appcontrol.ProfileDraft{Executable: inspection.Executable, Arguments: []string{}},
		WindowTitle: "Editor", WindowTitleMatch: "exact", WindowSelection: "unique", WindowClass: "EditorWindow",
		InputBackend: "postmessage", CaptureBackend: "gdi", ResolveTimeoutMilliseconds: 500,
	})
	withoutConsent, err := automationinstalled.Install([]automationinstalled.InstallationDraft{{Slot: "editor-input", Profile: profileDraft}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = withoutConsent.Close() })
	entry := withoutConsent.Entries()[0]
	binding := capability.Binding{
		ProviderID: entry.ProviderID, ProviderArtifactDigest: entry.ProviderArtifact, ProviderABI: automationinstalled.ProviderABI,
		TargetID: entry.TargetID, TargetKind: automationinstalled.TargetKind, ResourceKind: automationinstalled.KindInput, PluginInstanceID: "builtin",
	}
	policy, err := appbootstrap.NewBuiltinPolicy(func() time.Time { return time.Now().UTC() }, time.Minute, emptyAIInstallations(t), emptyHTTPInstallations(t), emptyApplicationInstallations(t), withoutConsent)
	if err != nil {
		t.Fatal(err)
	}
	decision, err := policy.Authorize(context.Background(), admission.PolicyRequest{Bindings: []capability.Binding{binding}})
	if err != nil || decision.Outcome != admission.PolicyConsentRequired {
		t.Fatalf("unconsented automation decision = %#v, %v", decision, err)
	}
	profile, err := automationinstalled.SealProfile(profileDraft)
	if err != nil {
		t.Fatal(err)
	}
	consent, err := automationinstalled.WorkflowConsentDigest("editor-input", profile)
	if err != nil {
		t.Fatal(err)
	}
	installed, err := automationinstalled.Install([]automationinstalled.InstallationDraft{{Slot: "editor-input", Profile: profileDraft, Consent: consent}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = installed.Close() })
	entry = installed.Entries()[0]
	binding.ProviderID, binding.ProviderArtifactDigest, binding.TargetID = entry.ProviderID, entry.ProviderArtifact, entry.TargetID
	policy, err = appbootstrap.NewBuiltinPolicy(func() time.Time { return time.Now().UTC() }, time.Minute, emptyAIInstallations(t), emptyHTTPInstallations(t), emptyApplicationInstallations(t), installed)
	if err != nil {
		t.Fatal(err)
	}
	decision, err = policy.Authorize(context.Background(), admission.PolicyRequest{Bindings: []capability.Binding{binding}})
	if err != nil || decision.Outcome != admission.PolicyApproved || len(decision.ConsentLineage) != 1 || decision.ConsentLineage[0] != consent {
		t.Fatalf("consented automation decision = %#v, %v", decision, err)
	}
	binding.ResourceKind = automationinstalled.KindWindow
	decision, err = policy.Authorize(context.Background(), admission.PolicyRequest{Bindings: []capability.Binding{binding}})
	if err != nil || decision.Outcome != admission.PolicyApproved || len(decision.ConsentLineage) != 1 || decision.ConsentLineage[0] != consent {
		t.Fatalf("consented window decision = %#v, %v", decision, err)
	}
	binding.ResourceKind = automationinstalled.KindCapture
	decision, err = policy.Authorize(context.Background(), admission.PolicyRequest{Bindings: []capability.Binding{binding}})
	if err != nil || decision.Outcome != admission.PolicyApproved || len(decision.ConsentLineage) != 1 || decision.ConsentLineage[0] != consent {
		t.Fatalf("consented capture decision = %#v, %v", decision, err)
	}
	binding.TargetKind = appcontrol.TargetKind
	decision, err = policy.Authorize(context.Background(), admission.PolicyRequest{Bindings: []capability.Binding{binding}})
	if err != nil || decision.Outcome != admission.PolicyDenied {
		t.Fatalf("forged automation decision = %#v, %v", decision, err)
	}
}

type testAICredentials struct{}

func (testAICredentials) Get(string) (string, error) { return "secret", nil }

type testCredentialAvailability map[string]bool

func (availability testCredentialAvailability) HasSlot(slot string) (bool, error) {
	return availability[slot], nil
}

type discardWorkflowLog struct{}

func (discardWorkflowLog) EmitWorkflowLog(context.Context, noderuntime.LogEntry) error { return nil }

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

func emptyHTTPInstallations(t *testing.T) httpegress.Installations {
	t.Helper()
	installations, err := httpegress.Install(nil)
	if err != nil {
		t.Fatal(err)
	}
	return installations
}

func emptyApplicationInstallations(t *testing.T) appcontrol.Installations {
	t.Helper()
	installations, err := appcontrol.Install(nil)
	if err != nil {
		t.Fatal(err)
	}
	return installations
}

func emptyAutomationInstallations(t *testing.T) automationinstalled.Installations {
	t.Helper()
	installations, err := automationinstalled.Install(nil)
	if err != nil {
		t.Fatal(err)
	}
	return installations
}

func testLimits() appbootstrap.Limits {
	return appbootstrap.Limits{
		MaxSources: 8, MaxPrograms: 8, MaxRuns: 8,
		MaxProgramCacheBytes:    8 << 20,
		MaxResourcePayloadBytes: 2 << 20,
		BlobChunkBytes:          64 << 10, BlobQueueCapacity: 2, StreamCapacity: 4, StreamChunkBytes: 64 << 10,
	}
}

type testWorkflowStorage struct {
	roots      storage.Roots
	foundation *catalog.Foundation
	blobs      *blob.Store
}

func newTestWorkflowStorage(t *testing.T) testWorkflowStorage {
	t.Helper()
	roots, err := storage.Resolve(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	foundation, err := catalog.Open(context.Background(), roots)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := foundation.Close(); err != nil {
			t.Errorf("close test Catalog = %v", err)
		}
	})
	blobs, err := blob.Open(
		roots.Objects,
		blob.Limits{MaxBlobBytes: 1 << 20, MaxTotalBytes: 8 << 20},
		foundation.Objects(),
	)
	if err != nil {
		t.Fatal(err)
	}
	return testWorkflowStorage{roots: roots, foundation: foundation, blobs: blobs}
}

func testDigest(t *testing.T, label string) artifact.Digest {
	t.Helper()
	digest, err := artifact.Sum("yotta/test/appbootstrap/v1", []byte(label))
	if err != nil {
		t.Fatal(err)
	}
	return digest
}

func approvedAIProfile(t *testing.T, draft ai.ModelProfileDraft) (ai.ModelProfileDraft, ai.EvalReportArtifact) {
	t.Helper()
	suite, err := ai.BuiltinEvalSuite()
	if err != nil {
		t.Fatal(err)
	}
	draft.Evaluation = ai.EvaluationUnverified
	draft.EvaluationSuite = ""
	draft.EvaluationReport = ""
	profile, err := ai.SealModelProfile(draft)
	if err != nil {
		t.Fatal(err)
	}
	subject, err := ai.EvaluationSubjectDigest(profile)
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := ai.NewEvalCandidate(subject, []artifact.Digest{suite.Machine().Baseline})
	if err != nil {
		t.Fatal(err)
	}
	observations := make([]ai.EvalObservation, 0, len(suite.Machine().Cases))
	for _, evalCase := range suite.Machine().Cases {
		observations = append(observations, ai.EvalObservation{
			CaseID: evalCase.ID, Output: append(json.RawMessage(nil), evalCase.Expected...), Refused: evalCase.RequireRefusal,
			InputTokens: 10, OutputTokens: 5, CostMicrounits: 100, LatencyMillis: 10,
		})
	}
	evidence, err := ai.GradeEvalSuite(suite, candidate, observations)
	if err != nil {
		t.Fatal(err)
	}
	draft.Evaluation = ai.EvaluationApproved
	draft.EvaluationSuite = suite.Digest()
	draft.EvaluationReport = evidence.Digest
	return draft, evidence
}
