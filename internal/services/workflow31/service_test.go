package workflow31_test

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
	"github.com/yottaapp/yotta/internal/nodes31"
	"github.com/yottaapp/yotta/internal/nodes31runtime"
	run31 "github.com/yottaapp/yotta/internal/run"
	"github.com/yottaapp/yotta/internal/scriptengine"
	"github.com/yottaapp/yotta/internal/services/workflow31"
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
	service, err := workflow31.NewService(runtime.Application)
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
		{Kind: authoring.CommandAddNode, AddNode: &authoring.AddNodeCommand{GraphID: "main", NodeTypeID: nodes31.ConcatNodeID, Handle: "concat", Position: schema.Position{}}},
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
	if err != nil || started.Run == nil || started.Run.Status != string(run31.StatusQueued) {
		t.Fatalf("StartRun() = %#v, %v", started, err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		view, err := service.GetRunTimeline(started.Run.RunID)
		if err != nil {
			t.Fatal(err)
		}
		if view.Status == string(run31.StatusSucceeded) {
			if len(view.Timeline) != 2 || view.Failure != nil {
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
	if _, err := workflow31.NewService(nil); err == nil {
		t.Fatal("NewService accepted nil Application")
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
		ScriptRuntime: scriptRuntime, LogEmitter: nodes31runtime.LogEmitterFunc(func(context.Context, nodes31runtime.LogEntry) error { return nil }),
		GrantTTL: 5 * time.Minute, OwnerCloseTimeout: time.Second, Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	return runtime
}
