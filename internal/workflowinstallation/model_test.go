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
}

func TestReadinessReturnsEveryBlockerAndSeparatesRunFromSchedule(t *testing.T) {
	release := testRelease(t)
	installation := InstallationRecord{
		ID: "installation-a", ReleaseID: release.ID, Name: "Installed",
		Lifecycle: LifecycleActive,
		CreatedAt: time.Date(2026, 7, 26, 1, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 7, 26, 1, 0, 0, 0, time.UTC),
	}
	configuration := NewConfiguration(installation)
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
	configuration.CredentialBindings["api"] = "credential-profile-a"
	configuration.RunConsentRelease = release.ID
	readyExceptSchedule := ReadinessEnvironment{
		Dependencies: []DependencyState{sourceDependency},
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
	})
	if err != nil {
		t.Fatal(err)
	}
	installation, err := module.InstallVerified(context.Background(), release, "")
	if err != nil {
		t.Fatal(err)
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

const testSourceJSON = `{
	"format":"yotta.workflow","version":"1",
	"workflow":{"id":"workflow-release","name":"Released workflow"},
	"revision":0,"entryGraph":"main",
	"graphs":[{"id":"main","kind":"main","nodes":[{
		"id":"example","nodeRef":{"nodeTypeId":"https://example.test/nodes/example","version":"1.0.0","semanticDigest":"sha256:1111111111111111111111111111111111111111111111111111111111111111"},
		"position":{"x":0,"y":0},"config":{},"bindings":{}
	}],"edges":[],"inputs":[],"outputs":[]}],
	"resources":[],
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
	releases       map[artifact.Digest]ReleaseRecord
	installations  map[string]InstallationRecord
	configurations map[string]Configuration
}

func (r *memoryRepository) Commit(_ context.Context, release ReleaseRecord, installation InstallationRecord) error {
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
	r.configurations[installation.ID] = NewConfiguration(installation)
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
