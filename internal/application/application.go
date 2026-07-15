// Package application owns the Yotta 3.1 command surface and its single local
// Run worker. GUI, CLI, MCP, schedules, and hotkeys call this package instead
// of constructing execution runtimes themselves.
package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/yottaapp/yotta/internal/admission"
	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/nodeauthoring"
	"github.com/yottaapp/yotta/internal/nodecatalog"
	"github.com/yottaapp/yotta/internal/resource"
	run31 "github.com/yottaapp/yotta/internal/run"
	"github.com/yottaapp/yotta/internal/workflow/authoring"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
	"github.com/yottaapp/yotta/internal/workflow/schema"
	"github.com/yottaapp/yotta/internal/workflowstore"
)

var (
	ErrNotStarted = errors.New("application is not started")
	ErrClosed     = errors.New("application is closed")
)

type Config struct {
	Catalog           nodecatalog.Snapshot
	Authoring         nodeauthoring.Snapshot
	CompilerBuild     artifact.Digest
	Sources           *workflowstore.SourceStore
	Programs          *workflowstore.ProgramStore
	Runs              *run31.Store
	Admitter          *admission.Admitter
	Executor          *compiler.Executor
	Providers         map[string]run31.InstalledProvider
	ResourceOptions   resource.Options
	OwnerCloseTimeout time.Duration
	Now               func() time.Time
	OnRunEvent        func(RunEvent)
}

type RunEvent struct {
	RunID      string
	Status     run31.Status
	Generation uint64
	Digest     artifact.Digest
	Err        error
}

type StartRunRequest struct {
	WorkflowID string
	Principal  string
	Selection  admission.Selection
}

type StartRunResult struct {
	SourceHash  artifact.Digest
	ProgramHash artifact.Digest
	Diagnostics []schema.Diagnostic
	Record      run31.Record
}

type ApplyPatchResult struct {
	Source         workflowstore.SourceSnapshot
	GeneratedNodes []authoring.GeneratedNode
}

type RunPreview struct {
	SourceHash     artifact.Digest        `json:"sourceHash,omitempty"`
	ProgramHash    artifact.Digest        `json:"programHash,omitempty"`
	Diagnostics    []schema.Diagnostic    `json:"diagnostics"`
	CapabilityPlan []capability.PlanEntry `json:"capabilityPlan"`
}

type jobState uint8

const (
	jobQueued jobState = iota + 1
	jobRunning
)

type runJob struct {
	state  jobState
	cancel context.CancelFunc
}

type lifecycleState uint8

const (
	stateNew lifecycleState = iota
	stateRunning
	stateClosed
)

type Application struct {
	catalog           nodecatalog.Snapshot
	authoring         nodeauthoring.Snapshot
	authoringEngine   *authoring.Engine
	compiler          *compiler.Compiler
	sources           *workflowstore.SourceStore
	programs          *workflowstore.ProgramStore
	runs              *run31.Store
	admitter          *admission.Admitter
	executor          *compiler.Executor
	providers         map[string]run31.InstalledProvider
	resourceOptions   resource.Options
	ownerCloseTimeout time.Duration
	now               func() time.Time
	onRunEvent        func(RunEvent)

	commandMu sync.RWMutex
	mu        sync.Mutex
	state     lifecycleState
	ctx       context.Context
	cancel    context.CancelFunc
	wake      chan struct{}
	queue     []string
	jobs      map[string]*runJob
	worker    sync.WaitGroup
}

