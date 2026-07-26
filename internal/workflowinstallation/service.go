package workflowinstallation

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sort"
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

// PreparedDerivedSource is an opaque, canonical editable Source candidate
// created from one Installation's exact immutable Release. Only this Module
// can construct its derivedFrom provenance.
type PreparedDerivedSource struct {
	originReleaseID artifact.Digest
	sourceArtifact  []byte
}

func (p PreparedDerivedSource) Valid() bool {
	return p.originReleaseID.Valid() && len(p.sourceArtifact) != 0
}

func (p PreparedDerivedSource) OriginReleaseID() artifact.Digest {
	if !p.Valid() {
		return ""
	}
	return p.originReleaseID
}

func (p PreparedDerivedSource) SourceArtifact() []byte {
	if !p.Valid() {
		return nil
	}
	return append([]byte(nil), p.sourceArtifact...)
}

type UpdateConflictKind string

const (
	UpdateConflictTarget     UpdateConflictKind = "target"
	UpdateConflictCredential UpdateConflictKind = "credential"
)

type UpdateConflict struct {
	Kind          UpdateConflictKind
	RequirementID string
}

type UpdateDiff struct {
	CurrentReleaseID    artifact.Digest
	CandidateReleaseID  artifact.Digest
	CurrentVersion      string
	CandidateVersion    string
	AddedTargets        []string
	ChangedTargets      []string
	RemovedTargets      []string
	AddedCredentials    []string
	ChangedCredentials  []string
	RemovedCredentials  []string
	GraphsChanged       bool
	ResourcesChanged    bool
	VariablesChanged    bool
	DependenciesChanged bool
}

// PreparedUpdate is an immutable staged transition from one exact Release and
// configuration generation to another. Only the Module can construct the
// candidate configuration committed by ApplyUpdate.
type PreparedUpdate struct{ state *preparedUpdateState }

type preparedUpdateState struct {
	expectedRelease    artifact.Digest
	expectedGeneration int64
	candidate          ReleaseRecord
	installation       InstallationRecord
	configuration      Configuration
	diff               UpdateDiff
	conflicts          []UpdateConflict
	readiness          ReadinessReport
}

func (p PreparedUpdate) Valid() bool {
	return p.state != nil && p.state.expectedRelease.Valid() &&
		p.state.expectedGeneration > 0 && p.state.candidate.ID.Valid() &&
		p.state.installation.ReleaseID == p.state.candidate.ID &&
		p.state.configuration.InstallationID == p.state.installation.ID &&
		p.state.configuration.Generation == p.state.expectedGeneration+1
}

func (p PreparedUpdate) Diff() UpdateDiff {
	if !p.Valid() {
		return UpdateDiff{}
	}
	return cloneUpdateDiff(p.state.diff)
}

func (p PreparedUpdate) Conflicts() []UpdateConflict {
	if !p.Valid() {
		return nil
	}
	return append([]UpdateConflict(nil), p.state.conflicts...)
}

func (p PreparedUpdate) CandidateReadiness() ReadinessReport {
	if !p.Valid() {
		return ReadinessReport{}
	}
	return cloneReadinessReport(p.state.readiness)
}

type Repository interface {
	Commit(context.Context, ReleaseRecord, InstallationRecord, Configuration) error
	SwitchRelease(
		context.Context,
		artifact.Digest,
		int64,
		ReleaseRecord,
		InstallationRecord,
		Configuration,
	) error
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
	Targets      func(context.Context) ([]TargetState, error)
	Credentials  func(context.Context) ([]CredentialState, error)
}

