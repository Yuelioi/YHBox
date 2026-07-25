package installationplan

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/workflow/schema"
)

func TestPlanIsCanonicalDeterministicAndMatchesExactSourceDependencies(t *testing.T) {
	source := testSource(t)
	draft := testDraft(t)
	draft.Packages = []NodePackageRelease{draft.Packages[1], draft.Packages[0]}
	first, err := SealForSource(source, draft)
	if err != nil {
		t.Fatal(err)
	}
	second, err := SealForSource(source, testDraft(t))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first.Bytes(), second.Bytes()) || first.Digest() != second.Digest() {
		t.Fatal("semantically equal Installation Plans have different identities")
	}
	reopened, err := Open(first.Bytes())
	if err != nil || reopened.Digest() != first.Digest() {
		t.Fatalf("Open() = %#v, %v", reopened.Machine(), err)
	}
	if err := reopened.ValidateSource(source); err != nil {
		t.Fatal(err)
	}
}

func TestPlanRejectsDependencyDriftAndLocalAuthorityFields(t *testing.T) {
	source := testSource(t)
	draft := testDraft(t)
	for name, mutate := range map[string]func(*Draft){
		"missing package": func(value *Draft) { value.Packages = value.Packages[:1] },
		"extra package": func(value *Draft) {
			value.Packages = append(value.Packages, value.Packages[0])
			value.Packages[2].PackageID = "https://example.test/publishers/acme/packages/extra/v1"
		},
		"manifest drift": func(value *Draft) {
			value.Packages[0].ManifestDigest = testDigest("9")
		},
		"publisher drift": func(value *Draft) {
			value.Packages[0].PublisherNamespace = "https://example.test/publishers/other"
		},
	} {
		t.Run(name, func(t *testing.T) {
			changed := testDraft(t)
			mutate(&changed)
			if _, err := SealForSource(source, changed); err == nil {
				t.Fatalf("SealForSource accepted %s", name)
			}
		})
	}
	plan, err := SealForSource(source, draft)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(plan.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	document["trustPolicy"] = map[string]any{"revision": 1}
	raw, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := artifact.Canonicalize(raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Open(canonical); err == nil {
		t.Fatal("Open accepted local trust state")
	}
}

func TestPlanRejectsNonCanonicalOrWrongArtifactIdentity(t *testing.T) {
	source := testSource(t)
	draft := testDraft(t)
	plan, err := SealForSource(source, draft)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Open(append([]byte(" "), plan.Bytes()...)); err == nil {
		t.Fatal("Open accepted non-canonical bytes")
	}
	draft.Workflow.Artifact.MediaType = NodePackageArtifactMediaType
	if _, err := SealForSource(source, draft); err == nil {
		t.Fatal("SealForSource accepted a Node Package media type for the Workflow artifact")
	}
}

func testDraft(t *testing.T) Draft {
	t.Helper()
	return Draft{
		Workflow: WorkflowRelease{
			PublisherNamespace: "https://example.test/publishers/acme",
			WorkflowID:         "workflow-plan",
			ReleaseVersion:     "1.2.3",
			ReleaseDigest:      testDigest("a"),
			SourceHash:         sourceHash(t),
			Artifact: ArtifactDescriptor{
				Digest: testDigest("b"), Size: 2048, MediaType: WorkflowArtifactMediaType,
			},
		},
		Packages: []NodePackageRelease{
			{
				PublisherNamespace: "https://example.test/publishers/acme",
				PackageID:          "https://example.test/publishers/acme/packages/alpha/v1",
				PackageVersion:     "1.0.0",
				ManifestDigest:     testDigest("1"),
				Artifact: ArtifactDescriptor{
					Digest: testDigest("c"), Size: 4096, MediaType: NodePackageArtifactMediaType,
				},
			},
			{
				PublisherNamespace: "https://example.test/publishers/acme",
				PackageID:          "https://example.test/publishers/acme/packages/beta/v1",
				PackageVersion:     "2.0.0",
				ManifestDigest:     testDigest("2"),
				Artifact: ArtifactDescriptor{
					Digest: testDigest("d"), Size: 8192, MediaType: NodePackageArtifactMediaType,
				},
			},
		},
	}
}

func sourceHash(t *testing.T) artifact.Digest {
	t.Helper()
	_, _, digest, diagnostics, err := schema.CanonicalSource(testSource(t))
	if err != nil || schema.HasErrors(diagnostics) {
		t.Fatalf("CanonicalSource() = %s, %#v, %v", digest, diagnostics, err)
	}
	return digest
}

func testSource(t *testing.T) []byte {
	t.Helper()
	raw := fmt.Sprintf(`{
		"format":"yotta.workflow","version":"1",
		"workflow":{"id":"workflow-plan","name":"Installation plan"},"revision":0,"entryGraph":"main",
		"graphs":[{"id":"main","kind":"main","nodes":[
			{"id":"alpha","nodeRef":{"nodeTypeId":"https://example.test/publishers/acme/nodes/alpha","version":"1.0.0","semanticDigest":"sha256:%s"},"position":{"x":0,"y":0},"config":{},"bindings":{}},
			{"id":"beta","nodeRef":{"nodeTypeId":"https://example.test/publishers/acme/nodes/beta","version":"1.0.0","semanticDigest":"sha256:%s"},"position":{"x":100,"y":0},"config":{},"bindings":{}}
		],"edges":[],"inputs":[],"outputs":[]}],
		"resources":[],"targetProfileDefinitions":[],"credentialRequirements":[],
		"dependencies":[
			{"publisherNamespace":"https://example.test/publishers/acme","packageId":"https://example.test/publishers/acme/packages/alpha/v1","packageVersion":"1.0.0","manifestDigest":"sha256:%s","nodeRefs":[{"nodeTypeId":"https://example.test/publishers/acme/nodes/alpha","version":"1.0.0","semanticDigest":"sha256:%s"}]},
			{"publisherNamespace":"https://example.test/publishers/acme","packageId":"https://example.test/publishers/acme/packages/beta/v1","packageVersion":"2.0.0","manifestDigest":"sha256:%s","nodeRefs":[{"nodeTypeId":"https://example.test/publishers/acme/nodes/beta","version":"1.0.0","semanticDigest":"sha256:%s"}]}
		],"variables":[]
	}`, strings.Repeat("3", 64), strings.Repeat("4", 64), strings.Repeat("1", 64), strings.Repeat("3", 64), strings.Repeat("2", 64), strings.Repeat("4", 64))
	canonical, err := artifact.Canonicalize([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	return canonical
}

func testDigest(character string) artifact.Digest {
	return artifact.Digest("sha256:" + strings.Repeat(character, 64))
}
