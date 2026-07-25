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
	if err := repository.Commit(context.Background(), release, first); err != nil {
		t.Fatal(err)
	}
	if err := repository.Commit(context.Background(), release, second); err != nil {
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
		len(configuration.TargetBindings) != 0 || len(configuration.CredentialBindings) != 0 {
		t.Fatalf("GetConfiguration() = %#v, found=%v, err=%v", configuration, found, err)
	}
	configuration.Generation = 2
	configuration.TargetBindings["desktop"] = "target-a"
	configuration.CredentialBindings["api"] = "credential-a"
	configuration.RunConsentRelease = release.ID
	configuration.UpdatedAt = now.Add(time.Minute)
	if err := repository.ReplaceConfiguration(context.Background(), 1, configuration); err != nil {
		t.Fatal(err)
	}
	reloaded, found, err := repository.GetConfiguration(context.Background(), first.ID)
	if err != nil || !found || reloaded.Generation != 2 ||
		reloaded.TargetBindings["desktop"] != "target-a" ||
		reloaded.CredentialBindings["api"] != "credential-a" ||
		reloaded.RunConsentRelease != release.ID {
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
	if err := repository.Commit(context.Background(), release, installation); err != nil {
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
	if err := repository.Commit(context.Background(), collision, other); !errors.Is(err, workflowinstallation.ErrReleaseConflict) {
		t.Fatalf("release collision error = %v", err)
	}
	if _, found, err := repository.GetInstallation(context.Background(), other.ID); err != nil || found {
		t.Fatalf("colliding commit published installation: found=%v err=%v", found, err)
	}
}

func catalogTestVerifiedRelease(t *testing.T) workflowinstallation.ReleaseRecord {
	t.Helper()
	raw := []byte(`{
		"format":"yotta.workflow","version":"1",
		"workflow":{"id":"release-workflow","name":"Released"},
		"revision":0,"entryGraph":"main",
		"graphs":[{"id":"main","kind":"main","nodes":[],"edges":[],"inputs":[],"outputs":[]}],
		"resources":[],"targetProfileDefinitions":[],"credentialRequirements":[],"dependencies":[],"variables":[]
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
		PublisherNamespace: "publisher-1", ReleaseVersion: "1.0.0",
		VerifiedAt: time.Date(2026, 7, 26, 1, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	return release
}