// Module is the command/query boundary for Workflow Installations. Verified
// release persistence and installation creation are one repository commit.
type Module struct {
	repository   Repository
	now          func() time.Time
	newID        func() string
	dependencies func(context.Context) ([]DependencyState, error)
	targets      func(context.Context) ([]TargetState, error)
	credentials  func(context.Context) ([]CredentialState, error)
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
	if options.Targets == nil {
		options.Targets = func(context.Context) ([]TargetState, error) { return nil, nil }
	}
	if options.Credentials == nil {
		options.Credentials = func(context.Context) ([]CredentialState, error) { return nil, nil }
	}
	return &Module{
		repository: repository, now: options.Now, newID: options.NewID,
		dependencies: options.Dependencies, targets: options.Targets, credentials: options.Credentials,
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
	configuration, err := NewConfigurationForRelease(installation, release)
	if err != nil {
		return InstallationRecord{}, err
	}
	if err := m.repository.Commit(ctx, release, installation, configuration); err != nil {
		return InstallationRecord{}, err
	}
	return installation, nil
}

// PrepareDerivedSource creates a new local authoring identity from the exact
// current Release without copying any Installation-local configuration or
// authority. Publishing the candidate remains the caller's separate Source
// Store commit.
func (m *Module) PrepareDerivedSource(
	ctx context.Context,
	installationID string,
	name string,
) (PreparedDerivedSource, error) {
	installation, err := m.Get(ctx, installationID)
	if err != nil {
		return PreparedDerivedSource{}, err
	}
	release, found, err := m.repository.GetRelease(ctx, installation.ReleaseID)
	if err != nil {
		return PreparedDerivedSource{}, err
	}
	if !found {
		return PreparedDerivedSource{}, errors.New("Workflow Installation references a missing Release")
	}
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 256 {
		return PreparedDerivedSource{}, errors.New("derived Workflow Source name is invalid")
	}
	source, diagnostics := schema.ParseSource(release.SourceArtifact)
	if schema.HasErrors(diagnostics) {
		return PreparedDerivedSource{}, errors.New("Workflow Release Source is invalid")
	}
	source.Workflow.ID = m.newID()
	source.Workflow.Name = name
	source.Workflow.CreatedAt = ""
	source.Workflow.UpdatedAt = ""
	source.DerivedFrom = &schema.WorkflowReleaseOrigin{
		ReleaseDigest: release.ID, SourceHash: release.SourceHash,
		AttestationDigest:  release.AttestationDigest,
		PublisherNamespace: release.PublisherNamespace,
		WorkflowID:         release.WorkflowID, ReleaseVersion: release.ReleaseVersion,
	}
	source.Revision = 0
	raw, err := artifact.Marshal(source)
	if err != nil {
		return PreparedDerivedSource{}, err
	}
	_, canonical, _, diagnostics, err := schema.CanonicalSource(raw)
	if err != nil {
		return PreparedDerivedSource{}, err
	}
	if schema.HasErrors(diagnostics) {
		return PreparedDerivedSource{}, errors.New("derived Workflow Source is invalid")
	}
	return PreparedDerivedSource{
		originReleaseID: release.ID,
		sourceArtifact:  canonical,
	}, nil
}

// PrepareUpdate validates a verified candidate Release, computes a
// deterministic diff, and projects the exact post-switch local configuration
// and readiness without changing durable Installation state.
func (m *Module) PrepareUpdate(
	ctx context.Context,
	installationID string,
	candidate ReleaseRecord,
) (PreparedUpdate, error) {
	if ctx == nil {
		return PreparedUpdate{}, errors.New("prepare Workflow Installation update requires a context")
	}
	if err := ctx.Err(); err != nil {
		return PreparedUpdate{}, err
	}
	if err := ValidateReleaseRecord(candidate); err != nil {
		return PreparedUpdate{}, err
	}
	installation, err := m.Get(ctx, installationID)
	if err != nil {
		return PreparedUpdate{}, err
	}
	current, found, err := m.repository.GetRelease(ctx, installation.ReleaseID)
	if err != nil {
		return PreparedUpdate{}, err
	}
	if !found {
		return PreparedUpdate{}, errors.New("Workflow Installation references a missing Release")
	}
	configuration, err := m.configuration(ctx, installation, current)
	if err != nil {
		return PreparedUpdate{}, err
	}
	if candidate.ID == current.ID {
		return PreparedUpdate{}, errors.New("Workflow Installation already uses the candidate Release")
	}
	if candidate.PublisherNamespace != current.PublisherNamespace ||
		candidate.WorkflowID != current.WorkflowID {
		return PreparedUpdate{}, errors.New("Workflow Installation update candidate has a different publisher or workflow identity")
	}
	if candidate.ReleaseVersion == current.ReleaseVersion {
		return PreparedUpdate{}, errors.New("Workflow Installation update candidate reuses the current immutable version")
	}
	currentSource, diagnostics := schema.ParseSource(current.SourceArtifact)
	if schema.HasErrors(diagnostics) {
		return PreparedUpdate{}, errors.New("current Workflow Release Source is invalid")
	}
	candidateSource, diagnostics := schema.ParseSource(candidate.SourceArtifact)
	if schema.HasErrors(diagnostics) {
		return PreparedUpdate{}, errors.New("candidate Workflow Release Source is invalid")
	}
	nextConfiguration, diff, conflicts, err := prepareUpdateConfiguration(
		configuration, current, candidate, currentSource, candidateSource,
	)
	if err != nil {
		return PreparedUpdate{}, err
	}
	now := m.now().UTC()
	nextInstallation := installation
	nextInstallation.ReleaseID = candidate.ID
	nextInstallation.UpdatedAt = now
	nextConfiguration.Generation = configuration.Generation + 1
	nextConfiguration.RunConsentRelease = ""
	nextConfiguration.ScheduleConsentRelease = ""
	nextConfiguration.UpdatedAt = now
	if err := ValidateInstallationRecord(nextInstallation); err != nil {
		return PreparedUpdate{}, err
	}
	if err := ValidateConfiguration(nextConfiguration); err != nil {
		return PreparedUpdate{}, err
	}
	dependencies, err := m.dependencies(ctx)
	if err != nil {
		return PreparedUpdate{}, err
	}
	targets, err := m.targets(ctx)
	if err != nil {
		return PreparedUpdate{}, err
	}
	credentials, err := m.credentials(ctx)
	if err != nil {
		return PreparedUpdate{}, err
	}
	readiness, err := EvaluateReadiness(
		nextInstallation, candidate, nextConfiguration,
		ReadinessEnvironment{Dependencies: dependencies, Targets: targets, Credentials: credentials},
	)
	if err != nil {
		return PreparedUpdate{}, err
	}
	return PreparedUpdate{state: &preparedUpdateState{
		expectedRelease: current.ID, expectedGeneration: configuration.Generation,
		candidate: CloneReleaseRecord(candidate), installation: nextInstallation,
		configuration: CloneConfiguration(nextConfiguration), diff: cloneUpdateDiff(diff),
		conflicts: append([]UpdateConflict(nil), conflicts...), readiness: cloneReadinessReport(readiness),
	}}, nil
}

// ApplyUpdate performs the single repository transaction captured by a
// conflict-free PreparedUpdate. Repository CAS rejects stale Release or
// configuration state.
func (m *Module) ApplyUpdate(
	ctx context.Context,
	prepared PreparedUpdate,
) (InstallationRecord, error) {
	if ctx == nil || !prepared.Valid() {
		return InstallationRecord{}, errors.New("prepared Workflow Installation update is invalid")
	}
	if err := ctx.Err(); err != nil {
		return InstallationRecord{}, err
	}
	if len(prepared.state.conflicts) != 0 {
		return InstallationRecord{}, errors.New("Workflow Installation update has unresolved local configuration conflicts")
	}
	if err := m.repository.SwitchRelease(
		ctx, prepared.state.expectedRelease, prepared.state.expectedGeneration,
		CloneReleaseRecord(prepared.state.candidate), prepared.state.installation,
		CloneConfiguration(prepared.state.configuration),
	); err != nil {
		return InstallationRecord{}, err
	}
	return prepared.state.installation, nil
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
	release, found, err := m.repository.GetRelease(ctx, installation.ReleaseID)
	if err != nil {
		return Configuration{}, err
	}
	if !found {
		return Configuration{}, errors.New("Workflow Installation references a missing Release")
	}
	return m.configuration(ctx, installation, release)
}

func (m *Module) configuration(
	ctx context.Context,
	installation InstallationRecord,
	release ReleaseRecord,
) (Configuration, error) {
	for attempt := 0; attempt < 2; attempt++ {
		configuration, found, err := m.repository.GetConfiguration(ctx, installation.ID)
		if err != nil {
			return Configuration{}, err
		}
		if !found {
			return Configuration{}, errors.New("Workflow Installation configuration is missing")
		}
		materialized, err := MaterializeTargetProfiles(configuration, release)
		if err != nil {
			return Configuration{}, err
		}
		if targetProfilesEqual(configuration.TargetProfiles, materialized.TargetProfiles) {
			return CloneConfiguration(configuration), nil
		}
		materialized.Generation++
		materialized.UpdatedAt = m.now().UTC()
		if err := m.repository.ReplaceConfiguration(ctx, configuration.Generation, materialized); err != nil {
			if errors.Is(err, ErrInstallationConflict) {
				continue
			}
			return Configuration{}, err
		}
		return CloneConfiguration(materialized), nil
	}
	return Configuration{}, ErrInstallationConflict
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
	configuration, err := m.configuration(ctx, installation, release)
	if err != nil {
		return InstallationRecord{}, ReleaseRecord{}, Configuration{}, ReadinessReport{}, err
	}
	dependencies, err := m.dependencies(ctx)
	if err != nil {
		return InstallationRecord{}, ReleaseRecord{}, Configuration{}, ReadinessReport{}, err
	}
	targets, err := m.targets(ctx)
	if err != nil {
		return InstallationRecord{}, ReleaseRecord{}, Configuration{}, ReadinessReport{}, err
	}
	credentials, err := m.credentials(ctx)
	if err != nil {
		return InstallationRecord{}, ReleaseRecord{}, Configuration{}, ReadinessReport{}, err
	}
	report, err := EvaluateReadiness(
		installation, release, configuration,
		ReadinessEnvironment{
			Dependencies: dependencies, Targets: targets, Credentials: credentials,
		},
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
		targets[targetDefault.Slot] = configuration.TargetProfiles[targetDefault.Slot].TargetInstallationID
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

type SettingsSnapshot struct {
	Configuration          Configuration
	TargetDefinitions      []schema.TargetProfileDefinition
	CredentialRequirements []schema.CredentialRequirement
	Credentials            []CredentialState
}

func (m *Module) Settings(ctx context.Context, installationID string) (SettingsSnapshot, error) {
	installation, err := m.Get(ctx, installationID)
	if err != nil {
		return SettingsSnapshot{}, err
	}
	release, found, err := m.repository.GetRelease(ctx, installation.ReleaseID)
	if err != nil {
		return SettingsSnapshot{}, err
	}
	if !found {
		return SettingsSnapshot{}, errors.New("Workflow Installation references a missing Release")
	}
	configuration, err := m.configuration(ctx, installation, release)
	if err != nil {
		return SettingsSnapshot{}, err
	}
	source, diagnostics := schema.ParseSource(release.SourceArtifact)
	if schema.HasErrors(diagnostics) {
		return SettingsSnapshot{}, errors.New("Workflow Release Source is invalid")
	}
	credentials, err := m.credentials(ctx)
	if err != nil {
		return SettingsSnapshot{}, err
	}
	return SettingsSnapshot{
		Configuration:          configuration,
		TargetDefinitions:      append([]schema.TargetProfileDefinition(nil), source.TargetProfileDefinitions...),
		CredentialRequirements: append([]schema.CredentialRequirement(nil), source.CredentialRequirements...),
		Credentials:            append([]CredentialState(nil), credentials...),
	}, nil
}

func (m *Module) UpdateCredentialBinding(
	ctx context.Context,
	installationID string,
	expectedGeneration int64,
	requirementSlot string,
	credentialBindingID string,
) (Configuration, error) {
	if !slotPattern.MatchString(requirementSlot) ||
		credentialBindingID != "" && !localReferencePattern.MatchString(credentialBindingID) {
		return Configuration{}, errors.New("Workflow credential binding update is invalid")
	}
	installation, err := m.Get(ctx, installationID)
	if err != nil {
		return Configuration{}, err
	}
	release, found, err := m.repository.GetRelease(ctx, installation.ReleaseID)
	if err != nil {
		return Configuration{}, err
	}
	if !found {
		return Configuration{}, errors.New("Workflow Installation references a missing Release")
	}
	current, err := m.configuration(ctx, installation, release)
	if err != nil {
		return Configuration{}, err
	}
	if current.Generation != expectedGeneration {
		return Configuration{}, ErrInstallationConflict
	}
	source, diagnostics := schema.ParseSource(release.SourceArtifact)
	if schema.HasErrors(diagnostics) {
		return Configuration{}, errors.New("Workflow Release Source is invalid")
	}
	var requirement *schema.CredentialRequirement
	for index := range source.CredentialRequirements {
		if source.CredentialRequirements[index].Slot == requirementSlot {
			requirement = &source.CredentialRequirements[index]
			break
		}
	}
	if requirement == nil {
		return Configuration{}, errors.New("Workflow credential requirement is unknown")
	}
	if credentialBindingID != "" {
		credentials, err := m.credentials(ctx)
		if err != nil {
			return Configuration{}, err
		}
		compatible := false
		for _, credential := range credentials {
			if credential.CredentialBindingID == credentialBindingID &&
				credential.Kind == requirement.Kind && credential.Available {
				compatible = true
				break
			}
		}
		if !compatible {
			return Configuration{}, errors.New("Workflow credential binding is unavailable or incompatible")
		}
	}
	next := CloneConfiguration(current)
	if credentialBindingID == "" {
		delete(next.CredentialBindings, requirementSlot)
	} else {
		next.CredentialBindings[requirementSlot] = credentialBindingID
	}
	next.Generation++
	next.UpdatedAt = m.now().UTC()
	if err := ValidateConfiguration(next); err != nil {
		return Configuration{}, err
	}
	if err := m.repository.ReplaceConfiguration(ctx, expectedGeneration, next); err != nil {
		return Configuration{}, err
	}
	return CloneConfiguration(next), nil
}

func (m *Module) UpdateTargetProfile(
	ctx context.Context,
	installationID string,
	expectedGeneration int64,
	definitionID string,
	settings []byte,
	targetInstallationID string,
) (Configuration, error) {
	if !slotPattern.MatchString(definitionID) ||
		targetInstallationID != "" && !localReferencePattern.MatchString(targetInstallationID) {
		return Configuration{}, errors.New("Workflow Target Profile update is invalid")
	}
	installation, err := m.Get(ctx, installationID)
	if err != nil {
		return Configuration{}, err
	}
	release, found, err := m.repository.GetRelease(ctx, installation.ReleaseID)
	if err != nil {
		return Configuration{}, err
	}
	if !found {
		return Configuration{}, errors.New("Workflow Installation references a missing Release")
	}
	current, err := m.configuration(ctx, installation, release)
	if err != nil {
		return Configuration{}, err
	}
	if current.Generation != expectedGeneration {
		return Configuration{}, ErrInstallationConflict
	}
	source, diagnostics := schema.ParseSource(release.SourceArtifact)
	if schema.HasErrors(diagnostics) {
		return Configuration{}, errors.New("Workflow Release Source is invalid")
	}
	var definition *schema.TargetProfileDefinition
	for index := range source.TargetProfileDefinitions {
		if source.TargetProfileDefinitions[index].ID == definitionID {
			definition = &source.TargetProfileDefinitions[index]
			break
		}
	}
	if definition == nil {
		return Configuration{}, errors.New("Workflow Target Profile definition is unknown")
	}
	canonical, err := schema.ValidateTargetProfileSettings(*definition, settings)
	if err != nil {
		return Configuration{}, fmt.Errorf("validate Workflow Target Profile settings: %w", err)
	}
	next := CloneConfiguration(current)
	profile := next.TargetProfiles[definitionID]
	profile.Settings = canonical
	profile.TargetInstallationID = targetInstallationID
	next.TargetProfiles[definitionID] = profile
	if targetInstallationID == "" {
		delete(next.TargetBindings, definitionID)
	} else {
		next.TargetBindings[definitionID] = targetInstallationID
	}
	next.Generation++
	next.UpdatedAt = m.now().UTC()
	if err := ValidateConfiguration(next); err != nil {
		return Configuration{}, err
	}
	if err := m.repository.ReplaceConfiguration(ctx, expectedGeneration, next); err != nil {
		return Configuration{}, err
	}
	return CloneConfiguration(next), nil
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
	release, found, err := m.repository.GetRelease(ctx, installation.ReleaseID)
	if err != nil {
		return Configuration{}, err
	}
	if !found {
		return Configuration{}, errors.New("Workflow Installation references a missing Release")
	}
	current, err := m.configuration(ctx, installation, release)
	if err != nil {
		return Configuration{}, err
	}
	if current.Generation != expectedGeneration {
		return Configuration{}, ErrInstallationConflict
	}
	if err := validateBindingUpdate(release, update); err != nil {
		return Configuration{}, err
	}
	next := CloneConfiguration(current)
	next.Generation++
	next.TargetBindings = cloneBindings(update.TargetBindings)
	next.CredentialBindings = cloneBindings(update.CredentialBindings)
	for definitionID, profile := range next.TargetProfiles {
		profile.TargetInstallationID = next.TargetBindings[definitionID]
		next.TargetProfiles[definitionID] = profile
	}
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
	release, found, err := m.repository.GetRelease(ctx, installation.ReleaseID)
	if err != nil {
		return Configuration{}, err
	}
	if !found {
		return Configuration{}, errors.New("Workflow Installation references a missing Release")
	}
	current, err := m.configuration(ctx, installation, release)
	if err != nil {
		return Configuration{}, err
	}
	if current.Generation != expectedGeneration {
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

func targetProfilesEqual(left, right map[string]TargetProfile) bool {
	if len(left) != len(right) {
		return false
	}
	for key, leftProfile := range left {
		rightProfile, found := right[key]
		if !found || leftProfile.DefinitionID != rightProfile.DefinitionID ||
			leftProfile.ReleaseID != rightProfile.ReleaseID ||
			leftProfile.TargetKind != rightProfile.TargetKind ||
			leftProfile.AdapterKind != rightProfile.AdapterKind ||
			leftProfile.ProfileVersion != rightProfile.ProfileVersion ||
			leftProfile.TargetInstallationID != rightProfile.TargetInstallationID ||
			!bytes.Equal(leftProfile.Settings, rightProfile.Settings) {
			return false
		}
	}
	return true
}

func prepareUpdateConfiguration(
	current Configuration,
	currentRelease ReleaseRecord,
	candidateRelease ReleaseRecord,
	currentSource schema.WorkflowSource,
	candidateSource schema.WorkflowSource,
) (Configuration, UpdateDiff, []UpdateConflict, error) {
	next := CloneConfiguration(current)
	next.TargetProfiles = make(map[string]TargetProfile, len(candidateSource.TargetProfileDefinitions))
	next.TargetBindings = make(map[string]string, len(candidateSource.TargetProfileDefinitions))
	next.CredentialBindings = make(map[string]string, len(candidateSource.CredentialRequirements))
	diff := UpdateDiff{
		CurrentReleaseID: currentRelease.ID, CandidateReleaseID: candidateRelease.ID,
		CurrentVersion: currentRelease.ReleaseVersion, CandidateVersion: candidateRelease.ReleaseVersion,
		GraphsChanged: currentSource.EntryGraph != candidateSource.EntryGraph ||
			!canonicalEqual(currentSource.Graphs, candidateSource.Graphs),
		ResourcesChanged:    !canonicalEqual(currentSource.Resources, candidateSource.Resources),
		VariablesChanged:    !canonicalEqual(currentSource.Variables, candidateSource.Variables),
		DependenciesChanged: !canonicalEqual(currentSource.Dependencies, candidateSource.Dependencies),
	}
	conflicts := make([]UpdateConflict, 0)
	currentTargets := make(map[string]schema.TargetProfileDefinition, len(currentSource.TargetProfileDefinitions))
	candidateTargets := make(map[string]schema.TargetProfileDefinition, len(candidateSource.TargetProfileDefinitions))
	for _, definition := range currentSource.TargetProfileDefinitions {
		currentTargets[definition.ID] = definition
	}
	for _, definition := range candidateSource.TargetProfileDefinitions {
		candidateTargets[definition.ID] = definition
		old, existed := currentTargets[definition.ID]
		if !existed {
			diff.AddedTargets = append(diff.AddedTargets, definition.ID)
			profile, err := newTargetProfile(candidateRelease.ID, definition)
			if err != nil {
				return Configuration{}, UpdateDiff{}, nil, err
			}
			next.TargetProfiles[definition.ID] = profile
			continue
		}
		if !canonicalEqual(old, definition) {
			diff.ChangedTargets = append(diff.ChangedTargets, definition.ID)
		}
		currentProfile := current.TargetProfiles[definition.ID]
		compatible := currentProfile.TargetKind == definition.TargetKind &&
			currentProfile.AdapterKind == definition.AdapterKind &&
			currentProfile.ProfileVersion == definition.ProfileVersion
		settings, settingsErr := schema.ValidateTargetProfileSettings(definition, currentProfile.Settings)
		if compatible && settingsErr == nil {
			currentProfile.ReleaseID = candidateRelease.ID
			currentProfile.Settings = settings
			next.TargetProfiles[definition.ID] = currentProfile
			if currentProfile.TargetInstallationID != "" {
				next.TargetBindings[definition.ID] = currentProfile.TargetInstallationID
			}
			continue
		}
		if targetProfileHasLocalValue(currentProfile, old) {
			conflicts = append(conflicts, UpdateConflict{Kind: UpdateConflictTarget, RequirementID: definition.ID})
		}
		profile, err := newTargetProfile(candidateRelease.ID, definition)
		if err != nil {
			return Configuration{}, UpdateDiff{}, nil, err
		}
		next.TargetProfiles[definition.ID] = profile
	}
	for _, definition := range currentSource.TargetProfileDefinitions {
		if _, found := candidateTargets[definition.ID]; found {
			continue
		}
		diff.RemovedTargets = append(diff.RemovedTargets, definition.ID)
		if targetProfileHasLocalValue(current.TargetProfiles[definition.ID], definition) {
			conflicts = append(conflicts, UpdateConflict{Kind: UpdateConflictTarget, RequirementID: definition.ID})
		}
	}
	currentCredentials := make(map[string]schema.CredentialRequirement, len(currentSource.CredentialRequirements))
	candidateCredentials := make(map[string]schema.CredentialRequirement, len(candidateSource.CredentialRequirements))
	for _, requirement := range currentSource.CredentialRequirements {
		currentCredentials[requirement.Slot] = requirement
	}
	for _, requirement := range candidateSource.CredentialRequirements {
		candidateCredentials[requirement.Slot] = requirement
		old, existed := currentCredentials[requirement.Slot]
		if !existed {
			diff.AddedCredentials = append(diff.AddedCredentials, requirement.Slot)
			continue
		}
		if !canonicalEqual(old, requirement) {
			diff.ChangedCredentials = append(diff.ChangedCredentials, requirement.Slot)
		}
		binding := current.CredentialBindings[requirement.Slot]
		if binding != "" && old.Kind != requirement.Kind {
			conflicts = append(conflicts, UpdateConflict{
				Kind: UpdateConflictCredential, RequirementID: requirement.Slot,
			})
			continue
		}
		if binding != "" {
			next.CredentialBindings[requirement.Slot] = binding
		}
	}
	for _, requirement := range currentSource.CredentialRequirements {
		if _, found := candidateCredentials[requirement.Slot]; found {
			continue
		}
		diff.RemovedCredentials = append(diff.RemovedCredentials, requirement.Slot)
		if current.CredentialBindings[requirement.Slot] != "" {
			conflicts = append(conflicts, UpdateConflict{
				Kind: UpdateConflictCredential, RequirementID: requirement.Slot,
			})
		}
	}
	sortUpdateDiff(&diff)
	sort.Slice(conflicts, func(i, j int) bool {
		if conflicts[i].Kind != conflicts[j].Kind {
			return conflicts[i].Kind < conflicts[j].Kind
		}
		return conflicts[i].RequirementID < conflicts[j].RequirementID
	})
	return next, diff, conflicts, nil
}

func newTargetProfile(
	releaseID artifact.Digest,
	definition schema.TargetProfileDefinition,
) (TargetProfile, error) {
	settings, err := schema.ValidateTargetProfileSettings(definition, definition.InitialDefaults)
	if err != nil {
		return TargetProfile{}, err
	}
	return TargetProfile{
		DefinitionID: definition.ID, ReleaseID: releaseID,
		TargetKind: definition.TargetKind, AdapterKind: definition.AdapterKind,
		ProfileVersion: definition.ProfileVersion, Settings: settings,
	}, nil
}

func targetProfileHasLocalValue(
	profile TargetProfile,
	definition schema.TargetProfileDefinition,
) bool {
	defaults, err := schema.ValidateTargetProfileSettings(definition, definition.InitialDefaults)
	return profile.TargetInstallationID != "" || err != nil || !bytes.Equal(profile.Settings, defaults)
}

func canonicalEqual(left, right any) bool {
	leftRaw, leftErr := artifact.Marshal(left)
	rightRaw, rightErr := artifact.Marshal(right)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftRaw, rightRaw)
}

func sortUpdateDiff(diff *UpdateDiff) {
	sort.Strings(diff.AddedTargets)
	sort.Strings(diff.ChangedTargets)
	sort.Strings(diff.RemovedTargets)
	sort.Strings(diff.AddedCredentials)
	sort.Strings(diff.ChangedCredentials)
	sort.Strings(diff.RemovedCredentials)
}

func cloneUpdateDiff(diff UpdateDiff) UpdateDiff {
	diff.AddedTargets = append([]string(nil), diff.AddedTargets...)
	diff.ChangedTargets = append([]string(nil), diff.ChangedTargets...)
	diff.RemovedTargets = append([]string(nil), diff.RemovedTargets...)
	diff.AddedCredentials = append([]string(nil), diff.AddedCredentials...)
	diff.ChangedCredentials = append([]string(nil), diff.ChangedCredentials...)
	diff.RemovedCredentials = append([]string(nil), diff.RemovedCredentials...)
	return diff
}

func cloneReadinessReport(report ReadinessReport) ReadinessReport {
	report.Blockers = append([]Blocker(nil), report.Blockers...)
	for index := range report.Blockers {
		report.Blockers[index].Blocks = append([]ExecutionScope(nil), report.Blockers[index].Blocks...)
	}
	return report
}