func New(config Config) (*Application, error) {
	if !config.Catalog.Valid() || !config.Authoring.Valid() || config.Authoring.CatalogHash() != config.Catalog.Hash() ||
		!config.CompilerBuild.Valid() || config.Sources == nil || config.Programs == nil ||
		config.Runs == nil || config.Admitter == nil || config.Executor == nil || config.OwnerCloseTimeout <= 0 {
		return nil, errors.New("application requires trusted contracts, stores, admission, executor, and owner timeout")
	}
	authoringEngine, err := authoring.New(config.Catalog, config.Authoring, nil)
	if err != nil {
		return nil, fmt.Errorf("construct authoring engine: %w", err)
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	providers := make(map[string]run31.InstalledProvider, len(config.Providers))
	for id, provider := range config.Providers {
		providers[id] = provider
	}
	return &Application{
		catalog: config.Catalog, authoring: config.Authoring, authoringEngine: authoringEngine,
		compiler: compiler.New(config.CompilerBuild), sources: config.Sources,
		programs: config.Programs, runs: config.Runs, admitter: config.Admitter, executor: config.Executor,
		providers: providers, resourceOptions: config.ResourceOptions, ownerCloseTimeout: config.OwnerCloseTimeout,
		now: config.Now, onRunEvent: config.OnRunEvent, state: stateNew, wake: make(chan struct{}, 1), jobs: make(map[string]*runJob),
	}, nil
}

// Start recovers stale process-owned states before accepting commands. A
// running Run is interrupted and an undelivered queued Run is cancelled; no
// effect is replayed from a guessed notification state.
func (a *Application) Start(ctx context.Context) error {
	if ctx == nil {
		return errors.New("application start context is required")
	}
	a.commandMu.Lock()
	a.mu.Lock()
	if a.state == stateClosed {
		a.mu.Unlock()
		a.commandMu.Unlock()
		return ErrClosed
	}
	if a.state == stateRunning {
		a.mu.Unlock()
		a.commandMu.Unlock()
		return nil
	}
	a.mu.Unlock()
	now := a.now().UTC()
	interrupted, err := a.runs.InterruptRunning(ctx, now)
	if err != nil {
		a.commandMu.Unlock()
		return fmt.Errorf("interrupt stale Runs: %w", err)
	}
	cancelled, err := a.runs.CancelQueued(ctx, now)
	if err != nil {
		a.commandMu.Unlock()
		return fmt.Errorf("cancel stale queued Runs: %w", err)
	}
	a.mu.Lock()
	if a.state != stateNew {
		a.mu.Unlock()
		a.commandMu.Unlock()
		return ErrClosed
	}
	a.ctx, a.cancel = context.WithCancel(context.Background())
	a.state = stateRunning
	a.worker.Add(1)
	go a.workerLoop()
	a.mu.Unlock()
	a.commandMu.Unlock()
	for _, record := range append(interrupted, cancelled...) {
		a.emit(record, nil)
	}
	return nil
}

// CompileSource compiles one durable revision. External authoring clients use
// this command instead of submitting an untrusted replacement document.
func (a *Application) CompileSource(ctx context.Context, workflowID string) (compiler.CompileResult, error) {
	if ctx == nil {
		return compiler.CompileResult{}, errors.New("compile Workflow Source context is required")
	}
	a.commandMu.RLock()
	defer a.commandMu.RUnlock()
	if err := a.requireRunning(); err != nil {
		return compiler.CompileResult{}, err
	}
	source, err := a.sources.Load(workflowID)
	if err != nil {
		return compiler.CompileResult{}, err
	}
	return a.compiler.CompileDraft(ctx, compiler.CompileRequest{SourceJSON: source.Artifact(), Catalog: a.catalog})
}

// ApplyPatch is the sole external mutation path for an existing Workflow
// Source. The command batch is reduced atomically and persisted with revision
// CAS; a failed command never publishes a partial source.
func (a *Application) ApplyPatch(ctx context.Context, request authoring.PatchRequest) (ApplyPatchResult, error) {
	if ctx == nil {
		return ApplyPatchResult{}, errors.New("apply Workflow patch context is required")
	}
	a.commandMu.RLock()
	defer a.commandMu.RUnlock()
	if err := a.requireRunning(); err != nil {
		return ApplyPatchResult{}, err
	}
	snapshot, err := a.sources.Load(request.WorkflowID)
	if err != nil {
		return ApplyPatchResult{}, err
	}
	if snapshot.Revision() != request.BaseRevision {
		return ApplyPatchResult{}, workflowstore.ErrSourceConflict
	}
	source, diagnostics := schema.ParseSource(snapshot.Artifact())
	if len(diagnostics) != 0 {
		return ApplyPatchResult{}, errors.New("stored Workflow Source failed strict reopen")
	}
	applied, err := a.authoringEngine.Apply(source, request.Commands)
	if err != nil {
		return ApplyPatchResult{}, err
	}
	next, saveErr := a.sources.Save(ctx, applied.Artifact, request.BaseRevision)
	return ApplyPatchResult{
		Source: next, GeneratedNodes: append([]authoring.GeneratedNode(nil), applied.GeneratedNodes...),
	}, saveErr
}

// PreviewRun performs the exact stored-source compilation used by StartRun and
// returns the frozen capability requirements without admission or effects.
func (a *Application) PreviewRun(ctx context.Context, workflowID string) (RunPreview, error) {
	compiled, err := a.CompileSource(ctx, workflowID)
	preview := RunPreview{
		SourceHash:     compiled.SourceHash,
		Diagnostics:    append([]schema.Diagnostic(nil), compiled.Diagnostics...),
		CapabilityPlan: []capability.PlanEntry{},
	}
	if err != nil {
		return preview, err
	}
	if program, ok := compiled.Program(); ok {
		preview.ProgramHash = program.Hash()
		preview.CapabilityPlan = program.CapabilityPlan().Entries()
	}
	return preview, nil
}

// CreateSource creates the only valid empty authoring root. IDs and structural
// defaults are host-owned so UI, CLI, and MCP cannot invent divergent source
// envelopes.
func (a *Application) CreateSource(ctx context.Context, name string) (workflowstore.SourceSnapshot, error) {
	if ctx == nil {
		return workflowstore.SourceSnapshot{}, errors.New("create Workflow Source context is required")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return workflowstore.SourceSnapshot{}, errors.New("workflow name is required")
	}
	a.commandMu.RLock()
	defer a.commandMu.RUnlock()
	if err := a.requireRunning(); err != nil {
		return workflowstore.SourceSnapshot{}, err
	}
	source := schema.WorkflowSource{
		Format: schema.Format, Version: schema.Version,
		Workflow: schema.Workflow{ID: uuid.NewString(), Name: name}, Revision: 0, EntryGraph: "main",
		Graphs:    []schema.Graph{{ID: "main", Kind: schema.GraphKindMain, Nodes: []schema.Node{}, Edges: []schema.Edge{}, Inputs: []schema.GraphPort{}, Outputs: []schema.GraphPort{}}},
		Variables: []schema.Variable{}, SecretRefs: []schema.SecretRef{},
	}
	raw, err := artifact.Marshal(source)
	if err != nil {
		return workflowstore.SourceSnapshot{}, err
	}
	return a.sources.Save(ctx, raw, -1)
}

func (a *Application) StartRun(ctx context.Context, request StartRunRequest) (StartRunResult, error) {
	if ctx == nil {
		return StartRunResult{}, errors.New("start Run context is required")
	}
	a.commandMu.RLock()
	defer a.commandMu.RUnlock()
	if err := a.requireRunning(); err != nil {
		return StartRunResult{}, err
	}
	source, err := a.sources.Load(request.WorkflowID)
	if err != nil {
		return StartRunResult{}, err
	}
	compiled, err := a.compiler.CompileDraft(ctx, compiler.CompileRequest{SourceJSON: source.Artifact(), Catalog: a.catalog})
	result := StartRunResult{SourceHash: compiled.SourceHash, Diagnostics: append([]schema.Diagnostic(nil), compiled.Diagnostics...)}
	if err != nil || schema.HasErrors(compiled.Diagnostics) {
		return result, err
	}
	program, ok := compiled.Program()
	if !ok {
		return result, errors.New("compiler returned no Program without diagnostics")
	}
	result.ProgramHash = program.Hash()
	if err := a.programs.Put(ctx, program); err != nil {
		return result, fmt.Errorf("persist Program before admission: %w", err)
	}
	admitted, err := a.admitter.Admit(ctx, admission.Request{
		Program: program, Principal: request.Principal, Selection: request.Selection,
	})
	result.Record = admitted.Record
	if err != nil {
		return result, err
	}
	runID := admitted.Record.Admission().RunID
	a.mu.Lock()
	if a.state != stateRunning {
		a.mu.Unlock()
		return result, ErrClosed
	}
	a.jobs[runID] = &runJob{state: jobQueued}
	a.queue = append(a.queue, runID)
	a.mu.Unlock()
	a.signalWorker()
	a.emit(admitted.Record, nil)
	return result, nil
}

func (a *Application) GetRun(runID string) (run31.Record, error) { return a.runs.Load(runID) }

func (a *Application) GetSource(workflowID string) (workflowstore.SourceSnapshot, error) {
	return a.sources.Load(workflowID)
}

func (a *Application) ListSources() []workflowstore.SourceSnapshot { return a.sources.List() }

func (a *Application) CatalogArtifact() []byte { return a.catalog.Bytes() }

func (a *Application) AuthoringProjection() nodeauthoring.Snapshot { return a.authoring }

func (a *Application) CancelRun(ctx context.Context, runID string) (run31.Record, error) {
	if ctx == nil {
		return run31.Record{}, errors.New("cancel Run context is required")
	}
	a.mu.Lock()
	job := a.jobs[runID]
	if job != nil && job.state == jobRunning {
		job.cancel()
		a.mu.Unlock()
		return a.runs.Load(runID)
	}
	if job != nil {
		delete(a.jobs, runID)
		a.removeQueuedLocked(runID)
	}
	a.mu.Unlock()
	current, err := a.runs.Load(runID)
	if err != nil || current.Status() != run31.StatusQueued {
		return current, err
	}
	next, err := current.Cancel(a.transitionTime(current.Admission().QueuedAt))
	if err != nil {
		return run31.Record{}, err
	}
	if err := a.runs.Update(ctx, current.Digest(), next); err != nil {
		return run31.Record{}, err
	}
	a.emit(next, nil)
	return next, nil
}

func (a *Application) CancelAll(ctx context.Context) error {
	if ctx == nil {
		return errors.New("cancel all Runs context is required")
	}
	a.commandMu.RLock()
	defer a.commandMu.RUnlock()
	if err := a.requireRunning(); err != nil {
		return err
	}
	a.mu.Lock()
	runIDs := make([]string, 0, len(a.jobs))
	for runID := range a.jobs {
		runIDs = append(runIDs, runID)
	}
	a.mu.Unlock()
	var cancelErr error
	for _, runID := range runIDs {
		if _, err := a.CancelRun(ctx, runID); err != nil && !errors.Is(err, run31.ErrRunConflict) {
			cancelErr = errors.Join(cancelErr, err)
		}
	}
	return cancelErr
}

func (a *Application) Close(ctx context.Context) error {
	if ctx == nil {
		return errors.New("application close context is required")
	}
	a.commandMu.Lock()
	a.mu.Lock()
	if a.state == stateClosed {
		a.mu.Unlock()
		a.commandMu.Unlock()
		return nil
	}
	if a.state == stateNew {
		a.state = stateClosed
		a.mu.Unlock()
		a.commandMu.Unlock()
		return nil
	}
	a.state = stateClosed
	queued := append([]string(nil), a.queue...)
	a.queue = nil
	for runID, job := range a.jobs {
		if job.state == jobRunning && job.cancel != nil {
			job.cancel()
		} else {
			delete(a.jobs, runID)
		}
	}
	a.cancel()
	a.mu.Unlock()
	a.commandMu.Unlock()
	var closeErr error
	for _, runID := range queued {
		if _, err := a.CancelRun(ctx, runID); err != nil && !errors.Is(err, run31.ErrRunConflict) {
			closeErr = errors.Join(closeErr, err)
		}
	}
	done := make(chan struct{})
	go func() { a.worker.Wait(); close(done) }()
	select {
	case <-done:
		return closeErr
	case <-ctx.Done():
		return errors.Join(closeErr, ctx.Err())
	}
}

func (a *Application) workerLoop() {
	defer a.worker.Done()
	for {
		runID, ctx, ok := a.nextJob()
		if !ok {
			return
		}
		err := a.execute(ctx, runID)
		a.mu.Lock()
		delete(a.jobs, runID)
		a.mu.Unlock()
		record, loadErr := a.runs.Load(runID)
		if loadErr == nil {
			a.emit(record, err)
		} else {
			a.emitRunID(runID, errors.Join(err, loadErr))
		}
	}
}

func (a *Application) nextJob() (string, context.Context, bool) {
	for {
		a.mu.Lock()
		if len(a.queue) != 0 {
			runID := a.queue[0]
			a.queue = a.queue[1:]
			job := a.jobs[runID]
			if job == nil {
				a.mu.Unlock()
				continue
			}
			jobCtx, cancel := context.WithCancel(a.ctx)
			job.state, job.cancel = jobRunning, cancel
			a.mu.Unlock()
			return runID, jobCtx, true
		}
		ctx := a.ctx
		a.mu.Unlock()
		select {
		case <-a.wake:
		case <-ctx.Done():
			return "", nil, false
		}
	}
}

func (a *Application) execute(ctx context.Context, runID string) error {
	record, err := a.runs.Load(runID)
	if err != nil || record.Status() != run31.StatusQueued {
		return errors.Join(err, errors.New("worker requires queued Run"))
	}
	program, bootstrapErr := a.programs.Load(record.Admission().ProgramHash)
	var grant capability.RunGrant
	if bootstrapErr == nil {
		grant, bootstrapErr = capability.OpenRunGrant(record.GrantArtifact(), program.CapabilityPlan(), a.catalog)
	}
	if ctx.Err() != nil {
		next, cancelErr := record.Cancel(a.transitionTime(record.Admission().QueuedAt))
		if cancelErr == nil {
			cancelErr = a.runs.Update(context.WithoutCancel(ctx), record.Digest(), next)
		}
		return errors.Join(ctx.Err(), cancelErr)
	}
	running, err := record.Start(a.transitionTime(record.Admission().QueuedAt))
	if err != nil {
		return err
	}
	if err := a.runs.Update(context.WithoutCancel(ctx), record.Digest(), running); err != nil {
		return err
	}
	journal, err := a.runs.OpenJournal(runID)
	if err != nil {
		return err
	}
	if ctx.Err() != nil {
		_, terminalErr := journal.Cancel(context.WithoutCancel(ctx), a.transitionTime(running.Admission().QueuedAt))
		return errors.Join(ctx.Err(), terminalErr)
	}
	if bootstrapErr != nil {
		_, terminalErr := journal.Fail(context.WithoutCancel(ctx), a.transitionTime(running.Admission().QueuedAt), run31.RunError{
			Code: "runtime.bootstrap_failed", Category: run31.ErrorCategoryInfrastructure,
		})
		return errors.Join(bootstrapErr, terminalErr)
	}
	owner, err := run31.NewOwner(ctx, grant, a.providers, a.resourceOptions)
	if err != nil {
		if ctx.Err() != nil {
			_, terminalErr := journal.Cancel(context.WithoutCancel(ctx), a.transitionTime(running.Admission().QueuedAt))
			return errors.Join(ctx.Err(), err, terminalErr)
		}
		_, terminalErr := journal.Fail(context.WithoutCancel(ctx), a.transitionTime(running.Admission().QueuedAt), run31.RunError{
			Code: "runtime.owner_failed", Category: run31.ErrorCategoryInfrastructure,
		})
		return errors.Join(err, terminalErr)
	}
	_, executionErr := a.executor.Run(ctx, program, owner, journal)
	closeCtx, cancel := context.WithTimeout(context.Background(), a.ownerCloseTimeout)
	closeErr := owner.Close(closeCtx)
	cancel()
	return errors.Join(executionErr, closeErr)
}

func (a *Application) transitionTime(notBefore time.Time) time.Time {
	now := a.now().UTC()
	if now.Before(notBefore) {
		return notBefore
	}
	return now
}

func (a *Application) requireRunning() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	switch a.state {
	case stateRunning:
		return nil
	case stateClosed:
		return ErrClosed
	default:
		return ErrNotStarted
	}
}

func (a *Application) signalWorker() {
	select {
	case a.wake <- struct{}{}:
	default:
	}
}

func (a *Application) removeQueuedLocked(runID string) {
	for index, queuedID := range a.queue {
		if queuedID == runID {
			a.queue = append(a.queue[:index], a.queue[index+1:]...)
			return
		}
	}
}

func (a *Application) emit(record run31.Record, err error) {
	if a.onRunEvent == nil || !record.Valid() {
		return
	}
	a.onRunEvent(RunEvent{
		RunID: record.Admission().RunID, Status: record.Status(), Generation: record.Generation(), Digest: record.Digest(), Err: err,
	})
}

func (a *Application) emitRunID(runID string, err error) {
	if a.onRunEvent != nil {
		a.onRunEvent(RunEvent{RunID: runID, Err: err})
	}
}
