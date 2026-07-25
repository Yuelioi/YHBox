package catalog

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/workflowinstallation"
)

// WorkflowInstallationRepository atomically commits a verified immutable
// Release projection and one local Installation. It never accepts unsigned
// Source imports as releases.
type WorkflowInstallationRepository struct {
	database *database
}

func (r *WorkflowInstallationRepository) Commit(
	ctx context.Context,
	release workflowinstallation.ReleaseRecord,
	installation workflowinstallation.InstallationRecord,
	configuration workflowinstallation.Configuration,
) error {
	if err := r.ready(); err != nil {
		return err
	}
	if ctx == nil {
		return errors.New("commit Workflow Installation requires a context")
	}
	if err := workflowinstallation.ValidateReleaseRecord(release); err != nil {
		return err
	}
	if err := workflowinstallation.ValidateInstallationRecord(installation); err != nil {
		return err
	}
	if installation.ReleaseID != release.ID {
		return errors.New("Workflow Installation release identity does not match committed Release")
	}
	if err := workflowinstallation.ValidateConfiguration(configuration); err != nil {
		return err
	}
	if configuration.InstallationID != installation.ID || configuration.Generation != 1 ||
		!configuration.UpdatedAt.Equal(installation.CreatedAt) {
		return errors.New("Workflow Installation initial configuration does not match Installation")
	}
	tx, err := r.database.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `
		INSERT INTO workflow_releases(
			release_id, source_hash, workflow_id, workflow_name, publisher_namespace,
			release_version, attestation_digest, source_artifact, verified_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(release_id) DO NOTHING
	`, release.ID.String(), release.SourceHash.String(), release.WorkflowID, release.WorkflowName,
		release.PublisherNamespace, release.ReleaseVersion, release.AttestationDigest.String(),
		release.SourceArtifact, release.VerifiedAt.Format(time.RFC3339Nano))
	if err != nil {
		return errors.Join(err, tx.Rollback())
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		return errors.Join(err, tx.Rollback())
	}
	if inserted == 0 {
		current, found, err := getWorkflowRelease(ctx, tx, release.ID)
		if err != nil {
			return errors.Join(err, tx.Rollback())
		}
		if !found || !equalWorkflowRelease(current, release) {
			return errors.Join(workflowinstallation.ErrReleaseConflict, tx.Rollback())
		}
	}
	result, err = tx.ExecContext(ctx, `
		INSERT INTO workflow_installations(
			installation_id, release_id, name, lifecycle, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(installation_id) DO NOTHING
	`, installation.ID, installation.ReleaseID.String(), installation.Name, string(installation.Lifecycle),
		installation.CreatedAt.Format(time.RFC3339Nano), installation.UpdatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return errors.Join(err, tx.Rollback())
	}
	inserted, err = result.RowsAffected()
	if err != nil {
		return errors.Join(err, tx.Rollback())
	}
	if inserted == 0 {
		return errors.Join(workflowinstallation.ErrInstallationConflict, tx.Rollback())
	}
	profiles, targets, credentials, err := marshalConfiguration(configuration)
	if err != nil {
		return errors.Join(err, tx.Rollback())
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO workflow_installation_configurations(
			installation_id, generation, target_profiles, target_bindings, credential_bindings,
			run_consent_release, schedule_consent_release, updated_at
		) VALUES (?, ?, ?, ?, ?, NULL, NULL, ?)
	`, configuration.InstallationID, configuration.Generation, profiles, targets, credentials,
		configuration.UpdatedAt.Format(time.RFC3339Nano)); err != nil {
		return errors.Join(err, tx.Rollback())
	}
	return tx.Commit()
}

func (r *WorkflowInstallationRepository) GetInstallation(
	ctx context.Context,
	installationID string,
) (workflowinstallation.InstallationRecord, bool, error) {
	if err := r.ready(); err != nil {
		return workflowinstallation.InstallationRecord{}, false, err
	}
	row := r.database.db.QueryRowContext(ctx, `
		SELECT installation_id, release_id, name, lifecycle, created_at, updated_at
		FROM workflow_installations WHERE installation_id = ?
	`, installationID)
	record, err := scanWorkflowInstallation(row)
	if errors.Is(err, sql.ErrNoRows) {
		return workflowinstallation.InstallationRecord{}, false, nil
	}
	return record, err == nil, err
}

func (r *WorkflowInstallationRepository) ListInstallations(
	ctx context.Context,
) ([]workflowinstallation.InstallationRecord, error) {
	if err := r.ready(); err != nil {
		return nil, err
	}
	rows, err := r.database.db.QueryContext(ctx, `
		SELECT installation_id, release_id, name, lifecycle, created_at, updated_at
		FROM workflow_installations ORDER BY installation_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]workflowinstallation.InstallationRecord, 0)
	for rows.Next() {
		record, err := scanWorkflowInstallation(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, record)
	}
	return result, rows.Err()
}

