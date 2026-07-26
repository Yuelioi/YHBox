package appbootstrap

import (
	"context"
	"errors"

	"github.com/yottaapp/yotta/internal/admission"
	appcore "github.com/yottaapp/yotta/internal/application"
	"github.com/yottaapp/yotta/internal/workflowinstallation"
	"github.com/yottaapp/yotta/internal/workflowstore"
)

// StartInstallationRun is the only composition command that turns a local
// Workflow Installation into a Run. Readiness, immutable release bytes, and
// local target/credential selections are prepared by one Module before the
// shared Application execution path is entered.
func (r *Runtime) StartInstallationRun(
	ctx context.Context,
	installationID string,
	scope workflowinstallation.ExecutionScope,
) (appcore.StartRunResult, error) {
	if r == nil || r.Application == nil || r.Installations == nil {
		return appcore.StartRunResult{}, errors.New("workflow installation runtime is unavailable")
	}
	prepared, err := r.Installations.PrepareExecution(ctx, installationID, scope)
	if err != nil {
		return appcore.StartRunResult{}, err
	}
	return r.Application.StartArtifactRun(ctx, appcore.StartArtifactRunRequest{
		SourceArtifact: prepared.SourceArtifact(),
		Principal:      "local-user",
		Selection: admission.Selection{
			Targets: prepared.TargetSelection(), Credentials: prepared.CredentialSelection(),
		},
	})
}

func (r *Runtime) ListWorkflowInstallations(ctx context.Context) ([]workflowinstallation.InstallationRecord, error) {
	if r == nil || r.Installations == nil {
		return nil, errors.New("workflow installation runtime is unavailable")
	}
	return r.Installations.List(ctx)
}

func (r *Runtime) DeriveWorkflowInstallationSource(
	ctx context.Context,
	installationID string,
	name string,
) (workflowstore.SourceSnapshot, error) {
	if r == nil || r.Application == nil || r.Installations == nil {
		return workflowstore.SourceSnapshot{}, errors.New("workflow installation runtime is unavailable")
	}
	prepared, err := r.Installations.PrepareDerivedSource(ctx, installationID, name)
	if err != nil {
		return workflowstore.SourceSnapshot{}, err
	}
	return r.Application.PublishImportedSource(ctx, prepared.SourceArtifact(), -1, "")
}

func (r *Runtime) PrepareWorkflowInstallationUpdate(
	ctx context.Context,
	installationID string,
	candidate workflowinstallation.ReleaseRecord,
) (workflowinstallation.PreparedUpdate, error) {
	if r == nil || r.Installations == nil {
		return workflowinstallation.PreparedUpdate{}, errors.New("workflow installation runtime is unavailable")
	}
	return r.Installations.PrepareUpdate(ctx, installationID, candidate)
}

func (r *Runtime) ApplyWorkflowInstallationUpdate(
	ctx context.Context,
	prepared workflowinstallation.PreparedUpdate,
) (workflowinstallation.InstallationRecord, error) {
	if r == nil || r.Installations == nil {
		return workflowinstallation.InstallationRecord{}, errors.New("workflow installation runtime is unavailable")
	}
	return r.Installations.ApplyUpdate(ctx, prepared)
}

func (r *Runtime) WorkflowInstallationReadiness(
	ctx context.Context,
	installationID string,
) (workflowinstallation.ReadinessReport, error) {
	if r == nil || r.Installations == nil {
		return workflowinstallation.ReadinessReport{}, errors.New("workflow installation runtime is unavailable")
	}
	return r.Installations.Readiness(ctx, installationID)
}

func (r *Runtime) WorkflowInstallationSettings(
	ctx context.Context,
	installationID string,
) (workflowinstallation.SettingsSnapshot, error) {
	if r == nil || r.Installations == nil {
		return workflowinstallation.SettingsSnapshot{}, errors.New("workflow installation runtime is unavailable")
	}
	return r.Installations.Settings(ctx, installationID)
}

func (r *Runtime) UpdateWorkflowInstallationTargetProfile(
	ctx context.Context,
	installationID string,
	expectedGeneration int64,
	definitionID string,
	settings []byte,
	targetInstallationID string,
) (workflowinstallation.SettingsSnapshot, error) {
	if r == nil || r.Installations == nil {
		return workflowinstallation.SettingsSnapshot{}, errors.New("workflow installation runtime is unavailable")
	}
	if _, err := r.Installations.UpdateTargetProfile(
		ctx, installationID, expectedGeneration, definitionID, settings, targetInstallationID,
	); err != nil {
		return workflowinstallation.SettingsSnapshot{}, err
	}
	return r.Installations.Settings(ctx, installationID)
}

func (r *Runtime) UpdateWorkflowInstallationCredentialBinding(
	ctx context.Context,
	installationID string,
	expectedGeneration int64,
	requirementSlot string,
	credentialBindingID string,
) (workflowinstallation.SettingsSnapshot, error) {
	if r == nil || r.Installations == nil {
		return workflowinstallation.SettingsSnapshot{}, errors.New("workflow installation runtime is unavailable")
	}
	if _, err := r.Installations.UpdateCredentialBinding(
		ctx, installationID, expectedGeneration, requirementSlot, credentialBindingID,
	); err != nil {
		return workflowinstallation.SettingsSnapshot{}, err
	}
	return r.Installations.Settings(ctx, installationID)
}

func (r *Runtime) GrantWorkflowInstallationConsent(
	ctx context.Context,
	installationID string,
	scope workflowinstallation.ExecutionScope,
) (workflowinstallation.ReadinessReport, error) {
	if r == nil || r.Installations == nil {
		return workflowinstallation.ReadinessReport{}, errors.New("workflow installation runtime is unavailable")
	}
	configuration, err := r.Installations.GetConfiguration(ctx, installationID)
	if err != nil {
		return workflowinstallation.ReadinessReport{}, err
	}
	if _, err := r.Installations.GrantConsent(
		ctx, installationID, configuration.Generation, scope,
	); err != nil {
		return workflowinstallation.ReadinessReport{}, err
	}
	return r.Installations.Readiness(ctx, installationID)
}
