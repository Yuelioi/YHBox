// Package application owns the Yotta command surface and its single local
// Run worker. GUI, CLI, MCP, schedules, and hotkeys call this package instead
// of constructing execution runtimes themselves.
package application

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/yottaapp/yotta/internal/admission"
	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/configvalidator"
	"github.com/yottaapp/yotta/internal/nodeauthoring"
	"github.com/yottaapp/yotta/internal/nodecatalog"
	"github.com/yottaapp/yotta/internal/nodes"
	"github.com/yottaapp/yotta/internal/resource"
	run "github.com/yottaapp/yotta/internal/run"
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
	ConfigValidators  configvalidator.Registry
	BlobVerifier      compiler.BlobVerifier
	Sources           *workflowstore.SourceStore
	Programs          *workflowstore.ProgramStore
	Runs              *run.Store
	Admitter          *admission.Admitter
	Executor          *compiler.Executor
	Providers         map[string]run.InstalledProvider
	ProviderLease     func() (func(), error)
	ResourceOptions   resource.Options
	OwnerCloseTimeout time.Duration
	Now               func() time.Time
	OnRunEvent        func(RunEvent)
	OnDebugEvent      func(DebugEvent)
}

type RunEvent struct {
	RunID      string
	Status     run.Status
	Generation uint64
	Digest     artifact.Digest
	Err        error
}

type DebugEvent struct {
	RunID    string
	Snapshot compiler.DebugSnapshot
}

type DebugAction string

const (
	DebugContinue DebugAction = "continue"
	DebugPause    DebugAction = "pause"
	DebugStep     DebugAction = "step"
)

const MaxDebugSessions = 128

type StartRunRequest struct {
	WorkflowID string
	Principal  string
	Selection  admission.Selection
}

type StartArtifactRunRequest struct {
	SourceArtifact []byte
	Principal      string
	Selection      admission.Selection
}

type StartRunResult struct {
	SourceHash  artifact.Digest
	ProgramHash artifact.Digest
	Diagnostics []schema.Diagnostic
	Record      run.Record
}

type ApplyPatchResult struct {
	Source         workflowstore.SourceSnapshot
	GeneratedNodes []authoring.GeneratedNode
}

// UnsafeStateMigrationError reports compiler errors introduced by changing a
// referenced state variable. The candidate is never persisted.
type UnsafeStateMigrationError struct {
	Diagnostics []schema.Diagnostic
}

func (e *UnsafeStateMigrationError) Error() string {
	if e == nil || len(e.Diagnostics) == 0 {
		return "state type migration is unsafe"
	}
	first := e.Diagnostics[0]
	location := strings.Join(first.GraphPath, "/")
	if first.NodeID != "" {
		if location != "" {
			location += "/"
		}
		location += first.NodeID
	}
	if location == "" {
		return fmt.Sprintf("state type migration introduces %s", first.Code)
	}
	return fmt.Sprintf("state type migration introduces %s at %s", first.Code, location)
}

// PreparedPatch is an application-sealed, exact Workflow Source transition.
// Its state is intentionally opaque: presentation and AI clients can review a
// candidate, but only Application can create or commit one.
type PreparedPatch struct{ state *preparedPatchState }

type preparedPatchState struct {
	workflowID           string
	baseRevision         int64
	baseHash             artifact.Digest
	candidate            []byte
	candidateHash        artifact.Digest
	generated            []authoring.GeneratedNode
	unsafeStateMigration []schema.Diagnostic
}

func (p PreparedPatch) Valid() bool {
	return p.state != nil && p.state.workflowID != "" && p.state.baseRevision >= 0 &&
		p.state.baseHash.Valid() && p.state.candidateHash.Valid() && len(p.state.candidate) != 0
}

func (p PreparedPatch) WorkflowID() string {
	if !p.Valid() {
		return ""
	}
	return p.state.workflowID
}

func (p PreparedPatch) BaseRevision() int64 {
	if !p.Valid() {
		return -1
	}
	return p.state.baseRevision
}

func (p PreparedPatch) BaseHash() artifact.Digest {
	if !p.Valid() {
		return ""
	}
	return p.state.baseHash
}

func (p PreparedPatch) CandidateHash() artifact.Digest {
	if !p.Valid() {
		return ""
	}
	return p.state.candidateHash
}

func (p PreparedPatch) CandidateArtifact() []byte {
	if !p.Valid() {
		return nil
	}
	return append([]byte(nil), p.state.candidate...)
}

func (p PreparedPatch) GeneratedNodes() []authoring.GeneratedNode {
	if !p.Valid() {
		return nil
	}
	return append([]authoring.GeneratedNode(nil), p.state.generated...)
}

