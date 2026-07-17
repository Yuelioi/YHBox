package application_test

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/admission"
	appcore "github.com/yottaapp/yotta/internal/application"
	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/nodeauthoring"
	"github.com/yottaapp/yotta/internal/noderuntime"
	"github.com/yottaapp/yotta/internal/nodes"
	"github.com/yottaapp/yotta/internal/resource"
	run "github.com/yottaapp/yotta/internal/run"
	"github.com/yottaapp/yotta/internal/scriptengine"
	"github.com/yottaapp/yotta/internal/workflow/authoring"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
	"github.com/yottaapp/yotta/internal/workflow/schema"
	"github.com/yottaapp/yotta/internal/workflowstore"
)

func TestApplicationRunsPersistedSourceThroughTheOnlyProgramWorker(t *testing.T) {
	now := time.Date(2026, 7, 15, 10, 0, 0, 0, time.UTC)
	application, sources, programs, _, events, _ := newTestApplication(t, now, nil)
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
	started, err := application.StartRun(context.Background(), appcore.StartRunRequest{WorkflowID: saved.Source.WorkflowID(), Principal: "user-1"})
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
			if event.RunID == started.Record.Admission().RunID && event.Status == run.StatusSucceeded {
				loaded, err := application.GetRun(event.RunID)
				if err != nil || loaded.Status() != run.StatusSucceeded || len(loaded.Journal()) != 4 {
					t.Fatalf("terminal Run = %#v, journal=%d, %v", loaded, len(loaded.Journal()), err)
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
	application, _, _, _, _, _ := newTestApplication(t, now, nil)
	if _, err := application.CreateSource(context.Background(), "before start"); err != appcore.ErrNotStarted {
		t.Fatalf("CreateSource before Start = %v", err)
	}
	if err := application.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := application.Start(context.Background()); err != appcore.ErrClosed {
		t.Fatalf("Start after Close = %v", err)
	}
}

func TestCreateSourceSeedsRunStartedRoot(t *testing.T) {
	now := time.Date(2026, 7, 17, 10, 45, 0, 0, time.UTC)
	application, _, _, _, _, _ := newTestApplication(t, now, nil)
	if err := application.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = application.Close(context.Background()) })
	if _, err := application.CreateSource(context.Background(), "  "); err == nil {
		t.Fatal("CreateSource accepted an empty workflow name")
	}

	created, err := application.CreateSource(context.Background(), "Runnable")
	if err != nil {
		t.Fatal(err)
	}
	var source schema.WorkflowSource
	if err := json.Unmarshal(created.Artifact(), &source); err != nil {
		t.Fatal(err)
	}
	if len(source.Graphs) != 1 || len(source.Graphs[0].Nodes) != 1 {
		t.Fatalf("created graph = %#v", source.Graphs)
	}
	root := source.Graphs[0].Nodes[0]
	if root.ID != "run-started" || root.NodeRef.NodeTypeID != nodes.RunStartedNodeID {
		t.Fatalf("created root = %#v", root)
	}
	loaded, err := application.GetSource(created.WorkflowID())
	if err != nil || loaded.Hash() != created.Hash() {
		t.Fatalf("GetSource = %#v, %v", loaded, err)
	}
	listed := application.ListSources()
	if len(listed) != 1 || listed[0].Hash() != created.Hash() {
		t.Fatalf("ListSources = %#v", listed)
	}
	if len(application.CatalogArtifact()) == 0 || !application.AuthoringProjection().Valid() {
		t.Fatal("application omitted its trusted authoring contracts")
	}
	compiled, err := application.CompileSource(context.Background(), created.WorkflowID())
	if err != nil || len(compiled.Diagnostics) != 0 {
		t.Fatalf("CompileSource = %#v, %v", compiled, err)
	}
	if _, ok := compiled.Program(); !ok {
		t.Fatal("new workflow did not compile to a runnable Program")
	}
	draft, err := application.CompileDraft(context.Background(), created.Artifact())
	if err != nil || draft.SourceHash != created.Hash() {
		t.Fatalf("CompileDraft = %#v, %v", draft, err)
	}
	preview, err := application.PreviewRun(context.Background(), created.WorkflowID())
	if err != nil || !preview.ProgramHash.Valid() || len(preview.Diagnostics) != 0 {
		t.Fatalf("PreviewRun = %#v, %v", preview, err)
	}
	if err := application.CancelAll(context.Background()); err != nil {
		t.Fatalf("CancelAll = %v", err)
	}
}

