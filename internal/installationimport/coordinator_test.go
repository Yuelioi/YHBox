package installationimport

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/installationplan"
	"github.com/yottaapp/yotta/internal/nodepackage"
	"github.com/yottaapp/yotta/internal/offlinepack"
	"github.com/yottaapp/yotta/internal/workflow/schema"
)

func TestOnlineAndOfflineStageTheSameCompletePlan(t *testing.T) {
	ctx := context.Background()
	workflowBytes := []byte("exact workflow artifact")
	packageBytes := []byte("exact signed node package")
	plan := testPlan(t, workflowBytes, packageBytes)
	payloads := map[artifact.Digest][]byte{
		testRawDigest(workflowBytes): workflowBytes,
		testRawDigest(packageBytes):  packageBytes,
	}
	open := func(
		_ context.Context,
		descriptor installationplan.ArtifactDescriptor,
	) (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(payloads[descriptor.Digest])), nil
	}
	online, err := StageOnline(ctx, t.TempDir(), plan, open)
	if err != nil {
		t.Fatal(err)
	}
	defer online.Close()

	packPath := filepath.Join(t.TempDir(), "transfer"+offlinepack.Extension)
	if err := WriteOfflinePack(
		ctx,
		packPath,
		plan,
		func(context.Context, installationplan.ArtifactDescriptor) error { return nil },
		open,
	); err != nil {
		t.Fatal(err)
	}
	offline, err := StageOffline(ctx, t.TempDir(), packPath)
	if err != nil {
		t.Fatal(err)
	}
	defer offline.Close()
	if online.Plan().Digest() != plan.Digest() || offline.Plan().Digest() != plan.Digest() {
		t.Fatal("online and offline staging changed the Installation Plan identity")
	}
	for _, session := range []*Session{online, offline} {
		path, err := session.WorkflowArtifact(ctx)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(raw, workflowBytes) {
			t.Fatalf("staged Workflow artifact = %q, %v", raw, err)
		}
	}
}

func TestStagingRequiresEveryExactArtifactAndCleansIncompleteState(t *testing.T) {
	ctx := context.Background()
	workflowBytes := []byte("workflow")
	packageBytes := []byte("package")
	plan := testPlan(t, workflowBytes, packageBytes)
	parent := t.TempDir()
	_, err := StageOnline(ctx, parent, plan, func(
		_ context.Context,
		descriptor installationplan.ArtifactDescriptor,
	) (io.ReadCloser, error) {
		if descriptor.MediaType == installationplan.NodePackageArtifactMediaType {
			return nil, errorsForTest("package unavailable")
		}
		return io.NopCloser(bytes.NewReader(workflowBytes)), nil
	})
	if err == nil {
		t.Fatal("incomplete online artifact set produced a staged Session")
	}
	entries, readErr := os.ReadDir(parent)
	if readErr != nil || len(entries) != 0 {
		t.Fatalf("incomplete staging residue = %#v, %v", entries, readErr)
	}
}

func TestOfflineExportRequiresEveryArtifactToBeRedistributable(t *testing.T) {
	ctx := context.Background()
	workflowBytes := []byte("workflow")
	packageBytes := []byte("package")
	plan := testPlan(t, workflowBytes, packageBytes)
	payloads := map[artifact.Digest][]byte{
		testRawDigest(workflowBytes): workflowBytes,
		testRawDigest(packageBytes):  packageBytes,
	}
	destination := filepath.Join(t.TempDir(), "forbidden"+offlinepack.Extension)
	err := WriteOfflinePack(
		ctx,
		destination,
		plan,
		func(
			_ context.Context,
			descriptor installationplan.ArtifactDescriptor,
		) error {
			if descriptor.MediaType == installationplan.NodePackageArtifactMediaType {
				return errorsForTest("release is delisted or redistribution is forbidden")
			}
			return nil
		},
		func(
			_ context.Context,
			descriptor installationplan.ArtifactDescriptor,
		) (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(payloads[descriptor.Digest])), nil
		},
	)
	if err == nil {
		t.Fatal("offline export claimed completeness for a forbidden dependency")
	}
	if _, statErr := os.Stat(destination); !os.IsNotExist(statErr) {
		t.Fatalf("ineligible offline pack was published: %v", statErr)
	}
}

