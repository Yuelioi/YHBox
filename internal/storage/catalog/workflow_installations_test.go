package catalog

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/workflow/schema"
	"github.com/yottaapp/yotta/internal/workflowinstallation"
)

func TestWorkflowInstallationRepositoryCommitsReleaseAndMultipleInstances(t *testing.T) {
	foundation, err := Open(context.Background(), testRoots(t))
	if err != nil {
		t.Fatal(err)
	}
	defer foundation.Close()
	repository := foundation.WorkflowInstallations()
	release := catalogTestVerifiedRelease(t)
	now := time.Date(2026, 7, 26, 2, 0, 0, 0, time.UTC)
	first := workflowinstallation.InstallationRecord{
		ID: "installation-a", ReleaseID: release.ID, Name: "First",
		Lifecycle: workflowinstallation.LifecycleActive, CreatedAt: now, UpdatedAt: now,
	}
	second := first
	second.ID = "installation-b"
	second.Name = "Second"
	firstConfiguration, err := workflowinstallation.NewConfigurationForRelease(first, release)
	if err != nil {
		t.Fatal(err)
	}
	secondConfiguration, err := workflowinstallation.NewConfigurationForRelease(second, release)
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.Commit(context.Background(), release, first, firstConfiguration); err != nil {
		t.Fatal(err)
	}
	if err := repository.Commit(context.Background(), release, second, secondConfiguration); err != nil {
		t.Fatal(err)
	}
	installations, err := repository.ListInstallations(context.Background())
	if err != nil || len(installations) != 2 || installations[0].ID != first.ID || installations[1].ID != second.ID {
		t.Fatalf("ListInstallations() = %#v, %v", installations, err)
	}
	loadedRelease, found, err := repository.GetRelease(context.Background(), release.ID)
	if err != nil || !found || loadedRelease.SourceHash != release.SourceHash ||
		string(loadedRelease.SourceArtifact) != string(release.SourceArtifact) {
		t.Fatalf("GetRelease() = %#v, found=%v, err=%v", loadedRelease, found, err)
	}
	configuration, found, err := repository.GetConfiguration(context.Background(), first.ID)
	if err != nil || !found || configuration.Generation != 1 ||
		len(configuration.TargetProfiles) != 1 || len(configuration.TargetBindings) != 0 ||
		len(configuration.CredentialBindings) != 0 {
		t.Fatalf("GetConfiguration() = %#v, found=%v, err=%v", configuration, found, err)
	}
	configuration.Generation = 2
	configuration.TargetBindings["desktop"] = "target-a"
	profile := configuration.TargetProfiles["desktop"]
	profile.TargetInstallationID = "target-a"
	profile.Settings = []byte(`{"windowTitle":"Changed"}`)
	configuration.TargetProfiles["desktop"] = profile
	configuration.CredentialBindings["api"] = "credential-a"
	configuration.UpdatedAt = now.Add(time.Minute)
	if err := repository.ReplaceConfiguration(context.Background(), 1, configuration); err != nil {
		t.Fatal(err)
	}
	reloaded, found, err := repository.GetConfiguration(context.Background(), first.ID)
	if err != nil || !found || reloaded.Generation != 2 ||
		reloaded.TargetBindings["desktop"] != "target-a" ||
		reloaded.TargetProfiles["desktop"].TargetInstallationID != "target-a" ||
		string(reloaded.TargetProfiles["desktop"].Settings) != `{"windowTitle":"Changed"}` ||
		reloaded.CredentialBindings["api"] != "credential-a" {
		t.Fatalf("reloaded configuration = %#v, found=%v, err=%v", reloaded, found, err)
	}
	if err := repository.ReplaceConfiguration(context.Background(), 1, configuration); !errors.Is(err, workflowinstallation.ErrInstallationConflict) {
		t.Fatalf("stale ReplaceConfiguration() error = %v", err)
	}
}