func TestPreparedPatchCommitsTheExactReviewedArtifactAndRejectsStaleBase(t *testing.T) {
	now := time.Date(2026, 7, 17, 9, 0, 0, 0, time.UTC)
	application, sources, _, _, _, _ := newTestApplication(t, now, nil)
	if err := application.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = application.Close(context.Background()) })
	created, err := application.CreateSource(context.Background(), "Prepared")
	if err != nil {
		t.Fatal(err)
	}
	request := authoring.PatchRequest{
		WorkflowID: created.WorkflowID(), BaseRevision: created.Revision(),
		Commands: []authoring.Command{{Kind: authoring.CommandAddNode, AddNode: &authoring.AddNodeCommand{
			GraphID: "main", NodeTypeID: nodes.ConcatNodeID, Handle: "reviewed", Position: schema.Position{X: 10, Y: 20},
		}}},
	}
	preview, err := application.PreparePatch(context.Background(), request)
	if err != nil || !preview.Patch.Valid() || preview.Patch.BaseHash() != created.Hash() || !preview.Patch.CandidateHash().Valid() {
		t.Fatalf("PreparePatch = %#v, %v", preview, err)
	}
	if current, loadErr := sources.Load(created.WorkflowID()); loadErr != nil || current.Hash() != created.Hash() {
		t.Fatalf("PreparePatch mutated durable source = %#v, %v", current, loadErr)
	}
	reviewed := preview.Patch.CandidateArtifact()
	committed, err := application.CommitPreparedPatch(context.Background(), preview.Patch)
	if err != nil || committed.Source.Hash() != preview.Patch.CandidateHash() || string(committed.Source.Artifact()) != string(reviewed) {
		t.Fatalf("CommitPreparedPatch = %#v, %v", committed, err)
	}
	if _, err := application.CommitPreparedPatch(context.Background(), preview.Patch); !errors.Is(err, workflowstore.ErrSourceConflict) {
		t.Fatalf("second CommitPreparedPatch = %v", err)
	}
}

