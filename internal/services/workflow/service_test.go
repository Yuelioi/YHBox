package workflow_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/ai"
	"github.com/yottaapp/yotta/internal/appbootstrap"
	"github.com/yottaapp/yotta/internal/appcontrol"
	automationinstalled "github.com/yottaapp/yotta/internal/automation/installed"
	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/httpegress"
	"github.com/yottaapp/yotta/internal/noderuntime"
	"github.com/yottaapp/yotta/internal/nodes"
	run "github.com/yottaapp/yotta/internal/run"
	"github.com/yottaapp/yotta/internal/scriptengine"
	"github.com/yottaapp/yotta/internal/services/workflow"
	"github.com/yottaapp/yotta/internal/workflow/authoring"
	"github.com/yottaapp/yotta/internal/workflow/schema"
)

func TestServiceProjectsProductionWorkflowLifecycle(t *testing.T) {
	now := time.Date(2026, 7, 17, 3, 0, 0, 0, time.UTC)
	runtime := workflowRuntime(t, now)
	if err := runtime.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := runtime.Close(ctx); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})
	service, err := workflow.NewService(runtime.Application)
	if err != nil {
		t.Fatal(err)
	}
	created, err := service.CreateSource(" Projection ")
	if err != nil || created.Name != "Projection" || created.SourceJSON == "" {
		t.Fatalf("CreateSource() = %#v, %v", created, err)
	}
	loaded, err := service.GetSource(created.WorkflowID)
	if err != nil || loaded.SourceHash != created.SourceHash || loaded.SourceJSON == "" {
		t.Fatalf("GetSource() = %#v, %v", loaded, err)
	}
	patched, err := service.ApplyPatch(created.WorkflowID, created.Revision, []authoring.Command{
		{Kind: authoring.CommandAddNode, AddNode: &authoring.AddNodeCommand{GraphID: "main", NodeTypeID: nodes.ConcatNodeID, Handle: "concat", Position: schema.Position{}}},
		{Kind: authoring.CommandBindValue, BindValue: &authoring.BindValueCommand{GraphID: "main", NodeID: "$concat", PortID: "a", Value: "a"}},
		{Kind: authoring.CommandBindValue, BindValue: &authoring.BindValueCommand{GraphID: "main", NodeID: "$concat", PortID: "b", Value: "b"}},
	})
	if err != nil || len(patched.GeneratedNodes) != 1 || patched.Source.Revision != 1 {
		t.Fatalf("ApplyPatch() = %#v, %v", patched, err)
	}
	listed, err := service.ListSources()
	if err != nil || len(listed) != 1 || listed[0].SourceJSON != "" || listed[0].SourceHash != patched.Source.SourceHash {
		t.Fatalf("ListSources() = %#v, %v", listed, err)
	}
	compiled, err := service.CompileSource(created.WorkflowID)
	if err != nil || !compiled.ProgramHash.Valid() || len(compiled.Diagnostics) != 0 {
		t.Fatalf("CompileSource() = %#v, %v", compiled, err)
	}
	preview, err := service.PreviewRun(created.WorkflowID)
	if err != nil || preview.ProgramHash != compiled.ProgramHash {
		t.Fatalf("PreviewRun() = %#v, %v", preview, err)
	}
	if !strings.Contains(service.GetCatalog(), `"version":"3.1"`) || !strings.Contains(service.GetAuthoringProjection(), `"format":"yotta.node-authoring-projection"`) {
		t.Fatal("service projections omitted their format identity")
	}
	started, err := service.StartRun(created.WorkflowID)
	if err != nil || started.Run == nil || started.Run.Status != string(run.StatusQueued) {
		t.Fatalf("StartRun() = %#v, %v", started, err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		view, err := service.GetRunTimeline(started.Run.RunID)
		if err != nil {
			t.Fatal(err)
		}
		if view.Status == string(run.StatusSucceeded) {
			if len(view.Timeline) != 4 || view.Failure != nil {
				t.Fatalf("terminal Run = %#v", view)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("Run remained %s", view.Status)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := service.CancelAllRuns(); err != nil {
		t.Fatal(err)
	}
}

func TestServiceRejectsMissingOrUntrustedApplication(t *testing.T) {
	if _, err := workflow.NewService(nil); err == nil {
		t.Fatal("NewService accepted nil Application")
	}
}

func TestServiceQueriesAndDeletesSourcesWithCASAndReferenceBlocking(t *testing.T) {
	runtime := workflowRuntime(t, time.Date(2026, 7, 17, 4, 0, 0, 0, time.UTC))
	if err := runtime.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := runtime.Close(ctx); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})
	service, err := workflow.NewService(runtime.Application, workflow.WithReferenceResolver(func(workflowID string) []workflow.SourceReference {
		if strings.HasSuffix(workflowID, "blocked") {
			return []workflow.SourceReference{{Kind: "schedule", ID: "nightly", Label: "Nightly"}}
		}
		return nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	alpha, err := service.CreateSource("Alpha")
	if err != nil {
		t.Fatal(err)
	}
	beta, err := service.CreateSource("Beta")
	if err != nil {
		t.Fatal(err)
	}
	page, err := service.QuerySources(workflow.SourceQuery{Search: "a", Sort: "name_desc", Page: 1, PageSize: 1})
	if err != nil || page.Total != 2 || len(page.Items) != 1 || page.Items[0].Name != "Beta" {
		t.Fatalf("QuerySources = %#v, %v", page, err)
	}

	blockedID := beta.WorkflowID + "-blocked"
	previews, err := service.PreviewDeleteSources([]string{alpha.WorkflowID, blockedID})
	if err == nil || len(previews) != 0 {
		// Preview is strict: a missing Source aborts rather than presenting stale confirmation data.
		t.Fatalf("PreviewDeleteSources missing source = %#v, %v", previews, err)
	}
	serviceWithReference, err := workflow.NewService(runtime.Application, workflow.WithReferenceResolver(func(workflowID string) []workflow.SourceReference {
		if workflowID == beta.WorkflowID {
			return []workflow.SourceReference{{Kind: "schedule", ID: "nightly", Label: "Nightly"}}
		}
		return nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	previews, err = serviceWithReference.PreviewDeleteSources([]string{alpha.WorkflowID, beta.WorkflowID})
	if err != nil || len(previews) != 2 || len(previews[0].References) != 0 || len(previews[1].References) != 1 {
		t.Fatalf("PreviewDeleteSources = %#v, %v", previews, err)
	}
	results := serviceWithReference.DeleteSources([]workflow.DeleteSourceRequest{
		{WorkflowID: alpha.WorkflowID, Revision: alpha.Revision, SourceHash: alpha.SourceHash},
		{WorkflowID: beta.WorkflowID, Revision: beta.Revision, SourceHash: beta.SourceHash},
	})
	if len(results) != 2 || !results[0].Deleted || results[1].Deleted || len(results[1].References) != 1 {
		t.Fatalf("DeleteSources = %#v", results)
	}
	if _, err := service.GetSource(alpha.WorkflowID); err == nil {
		t.Fatal("deleted Source still loads")
	}
	if _, err := service.GetSource(beta.WorkflowID); err != nil {
		t.Fatalf("referenced Source was deleted: %v", err)
	}
}

func workflowRuntime(t *testing.T, now time.Time) *appbootstrap.Runtime {
	t.Helper()
	aiInstallations, err := ai.Install(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	httpInstallations, err := httpegress.Install(nil)
	if err != nil {
		t.Fatal(err)
	}
	applicationInstallations, err := appcontrol.Install(nil)
	if err != nil {
		t.Fatal(err)
	}
	automationInstallations, err := automationinstalled.Install(nil)
	if err != nil {
		t.Fatal(err)
	}
	blobStore, err := blob.Open(filepath.Join(t.TempDir(), "blobs"), blob.Limits{MaxBlobBytes: 1 << 20, MaxTotalBytes: 8 << 20})
	if err != nil {
		t.Fatal(err)
	}
	scriptRuntime, err := scriptengine.NewRuntime(scriptengine.RuntimeOptions{
		Executable:         filepath.Join(t.TempDir(), scriptengine.WorkerExecutableName),
		ProcessMemoryBytes: scriptengine.DefaultMemoryBytes,
		JobMemoryBytes:     scriptengine.DefaultMemoryBytes,
	})
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := appbootstrap.Build(appbootstrap.Config{
		DataRoot: t.TempDir(), BlobStore: blobStore,
		Limits: appbootstrap.Limits{
			MaxSources: 8, MaxPrograms: 8, MaxRuns: 8, MaxResourcePayloadBytes: 2 << 20,
			BlobChunkBytes: 64 << 10, BlobQueueCapacity: 2, StreamCapacity: 4, StreamChunkBytes: 64 << 10,
		},
		AIInstallations: aiInstallations, HTTPInstallations: httpInstallations,
		ApplicationInstallations: applicationInstallations, AutomationInstallations: automationInstallations,
		ScriptRuntime: scriptRuntime, LogEmitter: noderuntime.LogEmitterFunc(func(context.Context, noderuntime.LogEntry) error { return nil }),
		GrantTTL: 5 * time.Minute, OwnerCloseTimeout: time.Second, Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	return runtime
}
