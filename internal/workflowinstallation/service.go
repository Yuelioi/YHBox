package workflowinstallation

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yottaapp/yotta/internal/apperr"
	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/workflow/schema"
)

var (
	ErrInstallationNotFound = errors.New("Workflow Installation not found")
	ErrInstallationConflict = errors.New("Workflow Installation conflict")
	ErrReleaseConflict      = errors.New("Workflow Release identity collision")
)

type NotReadyError struct {
	Report ReadinessReport
	Scope  ExecutionScope
}

func (e *NotReadyError) Error() string {
	if e == nil {
		return ""
	}
	return "Workflow Installation is not ready for " + string(e.Scope)
}

func (e *NotReadyError) RPCErrorEnvelope() apperr.Envelope {
	if e == nil {
		return apperr.Envelope{
			Code: apperr.CodeUnclassified, Category: apperr.CategoryInfrastructure,
			Message: "unknown Workflow Installation readiness error",
		}
	}
	blockers := make([]map[string]any, 0, len(e.Report.Blockers))
	for _, blocker := range e.Report.Blockers {
		scopes := make([]string, 0, len(blocker.Blocks))
		for _, scope := range blocker.Blocks {
			scopes = append(scopes, string(scope))
		}
		blockers = append(blockers, map[string]any{
			"kind": blocker.Kind, "requirementId": blocker.RequirementID,
			"expected": blocker.Expected, "blocks": scopes,
			"action": blocker.Action.Kind,
		})
	}
	return apperr.Envelope{
		Code: "workflow_installation.not_ready", Category: apperr.CategoryPolicy,
		Message: "Workflow Installation is not ready for execution",
		Details: map[string]any{
			"installationId": e.Report.InstallationID,
			"releaseId":      e.Report.ReleaseID,
			"scope":          e.Scope,
			"blockers":       blockers,
		},
	}
}

// PreparedExecution is an immutable, readiness-authorized projection consumed
// by the one application runtime. Callers cannot construct one directly.
type PreparedExecution struct{ state *preparedExecutionState }

type preparedExecutionState struct {
	installationID string
	releaseID      artifact.Digest
	sourceArtifact []byte
	targets        map[string]string
	credentials    map[string]string
}

func (e PreparedExecution) Valid() bool {
	return e.state != nil && installationIDPattern.MatchString(e.state.installationID) &&
		e.state.releaseID.Valid() && len(e.state.sourceArtifact) != 0
}

func (e PreparedExecution) InstallationID() string {
	if !e.Valid() {
		return ""
	}
	return e.state.installationID
}

func (e PreparedExecution) ReleaseID() artifact.Digest {
	if !e.Valid() {
		return ""
	}
	return e.state.releaseID
}

func (e PreparedExecution) SourceArtifact() []byte {
	if !e.Valid() {
		return nil
	}
	return append([]byte(nil), e.state.sourceArtifact...)
}

func (e PreparedExecution) TargetSelection() map[string]string {
	if !e.Valid() {
		return nil
	}
	return cloneBindings(e.state.targets)
}

func (e PreparedExecution) CredentialSelection() map[string]string {
	if !e.Valid() {
		return nil
	}
	return cloneBindings(e.state.credentials)
}

type Repository interface {
	Commit(context.Context, ReleaseRecord, InstallationRecord) error
	GetInstallation(context.Context, string) (InstallationRecord, bool, error)
	ListInstallations(context.Context) ([]InstallationRecord, error)
	GetRelease(context.Context, artifact.Digest) (ReleaseRecord, bool, error)
	GetConfiguration(context.Context, string) (Configuration, bool, error)
	ReplaceConfiguration(context.Context, int64, Configuration) error
}

type Options struct {
	Now          func() time.Time
	NewID        func() string
	Dependencies func(context.Context) ([]DependencyState, error)
}

// Module is the command/query boundary for Workflow Installations. Verified
// release persistence and installation creation are one repository commit.
type Module struct {
	repository   Repository
	now          func() time.Time
	newID        func() string
	dependencies func(context.Context) ([]DependencyState, error)
}

