package application_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/admission"
	app31 "github.com/yottaapp/yotta/internal/application"
	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/nodeauthoring"
	"github.com/yottaapp/yotta/internal/nodes31"
	"github.com/yottaapp/yotta/internal/nodes31runtime"
	"github.com/yottaapp/yotta/internal/resource"
	run31 "github.com/yottaapp/yotta/internal/run"
	"github.com/yottaapp/yotta/internal/workflow/authoring"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
	"github.com/yottaapp/yotta/internal/workflow/schema"
	"github.com/yottaapp/yotta/internal/workflowstore"
)

func TestApplicationRunsPersistedSourceThroughTheOnlyProgramWorker(t *testing.T) {
	now := time.Date(2026, 7, 15, 10, 0, 0, 0, time.UTC)
	application, sources, programs, _, events := newTestApplication(t, now, nil)
	if err := application.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := application.Close(ctx); err != nil {
			t.Errorf("Close = %v", err)
		}
	})
	saved := createConcatWorkflow(t, application)
	if saved.Source.Revision() != 1 {
		t.Fatalf("ApplyPatch = %#v", saved)
	}
	started, err := application.StartRun(context.Background(), app31.StartRunRequest{WorkflowID: saved.Source.WorkflowID(), Principal: "user-1"})
	if err != nil || !started.Record.Valid() || !started.ProgramHash.Valid() || len(started.Diagnostics) != 0 {
		t.Fatalf("StartRun = %#v, %v", started, err)
	}
	if started.SourceHash != saved.Source.Hash() {
		t.Fatalf("compiled Source hash = %s, stored = %s", started.SourceHash, saved.Source.Hash())
	}
	if _, err := programs.Load(started.ProgramHash); err != nil {
		t.Fatalf("Program was not durable before admission: %v", err)
	}
	deadline := time.After(5 * time.Second)
	for {
		select {
		case event := <-events:
			if event.RunID == started.Record.Admission().RunID && event.Status == run31.StatusSucceeded {
				loaded, err := application.GetRun(event.RunID)
				if err != nil || loaded.Status() != run31.StatusSucceeded || len(loaded.Journal()) != 2 {
					t.Fatalf("terminal Run = %#v, %v", loaded, err)
				}
				if listed := sources.List(); len(listed) != 1 || listed[0].Hash() != saved.Source.Hash() {
					t.Fatalf("Source list = %#v", listed)
				}
				return
			}
		case <-deadline:
			t.Fatal("Run did not reach succeeded")
		}
	}
}

func TestApplicationCommandsRequireLiveLifecycle(t *testing.T) {
	now := time.Date(2026, 7, 15, 10, 30, 0, 0, time.UTC)
	application, _, _, _, _ := newTestApplication(t, now, nil)
	if _, err := application.CreateSource(context.Background(), "before start"); err != app31.ErrNotStarted {
		t.Fatalf("CreateSource before Start = %v", err)
	}
	if err := application.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := application.Start(context.Background()); err != app31.ErrClosed {
		t.Fatalf("Start after Close = %v", err)
	}
}

func TestApplicationCancellationOwnsRunningWorkerAndPersistsTerminalState(t *testing.T) {
	now := time.Date(2026, 7, 15, 11, 0, 0, 0, time.UTC)
	invoked := make(chan struct{})
	adapter := func(ctx context.Context, _ compiler.Invocation) (compiler.AdapterResult, error) {
		close(invoked)
		<-ctx.Done()
		return compiler.AdapterResult{}, ctx.Err()
	}
	application, _, _, _, events := newTestApplication(t, now, adapter)
	if err := application.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	saved := createConcatWorkflow(t, application)
	started, err := application.StartRun(context.Background(), app31.StartRunRequest{WorkflowID: saved.Source.WorkflowID(), Principal: "user-1"})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-invoked:
	case <-time.After(5 * time.Second):
		t.Fatal("adapter was not invoked")
	}
	if _, err := application.CancelRun(context.Background(), started.Record.Admission().RunID); err != nil {
		t.Fatal(err)
	}
	deadline := time.After(5 * time.Second)
	for {
		select {
		case event := <-events:
			if event.RunID == started.Record.Admission().RunID && event.Status == run31.StatusCancelled {
				loaded, err := application.GetRun(event.RunID)
				if err != nil || loaded.Status() != run31.StatusCancelled {
					t.Fatalf("cancelled Run = %#v, %v", loaded, err)
				}
				if err := application.Close(context.Background()); err != nil {
					t.Fatal(err)
				}
				return
			}
		case <-deadline:
			t.Fatal("Run did not reach cancelled")
		}
	}
}

