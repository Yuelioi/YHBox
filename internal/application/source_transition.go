package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/workflow/authoring"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
	"github.com/yottaapp/yotta/internal/workflow/schema"
	"github.com/yottaapp/yotta/internal/workflowstore"
)

// These private methods own command reduction, state-migration safety,
// candidate sealing, and revision/hash CAS for Application.
func (a *Application) compileDraft(ctx context.Context, sourceJSON []byte) (compiler.CompileResult, error) {
	return a.compileDraftWithOverrides(ctx, sourceJSON, nil)
}

func (a *Application) compileDraftWithOverrides(ctx context.Context, sourceJSON []byte, overrides []compiler.ResourceOverride) (compiler.CompileResult, error) {
	return a.compiler.CompileDraft(ctx, compiler.CompileRequest{
		SourceJSON: sourceJSON, Catalog: a.catalog, BlobVerifier: a.blobVerifier, ResourceOverrides: overrides,
	})
}

// migrateCompatibleContracts publishes only exact node contract upgrades in
// the authoring migration registry.
func (a *Application) migrateCompatibleContracts(ctx context.Context) error {
	for _, snapshot := range a.sources.List() {
		source, diagnostics := schema.ParseSource(snapshot.Artifact())
		if len(diagnostics) != 0 {
			continue
		}
		commands := make([]authoring.Command, 0)
		for _, graph := range source.Graphs {
			for _, node := range graph.Nodes {
				projection, ok := a.authoring.Node(node.NodeRef.NodeTypeID)
				if !ok || projection.NodeRef == node.NodeRef {
					continue
				}
				command := authoring.Command{
					Kind: authoring.CommandUpgradeNodeContract,
					UpgradeNodeContract: &authoring.NodeCommand{
						GraphID: graph.ID,
						NodeID:  node.ID,
					},
				}
				if _, err := a.authoringEngine.Apply(source, []authoring.Command{command}); err != nil {
					continue
				}
				commands = append(commands, command)
				if len(commands) == authoring.MaxCommands {
					break
				}
			}
			if len(commands) == authoring.MaxCommands {
				break
			}
		}
		if len(commands) == 0 {
			continue
		}
		applied, err := a.authoringEngine.Apply(source, commands)
		if err != nil {
			continue
		}
		_, raw, err := stampWorkflowUpdate(applied.Source, a.now())
		if err != nil {
			return err
		}
		if _, err := a.sources.Save(ctx, raw, snapshot.Revision()); err != nil {
			return err
		}
	}
	return nil
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
	snapshot, source, candidate, err := a.reducePatch(request)
	if err != nil {
		return ApplyPatchResult{}, err
	}
	if referencedStateUpdate(source, candidate.Source, request.Commands) {
		baseline, err := a.compileDraft(ctx, snapshot.Artifact())
		if err != nil {
			return ApplyPatchResult{}, fmt.Errorf("compile state migration baseline: %w", err)
		}
		compiled, err := a.compileDraft(ctx, candidate.Artifact)
		if err != nil {
			return ApplyPatchResult{}, fmt.Errorf("compile state migration candidate: %w", err)
		}
		if introduced := introducedErrorDiagnostics(baseline.Diagnostics, compiled.Diagnostics); len(introduced) != 0 {
			return ApplyPatchResult{}, &UnsafeStateMigrationError{Diagnostics: introduced}
		}
	}
	next, saveErr := a.sources.Save(ctx, candidate.Artifact, request.BaseRevision)
	return ApplyPatchResult{
		Source: next, GeneratedNodes: append([]authoring.GeneratedNode(nil), candidate.GeneratedNodes...),
	}, saveErr
}

func (a *Application) reducePatch(request authoring.PatchRequest) (
	workflowstore.SourceSnapshot,
	schema.WorkflowSource,
	authoring.Result,
	error,
) {
	snapshot, err := a.sources.Load(request.WorkflowID)
	if err != nil {
		return workflowstore.SourceSnapshot{}, schema.WorkflowSource{}, authoring.Result{}, err
	}
	if snapshot.Revision() != request.BaseRevision {
		return workflowstore.SourceSnapshot{}, schema.WorkflowSource{}, authoring.Result{}, workflowstore.ErrSourceConflict
	}
	source, diagnostics := schema.ParseSource(snapshot.Artifact())
	if len(diagnostics) != 0 {
		return workflowstore.SourceSnapshot{}, schema.WorkflowSource{}, authoring.Result{}, errors.New("stored Workflow Source failed strict reopen")
	}
	applied, err := a.authoringEngine.Apply(source, request.Commands)
	if err != nil {
		return workflowstore.SourceSnapshot{}, schema.WorkflowSource{}, authoring.Result{}, err
	}
	applied.Source, applied.Artifact, err = stampWorkflowUpdate(applied.Source, a.now())
	if err != nil {
		return workflowstore.SourceSnapshot{}, schema.WorkflowSource{}, authoring.Result{}, err
	}
	return snapshot, source, applied, nil
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
	snapshot, source, candidate, err := a.reducePatch(request)
	if err != nil {
		return PreparePatchResult{}, err
	}
	_, _, candidateHash, candidateDiagnostics, err := schema.CanonicalSource(candidate.Artifact)
	if err != nil || len(candidateDiagnostics) != 0 {
		return PreparePatchResult{}, errors.New("prepared Workflow Source failed strict reopen")
	}
	prepared := PreparedPatch{state: &preparedPatchState{
		workflowID: request.WorkflowID, baseRevision: request.BaseRevision, baseHash: snapshot.Hash(),
		candidate: append([]byte(nil), candidate.Artifact...), candidateHash: candidateHash,
		generated: append([]authoring.GeneratedNode(nil), candidate.GeneratedNodes...),
	}}
	compiled, compileErr := a.compileDraft(ctx, candidate.Artifact)
	if referencedStateUpdate(source, candidate.Source, request.Commands) {
		baseline, err := a.compileDraft(ctx, snapshot.Artifact())
		if err != nil {
			return PreparePatchResult{}, fmt.Errorf("compile state migration baseline: %w", err)
		}
		prepared.state.unsafeStateMigration = introducedErrorDiagnostics(baseline.Diagnostics, compiled.Diagnostics)
	}
	result := PreparePatchResult{
		Patch: prepared, Diagnostics: append([]schema.Diagnostic(nil), compiled.Diagnostics...),
		CapabilityPlan: []capability.PlanEntry{},
	}
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
	return ApplyPatchResult{
		Source: next, GeneratedNodes: append([]authoring.GeneratedNode(nil), prepared.state.generated...),
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

func stampWorkflowUpdate(source schema.WorkflowSource, now time.Time) (schema.WorkflowSource, []byte, error) {
	updatedAt := now.UTC()
	for _, value := range []string{source.Workflow.CreatedAt, source.Workflow.UpdatedAt} {
		parsed, _ := time.Parse(time.RFC3339Nano, value)
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