func TestPreparedPatchCannotCommitAfterIndependentRevision(t *testing.T) {
	now := time.Date(2026, 7, 17, 9, 30, 0, 0, time.UTC)
	application, _, _, _, _, _ := newTestApplication(t, now, nil)
	if err := application.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = application.Close(context.Background()) })
	created, err := application.CreateSource(context.Background(), "Conflict")
	if err != nil {
		t.Fatal(err)
	}
	preview, err := application.PreparePatch(context.Background(), authoring.PatchRequest{
		WorkflowID: created.WorkflowID(), BaseRevision: created.Revision(),
		Commands: []authoring.Command{{Kind: authoring.CommandRenameWorkflow, RenameWorkflow: &authoring.RenameWorkflowCommand{Name: "AI candidate"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := application.ApplyPatch(context.Background(), authoring.PatchRequest{
		WorkflowID: created.WorkflowID(), BaseRevision: created.Revision(),
		Commands: []authoring.Command{{Kind: authoring.CommandRenameWorkflow, RenameWorkflow: &authoring.RenameWorkflowCommand{Name: "Human edit"}}},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := application.CommitPreparedPatch(context.Background(), preview.Patch); !errors.Is(err, workflowstore.ErrSourceConflict) {
		t.Fatalf("CommitPreparedPatch after human edit = %v", err)
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
	application, _, _, _, events, _ := newTestApplication(t, now, adapter)
	if err := application.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	saved := createConcatWorkflow(t, application)
	started, err := application.StartRun(context.Background(), appcore.StartRunRequest{WorkflowID: saved.Source.WorkflowID(), Principal: "user-1"})
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
			if event.RunID == started.Record.Admission().RunID && event.Status == run.StatusCancelled {
				loaded, err := application.GetRun(event.RunID)
				if err != nil || loaded.Status() != run.StatusCancelled {
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

func TestApplicationDebugRunPausesStepsAndCancelsThroughTheOnlyWorker(t *testing.T) {
	now := time.Date(2026, 7, 17, 13, 0, 0, 0, time.UTC)
	application, _, _, _, events, debugEvents := newTestApplication(t, now, nil)
	if err := application.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = application.Close(context.Background()) })
	saved := createConcatWorkflow(t, application)
	started, err := application.StartDebugRun(context.Background(), appcore.StartRunRequest{
		WorkflowID: saved.Source.WorkflowID(), Principal: "user-1",
	}, []compiler.DebugBreakpoint{})
	if err != nil {
		t.Fatal(err)
	}
	runID := started.Record.Admission().RunID
	concatID := saved.GeneratedNodes[0].NodeID
	first := waitApplicationDebug(t, debugEvents, runID, func(snapshot compiler.DebugSnapshot) bool {
		return snapshot.Status == compiler.DebugPaused && snapshot.NodeID == concatID
	})
	if first.GraphID != "main" || first.Attempt != 1 {
		t.Fatalf("first debug snapshot = %#v", first)
	}
	if len(first.Inputs) != 2 || !first.Inputs["a"].Redacted || !first.Inputs["b"].Redacted {
		t.Fatalf("bounded debug inputs = %#v", first.Inputs)
	}
	if _, err := application.ControlDebugRun(context.Background(), runID, appcore.DebugStep); err != nil {
		t.Fatal(err)
	}
	second := waitApplicationDebug(t, debugEvents, runID, func(snapshot compiler.DebugSnapshot) bool {
		return snapshot.Status == compiler.DebugPaused && snapshot.NodeID == "run-started"
	})
	if len(second.Inputs) != 0 {
		t.Fatalf("RunStarted debug inputs = %#v", second.Inputs)
	}
	if _, err := application.CancelRun(context.Background(), runID); err != nil {
		t.Fatal(err)
	}
	deadline := time.After(5 * time.Second)
	for {
		select {
		case event := <-events:
			if event.RunID == runID && event.Status == run.StatusCancelled {
				snapshot, snapshotErr := application.GetDebugSnapshot(runID)
				if snapshotErr != nil || snapshot.Status != compiler.DebugCompleted || snapshot.RunStatus != string(run.StatusCancelled) {
					t.Fatalf("completed debug snapshot = %#v, %v", snapshot, snapshotErr)
				}
				return
			}
		case <-deadline:
			t.Fatal("debug Run did not cancel")
		}
	}
}

func waitApplicationDebug(t *testing.T, events <-chan appcore.DebugEvent, runID string, matches func(compiler.DebugSnapshot) bool) compiler.DebugSnapshot {
	t.Helper()
	deadline := time.After(5 * time.Second)
	var last appcore.DebugEvent
	for {
		select {
		case event := <-events:
			last = event
			if event.RunID == runID && matches(event.Snapshot) {
				return event.Snapshot
			}
		case <-deadline:
			t.Fatalf("debug event did not reach expected state; last=%#v", last)
		}
	}
}

func newTestApplication(t *testing.T, now time.Time, adapterOverride compiler.Adapter) (*appcore.Application, *workflowstore.SourceStore, *workflowstore.ProgramStore, nodes.Builtins, chan appcore.RunEvent, chan appcore.DebugEvent) {
	t.Helper()
	builtins, err := nodes.Build()
	if err != nil {
		t.Fatal(err)
	}
	projection, err := nodeauthoring.Project(nodeauthoring.Input{
		Catalog: builtins.Catalog, Types: builtins.Types, Capabilities: builtins.Capabilities,
		Contracts: builtins.Contracts, GeneratorVersion: nodes.GeneratorVersion,
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
	programs, err := workflowstore.OpenProgramStore(filepath.Join(root, "programs"), builtins.Catalog, builtins.ConfigValidators, build, workflowstore.ProgramStoreOptions{MaxPrograms: 8})
	if err != nil {
		t.Fatal(err)
	}
	runs, err := run.OpenStore(filepath.Join(root, "runs"), builtins.Catalog, run.StoreOptions{MaxRecords: 8})
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
	adapters, err := noderuntime.Installed(builtins, noderuntime.Dependencies{
		Script: applicationScriptRuntime{},
		Log:    noderuntime.LogEmitterFunc(func(context.Context, noderuntime.LogEntry) error { return nil }),
	})
	if err != nil {
		t.Fatal(err)
	}
	if adapterOverride != nil {
		entry, ok := builtins.Catalog.Lookup(nodes.ConcatNodeID)
		if !ok {
			t.Fatal("Concat implementation is missing")
		}
		adapters[entry.Implementation.Entrypoint] = compiler.InstalledAdapter{Implementation: entry.Implementation, Run: adapterOverride}
	}
	executor := compiler.NewExecutor(builtins.Catalog, adapters, compiler.ExecutorOptions{Now: func() time.Time { return now }})
	events := make(chan appcore.RunEvent, 16)
	debugEvents := make(chan appcore.DebugEvent, 32)
	application, err := appcore.New(appcore.Config{
		Catalog: builtins.Catalog, Authoring: projection, CompilerBuild: build, ConfigValidators: builtins.ConfigValidators,
		Sources: sources, Programs: programs, Runs: runs,
		Admitter: admitter, Executor: executor, Providers: map[string]run.InstalledProvider{},
		ResourceOptions: resource.Options{Now: func() time.Time { return now }}, OwnerCloseTimeout: time.Second,
		Now: func() time.Time { return now }, OnRunEvent: func(event appcore.RunEvent) { events <- event },
		OnDebugEvent: func(event appcore.DebugEvent) { debugEvents <- event },
	})
	if err != nil {
		t.Fatal(err)
	}
	return application, sources, programs, builtins, events, debugEvents
}

type applicationScriptRuntime struct{}

func (applicationScriptRuntime) Execute(context.Context, scriptengine.Request) (scriptengine.Response, error) {
	return scriptengine.Response{}, errors.New("unexpected script execution in application test")
}

func createConcatWorkflow(t *testing.T, application *appcore.Application) appcore.ApplyPatchResult {
	t.Helper()
	created, err := application.CreateSource(context.Background(), "Application")
	if err != nil {
		t.Fatal(err)
	}
	result, err := application.ApplyPatch(context.Background(), authoring.PatchRequest{
		WorkflowID: created.WorkflowID(), BaseRevision: created.Revision(),
		Commands: []authoring.Command{
			{Kind: authoring.CommandAddNode, AddNode: &authoring.AddNodeCommand{
				GraphID: "main", NodeTypeID: nodes.ConcatNodeID, Handle: "concat", Position: schema.Position{},
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