func newTestApplication(t *testing.T, now time.Time, adapterOverride compiler.Adapter) (*app31.Application, *workflowstore.SourceStore, *workflowstore.ProgramStore, nodes31.Builtins, chan app31.RunEvent) {
	t.Helper()
	builtins, err := nodes31.Build()
	if err != nil {
		t.Fatal(err)
	}
	projection, err := nodeauthoring.Project(nodeauthoring.Input{
		Catalog: builtins.Catalog, Types: builtins.Types, Capabilities: builtins.Capabilities,
		Contracts: builtins.Contracts, GeneratorVersion: nodes31.GeneratorVersion,
	})
	if err != nil {
		t.Fatal(err)
	}
	build := testDigest(t, "application compiler")
	root := t.TempDir()
	sources, err := workflowstore.OpenSourceStore(filepath.Join(root, "sources"), workflowstore.SourceStoreOptions{MaxSources: 8})
	if err != nil {
		t.Fatal(err)
	}
	programs, err := workflowstore.OpenProgramStore(filepath.Join(root, "programs"), builtins.Catalog, build, workflowstore.ProgramStoreOptions{MaxPrograms: 8})
	if err != nil {
		t.Fatal(err)
	}
	runs, err := run31.OpenStore(filepath.Join(root, "runs"), builtins.Catalog, run31.StoreOptions{MaxRecords: 8})
	if err != nil {
		t.Fatal(err)
	}
	profile, err := admission.SealHostProfile(admission.HostProfileDraft{
		OS: "windows", Architecture: "amd64", HostAPIGeneration: "3.1",
	})
	if err != nil {
		t.Fatal(err)
	}
	policy := admission.PolicyFunc(func(context.Context, admission.PolicyRequest) (admission.PolicyDecision, error) {
		return admission.PolicyDecision{Outcome: admission.PolicyApproved, Generation: "test-policy", ExpiresAt: now.Add(time.Minute)}, nil
	})
	admitter, err := admission.New(builtins.Catalog, profile, runs, policy, admission.Options{
		Now: func() time.Time { return now }, MaxGrantTTL: 5 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	adapters, err := nodes31runtime.Installed(builtins)
	if err != nil {
		t.Fatal(err)
	}
	if adapterOverride != nil {
		entry, ok := builtins.Catalog.Lookup(nodes31.ConcatNodeID)
		if !ok {
			t.Fatal("Concat implementation is missing")
		}
		adapters[entry.Implementation.Entrypoint] = compiler.InstalledAdapter{Implementation: entry.Implementation, Run: adapterOverride}
	}
	executor := compiler.NewExecutor(builtins.Catalog, adapters, compiler.ExecutorOptions{Now: func() time.Time { return now }})
	events := make(chan app31.RunEvent, 16)
	application, err := app31.New(app31.Config{
		Catalog: builtins.Catalog, Authoring: projection, CompilerBuild: build,
		Sources: sources, Programs: programs, Runs: runs,
		Admitter: admitter, Executor: executor, Providers: map[string]run31.InstalledProvider{},
		ResourceOptions: resource.Options{Now: func() time.Time { return now }}, OwnerCloseTimeout: time.Second,
		Now: func() time.Time { return now }, OnRunEvent: func(event app31.RunEvent) { events <- event },
	})
	if err != nil {
		t.Fatal(err)
	}
	return application, sources, programs, builtins, events
}

func createConcatWorkflow(t *testing.T, application *app31.Application) app31.ApplyPatchResult {
	t.Helper()
	created, err := application.CreateSource(context.Background(), "Application")
	if err != nil {
		t.Fatal(err)
	}
	result, err := application.ApplyPatch(context.Background(), authoring.PatchRequest{
		WorkflowID: created.WorkflowID(), BaseRevision: created.Revision(),
		Commands: []authoring.Command{
			{Kind: authoring.CommandAddNode, AddNode: &authoring.AddNodeCommand{
				GraphID: "main", NodeTypeID: nodes31.ConcatNodeID, Handle: "concat", Position: schema.Position{},
			}},
			{Kind: authoring.CommandBindValue, BindValue: &authoring.BindValueCommand{GraphID: "main", NodeID: "$concat", PortID: "a", Value: "hello"}},
			{Kind: authoring.CommandBindValue, BindValue: &authoring.BindValueCommand{GraphID: "main", NodeID: "$concat", PortID: "b", Value: " world"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func testDigest(t *testing.T, label string) artifact.Digest {
	t.Helper()
	digest, err := artifact.Sum("yotta/test/application/v1", []byte(label))
	if err != nil {
		t.Fatal(err)
	}
	return digest
}