type PreparePatchResult struct {
	Patch          PreparedPatch
	Diagnostics    []schema.Diagnostic
	CapabilityPlan []capability.PlanEntry
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
	workflowID string
	state      jobState
	cancel     context.CancelFunc
	providers  map[string]run.InstalledProvider
	release    func()
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
	blobVerifier      compiler.BlobVerifier
	sources           *workflowstore.SourceStore
	programs          *workflowstore.ProgramStore
	runs              *run.Store
	admitter          *admission.Admitter
	executor          *compiler.Executor
	providers         map[string]run.InstalledProvider
	providerLease     func() (func(), error)
	resourceOptions   resource.Options
	ownerCloseTimeout time.Duration
	now               func() time.Time
	onRunEvent        func(RunEvent)
	onDebugEvent      func(DebugEvent)

	commandMu sync.RWMutex
	mu        sync.Mutex
	state     lifecycleState
	ctx       context.Context
	cancel    context.CancelFunc
	wake      chan struct{}
	queue     []string
	jobs      map[string]*runJob
	debug     map[string]*compiler.DebugController
	worker    sync.WaitGroup
}

// ReplaceExecutionEnvironment atomically switches admission and the provider
// generation used by future Runs. Already queued/running jobs keep the exact
// provider snapshot and lease acquired with their admission generation.
func (a *Application) ReplaceExecutionEnvironment(profile admission.HostProfile, policy admission.Policy, providers map[string]run.InstalledProvider, providerLease func() (func(), error)) error {
	if a == nil || !profile.Valid() || policy == nil || providers == nil {
		return errors.New("replacement execution environment is invalid")
	}
	next := make(map[string]run.InstalledProvider, len(providers))
	for id, provider := range providers {
		if id == "" || !provider.ArtifactDigest.Valid() || provider.ABI == "" || provider.Provider == nil {
			return errors.New("replacement execution environment contains an invalid provider")
		}
		next[id] = provider
	}
	a.commandMu.Lock()
	defer a.commandMu.Unlock()
	a.mu.Lock()
	closed := a.state == stateClosed
	a.mu.Unlock()
	if closed {
		return ErrClosed
	}
	if err := a.admitter.ReplaceEnvironment(profile, policy); err != nil {
		return err
	}
	a.providers = next
	a.providerLease = providerLease
	return nil
}

func (a *Application) acquireProviderSnapshot() (map[string]run.InstalledProvider, func(), error) {
	release := func() {}
	if a.providerLease != nil {
		var err error
		release, err = a.providerLease()
		if err != nil {
			return nil, nil, err
		}
		if release == nil {
			return nil, nil, errors.New("execution environment returned an invalid provider lease")
		}
	}
	providers := make(map[string]run.InstalledProvider, len(a.providers))
	for id, provider := range a.providers {
		providers[id] = provider
	}
	return providers, release, nil
}