func TestWorkflowInstallationRepositoryRejectsIdentityCollisionsAtomically(t *testing.T) {
	foundation, err := Open(context.Background(), testRoots(t))
	if err != nil {
		t.Fatal(err)
	}
	defer foundation.Close()
	repository := foundation.WorkflowInstallations()
	release := catalogTestVerifiedRelease(t)
	now := time.Date(2026, 7, 26, 2, 0, 0, 0, time.UTC)
	installation := workflowinstallation.InstallationRecord{
		ID: "installation-a", ReleaseID: release.ID, Name: "First",
		Lifecycle: workflowinstallation.LifecycleActive, CreatedAt: now, UpdatedAt: now,
	}
	configuration, err := workflowinstallation.NewConfigurationForRelease(installation, release)
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.Commit(context.Background(), release, installation, configuration); err != nil {
		t.Fatal(err)
	}
	collision := release
	collision.WorkflowName = "Changed"
	source, diagnostics := schema.ParseSource(collision.SourceArtifact)
	if schema.HasErrors(diagnostics) {
		t.Fatal("test source is invalid")
	}
	source.Workflow.Name = collision.WorkflowName
	collision.SourceArtifact, err = artifact.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	_, _, collision.SourceHash, _, err = schema.CanonicalSource(collision.SourceArtifact)
	if err != nil {
		t.Fatal(err)
	}
	other := installation
	other.ID = "installation-b"
	otherConfiguration, err := workflowinstallation.NewConfigurationForRelease(other, collision)
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.Commit(context.Background(), collision, other, otherConfiguration); !errors.Is(err, workflowinstallation.ErrReleaseConflict) {
		t.Fatalf("release collision error = %v", err)
	}
	if _, found, err := repository.GetInstallation(context.Background(), other.ID); err != nil || found {
		t.Fatalf("colliding commit published installation: found=%v err=%v", found, err)
	}
}

func TestWorkflowInstallationRepositorySwitchesReleaseAndConfigurationAtomically(t *testing.T) {
	foundation, err := Open(context.Background(), testRoots(t))
	if err != nil {
		t.Fatal(err)
	}
	defer foundation.Close()
	repository := foundation.WorkflowInstallations()
	current := catalogTestVerifiedRelease(t)
	candidate := catalogTestUpdatedRelease(t, current, "2.0.0")
	now := time.Date(2026, 7, 26, 9, 0, 0, 0, time.UTC)
	installation := workflowinstallation.InstallationRecord{
		ID: "installation-update", ReleaseID: current.ID, Name: "Installed",
		Lifecycle: workflowinstallation.LifecycleActive, CreatedAt: now, UpdatedAt: now,
	}
	configuration, err := workflowinstallation.NewConfigurationForRelease(installation, current)
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.Commit(context.Background(), current, installation, configuration); err != nil {
		t.Fatal(err)
	}
	if err := repository.CacheRelease(context.Background(), candidate); err != nil {
		t.Fatal(err)
	}
	beforeSwitch, found, err := repository.GetInstallation(context.Background(), installation.ID)
	if err != nil || !found || beforeSwitch.ReleaseID != current.ID {
		t.Fatalf("CacheRelease changed Installation = %#v, found=%v err=%v", beforeSwitch, found, err)
	}
	switchedAt := now.Add(time.Minute)
	nextInstallation := installation
	nextInstallation.ReleaseID = candidate.ID
	nextInstallation.PreviousReleaseID = current.ID
	nextInstallation.UpdatedAt = switchedAt
	nextConfiguration := workflowinstallation.CloneConfiguration(configuration)
	nextConfiguration.Generation++
	nextConfiguration.UpdatedAt = switchedAt
	profile := nextConfiguration.TargetProfiles["desktop"]
	profile.ReleaseID = candidate.ID
	nextConfiguration.TargetProfiles["desktop"] = profile
	if err := repository.SwitchRelease(
		context.Background(), current.ID, configuration.Generation,
		candidate, nextInstallation, nextConfiguration,
	); err != nil {
		t.Fatal(err)
	}
	loaded, found, err := repository.GetInstallation(context.Background(), installation.ID)
	if err != nil || !found || loaded.ReleaseID != candidate.ID ||
		loaded.PreviousReleaseID != current.ID || !loaded.UpdatedAt.Equal(switchedAt) {
		t.Fatalf("updated Installation = %#v, found=%v err=%v", loaded, found, err)
	}
	loadedConfiguration, found, err := repository.GetConfiguration(context.Background(), installation.ID)
	if err != nil || !found || loadedConfiguration.Generation != 2 ||
		loadedConfiguration.TargetProfiles["desktop"].ReleaseID != candidate.ID {
		t.Fatalf("updated configuration = %#v, found=%v err=%v", loadedConfiguration, found, err)
	}
	releases, err := repository.ListReleases(
		context.Background(), current.PublisherNamespace, current.WorkflowID,
	)
	if err != nil || len(releases) != 2 || releases[0].ID != candidate.ID || releases[1].ID != current.ID {
		t.Fatalf("ListReleases() = %#v, %v", releases, err)
	}

	staleCandidate := catalogTestUpdatedRelease(t, current, "3.0.0")
	staleInstallation := nextInstallation
	staleInstallation.ReleaseID = staleCandidate.ID
	staleInstallation.PreviousReleaseID = candidate.ID
	staleInstallation.UpdatedAt = switchedAt.Add(time.Minute)
	staleConfiguration := workflowinstallation.CloneConfiguration(nextConfiguration)
	staleConfiguration.UpdatedAt = staleInstallation.UpdatedAt
	profile = staleConfiguration.TargetProfiles["desktop"]
	profile.ReleaseID = staleCandidate.ID
	staleConfiguration.TargetProfiles["desktop"] = profile
	if err := repository.SwitchRelease(
		context.Background(), candidate.ID, configuration.Generation,
		staleCandidate, staleInstallation, staleConfiguration,
	); !errors.Is(err, workflowinstallation.ErrInstallationConflict) {
		t.Fatalf("stale SwitchRelease error = %v", err)
	}
	if _, found, err := repository.GetRelease(context.Background(), staleCandidate.ID); err != nil || found {
		t.Fatalf("stale transaction published candidate Release: found=%v err=%v", found, err)
	}
	loaded, found, err = repository.GetInstallation(context.Background(), installation.ID)
	if err != nil || !found || loaded.ReleaseID != candidate.ID {
		t.Fatalf("stale transaction changed Installation: %#v found=%v err=%v", loaded, found, err)
	}
}