func New(repository Repository, options Options) (*Module, error) {
	if repository == nil {
		return nil, errors.New("Workflow Installation module requires a repository")
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.NewID == nil {
		options.NewID = uuid.NewString
	}
	if options.Dependencies == nil {
		options.Dependencies = func(context.Context) ([]DependencyState, error) { return nil, nil }
	}
	return &Module{
		repository: repository, now: options.Now, newID: options.NewID,
		dependencies: options.Dependencies,
	}, nil
}

func (m *Module) InstallVerified(
	ctx context.Context,
	release ReleaseRecord,
	name string,
) (InstallationRecord, error) {
	if ctx == nil {
		return InstallationRecord{}, errors.New("install verified Workflow Release requires a context")
	}
	if err := ctx.Err(); err != nil {
		return InstallationRecord{}, err
	}
	if err := ValidateReleaseRecord(release); err != nil {
		return InstallationRecord{}, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = release.WorkflowName
	}
	now := m.now().UTC()
	installation := InstallationRecord{
		ID: m.newID(), ReleaseID: release.ID, Name: name,
		Lifecycle: LifecycleActive, CreatedAt: now, UpdatedAt: now,
	}
	if err := ValidateInstallationRecord(installation); err != nil {
		return InstallationRecord{}, err
	}
	if err := m.repository.Commit(ctx, release, installation); err != nil {
		return InstallationRecord{}, err
	}
	return installation, nil
}

func (m *Module) Get(ctx context.Context, installationID string) (InstallationRecord, error) {
	if ctx == nil || !installationIDPattern.MatchString(installationID) {
		return InstallationRecord{}, ErrInstallationNotFound
	}
	installation, found, err := m.repository.GetInstallation(ctx, installationID)
	if err != nil {
		return InstallationRecord{}, err
	}
	if !found {
		return InstallationRecord{}, ErrInstallationNotFound
	}
	return installation, nil
}

func (m *Module) List(ctx context.Context) ([]InstallationRecord, error) {
	if ctx == nil {
		return nil, errors.New("list Workflow Installations requires a context")
	}
	return m.repository.ListInstallations(ctx)
}

func (m *Module) GetConfiguration(ctx context.Context, installationID string) (Configuration, error) {
	installation, err := m.Get(ctx, installationID)
	if err != nil {
		return Configuration{}, err
	}
	configuration, found, err := m.repository.GetConfiguration(ctx, installation.ID)
	if err != nil {
		return Configuration{}, err
	}
	if !found {
		return Configuration{}, errors.New("Workflow Installation configuration is missing")
	}
	return CloneConfiguration(configuration), nil
}

func (m *Module) Readiness(
	ctx context.Context,
	installationID string,
) (ReadinessReport, error) {
	_, _, _, report, err := m.executionState(ctx, installationID)
	return report, err
}

func (m *Module) executionState(
	ctx context.Context,
	installationID string,
) (InstallationRecord, ReleaseRecord, Configuration, ReadinessReport, error) {
	installation, err := m.Get(ctx, installationID)
	if err != nil {
		return InstallationRecord{}, ReleaseRecord{}, Configuration{}, ReadinessReport{}, err
	}
	release, found, err := m.repository.GetRelease(ctx, installation.ReleaseID)
	if err != nil {
		return InstallationRecord{}, ReleaseRecord{}, Configuration{}, ReadinessReport{}, err
	}
	if !found {
		return InstallationRecord{}, ReleaseRecord{}, Configuration{}, ReadinessReport{},
			errors.New("Workflow Installation references a missing Release")
	}
	configuration, found, err := m.repository.GetConfiguration(ctx, installation.ID)
	if err != nil {
		return InstallationRecord{}, ReleaseRecord{}, Configuration{}, ReadinessReport{}, err
	}
	if !found {
		return InstallationRecord{}, ReleaseRecord{}, Configuration{}, ReadinessReport{},
			errors.New("Workflow Installation configuration is missing")
	}
	dependencies, err := m.dependencies(ctx)
	if err != nil {
		return InstallationRecord{}, ReleaseRecord{}, Configuration{}, ReadinessReport{}, err
	}
	report, err := EvaluateReadiness(
		installation, release, configuration, ReadinessEnvironment{Dependencies: dependencies},
	)
	if err != nil {
		return InstallationRecord{}, ReleaseRecord{}, Configuration{}, ReadinessReport{}, err
	}
	return installation, release, configuration, report, nil
}

func (m *Module) PrepareExecution(
	ctx context.Context,
	installationID string,
	scope ExecutionScope,
) (PreparedExecution, error) {
	if scope != ScopeRun && scope != ScopeSchedule {
		return PreparedExecution{}, errors.New("Workflow Installation execution scope is invalid")
	}
	installation, release, configuration, report, err := m.executionState(ctx, installationID)
	if err != nil {
		return PreparedExecution{}, err
	}
	allowed := report.RunAllowed
	if scope == ScopeSchedule {
		allowed = report.ScheduleAllowed
	}
	if !allowed {
		return PreparedExecution{}, &NotReadyError{Report: report, Scope: scope}
	}
	source, diagnostics := schema.ParseSource(release.SourceArtifact)
	if schema.HasErrors(diagnostics) {
		return PreparedExecution{}, errors.New("Workflow Release Source is invalid")
	}
	targets := make(map[string]string, len(source.TargetDefaults))
	for _, targetDefault := range source.TargetDefaults {
		targets[targetDefault.Slot] = configuration.TargetBindings[targetDefault.Slot]
	}
	return PreparedExecution{state: &preparedExecutionState{
		installationID: installation.ID, releaseID: release.ID,
		sourceArtifact: append([]byte(nil), release.SourceArtifact...),
		targets:        targets, credentials: cloneBindings(configuration.CredentialBindings),
	}}, nil
}

type BindingUpdate struct {
	TargetBindings     map[string]string
	CredentialBindings map[string]string
}

func (m *Module) ReplaceBindings(
	ctx context.Context,
	installationID string,
	expectedGeneration int64,
	update BindingUpdate,
) (Configuration, error) {
	installation, err := m.Get(ctx, installationID)
	if err != nil {
		return Configuration{}, err
	}
	current, found, err := m.repository.GetConfiguration(ctx, installation.ID)
	if err != nil {
		return Configuration{}, err
	}
	if !found || current.Generation != expectedGeneration {
		return Configuration{}, ErrInstallationConflict
	}
	release, found, err := m.repository.GetRelease(ctx, installation.ReleaseID)
	if err != nil {
		return Configuration{}, err
	}
	if !found {
		return Configuration{}, errors.New("Workflow Installation references a missing Release")
	}
	if err := validateBindingUpdate(release, update); err != nil {
		return Configuration{}, err
	}
	next := CloneConfiguration(current)
	next.Generation++
	next.TargetBindings = cloneBindings(update.TargetBindings)
	next.CredentialBindings = cloneBindings(update.CredentialBindings)
	next.UpdatedAt = m.now().UTC()
	if err := ValidateConfiguration(next); err != nil {
		return Configuration{}, err
	}
	if err := m.repository.ReplaceConfiguration(ctx, expectedGeneration, next); err != nil {
		return Configuration{}, err
	}
	return CloneConfiguration(next), nil
}

func (m *Module) GrantConsent(
	ctx context.Context,
	installationID string,
	expectedGeneration int64,
	scope ExecutionScope,
) (Configuration, error) {
	installation, err := m.Get(ctx, installationID)
	if err != nil {
		return Configuration{}, err
	}
	current, found, err := m.repository.GetConfiguration(ctx, installation.ID)
	if err != nil {
		return Configuration{}, err
	}
	if !found || current.Generation != expectedGeneration {
		return Configuration{}, ErrInstallationConflict
	}
	next := CloneConfiguration(current)
	next.Generation++
	switch scope {
	case ScopeRun:
		next.RunConsentRelease = installation.ReleaseID
	case ScopeSchedule:
		next.ScheduleConsentRelease = installation.ReleaseID
	default:
		return Configuration{}, errors.New("Workflow Installation consent scope is invalid")
	}
	next.UpdatedAt = m.now().UTC()
	if err := m.repository.ReplaceConfiguration(ctx, expectedGeneration, next); err != nil {
		return Configuration{}, err
	}
	return CloneConfiguration(next), nil
}

func validateBindingUpdate(release ReleaseRecord, update BindingUpdate) error {
	source, diagnostics := schema.ParseSource(release.SourceArtifact)
	if schema.HasErrors(diagnostics) {
		return errors.New("Workflow Release Source is invalid")
	}
	targetSlots := make(map[string]struct{}, len(source.TargetProfileDefinitions))
	for _, definition := range source.TargetProfileDefinitions {
		targetSlots[definition.ID] = struct{}{}
	}
	for slot := range update.TargetBindings {
		if _, ok := targetSlots[slot]; !ok {
			return errors.New("Workflow Installation target binding references an unknown definition")
		}
	}
	credentialSlots := make(map[string]struct{}, len(source.CredentialRequirements))
	for _, requirement := range source.CredentialRequirements {
		credentialSlots[requirement.Slot] = struct{}{}
	}
	for slot := range update.CredentialBindings {
		if _, ok := credentialSlots[slot]; !ok {
			return errors.New("Workflow Installation credential binding references an unknown requirement")
		}
	}
	return nil
}
