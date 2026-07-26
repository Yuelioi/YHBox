// Package workflowinstallation owns verified Workflow Release projections,
// local Workflow Installations, and their current execution readiness.
package workflowinstallation

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/workflow/schema"
)

type Lifecycle string

const (
	LifecycleActive   Lifecycle = "active"
	LifecycleArchived Lifecycle = "archived"
)

var (
	installationIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)
	localReferencePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]{0,255}$`)
	slotPattern           = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$`)
	profileVersionPattern = regexp.MustCompile(`^[1-9][0-9]*$`)
	semverPattern         = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-((?:0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*)(?:\.(?:0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*))*))?(?:\+([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$`)
)

// VerificationReceipt is produced by a trust boundary after it has verified
// the exact release artifact and its Publisher Attestation. This package does
// not turn an unsigned Workflow Source or private bundle into a verified
// release.
type VerificationReceipt struct {
	ReleaseDigest      artifact.Digest
	AttestationDigest  artifact.Digest
	PublisherNamespace string
	ReleaseVersion     string
	VerifiedAt         time.Time
}

// ReleaseRecord is the durable local projection of one verified, immutable
// Workflow Release. SourceArtifact is canonical Workflow Source JSON.
type ReleaseRecord struct {
	ID                 artifact.Digest
	SourceHash         artifact.Digest
	WorkflowID         string
	WorkflowName       string
	PublisherNamespace string
	ReleaseVersion     string
	AttestationDigest  artifact.Digest
	SourceArtifact     []byte
	VerifiedAt         time.Time
}

func NewVerifiedRelease(sourceArtifact []byte, receipt VerificationReceipt) (ReleaseRecord, error) {
	source, canonical, sourceHash, diagnostics, err := schema.CanonicalSource(sourceArtifact)
	if err != nil {
		return ReleaseRecord{}, fmt.Errorf("open verified Workflow Source: %w", err)
	}
	if schema.HasErrors(diagnostics) {
		return ReleaseRecord{}, errors.New("verified Workflow Source has schema diagnostics")
	}
	if !bytes.Equal(sourceArtifact, canonical) {
		return ReleaseRecord{}, errors.New("verified Workflow Source must use canonical bytes")
	}
	record := ReleaseRecord{
		ID: receipt.ReleaseDigest, SourceHash: sourceHash,
		WorkflowID: source.Workflow.ID, WorkflowName: source.Workflow.Name,
		PublisherNamespace: receipt.PublisherNamespace, ReleaseVersion: receipt.ReleaseVersion,
		AttestationDigest: receipt.AttestationDigest,
		SourceArtifact:    append([]byte(nil), canonical...),
		VerifiedAt:        receipt.VerifiedAt,
	}
	if err := ValidateReleaseRecord(record); err != nil {
		return ReleaseRecord{}, err
	}
	return record, nil
}

func ValidateReleaseRecord(record ReleaseRecord) error {
	if !record.ID.Valid() || !record.SourceHash.Valid() || !record.AttestationDigest.Valid() {
		return errors.New("Workflow Release requires exact release, source, and attestation digests")
	}
	if strings.TrimSpace(record.WorkflowID) == "" || strings.TrimSpace(record.WorkflowName) == "" ||
		record.WorkflowID != strings.TrimSpace(record.WorkflowID) || record.WorkflowName != strings.TrimSpace(record.WorkflowName) {
		return errors.New("Workflow Release identity and name are invalid")
	}
	if strings.TrimSpace(record.PublisherNamespace) == "" ||
		record.PublisherNamespace != strings.TrimSpace(record.PublisherNamespace) ||
		len(record.PublisherNamespace) > 1024 ||
		schema.ValidatePublisherNamespace(record.PublisherNamespace) != nil {
		return errors.New("Workflow Release publisher namespace is invalid")
	}
	if !semverPattern.MatchString(record.ReleaseVersion) {
		return errors.New("Workflow Release version must be SemVer")
	}
	if record.VerifiedAt.IsZero() || record.VerifiedAt.Location() != time.UTC {
		return errors.New("Workflow Release verification time must be UTC")
	}
	source, canonical, sourceHash, diagnostics, err := schema.CanonicalSource(record.SourceArtifact)
	if err != nil || schema.HasErrors(diagnostics) || !bytes.Equal(record.SourceArtifact, canonical) ||
		sourceHash != record.SourceHash || source.Workflow.ID != record.WorkflowID ||
		source.Workflow.Name != record.WorkflowName {
		return errors.New("Workflow Release Source projection is inconsistent")
	}
	return nil
}

func CloneReleaseRecord(record ReleaseRecord) ReleaseRecord {
	record.SourceArtifact = append([]byte(nil), record.SourceArtifact...)
	return record
}

type InstallationRecord struct {
	ID                string
	ReleaseID         artifact.Digest
	PreviousReleaseID artifact.Digest
	Name              string
	Lifecycle         Lifecycle
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func ValidateInstallationRecord(record InstallationRecord) error {
	if !installationIDPattern.MatchString(record.ID) || !record.ReleaseID.Valid() ||
		(record.PreviousReleaseID != "" && (!record.PreviousReleaseID.Valid() ||
			record.PreviousReleaseID == record.ReleaseID)) ||
		strings.TrimSpace(record.Name) == "" || record.Name != strings.TrimSpace(record.Name) ||
		(record.Lifecycle != LifecycleActive && record.Lifecycle != LifecycleArchived) ||
		record.CreatedAt.IsZero() || record.UpdatedAt.IsZero() ||
		record.CreatedAt.Location() != time.UTC || record.UpdatedAt.Location() != time.UTC ||
		record.UpdatedAt.Before(record.CreatedAt) {
		return errors.New("Workflow Installation record is invalid")
	}
	return nil
}

type Configuration struct {
	InstallationID         string
	Generation             int64
	TargetProfiles         map[string]TargetProfile
	TargetBindings         map[string]string
	CredentialBindings     map[string]string
	RunConsentRelease      artifact.Digest
	ScheduleConsentRelease artifact.Digest
	UpdatedAt              time.Time
}

type TargetProfile struct {
	DefinitionID         string          `json:"definitionId"`
	ReleaseID            artifact.Digest `json:"releaseId"`
	TargetKind           string          `json:"targetKind"`
	AdapterKind          string          `json:"adapterKind"`
	ProfileVersion       string          `json:"profileVersion"`
	Settings             json.RawMessage `json:"settings"`
	TargetInstallationID string          `json:"targetInstallationId,omitempty"`
}

func NewConfiguration(installation InstallationRecord) Configuration {
	return Configuration{
		InstallationID: installation.ID, Generation: 1,
		TargetProfiles: map[string]TargetProfile{},
		TargetBindings: map[string]string{}, CredentialBindings: map[string]string{},
		UpdatedAt: installation.CreatedAt,
	}
}

func NewConfigurationForRelease(
	installation InstallationRecord,
	release ReleaseRecord,
) (Configuration, error) {
	configuration := NewConfiguration(installation)
	return MaterializeTargetProfiles(configuration, release)
}

func MaterializeTargetProfiles(
	configuration Configuration,
	release ReleaseRecord,
) (Configuration, error) {
	source, diagnostics := schema.ParseSource(release.SourceArtifact)
	if schema.HasErrors(diagnostics) {
		return Configuration{}, errors.New("Workflow Release Source is invalid")
	}
	next := CloneConfiguration(configuration)
	if next.TargetProfiles == nil {
		next.TargetProfiles = map[string]TargetProfile{}
	}
	definitionIDs := make(map[string]struct{}, len(source.TargetProfileDefinitions))
	for _, definition := range source.TargetProfileDefinitions {
		definitionIDs[definition.ID] = struct{}{}
		if current, found := next.TargetProfiles[definition.ID]; found {
			if current.ReleaseID != release.ID || current.DefinitionID != definition.ID ||
				current.TargetKind != definition.TargetKind || current.AdapterKind != definition.AdapterKind ||
				current.ProfileVersion != definition.ProfileVersion {
				return Configuration{}, errors.New("Workflow Target Profile does not match its Release definition")
			}
			settings, err := schema.ValidateTargetProfileSettings(definition, current.Settings)
			if err != nil {
				return Configuration{}, fmt.Errorf(
					"validate materialized Workflow Target Profile %q: %w",
					definition.ID, err,
				)
			}
			if !bytes.Equal(current.Settings, settings) {
				return Configuration{}, errors.New("materialized Workflow Target Profile settings are not canonical")
			}
			continue
		}
		settings, err := schema.ValidateTargetProfileSettings(definition, definition.InitialDefaults)
		if err != nil {
			return Configuration{}, fmt.Errorf("materialize Workflow Target Profile %q: %w", definition.ID, err)
		}
		next.TargetProfiles[definition.ID] = TargetProfile{
			DefinitionID: definition.ID, ReleaseID: release.ID,
			TargetKind: definition.TargetKind, AdapterKind: definition.AdapterKind,
			ProfileVersion: definition.ProfileVersion, Settings: settings,
			TargetInstallationID: next.TargetBindings[definition.ID],
		}
	}
	for definitionID := range next.TargetProfiles {
		if _, found := definitionIDs[definitionID]; !found {
			return Configuration{}, errors.New("Workflow Target Profile references an unknown Release definition")
		}
	}
	return next, nil
}

func ValidateConfiguration(configuration Configuration) error {
	if !installationIDPattern.MatchString(configuration.InstallationID) ||
		configuration.Generation < 1 || configuration.UpdatedAt.IsZero() ||
		configuration.UpdatedAt.Location() != time.UTC ||
		(configuration.RunConsentRelease != "" && !configuration.RunConsentRelease.Valid()) ||
		(configuration.ScheduleConsentRelease != "" && !configuration.ScheduleConsentRelease.Valid()) {
		return errors.New("Workflow Installation configuration is invalid")
	}
	for definitionID, profile := range configuration.TargetProfiles {
		canonical, err := artifact.Canonicalize(profile.Settings)
		if !slotPattern.MatchString(definitionID) || definitionID != profile.DefinitionID ||
			!profile.ReleaseID.Valid() || !slotPattern.MatchString(profile.TargetKind) ||
			!slotPattern.MatchString(profile.AdapterKind) || !profileVersionPattern.MatchString(profile.ProfileVersion) ||
			err != nil || !bytes.Equal(profile.Settings, canonical) ||
			profile.TargetInstallationID != "" && !localReferencePattern.MatchString(profile.TargetInstallationID) {
			return errors.New("Workflow Target Profile is invalid")
		}
		if configuration.TargetBindings[definitionID] != profile.TargetInstallationID {
			return errors.New("Workflow Target Profile binding is inconsistent")
		}
	}
	for slot, reference := range configuration.TargetBindings {
		if !slotPattern.MatchString(slot) || !localReferencePattern.MatchString(reference) {
			return errors.New("Workflow Installation target binding is invalid")
		}
		if _, found := configuration.TargetProfiles[slot]; len(configuration.TargetProfiles) != 0 && !found {
			return errors.New("Workflow Installation target binding has no materialized profile")
		}
	}
	for slot, reference := range configuration.CredentialBindings {
		if !slotPattern.MatchString(slot) || !localReferencePattern.MatchString(reference) {
			return errors.New("Workflow Installation credential binding is invalid")
		}
	}
	return nil
}

func CloneConfiguration(configuration Configuration) Configuration {
	configuration.TargetProfiles = cloneTargetProfiles(configuration.TargetProfiles)
	configuration.TargetBindings = cloneBindings(configuration.TargetBindings)
	configuration.CredentialBindings = cloneBindings(configuration.CredentialBindings)
	return configuration
}

func cloneTargetProfiles(source map[string]TargetProfile) map[string]TargetProfile {
	result := make(map[string]TargetProfile, len(source))
	for key, value := range source {
		value.Settings = append(json.RawMessage(nil), value.Settings...)
		result[key] = value
	}
	return result
}

func cloneBindings(source map[string]string) map[string]string {
	result := make(map[string]string, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

type BlockerKind string

const (
	BlockerDependency      BlockerKind = "dependency"
	BlockerTarget          BlockerKind = "target"
	BlockerCredential      BlockerKind = "credential"
	BlockerRunConsent      BlockerKind = "run-consent"
	BlockerScheduleConsent BlockerKind = "schedule-consent"
)

type RepairActionKind string

const (
	ActionInstallDependency RepairActionKind = "install-dependency"
	ActionConfigureTarget   RepairActionKind = "configure-target"
	ActionBindCredential    RepairActionKind = "bind-credential"
	ActionGrantRunConsent   RepairActionKind = "grant-run-consent"
	ActionGrantSchedule     RepairActionKind = "grant-schedule-consent"
)

type ExecutionScope string

const (
	ScopeRun      ExecutionScope = "run"
	ScopeSchedule ExecutionScope = "schedule"
)

type RepairAction struct {
	Kind          RepairActionKind
	RequirementID string
}

type Blocker struct {
	Kind          BlockerKind
	RequirementID string
	Expected      string
	Blocks        []ExecutionScope
	Action        RepairAction
}

type DependencyState struct {
	PublisherNamespace string
	PackageID          string
	PackageVersion     string
	ManifestDigest     artifact.Digest
	Enabled            bool
}

type TargetState struct {
	TargetInstallationID string
	TargetKind           string
	AdapterKind          string
	ProfileVersion       string
	Available            bool
	Authorized           bool
}

type CredentialState struct {
	CredentialBindingID string
	Kind                string
	Label               string
	Available           bool
}

// ReadinessEnvironment is a projection of installation-local bindings and
// exact host package state. Secret material is never part of this value.
type ReadinessEnvironment struct {
	Dependencies []DependencyState
	Targets      []TargetState
	Credentials  []CredentialState
}

type ReadinessReport struct {
	InstallationID           string
	ReleaseID                artifact.Digest
	Lifecycle                Lifecycle
	LifecycleAllowsExecution bool
	RunAllowed               bool
	ScheduleAllowed          bool
	Blockers                 []Blocker
}

func EvaluateReadiness(
	installation InstallationRecord,
	release ReleaseRecord,
	configuration Configuration,
	environment ReadinessEnvironment,
) (ReadinessReport, error) {
	if err := ValidateInstallationRecord(installation); err != nil {
		return ReadinessReport{}, err
	}
	if err := ValidateReleaseRecord(release); err != nil {
		return ReadinessReport{}, err
	}
	if installation.ReleaseID != release.ID {
		return ReadinessReport{}, errors.New("Workflow Installation does not reference the supplied Release")
	}
	if err := ValidateConfiguration(configuration); err != nil {
		return ReadinessReport{}, err
	}
	if configuration.InstallationID != installation.ID {
		return ReadinessReport{}, errors.New("Workflow Installation configuration identity does not match")
	}
	source, diagnostics := schema.ParseSource(release.SourceArtifact)
	if schema.HasErrors(diagnostics) {
		return ReadinessReport{}, errors.New("Workflow Release Source is invalid")
	}

	blockers := make([]Blocker, 0,
		len(source.Dependencies)+len(source.TargetProfileDefinitions)+len(source.CredentialRequirements)+2)
	dependencies := make(map[string]DependencyState, len(environment.Dependencies))
	for _, current := range environment.Dependencies {
		dependencies[current.PackageID] = current
	}
	for _, required := range source.Dependencies {
		current, ok := dependencies[required.PackageID]
		if ok && current.Enabled && current.PublisherNamespace == required.PublisherNamespace &&
			current.PackageVersion == required.PackageVersion && current.ManifestDigest == required.ManifestDigest {
			continue
		}
		blockers = append(blockers, newBlocker(
			BlockerDependency, required.PackageID,
			fmt.Sprintf("%s@%s#%s", required.PublisherNamespace, required.PackageVersion, required.ManifestDigest),
			ActionInstallDependency, ScopeRun, ScopeSchedule,
		))
	}
	for _, definition := range source.TargetProfileDefinitions {
		profile, materialized := configuration.TargetProfiles[definition.ID]
		targetAvailable := false
		for _, target := range environment.Targets {
			if target.TargetInstallationID == profile.TargetInstallationID &&
				target.TargetKind == definition.TargetKind &&
				target.AdapterKind == definition.AdapterKind &&
				target.ProfileVersion == definition.ProfileVersion &&
				target.Available && target.Authorized {
				targetAvailable = true
				break
			}
		}
		if materialized && profile.ReleaseID == release.ID &&
			profile.TargetKind == definition.TargetKind && profile.AdapterKind == definition.AdapterKind &&
			profile.ProfileVersion == definition.ProfileVersion &&
			strings.TrimSpace(profile.TargetInstallationID) != "" && targetAvailable {
			continue
		}
		blockers = append(blockers, newBlocker(
			BlockerTarget, definition.ID, definition.TargetKind+"/"+definition.AdapterKind,
			ActionConfigureTarget, ScopeRun, ScopeSchedule,
		))
	}
	for _, requirement := range source.CredentialRequirements {
		bindingID := configuration.CredentialBindings[requirement.Slot]
		credentialAvailable := false
		for _, credential := range environment.Credentials {
			if credential.CredentialBindingID == bindingID &&
				credential.Kind == requirement.Kind && credential.Available {
				credentialAvailable = true
				break
			}
		}
		if strings.TrimSpace(bindingID) != "" && credentialAvailable {
			continue
		}
		blockers = append(blockers, newBlocker(
			BlockerCredential, requirement.Slot, requirement.Kind,
			ActionBindCredential, ScopeRun, ScopeSchedule,
		))
	}
	if configuration.RunConsentRelease != release.ID {
		blockers = append(blockers, newBlocker(
			BlockerRunConsent, "workflow-execution", release.ID.String(),
			ActionGrantRunConsent, ScopeRun,
		))
	}
	if configuration.ScheduleConsentRelease != release.ID {
		blockers = append(blockers, newBlocker(
			BlockerScheduleConsent, "scheduled-execution", release.ID.String(),
			ActionGrantSchedule, ScopeSchedule,
		))
	}
	sort.SliceStable(blockers, func(i, j int) bool {
		if blockers[i].Kind != blockers[j].Kind {
			return blockers[i].Kind < blockers[j].Kind
		}
		return blockers[i].RequirementID < blockers[j].RequirementID
	})

	active := installation.Lifecycle == LifecycleActive
	report := ReadinessReport{
		InstallationID: installation.ID, ReleaseID: release.ID, Lifecycle: installation.Lifecycle,
		LifecycleAllowsExecution: active, RunAllowed: active, ScheduleAllowed: active,
		Blockers: blockers,
	}
	for _, blocker := range blockers {
		for _, scope := range blocker.Blocks {
			switch scope {
			case ScopeRun:
				report.RunAllowed = false
			case ScopeSchedule:
				report.ScheduleAllowed = false
			}
		}
	}
	return report, nil
}

func newBlocker(
	kind BlockerKind,
	requirementID string,
	expected string,
	action RepairActionKind,
	scopes ...ExecutionScope,
) Blocker {
	return Blocker{
		Kind: kind, RequirementID: requirementID, Expected: expected,
		Blocks: append([]ExecutionScope(nil), scopes...),
		Action: RepairAction{Kind: action, RequirementID: requirementID},
	}
}