func catalogTestVerifiedRelease(t *testing.T) workflowinstallation.ReleaseRecord {
	t.Helper()
	raw := []byte(`{
		"format":"yotta.workflow","version":"1",
		"workflow":{"id":"release-workflow","name":"Released"},
		"revision":0,"entryGraph":"main",
		"graphs":[{"id":"main","kind":"main","nodes":[],"edges":[],"inputs":[],"outputs":[]}],
		"resources":[],
		"targetDefaults":[{"target":"desktop-target","slot":"desktop"}],
		"targetProfileDefinitions":[{
			"id":"desktop","name":"Desktop","targetKind":"desktop","adapterKind":"windows",
			"profileVersion":"1","settingsSchemaRoot":"https://example.test/desktop/v1",
			"settingsSchemaBundle":[{"id":"https://example.test/desktop/v1","schema":{"$schema":"https://json-schema.org/draft/2020-12/schema","$id":"https://example.test/desktop/v1","type":"object","properties":{"windowTitle":{"type":"string"}},"additionalProperties":false}}],
			"initialDefaults":{"windowTitle":"Released"},"discoveryHints":[]
		}],
		"credentialRequirements":[],"dependencies":[],"variables":[]
	}`)
	canonical, err := artifact.Canonicalize(raw)
	if err != nil {
		t.Fatal(err)
	}
	releaseID, err := artifact.Sum("yotta/test/workflow-release/v1", []byte("release"))
	if err != nil {
		t.Fatal(err)
	}
	attestationID, err := artifact.Sum("yotta/test/attestation/v1", []byte("attestation"))
	if err != nil {
		t.Fatal(err)
	}
	release, err := workflowinstallation.NewVerifiedRelease(canonical, workflowinstallation.VerificationReceipt{
		ReleaseDigest: releaseID, AttestationDigest: attestationID,
		PublisherNamespace: "https://example.test/publishers/acme", ReleaseVersion: "1.0.0",
		VerifiedAt: time.Date(2026, 7, 26, 1, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	return release
}

func catalogTestUpdatedRelease(
	t *testing.T,
	current workflowinstallation.ReleaseRecord,
	version string,
) workflowinstallation.ReleaseRecord {
	t.Helper()
	source, diagnostics := schema.ParseSource(current.SourceArtifact)
	if schema.HasErrors(diagnostics) {
		t.Fatalf("current Source diagnostics = %#v", diagnostics)
	}
	source.Workflow.Name = "Released " + version
	raw, err := artifact.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	releaseID, err := artifact.Sum("yotta/test/workflow-release/v1", raw)
	if err != nil {
		t.Fatal(err)
	}
	attestationID, err := artifact.Sum("yotta/test/attestation/v1", raw)
	if err != nil {
		t.Fatal(err)
	}
	release, err := workflowinstallation.NewVerifiedRelease(raw, workflowinstallation.VerificationReceipt{
		ReleaseDigest: releaseID, AttestationDigest: attestationID,
		PublisherNamespace: current.PublisherNamespace, ReleaseVersion: version,
		VerifiedAt: current.VerifiedAt.Add(time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	return release
}
