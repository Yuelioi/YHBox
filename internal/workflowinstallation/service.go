package workflowinstallation

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/workflow/schema"
)

var (
	ErrInstallationNotFound = errors.New("Workflow Installation not found")
	ErrInstallationConflict = errors.New("Workflow Installation conflict")
	ErrReleaseConflict      = errors.New("Workflow Release identity collision")
)

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
	installation, err := m.Get(ctx, installationID)
	if err != nil {
		return ReadinessReport{}, err
	}
	release, found, err := m.repository.GetRelease(ctx, installation.ReleaseID)
	if err != nil {
		return ReadinessReport{}, err
	}
	if !found {
		return ReadinessReport{}, errors.New("Workflow Installation references a missing Release")
	}
	configuration, found, err := m.repository.GetConfiguration(ctx, installation.ID)
	if err != nil {
		return ReadinessReport{}, err
	}
	if !found {
		return ReadinessReport{}, errors.New("Workflow Installation configuration is missing")
	}
	dependencies, err := m.dependencies(ctx)
	if err != nil {
		return ReadinessReport{}, err
	}
	return EvaluateReadiness(installation, release, configuration, ReadinessEnvironment{Dependencies: dependencies})
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
