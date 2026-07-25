package workflowinstallation

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/workflow/schema"
)

func TestInstallVerifiedCreatesIndependentInstallationsBeforeReadiness(t *testing.T) {
	release := testRelease(t)
	repository := &memoryRepository{
		releases:      map[artifact.Digest]ReleaseRecord{},
		installations: map[string]InstallationRecord{},
	}
	ids := []string{"installation-a", "installation-b"}
	module, err := New(repository, Options{
		Now: func() time.Time { return time.Date(2026, 7, 26, 1, 0, 0, 0, time.UTC) },
		NewID: func() string {
			id := ids[0]
			ids = ids[1:]
			return id
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	first, err := module.InstallVerified(context.Background(), release, "")
	if err != nil {
		t.Fatal(err)
	}
	second, err := module.InstallVerified(context.Background(), release, "Second copy")
	if err != nil {
		t.Fatal(err)
	}
	if first.ID == second.ID || first.ReleaseID != release.ID || second.ReleaseID != release.ID ||
		first.Name != release.WorkflowName || second.Name != "Second copy" ||
		len(repository.releases) != 1 || len(repository.installations) != 2 {
		t.Fatalf("installations = %#v / %#v, repository = %#v", first, second, repository)
	}
	firstConfiguration, err := module.GetConfiguration(context.Background(), first.ID)
	if err != nil || len(firstConfiguration.TargetProfiles) != 1 ||
		firstConfiguration.TargetProfiles["desktop"].ReleaseID != release.ID ||
		string(firstConfiguration.TargetProfiles["desktop"].Settings) != `{}` {
		t.Fatalf("materialized configuration = %#v, %v", firstConfiguration, err)
	}
	secondConfiguration, err := module.GetConfiguration(context.Background(), second.ID)
	if err != nil {
		t.Fatal(err)
	}
	firstProfile := firstConfiguration.TargetProfiles["desktop"]
	firstProfile.Settings[0] = 'x'
	if secondConfiguration.TargetProfiles["desktop"].Settings[0] == 'x' {
		t.Fatal("Workflow Installations share mutable Target Profile settings")
	}
}

func TestUpdateTargetProfileValidatesExactReleaseSchemaAndGeneration(t *testing.T) {
	release := testRelease(t)
	repository := &memoryRepository{
		releases: map[artifact.Digest]ReleaseRecord{}, installations: map[string]InstallationRecord{},
	}
	module, err := New(repository, Options{
		Now:   func() time.Time { return time.Date(2026, 7, 26, 2, 0, 0, 0, time.UTC) },
		NewID: func() string { return "installation-profile" },
	})
	if err != nil {
		t.Fatal(err)
	}
	installation, err := module.InstallVerified(context.Background(), release, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := module.UpdateTargetProfile(
		context.Background(), installation.ID, 1, "desktop",
		[]byte(`{"unknown":true}`), "target-a",
	); err == nil {
		t.Fatal("UpdateTargetProfile accepted settings outside the exact Release schema")
	}
	updated, err := module.UpdateTargetProfile(
		context.Background(), installation.ID, 1, "desktop", []byte(`{ }`), "target-a",
	)
	if err != nil || updated.Generation != 2 ||
		updated.TargetProfiles["desktop"].TargetInstallationID != "target-a" ||
		updated.TargetBindings["desktop"] != "target-a" ||
		string(updated.TargetProfiles["desktop"].Settings) != `{}` {
		t.Fatalf("UpdateTargetProfile() = %#v, %v", updated, err)
	}
	if _, err := module.UpdateTargetProfile(
		context.Background(), installation.ID, 1, "desktop", []byte(`{}`), "target-b",
	); !errors.Is(err, ErrInstallationConflict) {
		t.Fatalf("stale UpdateTargetProfile error = %v", err)
	}
}

func TestMaterializeTargetProfilesRejectsPersistedSchemaDriftAndUnknownDefinitions(t *testing.T) {
	release := testRelease(t)
	installation := InstallationRecord{
		ID: "installation-profile-validation", ReleaseID: release.ID, Name: "Installed",
		Lifecycle: LifecycleActive,
		CreatedAt: time.Date(2026, 7, 26, 2, 15, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 7, 26, 2, 15, 0, 0, time.UTC),
	}
	configuration, err := NewConfigurationForRelease(installation, release)
	if err != nil {
		t.Fatal(err)
	}
	drifted := CloneConfiguration(configuration)
	profile := drifted.TargetProfiles["desktop"]
	profile.Settings = []byte(`{"unknown":true}`)
	drifted.TargetProfiles["desktop"] = profile
	if _, err := MaterializeTargetProfiles(drifted, release); err == nil {
		t.Fatal("MaterializeTargetProfiles accepted settings outside the Release schema")
	}
	unknown := CloneConfiguration(configuration)
	profile.DefinitionID = "unknown"
	profile.Settings = []byte(`{}`)
	unknown.TargetProfiles["unknown"] = profile
	if _, err := MaterializeTargetProfiles(unknown, release); err == nil {
		t.Fatal("MaterializeTargetProfiles accepted an unknown Release definition")
	}
}

func TestUpdateCredentialBindingRequiresCompatibleAvailableSecureProfile(t *testing.T) {
	release := testRelease(t)
	repository := &memoryRepository{
		releases: map[artifact.Digest]ReleaseRecord{}, installations: map[string]InstallationRecord{},
	}
	module, err := New(repository, Options{
		Now:   func() time.Time { return time.Date(2026, 7, 26, 2, 20, 0, 0, time.UTC) },
		NewID: func() string { return "installation-credential" },
		Credentials: func(context.Context) ([]CredentialState, error) {
			return []CredentialState{
				testCredentialState("credential-api"),
				{
					CredentialBindingID: "credential-other",
					Kind:                "https://example.test/credentials/other/v1",
					Label:               "Other", Available: true,
				},
				{
					CredentialBindingID: "credential-missing",
					Kind:                "https://example.test/credentials/api/v1",
					Label:               "Missing", Available: false,
				},
			}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	installation, err := module.InstallVerified(context.Background(), release, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, bindingID := range []string{"credential-other", "credential-missing", "credential-unknown"} {
		if _, err := module.UpdateCredentialBinding(
			context.Background(), installation.ID, 1, "api", bindingID,
		); err == nil {
			t.Fatalf("UpdateCredentialBinding accepted %q", bindingID)
		}
	}
	updated, err := module.UpdateCredentialBinding(
		context.Background(), installation.ID, 1, "api", "credential-api",
	)
	if err != nil || updated.Generation != 2 ||
		updated.CredentialBindings["api"] != "credential-api" {
		t.Fatalf("UpdateCredentialBinding() = %#v, %v", updated, err)
	}
	if _, err := module.UpdateCredentialBinding(
		context.Background(), installation.ID, 1, "api", "",
	); !errors.Is(err, ErrInstallationConflict) {
		t.Fatalf("stale UpdateCredentialBinding error = %v", err)
	}
}

func TestGetConfigurationMaterializesProfilesForPreProfileConfiguration(t *testing.T) {
	release := testRelease(t)
	repository := &memoryRepository{
		releases: map[artifact.Digest]ReleaseRecord{}, installations: map[string]InstallationRecord{},
	}
	module, err := New(repository, Options{
		Now:   func() time.Time { return time.Date(2026, 7, 26, 2, 30, 0, 0, time.UTC) },
		NewID: func() string { return "installation-migrated" },
	})
	if err != nil {
		t.Fatal(err)
	}
	installation, err := module.InstallVerified(context.Background(), release, "")
	if err != nil {
		t.Fatal(err)
	}
	legacy := repository.configurations[installation.ID]
	legacy.TargetProfiles = map[string]TargetProfile{}
	legacy.TargetBindings["desktop"] = "target-existing"
	repository.configurations[installation.ID] = legacy

	materialized, err := module.GetConfiguration(context.Background(), installation.ID)
	if err != nil || materialized.Generation != 2 ||
		materialized.TargetProfiles["desktop"].TargetInstallationID != "target-existing" {
		t.Fatalf("GetConfiguration() = %#v, %v", materialized, err)
	}
}

func TestReadinessReturnsEveryBlockerAndSeparatesRunFromSchedule(t *testing.T) {
	release := testRelease(t)
	installation := InstallationRecord{
		ID: "installation-a", ReleaseID: release.ID, Name: "Installed",
		Lifecycle: LifecycleActive,
		CreatedAt: time.Date(2026, 7, 26, 1, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 7, 26, 1, 0, 0, 0, time.UTC),
	}
	configuration, err := NewConfigurationForRelease(installation, release)
	if err != nil {
		t.Fatal(err)
	}
	report, err := EvaluateReadiness(installation, release, configuration, ReadinessEnvironment{})
	if err != nil {
		t.Fatal(err)
	}
	wantKinds := []BlockerKind{
		BlockerCredential, BlockerDependency, BlockerRunConsent, BlockerScheduleConsent, BlockerTarget,
	}
	gotKinds := make([]BlockerKind, 0, len(report.Blockers))
	for _, blocker := range report.Blockers {
		gotKinds = append(gotKinds, blocker.Kind)
	}
	if len(report.Blockers) != 5 || report.RunAllowed || report.ScheduleAllowed {
		t.Fatalf("blocked report = %#v, kinds=%v want=%v", report, gotKinds, wantKinds)
	}

	sourceDependency := testDependency(t)
	configuration.TargetBindings["desktop"] = "target-installation-a"
	profile := configuration.TargetProfiles["desktop"]
	profile.TargetInstallationID = "target-installation-a"
	configuration.TargetProfiles["desktop"] = profile
	configuration.CredentialBindings["api"] = "credential-profile-a"
	configuration.RunConsentRelease = release.ID
	readyExceptSchedule := ReadinessEnvironment{
		Dependencies: []DependencyState{sourceDependency},
		Targets:      []TargetState{testTargetState("target-installation-a")},
		Credentials:  []CredentialState{testCredentialState("credential-profile-a")},
	}
	report, err = EvaluateReadiness(installation, release, configuration, readyExceptSchedule)
	if err != nil {
		t.Fatal(err)
	}
	if !report.RunAllowed || report.ScheduleAllowed || len(report.Blockers) != 1 ||
		report.Blockers[0].Kind != BlockerScheduleConsent {
		t.Fatalf("schedule-only blocker report = %#v", report)
	}
	configuration.ScheduleConsentRelease = release.ID
	report, err = EvaluateReadiness(installation, release, configuration, readyExceptSchedule)
	if err != nil || !report.RunAllowed || !report.ScheduleAllowed || len(report.Blockers) != 0 {
		t.Fatalf("ready report = %#v, %v", report, err)
	}
	readyExceptSchedule.Targets[0].Authorized = false
	report, err = EvaluateReadiness(installation, release, configuration, readyExceptSchedule)
	if err != nil || report.RunAllowed || report.ScheduleAllowed || len(report.Blockers) != 1 ||
		report.Blockers[0].Kind != BlockerTarget {
		t.Fatalf("unauthorized target report = %#v, %v", report, err)
	}
	readyExceptSchedule.Targets[0].Authorized = true
	configuration.RunConsentRelease = ""
	report, err = EvaluateReadiness(installation, release, configuration, readyExceptSchedule)
	if err != nil || report.RunAllowed || !report.ScheduleAllowed || len(report.Blockers) != 1 ||
		report.Blockers[0].Kind != BlockerRunConsent {
		t.Fatalf("run-only blocker report = %#v, %v", report, err)
	}
}

func TestModulePersistsBindingsAndExactReleaseConsentsWithGenerationCAS(t *testing.T) {
	release := testRelease(t)
	repository := &memoryRepository{
		releases:      map[artifact.Digest]ReleaseRecord{},
		installations: map[string]InstallationRecord{},
	}
	now := time.Date(2026, 7, 26, 3, 0, 0, 0, time.UTC)
	module, err := New(repository, Options{
		Now: func() time.Time {
			now = now.Add(time.Second)
			return now
		},
		NewID: func() string { return "installation-config" },
		Dependencies: func(context.Context) ([]DependencyState, error) {
			return []DependencyState{testDependency(t)}, nil
		},
		Targets: func(context.Context) ([]TargetState, error) {
			return []TargetState{testTargetState("target-a")}, nil
		},
		Credentials: func(context.Context) ([]CredentialState, error) {
			return []CredentialState{testCredentialState("credential-a")}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	installation, err := module.InstallVerified(context.Background(), release, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := module.PrepareExecution(context.Background(), installation.ID, ScopeRun); err == nil {
		t.Fatal("PrepareExecution accepted an unready Installation")
	} else {
		var notReady *NotReadyError
		if !errors.As(err, &notReady) || notReady.RPCErrorEnvelope().Code != "workflow_installation.not_ready" {
			t.Fatalf("PrepareExecution error = %T %v", err, err)
		}
	}
	configuration, err := module.ReplaceBindings(context.Background(), installation.ID, 1, BindingUpdate{
		TargetBindings:     map[string]string{"desktop": "target-a"},
		CredentialBindings: map[string]string{"api": "credential-a"},
	})
	if err != nil || configuration.Generation != 2 {
		t.Fatalf("ReplaceBindings() = %#v, %v", configuration, err)
	}
	if _, err := module.ReplaceBindings(context.Background(), installation.ID, 1, BindingUpdate{}); !errors.Is(err, ErrInstallationConflict) {
		t.Fatalf("stale ReplaceBindings() error = %v", err)
	}
	configuration, err = module.GrantConsent(context.Background(), installation.ID, 2, ScopeRun)
	if err != nil || configuration.RunConsentRelease != release.ID || configuration.Generation != 3 {
		t.Fatalf("GrantConsent(run) = %#v, %v", configuration, err)
	}
	configuration, err = module.GrantConsent(context.Background(), installation.ID, 3, ScopeSchedule)
	if err != nil || configuration.ScheduleConsentRelease != release.ID || configuration.Generation != 4 {
		t.Fatalf("GrantConsent(schedule) = %#v, %v", configuration, err)
	}
	report, err := module.Readiness(context.Background(), installation.ID)
	if err != nil || !report.RunAllowed || !report.ScheduleAllowed {
		t.Fatalf("Readiness() = %#v, %v", report, err)
	}
	readsBeforePrepare := repository.configurationReads
	prepared, err := module.PrepareExecution(context.Background(), installation.ID, ScopeSchedule)
	if err != nil || !prepared.Valid() || prepared.InstallationID() != installation.ID ||
		prepared.ReleaseID() != release.ID || len(prepared.SourceArtifact()) == 0 ||
		prepared.TargetSelection()["desktop"] != "target-a" ||
		prepared.CredentialSelection()["api"] != "credential-a" {
		t.Fatalf("PrepareExecution() = %#v, %v", prepared, err)
	}
	if repository.configurationReads != readsBeforePrepare+1 {
		t.Fatalf(
			"PrepareExecution configuration reads = %d, want one readiness snapshot",
			repository.configurationReads-readsBeforePrepare,
		)
	}
	artifactCopy := prepared.SourceArtifact()
	artifactCopy[0] = 'x'
	if prepared.SourceArtifact()[0] == 'x' {
		t.Fatal("PreparedExecution leaked mutable Source bytes")
	}
}

func TestNewVerifiedReleaseRejectsNonCanonicalAndInconsistentReceipts(t *testing.T) {
	raw := []byte(testSourceJSON)
	canonical, err := artifact.Canonicalize(raw)
	if err != nil {
		t.Fatal(err)
	}
	receipt := testReceipt(t)
	if _, err := NewVerifiedRelease(append([]byte(" "), canonical...), receipt); err == nil {
		t.Fatal("accepted non-canonical source bytes")
	}
	receipt.ReleaseDigest = ""
	if _, err := NewVerifiedRelease(canonical, receipt); err == nil {
		t.Fatal("accepted receipt without exact release identity")
	}
}

func testRelease(t *testing.T) ReleaseRecord {
	t.Helper()
	canonical, err := artifact.Canonicalize([]byte(testSourceJSON))
	if err != nil {
		t.Fatal(err)
	}
	release, err := NewVerifiedRelease(canonical, testReceipt(t))
	if err != nil {
		_, diagnostics := schema.ParseSource(canonical)
		t.Fatalf("%v; diagnostics=%#v", err, diagnostics)
	}
	return release
}

func testReceipt(t *testing.T) VerificationReceipt {
	t.Helper()
	releaseDigest, err := artifact.Sum("yotta/test/workflow-release/v1", []byte("release"))
	if err != nil {
		t.Fatal(err)
	}
	attestationDigest, err := artifact.Sum("yotta/test/publisher-attestation/v1", []byte("attestation"))
	if err != nil {
		t.Fatal(err)
	}
	return VerificationReceipt{
		ReleaseDigest: releaseDigest, AttestationDigest: attestationDigest,
		PublisherNamespace: "publisher-1", ReleaseVersion: "1.2.3",
		VerifiedAt: time.Date(2026, 7, 26, 0, 0, 0, 0, time.UTC),
	}
}

func testDependency(t *testing.T) DependencyState {
	t.Helper()
	return DependencyState{
		PublisherNamespace: "https://example.test/publishers/acme",
		PackageID:          "https://example.test/publishers/acme/packages/example/v1",
		PackageVersion:     "2.0.0", ManifestDigest: artifact.Digest("sha256:2222222222222222222222222222222222222222222222222222222222222222"), Enabled: true,
	}
}

func testTargetState(targetInstallationID string) TargetState {
	return TargetState{
		TargetInstallationID: targetInstallationID,
		TargetKind:           "desktop", AdapterKind: "windows", ProfileVersion: "1",
		Available: true, Authorized: true,
	}
}

func testCredentialState(bindingID string) CredentialState {
	return CredentialState{
		CredentialBindingID: bindingID,
		Kind:                "https://example.test/credentials/api/v1",
		Label:               "API credential", Available: true,
	}
}

const testSourceJSON = `{
	"format":"yotta.workflow","version":"1",
	"workflow":{"id":"workflow-release","name":"Released workflow"},
	"revision":0,"entryGraph":"main",
	"graphs":[{"id":"main","kind":"main","nodes":[{
		"id":"example","nodeRef":{"nodeTypeId":"https://example.test/nodes/example","version":"1.0.0","semanticDigest":"sha256:1111111111111111111111111111111111111111111111111111111111111111"},
		"position":{"x":0,"y":0},"config":{},"bindings":{}
	}],"edges":[],"inputs":[],"outputs":[]}],
	"resources":[],
	"targetDefaults":[{"target":"desktop-target","slot":"desktop"}],
	"targetProfileDefinitions":[{
		"id":"desktop","name":"Desktop","targetKind":"desktop","adapterKind":"windows",
		"profileVersion":"1","settingsSchemaRoot":"https://example.test/desktop/v1",
		"settingsSchemaBundle":[{"id":"https://example.test/desktop/v1","schema":{"$schema":"https://json-schema.org/draft/2020-12/schema","$id":"https://example.test/desktop/v1","type":"object","additionalProperties":false}}],
		"initialDefaults":{},"discoveryHints":[]
	}],
	"credentialRequirements":[{"slot":"api","kind":"https://example.test/credentials/api/v1","purpose":"Call API"}],
	"dependencies":[{
		"publisherNamespace":"https://example.test/publishers/acme","packageId":"https://example.test/publishers/acme/packages/example/v1","packageVersion":"2.0.0",
		"manifestDigest":"sha256:2222222222222222222222222222222222222222222222222222222222222222",
		"nodeRefs":[{"nodeTypeId":"https://example.test/nodes/example","version":"1.0.0","semanticDigest":"sha256:1111111111111111111111111111111111111111111111111111111111111111"}]
	}],
	"variables":[]
}`

type memoryRepository struct {
	releases           map[artifact.Digest]ReleaseRecord
	installations      map[string]InstallationRecord
	configurations     map[string]Configuration
	configurationReads int
}

func (r *memoryRepository) Commit(
	_ context.Context,
	release ReleaseRecord,
	installation InstallationRecord,
	configuration Configuration,
) error {
	if current, found := r.releases[release.ID]; found {
		if current.SourceHash != release.SourceHash || current.AttestationDigest != release.AttestationDigest {
			return ErrReleaseConflict
		}
	}
	if _, found := r.installations[installation.ID]; found {
		return ErrInstallationConflict
	}
	r.releases[release.ID] = CloneReleaseRecord(release)
	r.installations[installation.ID] = installation
	if r.configurations == nil {
		r.configurations = map[string]Configuration{}
	}
	r.configurations[installation.ID] = CloneConfiguration(configuration)
	return nil
}

func (r *memoryRepository) GetInstallation(_ context.Context, id string) (InstallationRecord, bool, error) {
	value, found := r.installations[id]
	return value, found, nil
}

func (r *memoryRepository) ListInstallations(context.Context) ([]InstallationRecord, error) {
	result := make([]InstallationRecord, 0, len(r.installations))
	for _, value := range r.installations {
		result = append(result, value)
	}
	return result, nil
}

func (r *memoryRepository) GetRelease(_ context.Context, id artifact.Digest) (ReleaseRecord, bool, error) {
	value, found := r.releases[id]
	return CloneReleaseRecord(value), found, nil
}

func (r *memoryRepository) GetConfiguration(_ context.Context, id string) (Configuration, bool, error) {
	r.configurationReads++
	value, found := r.configurations[id]
	return CloneConfiguration(value), found, nil
}

func (r *memoryRepository) ReplaceConfiguration(_ context.Context, expected int64, next Configuration) error {
	current, found := r.configurations[next.InstallationID]
	if !found || current.Generation != expected || next.Generation != expected+1 {
		return ErrInstallationConflict
	}
	r.configurations[next.InstallationID] = CloneConfiguration(next)
	return nil
}

var _ Repository = (*memoryRepository)(nil)
