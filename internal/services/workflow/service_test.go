package workflow_test

import (
	"context"
	"fmt"
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

func TestServiceQueriesOneThousandSourcesWithBoundedPages(t *testing.T) {
	runtime := workflowRuntime(t, time.Date(2026, 7, 18, 13, 0, 0, 0, time.UTC), 1_000)
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
	for index := 0; index < 1_000; index++ {
		name := fmt.Sprintf("Workflow %04d", index)
		if index == 100 || index == 900 {
			name += " Needle"
		}
		request := workflow.CreateSourceRequest{
			Name: name, Description: fmt.Sprintf("Fixture workflow %04d", index),
			Category: []string{"Even", "Odd"}[index%2], Tags: []string{"common"},
		}
		if index == 100 || index == 900 {
			request.Tags = append(request.Tags, "needle")
		}
		if _, err := service.CreateSourceWithMetadata(request); err != nil {
			t.Fatalf("CreateSource(%d): %v", index, err)
		}
	}

	started := time.Now()
	page, err := service.QuerySources(workflow.SourceQuery{Sort: "name_asc", Page: 10, PageSize: 100})
	if err != nil || page.Total != 1_000 || len(page.Items) != 100 || page.Items[0].Name != "Workflow 0900 Needle" {
		t.Fatalf("QuerySources page = %#v, %v", page, err)
	}
	search, err := service.QuerySources(workflow.SourceQuery{Search: "needle", Sort: "name_asc", Page: 1, PageSize: 20})
	if err != nil || search.Total != 2 || len(search.Items) != 2 {
		t.Fatalf("QuerySources search = %#v, %v", search, err)
	}
	filtered, err := service.QuerySources(workflow.SourceQuery{
		Category: "even", Tags: []string{"NEEDLE"}, Sort: "nodes_desc", Page: 1, PageSize: 20,
	})
	if err != nil || filtered.Total != 2 || len(filtered.Items) != 2 || filtered.Items[0].NodeCount != 1 {
		t.Fatalf("QuerySources facets = %#v, %v", filtered, err)
	}
	recent, err := service.QuerySources(workflow.SourceQuery{
		CreatedSince: time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC).Format(time.RFC3339),
		UpdatedSince: time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC).Format(time.RFC3339),
		Sort:         "updated_desc", Page: 1, PageSize: 20,
	})
	if err != nil || recent.Total != 1_000 || recent.Items[0].CreatedAt == "" || recent.Items[0].UpdatedAt == "" {
		t.Fatalf("QuerySources date filters = %#v, %v", recent, err)
	}
	future, err := service.QuerySources(workflow.SourceQuery{
		CreatedSince: time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
		Sort:         "created_desc", Page: 1, PageSize: 20,
	})
	if err != nil || future.Total != 0 {
		t.Fatalf("QuerySources future date filter = %#v, %v", future, err)
	}
	if _, err := service.QuerySources(workflow.SourceQuery{CreatedSince: "invalid", Page: 1}); err == nil {
		t.Fatal("QuerySources accepted an invalid date filter")
	}
	if len(filtered.Categories) != 2 || filtered.Categories[0].Value != "Even" || filtered.Categories[0].Count != 500 ||
		len(filtered.Tags) != 2 || filtered.Tags[0].Value != "common" || filtered.Tags[0].Count != 1_000 {
		t.Fatalf("QuerySources facet values = categories %#v, tags %#v", filtered.Categories, filtered.Tags)
	}
	if elapsed := time.Since(started); elapsed > 5*time.Second {
		t.Fatalf("querying 1000 workflow sources took %s", elapsed)
	}
}

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
	if err != nil || created.Name != "Projection" || created.SourceJSON == "" ||
		created.CreatedAt != now.Format(time.RFC3339Nano) || created.UpdatedAt != created.CreatedAt {
		t.Fatalf("CreateSource() = %#v, %v", created, err)
	}
	metadata, err := service.UpdateSourceMetadata(created.WorkflowID, created.Revision, workflow.UpdateSourceMetadataRequest{
		Name: "Projection", Description: "Service projection", Category: "Tests", Tags: []string{"workflow", "Workflow"},
	})
	if err != nil || metadata.Description != "Service projection" || metadata.Category != "Tests" || len(metadata.Tags) != 1 || metadata.Revision != 1 {
		t.Fatalf("UpdateSourceMetadata() = %#v, %v", metadata, err)
	}
	batch := service.BatchUpdateSourceMetadata([]workflow.BatchUpdateSourceMetadataRequest{
		{WorkflowID: created.WorkflowID, BaseRevision: metadata.Revision, Category: "Batch", Tags: []string{"reviewed"}},
	})
	if len(batch) != 1 || !batch[0].Updated || batch[0].Error != "" {
		t.Fatalf("BatchUpdateSourceMetadata() = %#v", batch)
	}
	created, err = service.GetSource(created.WorkflowID)
	if err != nil || created.Name != "Projection" || created.Description != "Service projection" || created.Category != "Batch" || len(created.Tags) != 1 || created.Tags[0] != "reviewed" {
		t.Fatalf("batch metadata source = %#v, %v", created, err)
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
	if err != nil || len(patched.GeneratedNodes) != 1 || patched.Source.Revision != 3 || patched.Source.NodeCount != 2 {
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

func TestServiceExposesWorkflowSourcePortabilityWithoutMachineInstallations(t *testing.T) {
	runtime := workflowRuntime(t, time.Date(2026, 7, 17, 3, 30, 0, 0, time.UTC))
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
	service, err := workflow.NewService(runtime.Application, workflow.WithBundleManager(runtime.Bundles))
	if err != nil {
		t.Fatal(err)
	}
	created, err := service.CreateSource("Portable")
	if err != nil {
		t.Fatal(err)
	}
	archivePath := filepath.Join(t.TempDir(), created.WorkflowID+".yotta-workflow")
	exported, err := service.ExportSourceBundle(created.WorkflowID, archivePath)
	if err != nil || !exported.Exported || exported.Path != archivePath {
		t.Fatalf("ExportSourceBundle() = %#v, %v", exported, err)
	}
	info, err := service.InspectSourceBundle(archivePath)
	if err != nil || info.WorkflowID != created.WorkflowID || info.SourceHash != created.SourceHash {
		t.Fatalf("InspectSourceBundle() = %#v, %v", info, err)
	}
	imported, err := service.ImportSourceBundle(archivePath)
	if err != nil || imported.WorkflowID == created.WorkflowID || imported.Name != created.Name || imported.Revision != 0 {
		t.Fatalf("ImportSourceBundle() = %#v, %v", imported, err)
	}
	replaced, err := service.ReplaceSourceFromBundle(archivePath, imported.WorkflowID, imported.Revision, imported.SourceHash)
	if err != nil || replaced.WorkflowID != imported.WorkflowID || replaced.Revision != 1 {
		t.Fatalf("ReplaceSourceFromBundle() = %#v, %v", replaced, err)
	}
	batchDirectory := t.TempDir()
	results := service.ExportSourceBundles([]string{created.WorkflowID, imported.WorkflowID}, batchDirectory)
	if len(results) != 2 || !results[0].Exported || !results[1].Exported {
		t.Fatalf("ExportSourceBundles() = %#v", results)
	}
	repeated := service.ExportSourceBundles([]string{created.WorkflowID}, batchDirectory)
	if len(repeated) != 1 || repeated[0].Exported || repeated[0].Error != "destination already exists" {
		t.Fatalf("repeated ExportSourceBundles() = %#v", repeated)
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

func workflowRuntime(t *testing.T, now time.Time, maxSources ...int) *appbootstrap.Runtime {
	t.Helper()
	sourceLimit := 8
	if len(maxSources) != 0 {
		sourceLimit = maxSources[0]
	}
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
			MaxSources: sourceLimit, MaxPrograms: 8, MaxRuns: 8, MaxResourcePayloadBytes: 2 << 20,
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