func TestTrustInstallAndExecutionConsentRemainSeparateActions(t *testing.T) {
	ctx := context.Background()
	workflowBytes := []byte("workflow")
	packageBytes := []byte("signed package")
	plan := testPlan(t, workflowBytes, packageBytes)
	payloads := map[artifact.Digest][]byte{
		testRawDigest(workflowBytes): workflowBytes,
		testRawDigest(packageBytes):  packageBytes,
	}
	session, err := StageOnline(ctx, t.TempDir(), plan, func(
		_ context.Context,
		descriptor installationplan.ArtifactDescriptor,
	) (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(payloads[descriptor.Digest])), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	release := plan.Machine().Packages[0]
	store := &fakePackageStore{trust: nodepackage.ArchiveTrust{
		PublisherNamespace: release.PublisherNamespace,
		PackageID:          release.PackageID,
		PackageVersion:     release.PackageVersion,
		ManifestDigest:     release.ManifestDigest,
		PublisherKeyID:     artifact.Digest("sha256:" + strings.Repeat("9", 64)),
	}}
	candidates, err := session.InspectPackages(ctx, store)
	if err != nil || len(candidates) != 1 || candidates[0].TrustGranted ||
		store.grantCalls != 0 || store.installCalls != 0 || store.executionConsent {
		t.Fatalf("initial candidates = %#v, err=%v, store=%#v", candidates, err, store)
	}
	if _, err := session.InstallPackage(ctx, store, release.PackageID); err == nil {
		t.Fatal("package installed before its separate publisher trust action")
	}
	if store.installCalls != 0 {
		t.Fatal("failed precondition invoked package installation")
	}
	trusted, err := session.GrantPublisherTrust(ctx, store, release.PackageID)
	if err != nil || !trusted.TrustGranted || store.installCalls != 0 || store.executionConsent {
		t.Fatalf("trust action = %#v, %v, store=%#v", trusted, err, store)
	}
	installed, err := session.InstallPackage(ctx, store, release.PackageID)
	if err != nil || installed.Current != release.ManifestDigest ||
		store.installCalls != 1 || store.executionConsent {
		t.Fatalf("install action = %#v, %v, store=%#v", installed, err, store)
	}
}

func TestSessionRevalidatesStagedBytesAndSignedPlanIdentity(t *testing.T) {
	ctx := context.Background()
	workflowBytes := []byte("workflow")
	packageBytes := []byte("package")
	plan := testPlan(t, workflowBytes, packageBytes)
	payloads := map[artifact.Digest][]byte{
		testRawDigest(workflowBytes): workflowBytes,
		testRawDigest(packageBytes):  packageBytes,
	}
	session, err := StageOnline(ctx, t.TempDir(), plan, func(
		_ context.Context,
		descriptor installationplan.ArtifactDescriptor,
	) (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(payloads[descriptor.Digest])), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	workflowPath, err := session.WorkflowArtifact(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(workflowPath, []byte("tampered"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := session.WorkflowArtifact(ctx); err == nil {
		t.Fatal("Session returned a tampered staged Workflow artifact")
	}

	release := plan.Machine().Packages[0]
	store := &fakePackageStore{trust: nodepackage.ArchiveTrust{
		PublisherNamespace: release.PublisherNamespace,
		PackageID:          release.PackageID,
		PackageVersion:     release.PackageVersion,
		ManifestDigest:     artifact.Digest("sha256:" + strings.Repeat("8", 64)),
		PublisherKeyID:     artifact.Digest("sha256:" + strings.Repeat("9", 64)),
	}}
	if _, err := session.InspectPackages(ctx, store); err == nil {
		t.Fatal("Session accepted a signed package identity outside its Plan")
	}
}

type fakePackageStore struct {
	trust            nodepackage.ArchiveTrust
	installed        nodepackage.PackageInstallation
	grantCalls       int
	installCalls     int
	executionConsent bool
}

func (s *fakePackageStore) InspectArchiveTrust(
	context.Context,
	string,
) (nodepackage.ArchiveTrust, error) {
	trust := s.trust
	trust.Granted = s.grantCalls != 0
	return trust, nil
}

func (s *fakePackageStore) GrantPackageTrust(
	_ context.Context,
	keyID artifact.Digest,
	packageID string,
) error {
	if keyID != s.trust.PublisherKeyID || packageID != s.trust.PackageID {
		return errorsForTest("wrong trust scope")
	}
	s.grantCalls++
	return nil
}

func (s *fakePackageStore) InstallArchive(
	context.Context,
	string,
) (nodepackage.PackageInstallation, error) {
	s.installCalls++
	s.installed = nodepackage.PackageInstallation{
		PackageID: s.trust.PackageID, PublisherNamespace: s.trust.PublisherNamespace,
		Current: s.trust.ManifestDigest, Enabled: true,
		Releases: []nodepackage.PackageRelease{{
			ManifestDigest: s.trust.ManifestDigest, PackageVersion: s.trust.PackageVersion,
		}},
	}
	return s.installed, nil
}

func (s *fakePackageStore) Get(packageID string) (nodepackage.PackageInstallation, bool) {
	return s.installed, s.installed.PackageID == packageID
}

type errorsForTest string

func (e errorsForTest) Error() string { return string(e) }

func testPlan(t *testing.T, workflowBytes, packageBytes []byte) installationplan.Plan {
	t.Helper()
	nodeDigest := artifact.Digest("sha256:" + strings.Repeat("1", 64))
	manifestDigest := artifact.Digest("sha256:" + strings.Repeat("2", 64))
	sourceRaw := []byte(fmt.Sprintf(`{
		"format":"yotta.workflow","version":"1",
		"workflow":{"id":"import-workflow","name":"Import"},"revision":0,"entryGraph":"main",
		"graphs":[{"id":"main","kind":"main","nodes":[{"id":"node","nodeRef":{"nodeTypeId":"https://example.test/publishers/acme/nodes/example","version":"1.0.0","semanticDigest":%q},"position":{"x":0,"y":0},"config":{},"bindings":{}}],"edges":[],"inputs":[],"outputs":[]}],
		"resources":[],"targetProfileDefinitions":[],"credentialRequirements":[],
		"dependencies":[{"publisherNamespace":"https://example.test/publishers/acme","packageId":"https://example.test/publishers/acme/packages/example/v1","packageVersion":"1.0.0","manifestDigest":%q,"nodeRefs":[{"nodeTypeId":"https://example.test/publishers/acme/nodes/example","version":"1.0.0","semanticDigest":%q}]}],
		"variables":[]
	}`, nodeDigest, manifestDigest, nodeDigest))
	source, err := artifact.Canonicalize(sourceRaw)
	if err != nil {
		t.Fatal(err)
	}
	_, _, sourceHash, diagnostics, err := schema.CanonicalSource(source)
	if err != nil || schema.HasErrors(diagnostics) {
		t.Fatalf("CanonicalSource() = %s, %#v, %v", sourceHash, diagnostics, err)
	}
	plan, err := installationplan.SealForSource(source, installationplan.Draft{
		Workflow: installationplan.WorkflowRelease{
			PublisherNamespace: "https://example.test/publishers/acme",
			WorkflowID:         "import-workflow", ReleaseVersion: "1.0.0",
			ReleaseDigest: artifact.Digest("sha256:" + strings.Repeat("3", 64)),
			SourceHash:    sourceHash,
			Artifact: installationplan.ArtifactDescriptor{
				Digest: testRawDigest(workflowBytes), Size: int64(len(workflowBytes)),
				MediaType: installationplan.WorkflowArtifactMediaType,
			},
		},
		Packages: []installationplan.NodePackageRelease{{
			PublisherNamespace: "https://example.test/publishers/acme",
			PackageID:          "https://example.test/publishers/acme/packages/example/v1",
			PackageVersion:     "1.0.0", ManifestDigest: manifestDigest,
			Artifact: installationplan.ArtifactDescriptor{
				Digest: testRawDigest(packageBytes), Size: int64(len(packageBytes)),
				MediaType: installationplan.NodePackageArtifactMediaType,
			},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

func testRawDigest(raw []byte) artifact.Digest {
	sum := sha256.Sum256(raw)
	return artifact.Digest("sha256:" + hex.EncodeToString(sum[:]))
}