func (r *WorkflowInstallationRepository) GetRelease(
	ctx context.Context,
	releaseID artifact.Digest,
) (workflowinstallation.ReleaseRecord, bool, error) {
	if err := r.ready(); err != nil {
		return workflowinstallation.ReleaseRecord{}, false, err
	}
	return getWorkflowRelease(ctx, r.database.db, releaseID)
}

func (r *WorkflowInstallationRepository) GetConfiguration(
	ctx context.Context,
	installationID string,
) (workflowinstallation.Configuration, bool, error) {
	if err := r.ready(); err != nil {
		return workflowinstallation.Configuration{}, false, err
	}
	return getWorkflowInstallationConfiguration(ctx, r.database.db, installationID)
}

func (r *WorkflowInstallationRepository) ReplaceConfiguration(
	ctx context.Context,
	expectedGeneration int64,
	next workflowinstallation.Configuration,
) error {
	if err := r.ready(); err != nil {
		return err
	}
	if ctx == nil || expectedGeneration < 1 || next.Generation != expectedGeneration+1 {
		return workflowinstallation.ErrInstallationConflict
	}
	if err := workflowinstallation.ValidateConfiguration(next); err != nil {
		return err
	}
	profiles, targets, credentials, err := marshalConfiguration(next)
	if err != nil {
		return err
	}
	result, err := r.database.db.ExecContext(ctx, `
		UPDATE workflow_installation_configurations
		SET generation = ?, target_profiles = ?, target_bindings = ?, credential_bindings = ?,
		    run_consent_release = ?, schedule_consent_release = ?, updated_at = ?
		WHERE installation_id = ? AND generation = ?
	`, next.Generation, profiles, targets, credentials, nullableDigest(next.RunConsentRelease),
		nullableDigest(next.ScheduleConsentRelease), next.UpdatedAt.Format(time.RFC3339Nano),
		next.InstallationID, expectedGeneration)
	if err != nil {
		return err
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if updated != 1 {
		return workflowinstallation.ErrInstallationConflict
	}
	return nil
}

func (r *WorkflowInstallationRepository) ready() error {
	if r == nil || r.database == nil || r.database.db == nil {
		return errors.New("Workflow Installation repository is closed")
	}
	return nil
}

type rowScanner interface {
	Scan(...any) error
}

func scanWorkflowInstallation(row rowScanner) (workflowinstallation.InstallationRecord, error) {
	var record workflowinstallation.InstallationRecord
	var releaseID, lifecycle, createdAt, updatedAt string
	if err := row.Scan(&record.ID, &releaseID, &record.Name, &lifecycle, &createdAt, &updatedAt); err != nil {
		return workflowinstallation.InstallationRecord{}, err
	}
	record.ReleaseID = artifact.Digest(releaseID)
	record.Lifecycle = workflowinstallation.Lifecycle(lifecycle)
	var err error
	record.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return workflowinstallation.InstallationRecord{}, ErrSchemaDrift
	}
	record.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt)
	if err != nil || workflowinstallation.ValidateInstallationRecord(record) != nil {
		return workflowinstallation.InstallationRecord{}, ErrSchemaDrift
	}
	return record, nil
}

func getWorkflowRelease(
	ctx context.Context,
	query interface {
		QueryRowContext(context.Context, string, ...any) *sql.Row
	},
	releaseID artifact.Digest,
) (workflowinstallation.ReleaseRecord, bool, error) {
	if ctx == nil || !releaseID.Valid() {
		return workflowinstallation.ReleaseRecord{}, false, errors.New("Workflow Release identity is invalid")
	}
	row := query.QueryRowContext(ctx, `
		SELECT release_id, source_hash, workflow_id, workflow_name, publisher_namespace,
		       release_version, attestation_digest, source_artifact, verified_at
		FROM workflow_releases WHERE release_id = ?
	`, releaseID.String())
	var record workflowinstallation.ReleaseRecord
	var rawID, rawSourceHash, rawAttestation, rawVerifiedAt string
	if err := row.Scan(
		&rawID, &rawSourceHash, &record.WorkflowID, &record.WorkflowName, &record.PublisherNamespace,
		&record.ReleaseVersion, &rawAttestation, &record.SourceArtifact, &rawVerifiedAt,
	); errors.Is(err, sql.ErrNoRows) {
		return workflowinstallation.ReleaseRecord{}, false, nil
	} else if err != nil {
		return workflowinstallation.ReleaseRecord{}, false, err
	}
	record.ID = artifact.Digest(rawID)
	record.SourceHash = artifact.Digest(rawSourceHash)
	record.AttestationDigest = artifact.Digest(rawAttestation)
	record.VerifiedAt, _ = time.Parse(time.RFC3339Nano, rawVerifiedAt)
	if workflowinstallation.ValidateReleaseRecord(record) != nil {
		return workflowinstallation.ReleaseRecord{}, false, ErrSchemaDrift
	}
	return workflowinstallation.CloneReleaseRecord(record), true, nil
}