func New(config Config) (*Application, error) {
	if !config.Catalog.Valid() || !config.Authoring.Valid() || config.Authoring.CatalogHash() != config.Catalog.Hash() ||
		!config.CompilerBuild.Valid() || !config.ConfigValidators.Valid() || config.BlobVerifier == nil || config.Sources == nil || config.Programs == nil ||
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
	providers := make(map[string]run.InstalledProvider, len(config.Providers))
	for id, provider := range config.Providers {
		providers[id] = provider
	}
	return &Application{
		catalog: config.Catalog, authoring: config.Authoring, authoringEngine: authoringEngine,
		compiler: compiler.New(config.CompilerBuild, config.ConfigValidators), blobVerifier: config.BlobVerifier, sources: config.Sources,
		programs: config.Programs, runs: config.Runs, admitter: config.Admitter, executor: config.Executor,
		providers: providers, providerLease: config.ProviderLease, resourceOptions: config.ResourceOptions, ownerCloseTimeout: config.OwnerCloseTimeout,
		now: config.Now, onRunEvent: config.OnRunEvent, onDebugEvent: config.OnDebugEvent,
		state: stateNew, wake: make(chan struct{}, 1), jobs: make(map[string]*runJob), debug: make(map[string]*compiler.DebugController),
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
	return a.compileDraft(ctx, source.Artifact())
}

// CompileDraft validates and compiles an in-memory Source through the same
// trusted Catalog/compiler path without publishing a Source or Program.
func (a *Application) CompileDraft(ctx context.Context, sourceJSON []byte) (compiler.CompileResult, error) {
	if ctx == nil {
		return compiler.CompileResult{}, errors.New("compile Workflow Source context is required")
	}
	a.commandMu.RLock()
	defer a.commandMu.RUnlock()
	if err := a.requireRunning(); err != nil {
		return compiler.CompileResult{}, err
	}
	return a.compileDraft(ctx, append([]byte(nil), sourceJSON...))
}

func (a *Application) compileDraft(ctx context.Context, sourceJSON []byte) (compiler.CompileResult, error) {
	return a.compiler.CompileDraft(ctx, compiler.CompileRequest{
		SourceJSON: sourceJSON, Catalog: a.catalog, BlobVerifier: a.blobVerifier,
	})
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
	candidateSource, candidateArtifact, err := stampWorkflowUpdate(applied.Source, a.now())
	if err != nil {
		return ApplyPatchResult{}, err
	}
	if referencedStateUpdate(source, candidateSource, request.Commands) {
		baseline, err := a.compileDraft(ctx, snapshot.Artifact())
		if err != nil {
			return ApplyPatchResult{}, fmt.Errorf("compile state migration baseline: %w", err)
		}
		candidate, err := a.compileDraft(ctx, candidateArtifact)
		if err != nil {
			return ApplyPatchResult{}, fmt.Errorf("compile state migration candidate: %w", err)
		}
		if introduced := introducedErrorDiagnostics(baseline.Diagnostics, candidate.Diagnostics); len(introduced) != 0 {
			return ApplyPatchResult{}, &UnsafeStateMigrationError{Diagnostics: introduced}
		}
	}
	next, saveErr := a.sources.Save(ctx, candidateArtifact, request.BaseRevision)
	return ApplyPatchResult{
		Source: next, GeneratedNodes: append([]authoring.GeneratedNode(nil), applied.GeneratedNodes...),
	}, saveErr
}

func referencedStateUpdate(before, after schema.WorkflowSource, commands []authoring.Command) bool {
	updated := make(map[string]bool)
	for _, command := range commands {
		if command.Kind == authoring.CommandUpdateStateVariable && command.UpdateStateVariable != nil {
			updated[command.UpdateStateVariable.Name] = true
		}
	}
	for name := range updated {
		if workflowReferencesState(before, name) || workflowReferencesState(after, name) {
			return true
		}
	}
	return false
}

func workflowReferencesState(source schema.WorkflowSource, name string) bool {
	for _, graph := range source.Graphs {
		for _, node := range graph.Nodes {
			if strings.Contains(node.NodeRef.NodeTypeID, "/nodes/state/") && node.Config["variable"] == name {
				return true
			}
		}
	}
	return false
}

func introducedErrorDiagnostics(baseline, candidate []schema.Diagnostic) []schema.Diagnostic {
	counts := make(map[string]int)
	for _, diagnostic := range baseline {
		if diagnostic.Severity == schema.SeverityError {
			counts[diagnosticIdentity(diagnostic)]++
		}
	}
	introduced := make([]schema.Diagnostic, 0)
	for _, diagnostic := range candidate {
		if diagnostic.Severity != schema.SeverityError {
			continue
		}
		key := diagnosticIdentity(diagnostic)
		if counts[key] > 0 {
			counts[key]--
			continue
		}
		introduced = append(introduced, diagnostic)
	}
	return introduced
}

func diagnosticIdentity(diagnostic schema.Diagnostic) string {
	return strings.Join([]string{
		diagnostic.Code,
		strings.Join(diagnostic.GraphPath, "\x1f"),
		diagnostic.NodeID,
		strings.Join(diagnostic.FieldPath, "\x1f"),
	}, "\x1e")
}

// PreparePatch reduces and compiles an exact candidate without publishing it.
// The returned opaque patch can later be committed once, provided the durable
// base revision and hash are still unchanged.
func (a *Application) PreparePatch(ctx context.Context, request authoring.PatchRequest) (PreparePatchResult, error) {
	if ctx == nil {
		return PreparePatchResult{}, errors.New("prepare Workflow patch context is required")
	}
	a.commandMu.RLock()
	defer a.commandMu.RUnlock()
	if err := a.requireRunning(); err != nil {
		return PreparePatchResult{}, err
	}
	snapshot, err := a.sources.Load(request.WorkflowID)
	if err != nil {
		return PreparePatchResult{}, err
	}
	if snapshot.Revision() != request.BaseRevision {
		return PreparePatchResult{}, workflowstore.ErrSourceConflict
	}
	source, diagnostics := schema.ParseSource(snapshot.Artifact())
	if len(diagnostics) != 0 {
		return PreparePatchResult{}, errors.New("stored Workflow Source failed strict reopen")
	}
	applied, err := a.authoringEngine.Apply(source, request.Commands)
	if err != nil {
		return PreparePatchResult{}, err
	}
	candidateSource, candidateArtifact, err := stampWorkflowUpdate(applied.Source, a.now())
	if err != nil {
		return PreparePatchResult{}, err
	}
	_, _, candidateHash, candidateDiagnostics, err := schema.CanonicalSource(candidateArtifact)
	if err != nil || len(candidateDiagnostics) != 0 {
		return PreparePatchResult{}, errors.New("prepared Workflow Source failed strict reopen")
	}
	prepared := PreparedPatch{state: &preparedPatchState{
		workflowID: request.WorkflowID, baseRevision: request.BaseRevision, baseHash: snapshot.Hash(),
		candidate: append([]byte(nil), candidateArtifact...), candidateHash: candidateHash,
		generated: append([]authoring.GeneratedNode(nil), applied.GeneratedNodes...),
	}}
	compiled, compileErr := a.compileDraft(ctx, candidateArtifact)
	if referencedStateUpdate(source, candidateSource, request.Commands) {
		baseline, err := a.compileDraft(ctx, snapshot.Artifact())
		if err != nil {
			return PreparePatchResult{}, fmt.Errorf("compile state migration baseline: %w", err)
		}
		prepared.state.unsafeStateMigration = introducedErrorDiagnostics(baseline.Diagnostics, compiled.Diagnostics)
	}
	result := PreparePatchResult{Patch: prepared, Diagnostics: append([]schema.Diagnostic(nil), compiled.Diagnostics...), CapabilityPlan: []capability.PlanEntry{}}
	if program, ok := compiled.Program(); ok {
		result.CapabilityPlan = program.CapabilityPlan().Entries()
	}
	return result, compileErr
}

// CommitPreparedPatch publishes the exact artifact that was reviewed. It does
// not replay commands, and therefore cannot silently change generated node IDs.
func (a *Application) CommitPreparedPatch(ctx context.Context, prepared PreparedPatch) (ApplyPatchResult, error) {
	if ctx == nil {
		return ApplyPatchResult{}, errors.New("commit prepared Workflow patch context is required")
	}
	if !prepared.Valid() {
		return ApplyPatchResult{}, errors.New("prepared Workflow patch is invalid")
	}
	if len(prepared.state.unsafeStateMigration) != 0 {
		return ApplyPatchResult{}, &UnsafeStateMigrationError{Diagnostics: append([]schema.Diagnostic(nil), prepared.state.unsafeStateMigration...)}
	}
	a.commandMu.RLock()
	defer a.commandMu.RUnlock()
	if err := a.requireRunning(); err != nil {
		return ApplyPatchResult{}, err
	}
	current, err := a.sources.Load(prepared.state.workflowID)
	if err != nil {
		return ApplyPatchResult{}, err
	}
	if current.Revision() != prepared.state.baseRevision || current.Hash() != prepared.state.baseHash {
		return ApplyPatchResult{}, workflowstore.ErrSourceConflict
	}
	next, saveErr := a.sources.Save(ctx, prepared.state.candidate, prepared.state.baseRevision)
	return ApplyPatchResult{Source: next, GeneratedNodes: prepared.GeneratedNodes()}, saveErr
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

// CreateSource creates the only valid authoring root. IDs and structural
// defaults are host-owned so UI, CLI, and MCP cannot invent divergent source
// envelopes.
func (a *Application) CreateSource(ctx context.Context, name string) (workflowstore.SourceSnapshot, error) {
	return a.CreateSourceWithMetadata(ctx, authoring.WorkflowMetadata{Name: name})
}

func (a *Application) CreateSourceWithMetadata(ctx context.Context, requested authoring.WorkflowMetadata) (workflowstore.SourceSnapshot, error) {
	if ctx == nil {
		return workflowstore.SourceSnapshot{}, errors.New("create Workflow Source context is required")
	}
	metadata, err := authoring.NormalizeWorkflowMetadata(requested)
	if err != nil {
		return workflowstore.SourceSnapshot{}, err
	}
	a.commandMu.RLock()
	defer a.commandMu.RUnlock()
	if err := a.requireRunning(); err != nil {
		return workflowstore.SourceSnapshot{}, err
	}
	runStarted, ok := a.catalog.Lookup(nodes.RunStartedNodeID)
	if !ok {
		return workflowstore.SourceSnapshot{}, errors.New("catalog is missing the RunStarted node")
	}
	source := schema.WorkflowSource{
		Format: schema.Format, Version: schema.Version,
		Workflow: schema.Workflow{
			ID: uuid.NewString(), Name: metadata.Name, Description: metadata.Description,
			Category: metadata.Category, Tags: metadata.Tags,
		},
		Revision: 0, EntryGraph: "main",
		Graphs: []schema.Graph{{
			ID: "main", Kind: schema.GraphKindMain,
			Nodes: []schema.Node{{
				ID: "run-started", NodeRef: runStarted.Contract.NodeRef(), Position: schema.Position{X: 120, Y: 160},
				Config: map[string]any{}, Bindings: map[string]schema.InputBinding{},
			}},
			Edges: []schema.Edge{}, Inputs: []schema.GraphPort{}, Outputs: []schema.GraphPort{},
		}},
		Resources: []schema.WorkflowResource{}, TargetProfileDefinitions: []schema.TargetProfileDefinition{},
		CredentialRequirements: []schema.CredentialRequirement{}, Dependencies: []schema.NodePackageDependency{}, Variables: []schema.Variable{},
	}
	timestamp := formatWorkflowTimestamp(a.now())
	source.Workflow.CreatedAt = timestamp
	source.Workflow.UpdatedAt = timestamp
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
	return a.startRun(ctx, request, false, nil)
}

// StartArtifactRun executes immutable Source bytes that another trusted
// module has already authorized. It shares the compiler, admission, ledger,
// provider, and worker path with local editable Source runs.
func (a *Application) StartArtifactRun(ctx context.Context, request StartArtifactRunRequest) (StartRunResult, error) {
	if ctx == nil {
		return StartRunResult{}, errors.New("start artifact Run context is required")
	}
	if len(request.SourceArtifact) == 0 {
		return StartRunResult{}, errors.New("start artifact Run requires Workflow Source bytes")
	}
	a.commandMu.RLock()
	defer a.commandMu.RUnlock()
	return a.startRunArtifact(ctx, request.SourceArtifact, request.Principal, request.Selection, "", false, nil)
}

func (a *Application) StartDebugRun(ctx context.Context, request StartRunRequest, breakpoints []compiler.DebugBreakpoint) (StartRunResult, error) {
	if ctx == nil {
		return StartRunResult{}, errors.New("start debug Run context is required")
	}
	a.commandMu.Lock()
	defer a.commandMu.Unlock()
	a.mu.Lock()
	for runID, control := range a.debug {
		if control.Snapshot().Status == compiler.DebugCompleted {
			delete(a.debug, runID)
		}
	}
	full := len(a.debug) >= MaxDebugSessions
	a.mu.Unlock()
	if full {
		return StartRunResult{}, errors.New("debug session budget exceeded")
	}
	if len(breakpoints) > compiler.MaxDebugQueueEntries {
		return StartRunResult{}, errors.New("debug breakpoint budget exceeded")
	}
	for _, breakpoint := range breakpoints {
		if breakpoint.GraphID == "" || breakpoint.NodeID == "" {
			return StartRunResult{}, errors.New("debug breakpoint requires graph and node")
		}
	}
	return a.startRun(ctx, request, true, breakpoints)
}

func (a *Application) startRun(ctx context.Context, request StartRunRequest, debug bool, breakpoints []compiler.DebugBreakpoint) (StartRunResult, error) {
	if err := a.requireRunning(); err != nil {
		return StartRunResult{}, err
	}
	source, err := a.sources.Load(request.WorkflowID)
	if err != nil {
		return StartRunResult{}, err
	}
	return a.startRunArtifact(ctx, source.Artifact(), request.Principal, request.Selection, request.WorkflowID, debug, breakpoints)
}

func (a *Application) startRunArtifact(
	ctx context.Context,
	sourceArtifact []byte,
	principal string,
	selection admission.Selection,
	sourceWorkflowID string,
	debug bool,
	breakpoints []compiler.DebugBreakpoint,
) (StartRunResult, error) {
	if err := a.requireRunning(); err != nil {
		return StartRunResult{}, err
	}
	compiled, err := a.compileDraft(ctx, sourceArtifact)
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
	providers, releaseProviders, err := a.acquireProviderSnapshot()
	if err != nil {
		return result, fmt.Errorf("lease execution environment: %w", err)
	}
	leasePublished := false
	defer func() {
		if !leasePublished {
			releaseProviders()
		}
	}()
	admitted, err := a.admitter.Admit(ctx, admission.Request{
		Program: program, Principal: principal, Selection: selection,
	})
	result.Record = admitted.Record
	if err != nil {
		return result, err
	}
	runID := admitted.Record.Admission().RunID
	var control *compiler.DebugController
	if debug {
		control, err = compiler.NewDebugController(compiler.DebugControllerOptions{
			StartPaused: true, Breakpoints: breakpoints,
			OnChanged: func(snapshot compiler.DebugSnapshot) {
				a.emitDebug(DebugEvent{RunID: runID, Snapshot: snapshot})
			},
		})
		if err != nil {
			return result, err
		}
	}
	a.mu.Lock()
	if a.state != stateRunning {
		a.mu.Unlock()
		return result, ErrClosed
	}
	a.jobs[runID] = &runJob{
		workflowID: sourceWorkflowID, state: jobQueued,
		providers: providers, release: releaseProviders,
	}
	if control != nil {
		a.debug[runID] = control
	}
	a.queue = append(a.queue, runID)
	a.mu.Unlock()
	leasePublished = true
	a.signalWorker()
	a.emit(admitted.Record, nil)
	return result, nil
}

func (a *Application) GetDebugSnapshot(runID string) (compiler.DebugSnapshot, error) {
	a.mu.Lock()
	control := a.debug[runID]
	a.mu.Unlock()
	if control == nil {
		return compiler.DebugSnapshot{}, errors.New("debug session does not exist")
	}
	return control.Snapshot(), nil
}

func (a *Application) ControlDebugRun(ctx context.Context, runID string, action DebugAction) (compiler.DebugSnapshot, error) {
	if ctx == nil {
		return compiler.DebugSnapshot{}, errors.New("debug control context is required")
	}
	a.mu.Lock()
	control := a.debug[runID]
	a.mu.Unlock()
	if control == nil {
		return compiler.DebugSnapshot{}, errors.New("debug session does not exist")
	}
	var err error
	switch action {
	case DebugContinue:
		err = control.Continue()
	case DebugPause:
		err = control.Pause()
	case DebugStep:
		err = control.Step()
	default:
		err = errors.New("unknown debug action")
	}
	return control.Snapshot(), err
}

func (a *Application) SetDebugBreakpoints(ctx context.Context, runID string, breakpoints []compiler.DebugBreakpoint) (compiler.DebugSnapshot, error) {
	if ctx == nil {
		return compiler.DebugSnapshot{}, errors.New("set debug breakpoints context is required")
	}
	a.mu.Lock()
	control := a.debug[runID]
	a.mu.Unlock()
	if control == nil {
		return compiler.DebugSnapshot{}, errors.New("debug session does not exist")
	}
	if err := control.SetBreakpoints(breakpoints); err != nil {
		return compiler.DebugSnapshot{}, err
	}
	return control.Snapshot(), nil
}

func (a *Application) GetRun(runID string) (run.Record, error) { return a.runs.Load(runID) }

func (a *Application) GetRunTimelinePage(ctx context.Context, runID string, page, pageSize int) (run.TimelinePage, error) {
	return a.runs.TimelinePage(ctx, runID, page, pageSize)
}

func (a *Application) GetSource(workflowID string) (workflowstore.SourceSnapshot, error) {
	return a.sources.Load(workflowID)
}

func (a *Application) ListSources() []workflowstore.SourceSnapshot { return a.sources.List() }

func (a *Application) ListSourceRecoveries() []workflowstore.SourceRecovery {
	return a.sources.ListRecoveries()
}

func (a *Application) RepairSourceRecovery(ctx context.Context, recoveryID artifact.Digest, raw []byte) (workflowstore.SourceSnapshot, error) {
	return a.sources.RepairRecovery(ctx, recoveryID, raw)
}

func (a *Application) DeleteSourceRecovery(ctx context.Context, recoveryID artifact.Digest) error {
	return a.sources.DeleteRecovery(ctx, recoveryID)
}

func (a *Application) WithDurableBlobReferences(ctx context.Context, visit func([]blob.BlobRef) error) error {
	if ctx == nil || visit == nil {
		return errors.New("durable blob inventory requires context and visitor")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	a.commandMu.Lock()
	defer a.commandMu.Unlock()
	if err := a.requireRunning(); err != nil {
		return err
	}
	refs := make([]blob.BlobRef, 0)
	for _, listed := range a.sources.List() {
		snapshot, err := a.sources.Load(listed.WorkflowID())
		if err != nil {
			return err
		}
		document, diagnostics := schema.ParseSource(snapshot.Artifact())
		if len(diagnostics) != 0 {
			return errors.New("stored Workflow Source failed blob inventory reopen")
		}
		sourceRefs, err := schema.BlobReferences(document)
		if err != nil {
			return err
		}
		refs = append(refs, sourceRefs...)
	}
	programs, err := a.programs.List()
	if err != nil {
		return err
	}
	for _, program := range programs {
		programRefs, err := program.BlobReferences(a.catalog)
		if err != nil {
			return err
		}
		refs = append(refs, programRefs...)
	}
	runRefs, err := a.runs.BlobReferences(ctx)
	if err != nil {
		return err
	}
	refs = append(refs, runRefs...)
	return visit(uniqueBlobReferences(refs))
}

func uniqueBlobReferences(source []blob.BlobRef) []blob.BlobRef {
	seen := make(map[blob.BlobRef]struct{}, len(source))
	result := make([]blob.BlobRef, 0, len(source))
	for _, ref := range source {
		if _, duplicate := seen[ref]; duplicate {
			continue
		}
		seen[ref] = struct{}{}
		result = append(result, ref)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Digest == result[j].Digest {
			return result[i].MediaType < result[j].MediaType
		}
		return result[i].Digest < result[j].Digest
	})
	return result
}

func (a *Application) PublishImportedSource(ctx context.Context, raw []byte, baseRevision int64, expectedHash artifact.Digest) (workflowstore.SourceSnapshot, error) {
	if ctx == nil {
		return workflowstore.SourceSnapshot{}, errors.New("import Workflow Source context is required")
	}
	a.commandMu.Lock()
	defer a.commandMu.Unlock()
	if err := a.requireRunning(); err != nil {
		return workflowstore.SourceSnapshot{}, err
	}
	document, diagnostics := schema.ParseSource(raw)
	if len(diagnostics) != 0 {
		return workflowstore.SourceSnapshot{}, errors.New("imported Workflow Source is invalid")
	}
	if baseRevision == -1 {
		if expectedHash != "" || document.Revision != 0 {
			return workflowstore.SourceSnapshot{}, errors.New("new imported Workflow Source has invalid identity")
		}
		timestamp := formatWorkflowTimestamp(a.now())
		document.Workflow.CreatedAt = timestamp
		document.Workflow.UpdatedAt = timestamp
	} else if baseRevision >= 0 {
		if !expectedHash.Valid() || document.Revision != baseRevision+1 {
			return workflowstore.SourceSnapshot{}, errors.New("replacement import requires exact next revision")
		}
		current, err := a.sources.Load(document.Workflow.ID)
		if err != nil {
			return workflowstore.SourceSnapshot{}, err
		}
		if current.Revision() != baseRevision || current.Hash() != expectedHash {
			return workflowstore.SourceSnapshot{}, workflowstore.ErrSourceConflict
		}
		currentDocument, currentDiagnostics := schema.ParseSource(current.Artifact())
		if len(currentDiagnostics) != 0 {
			return workflowstore.SourceSnapshot{}, errors.New("stored Workflow Source failed strict reopen")
		}
		document.Workflow.CreatedAt = currentDocument.Workflow.CreatedAt
		document.Workflow.UpdatedAt = currentDocument.Workflow.UpdatedAt
		updatedDocument, _, err := stampWorkflowUpdate(document, a.now())
		if err != nil {
			return workflowstore.SourceSnapshot{}, err
		}
		document = updatedDocument
		if active := a.ActiveSourceRuns(document.Workflow.ID); len(active) != 0 {
			return workflowstore.SourceSnapshot{}, fmt.Errorf("workflow source has %d active run(s)", len(active))
		}
	} else {
		return workflowstore.SourceSnapshot{}, errors.New("imported Workflow Source base revision is invalid")
	}
	canonical, err := artifact.Marshal(document)
	if err != nil {
		return workflowstore.SourceSnapshot{}, err
	}
	return a.sources.Save(ctx, canonical, baseRevision)
}

func stampWorkflowUpdate(source schema.WorkflowSource, now time.Time) (schema.WorkflowSource, []byte, error) {
	updatedAt := now.UTC()
	for _, value := range []string{source.Workflow.CreatedAt, source.Workflow.UpdatedAt} {
		if value == "" {
			continue
		}
		parsed, err := time.Parse(time.RFC3339Nano, value)
		if err != nil {
			return schema.WorkflowSource{}, nil, fmt.Errorf("stamp Workflow Source timestamp: %w", err)
		}
		if parsed.After(updatedAt) {
			updatedAt = parsed
		}
	}
	source.Workflow.UpdatedAt = formatWorkflowTimestamp(updatedAt)
	raw, err := artifact.Marshal(source)
	if err != nil {
		return schema.WorkflowSource{}, nil, err
	}
	return source, raw, nil
}

func formatWorkflowTimestamp(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

// ActiveSourceRuns returns the queued/running run IDs currently retaining one
// Workflow Source. Historical Run records are intentionally not references:
// they retain immutable Program and journal facts after Source deletion.
func (a *Application) ActiveSourceRuns(workflowID string) []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	runIDs := make([]string, 0)
	for runID, job := range a.jobs {
		if job.workflowID == workflowID {
			runIDs = append(runIDs, runID)
		}
	}
	sort.Strings(runIDs)
	return runIDs
}

// DeleteSource deletes only the exact revision/hash reviewed by the caller
// and refuses while an in-process Run still retains the Source identity.
func (a *Application) DeleteSource(ctx context.Context, workflowID string, revision int64, hash artifact.Digest) error {
	if ctx == nil {
		return errors.New("delete Workflow Source context is required")
	}
	a.commandMu.Lock()
	defer a.commandMu.Unlock()
	if err := a.requireRunning(); err != nil {
		return err
	}
	if active := a.ActiveSourceRuns(workflowID); len(active) != 0 {
		return fmt.Errorf("workflow source has %d active run(s)", len(active))
	}
	return a.sources.Delete(ctx, workflowID, revision, hash)
}

func (a *Application) CatalogArtifact() []byte { return a.catalog.Bytes() }

func (a *Application) AuthoringProjection() nodeauthoring.Snapshot { return a.authoring }

func (a *Application) CancelRun(ctx context.Context, runID string) (run.Record, error) {
	if ctx == nil {
		return run.Record{}, errors.New("cancel Run context is required")
	}
	a.mu.Lock()
	job := a.jobs[runID]
	if job != nil && job.state == jobRunning {
		job.cancel()
		a.mu.Unlock()
		return a.runs.Load(runID)
	}
	var releaseProviders func()
	if job != nil {
		releaseProviders = job.release
		delete(a.jobs, runID)
		a.removeQueuedLocked(runID)
	}
	a.mu.Unlock()
	if releaseProviders != nil {
		releaseProviders()
	}
	current, err := a.runs.Load(runID)
	if err != nil || current.Status() != run.StatusQueued {
		return current, err
	}
	next, err := current.Cancel(a.transitionTime(current.Admission().QueuedAt))
	if err != nil {
		return run.Record{}, err
	}
	if err := a.runs.Update(ctx, current.Digest(), next); err != nil {
		return run.Record{}, err
	}
	a.mu.Lock()
	control := a.debug[runID]
	a.mu.Unlock()
	if control != nil {
		control.Complete(string(next.Status()))
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
		if _, err := a.CancelRun(ctx, runID); err != nil && !errors.Is(err, run.ErrRunConflict) {
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
	queuedReleases := make([]func(), 0, len(queued))
	a.queue = nil
	for runID, job := range a.jobs {
		if job.state == jobRunning && job.cancel != nil {
			job.cancel()
		} else {
			if job.release != nil {
				queuedReleases = append(queuedReleases, job.release)
			}
			delete(a.jobs, runID)
		}
	}
	a.cancel()
	a.mu.Unlock()
	a.commandMu.Unlock()
	for _, release := range queuedReleases {
		release()
	}
	var closeErr error
	for _, runID := range queued {
		if _, err := a.CancelRun(ctx, runID); err != nil && !errors.Is(err, run.ErrRunConflict) {
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
		job := a.jobs[runID]
		delete(a.jobs, runID)
		a.mu.Unlock()
		if job != nil && job.release != nil {
			job.release()
		}
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
	a.mu.Lock()
	job := a.jobs[runID]
	a.mu.Unlock()
	if job == nil || job.providers == nil {
		return errors.New("run has no leased execution environment")
	}
	record, err := a.runs.Load(runID)
	if err != nil || record.Status() != run.StatusQueued {
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
		_, terminalErr := journal.Fail(context.WithoutCancel(ctx), a.transitionTime(running.Admission().QueuedAt), run.RunError{
			Code: "runtime.bootstrap_failed", Category: run.ErrorCategoryInfrastructure,
		})
		return errors.Join(bootstrapErr, terminalErr)
	}
	owner, err := run.NewOwner(ctx, grant, job.providers, a.resourceOptions)
	if err != nil {
		if ctx.Err() != nil {
			_, terminalErr := journal.Cancel(context.WithoutCancel(ctx), a.transitionTime(running.Admission().QueuedAt))
			return errors.Join(ctx.Err(), err, terminalErr)
		}
		_, terminalErr := journal.Fail(context.WithoutCancel(ctx), a.transitionTime(running.Admission().QueuedAt), run.RunError{
			Code: "runtime.owner_failed", Category: run.ErrorCategoryInfrastructure,
		})
		return errors.Join(err, terminalErr)
	}
	a.mu.Lock()
	control := a.debug[runID]
	a.mu.Unlock()
	var executionErr error
	if control == nil {
		_, executionErr = a.executor.Run(ctx, program, owner, journal)
	} else {
		_, executionErr = a.executor.RunDebug(ctx, program, owner, journal, control)
	}
	closeCtx, cancel := context.WithTimeout(context.Background(), a.ownerCloseTimeout)
	closeErr := owner.Close(closeCtx)
	cancel()
	if control != nil {
		status := "UNKNOWN"
		if current := journal.Current(); current.Valid() {
			status = string(current.Status())
		}
		control.Complete(status)
	}
	return errors.Join(executionErr, closeErr)
}

func (a *Application) emitDebug(event DebugEvent) {
	if a.onDebugEvent != nil && event.RunID != "" {
		a.onDebugEvent(event)
	}
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

func (a *Application) emit(record run.Record, err error) {
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
