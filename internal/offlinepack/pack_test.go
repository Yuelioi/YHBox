package offlinepack

import (
	"archive/zip"
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
	"github.com/yottaapp/yotta/internal/workflow/schema"
)

func TestPackRoundTripsExactPlanAndOriginalArtifactBytes(t *testing.T) {
	ctx := context.Background()
	workflowBytes := []byte("exact signed workflow release bytes")
	packageBytes := []byte("exact signed node package bytes")
	plan := testPlan(t, workflowBytes, packageBytes)
	payloads := map[artifact.Digest][]byte{
		rawSHA256(workflowBytes): workflowBytes,
		rawSHA256(packageBytes):  packageBytes,
	}
	open := func(_ context.Context, descriptor installationplan.ArtifactDescriptor) (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(payloads[descriptor.Digest])), nil
	}
	first := filepath.Join(t.TempDir(), "first"+Extension)
	if err := Write(ctx, first, plan, open); err != nil {
		t.Fatal(err)
	}
	inspected, err := Inspect(ctx, first)
	if err != nil || inspected.Digest() != plan.Digest() {
		t.Fatalf("Inspect() = %s, %v", inspected.Digest(), err)
	}
	second := filepath.Join(t.TempDir(), "second"+Extension)
	if err := Write(ctx, second, plan, open); err != nil {
		t.Fatal(err)
	}
	firstBytes, firstErr := os.ReadFile(first)
	secondBytes, secondErr := os.ReadFile(second)
	if firstErr != nil || secondErr != nil || !bytes.Equal(firstBytes, secondBytes) {
		t.Fatalf("offline pack bytes are not deterministic: %v, %v", firstErr, secondErr)
	}
}

func TestPackRejectsMissingExtraTamperedAndChangedSourceArtifacts(t *testing.T) {
	ctx := context.Background()
	workflowBytes := []byte("workflow release")
	packageBytes := []byte("node package release")
	plan := testPlan(t, workflowBytes, packageBytes)
	payloads := map[artifact.Digest][]byte{
		rawSHA256(workflowBytes): workflowBytes,
		rawSHA256(packageBytes):  packageBytes,
	}
	valid := filepath.Join(t.TempDir(), "valid"+Extension)
	if err := Write(ctx, valid, plan, func(
		_ context.Context,
		descriptor installationplan.ArtifactDescriptor,
	) (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(payloads[descriptor.Digest])), nil
	}); err != nil {
		t.Fatal(err)
	}
	entries := readZip(t, valid)
	packagePath := artifactPath(rawSHA256(packageBytes))
	tests := map[string]func(map[string][]byte){
		"missing":  func(value map[string][]byte) { delete(value, packagePath) },
		"extra":    func(value map[string][]byte) { value["trust-policy.json"] = []byte(`{}`) },
		"tampered": func(value map[string][]byte) { value[packagePath] = []byte("changed") },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			changed := cloneEntries(entries)
			mutate(changed)
			archivePath := filepath.Join(t.TempDir(), name+Extension)
			writeZip(t, archivePath, changed)
			if _, err := Inspect(ctx, archivePath); err == nil {
				t.Fatalf("Inspect accepted %s pack", name)
			}
		})
	}
	if err := Write(ctx, filepath.Join(t.TempDir(), "changed"+Extension), plan, func(
		_ context.Context,
		descriptor installationplan.ArtifactDescriptor,
	) (io.ReadCloser, error) {
		if descriptor.Digest == rawSHA256(packageBytes) {
			return io.NopCloser(strings.NewReader("changed")), nil
		}
		return io.NopCloser(bytes.NewReader(payloads[descriptor.Digest])), nil
	}); err == nil {
		t.Fatal("Write accepted changed artifact bytes")
	}
}

func testPlan(t *testing.T, workflowBytes, packageBytes []byte) installationplan.Plan {
	t.Helper()
	nodeDigest := artifact.Digest("sha256:" + strings.Repeat("1", 64))
	manifestDigest := artifact.Digest("sha256:" + strings.Repeat("2", 64))
	sourceRaw := []byte(fmt.Sprintf(`{
		"format":"yotta.workflow","version":"1",
		"workflow":{"id":"offline-workflow","name":"Offline"},"revision":0,"entryGraph":"main",
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
			WorkflowID:         "offline-workflow", ReleaseVersion: "1.0.0",
			ReleaseDigest: artifact.Digest("sha256:" + strings.Repeat("3", 64)),
			SourceHash:    sourceHash,
			Artifact: installationplan.ArtifactDescriptor{
				Digest: rawSHA256(workflowBytes), Size: int64(len(workflowBytes)),
				MediaType: installationplan.WorkflowArtifactMediaType,
			},
		},
		Packages: []installationplan.NodePackageRelease{{
			PublisherNamespace: "https://example.test/publishers/acme",
			PackageID:          "https://example.test/publishers/acme/packages/example/v1",
			PackageVersion:     "1.0.0", ManifestDigest: manifestDigest,
			Artifact: installationplan.ArtifactDescriptor{
				Digest: rawSHA256(packageBytes), Size: int64(len(packageBytes)),
				MediaType: installationplan.NodePackageArtifactMediaType,
			},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

func rawSHA256(raw []byte) artifact.Digest {
	sum := sha256.Sum256(raw)
	return artifact.Digest("sha256:" + hex.EncodeToString(sum[:]))
}

func readZip(t *testing.T, source string) map[string][]byte {
	t.Helper()
	reader, err := zip.OpenReader(source)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	result := make(map[string][]byte, len(reader.File))
	for _, entry := range reader.File {
		file, err := entry.Open()
		if err != nil {
			t.Fatal(err)
		}
		result[entry.Name], err = io.ReadAll(file)
		closeErr := file.Close()
		if err != nil || closeErr != nil {
			t.Fatalf("read %s: %v, close: %v", entry.Name, err, closeErr)
		}
	}
	return result
}

func writeZip(t *testing.T, destination string, entries map[string][]byte) {
	t.Helper()
	file, err := os.Create(destination)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	for name, raw := range entries {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write(raw); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func cloneEntries(source map[string][]byte) map[string][]byte {
	result := make(map[string][]byte, len(source))
	for name, raw := range source {
		result[name] = append([]byte(nil), raw...)
	}
	return result
}