func equalWorkflowRelease(left, right workflowinstallation.ReleaseRecord) bool {
	return left.ID == right.ID && left.SourceHash == right.SourceHash &&
		left.WorkflowID == right.WorkflowID && left.WorkflowName == right.WorkflowName &&
		left.PublisherNamespace == right.PublisherNamespace && left.ReleaseVersion == right.ReleaseVersion &&
		left.AttestationDigest == right.AttestationDigest && left.VerifiedAt.Equal(right.VerifiedAt) &&
		bytes.Equal(left.SourceArtifact, right.SourceArtifact)
}

func getWorkflowInstallationConfiguration(
	ctx context.Context,
	query interface {
		QueryRowContext(context.Context, string, ...any) *sql.Row
	},
	installationID string,
) (workflowinstallation.Configuration, bool, error) {
	if ctx == nil || strings.TrimSpace(installationID) == "" {
		return workflowinstallation.Configuration{}, false, errors.New("Workflow Installation identity is invalid")
	}
	row := query.QueryRowContext(ctx, `
		SELECT installation_id, generation, target_profiles, target_bindings, credential_bindings,
		       run_consent_release, schedule_consent_release, updated_at
		FROM workflow_installation_configurations WHERE installation_id = ?
	`, installationID)
	var configuration workflowinstallation.Configuration
	var profiles, targets, credentials []byte
	var runConsent, scheduleConsent sql.NullString
	var updatedAt string
	if err := row.Scan(
		&configuration.InstallationID, &configuration.Generation, &profiles, &targets, &credentials,
		&runConsent, &scheduleConsent, &updatedAt,
	); errors.Is(err, sql.ErrNoRows) {
		return workflowinstallation.Configuration{}, false, nil
	} else if err != nil {
		return workflowinstallation.Configuration{}, false, err
	}
	if err := decodeCanonicalTargetProfiles(profiles, &configuration.TargetProfiles); err != nil {
		return workflowinstallation.Configuration{}, false, ErrSchemaDrift
	}
	if err := decodeCanonicalBindings(targets, &configuration.TargetBindings); err != nil {
		return workflowinstallation.Configuration{}, false, ErrSchemaDrift
	}
	if err := decodeCanonicalBindings(credentials, &configuration.CredentialBindings); err != nil {
		return workflowinstallation.Configuration{}, false, ErrSchemaDrift
	}
	if runConsent.Valid {
		configuration.RunConsentRelease = artifact.Digest(runConsent.String)
	}
	if scheduleConsent.Valid {
		configuration.ScheduleConsentRelease = artifact.Digest(scheduleConsent.String)
	}
	configuration.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)
	if workflowinstallation.ValidateConfiguration(configuration) != nil {
		return workflowinstallation.Configuration{}, false, ErrSchemaDrift
	}
	return workflowinstallation.CloneConfiguration(configuration), true, nil
}

func marshalConfiguration(configuration workflowinstallation.Configuration) ([]byte, []byte, []byte, error) {
	profiles, err := artifact.Marshal(configuration.TargetProfiles)
	if err != nil {
		return nil, nil, nil, err
	}
	targets, err := artifact.Marshal(configuration.TargetBindings)
	if err != nil {
		return nil, nil, nil, err
	}
	credentials, err := artifact.Marshal(configuration.CredentialBindings)
	if err != nil {
		return nil, nil, nil, err
	}
	return profiles, targets, credentials, nil
}

func decodeCanonicalBindings(raw []byte, target *map[string]string) error {
	canonical, err := artifact.Canonicalize(raw)
	if err != nil || !bytes.Equal(raw, canonical) {
		return errors.New("bindings are not canonical")
	}
	if err := json.Unmarshal(raw, target); err != nil || *target == nil {
		return errors.New("bindings are invalid")
	}
	return nil
}

func decodeCanonicalTargetProfiles(raw []byte, target *map[string]workflowinstallation.TargetProfile) error {
	canonical, err := artifact.Canonicalize(raw)
	if err != nil || !bytes.Equal(raw, canonical) {
		return errors.New("Workflow Target Profiles are not canonical")
	}
	if err := json.Unmarshal(raw, target); err != nil || *target == nil {
		return errors.New("Workflow Target Profiles are invalid")
	}
	return nil
}

func nullableDigest(value artifact.Digest) any {
	if value == "" {
		return nil
	}
	return value.String()
}

var _ workflowinstallation.Repository = (*WorkflowInstallationRepository)(nil)
