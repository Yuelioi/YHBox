package workflowbundle

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/workflow/schema"
	"github.com/yottaapp/yotta/internal/workflowstore"
)

type testSourceRepository struct{ store *workflowstore.SourceStore }

func (r testSourceRepository) GetSource(workflowID string) (workflowstore.SourceSnapshot, error) {
	return r.store.Load(workflowID)
}

func (r testSourceRepository) PublishImportedSource(ctx context.Context, raw []byte, baseRevision int64, expectedHash artifact.Digest) (workflowstore.SourceSnapshot, error) {
	if baseRevision >= 0 {
		var identity struct {
			Workflow struct {
				ID string `json:"id"`
			} `json:"workflow"`
		}
		if err := json.Unmarshal(raw, &identity); err != nil {
			return workflowstore.SourceSnapshot{}, err
		}
		current, err := r.store.Load(identity.Workflow.ID)
		if err != nil {
			return workflowstore.SourceSnapshot{}, err
		}
		if current.Revision() != baseRevision || current.Hash() != expectedHash {
			return workflowstore.SourceSnapshot{}, workflowstore.ErrSourceConflict
		}
	}
	return r.store.Save(ctx, raw, baseRevision)
}

func TestManagerRoundTripsCanonicalSourceAndReferencedBlobs(t *testing.T) {
	ctx := context.Background()
	sourceStore := openTestSources(t)
	sourceBlobs := openTestBlobs(t)
	ref, err := sourceBlobs.Put(ctx, "application/vnd.yotta.macro+json", strings.NewReader("portable payload"))
	if err != nil {
		t.Fatal(err)
	}
	source := testResourceSource("source_workflow", "Portable source", ref)
	original, err := sourceStore.Save(ctx, source, -1)
	if err != nil {
		t.Fatal(err)
	}
	sourceManager, err := New(testSourceRepository{sourceStore}, sourceBlobs)
	if err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(t.TempDir(), "portable"+Extension)
	exported, err := sourceManager.Export(ctx, original.WorkflowID(), destination)
	if err != nil {
		t.Fatal(err)
	}
	if exported.Info.SourceHash != original.Hash() || exported.Info.ResourceCount != 1 || exported.Info.DependencyCount != 1 ||
		exported.Info.BlobCount != 1 || exported.Info.BlobBytes != ref.Size {
		t.Fatalf("export info = %#v", exported.Info)
	}
	inspected, err := sourceManager.Inspect(ctx, destination)
	if err != nil || inspected.WorkflowID != original.WorkflowID() {
		t.Fatalf("Inspect() = %#v, %v", inspected, err)
	}
	targetStore := openTestSources(t)
	targetBlobs := openTestBlobs(t)
	targetManager, err := New(testSourceRepository{targetStore}, targetBlobs)
	if err != nil {
		t.Fatal(err)
	}
	targetManager.newID = func() string { return "imported_workflow" }
	imported, err := targetManager.Import(ctx, ImportRequest{Path: destination, Mode: ImportCopy})
	if err != nil {
		t.Fatal(err)
	}
	if imported.Source.WorkflowID() != "imported_workflow" || imported.Source.Revision() != 0 || imported.Source.Hash() == original.Hash() {
		t.Fatalf("imported source = id %q revision %d hash %q", imported.Source.WorkflowID(), imported.Source.Revision(), imported.Source.Hash())
	}
	if err := targetBlobs.Verify(ctx, ref); err != nil {
		t.Fatalf("imported blob failed verification: %v", err)
	}
}

func TestManagerReplaceRequiresExactTargetIdentity(t *testing.T) {
	ctx := context.Background()
	sources := openTestSources(t)
	blobs := openTestBlobs(t)
	manager, err := New(testSourceRepository{sources}, blobs)
	if err != nil {
		t.Fatal(err)
	}
	original, err := sources.Save(ctx, testSource("source_workflow", "Replacement", blob.BlobRef{}), -1)
	if err != nil {
		t.Fatal(err)
	}
	archivePath := filepath.Join(t.TempDir(), "replacement"+Extension)
	if _, err := manager.Export(ctx, original.WorkflowID(), archivePath); err != nil {
		t.Fatal(err)
	}
	target, err := sources.Save(ctx, testSource("target_workflow", "Old name", blob.BlobRef{}), -1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Import(ctx, ImportRequest{
		Path: archivePath, Mode: ImportReplace, TargetWorkflowID: target.WorkflowID(),
		ExpectedRevision: target.Revision(), ExpectedSourceHash: artifact.Digest("sha256:" + strings.Repeat("f", 64)),
	}); !errors.Is(err, workflowstore.ErrSourceConflict) {
		t.Fatalf("wrong expected hash error = %v", err)
	}
	result, err := manager.Import(ctx, ImportRequest{
		Path: archivePath, Mode: ImportReplace, TargetWorkflowID: target.WorkflowID(),
		ExpectedRevision: target.Revision(), ExpectedSourceHash: target.Hash(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Source.WorkflowID() != target.WorkflowID() || result.Source.Revision() != target.Revision()+1 {
		t.Fatalf("replacement identity = %q revision %d", result.Source.WorkflowID(), result.Source.Revision())
	}
}

func TestManagerRejectsUndeclaredAndCorruptArchiveEntries(t *testing.T) {
	ctx := context.Background()
	sources := openTestSources(t)
	blobs := openTestBlobs(t)
	manager, err := New(testSourceRepository{sources}, blobs)
	if err != nil {
		t.Fatal(err)
	}
	ref, err := blobs.Put(ctx, "application/octet-stream", strings.NewReader("expected"))
	if err != nil {
		t.Fatal(err)
	}
	raw := testSource("source_workflow", "Portable source", ref)
	snapshot, err := sources.Save(ctx, raw, -1)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := json.Marshal(Manifest{Format: Format, Version: Version, WorkflowID: snapshot.WorkflowID(), SourceHash: snapshot.Hash(), Dependencies: []schema.NodePackageDependency{}, Blobs: []blob.BlobRef{ref}})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		extra   string
		payload string
	}{
		{name: "undeclared", extra: "unexpected.json", payload: "expected"},
		{name: "traversal", extra: "../outside", payload: "expected"},
		{name: "corrupt blob", payload: "tampered"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			archivePath := filepath.Join(t.TempDir(), "invalid.zip")
			entries := map[string][]byte{
				ManifestPath: manifest, SourcePath: snapshot.Artifact(), blobEntryPath(ref.Digest): []byte(test.payload),
			}
			if test.extra != "" {
				entries[test.extra] = []byte("extra")
			}
			writeTestZip(t, archivePath, entries)
			if _, err := manager.Inspect(ctx, archivePath); err == nil {
				t.Fatal("Inspect() accepted an invalid archive")
			}
		})
	}
}

func openTestSources(t *testing.T) *workflowstore.SourceStore {
	t.Helper()
	store, err := workflowstore.OpenSourceStore(filepath.Join(t.TempDir(), "sources"), workflowstore.SourceStoreOptions{MaxSources: 16})
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func openTestBlobs(t *testing.T) *blob.Store {
	t.Helper()
	store, err := blob.Open(filepath.Join(t.TempDir(), "blobs"), blob.Limits{MaxBlobBytes: 1 << 20, MaxTotalBytes: 8 << 20})
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func testSource(workflowID, name string, ref blob.BlobRef) []byte {
	binding := ""
	if ref.Digest.Valid() {
		encoded, _ := json.Marshal(ref)
		binding = `"asset":{"kind":"blob","blob":` + string(encoded) + `}`
	}
	return []byte(fmt.Sprintf(`{"format":"yotta.workflow","version":"3.1","workflow":{"id":%q,"name":%q},"revision":0,"entryGraph":"main","graphs":[{"id":"main","kind":"main","nodes":[{"id":"node_1","nodeRef":{"nodeTypeId":"https://schemas.yotta.dev/nodes/test","version":"1.0.0","semanticDigest":"sha256:%s"},"position":{"x":0,"y":0},"config":{},"bindings":{%s}}],"edges":[],"inputs":[],"outputs":[]}],"variables":[],"resources":[],"targetProfileDefinitions":[],"credentialRequirements":[],"dependencies":[]}`, workflowID, name, strings.Repeat("1", 64), binding))
}

func testResourceSource(workflowID, name string, ref blob.BlobRef) []byte {
	raw := string(testSource(workflowID, name, ref))
	encoded, _ := json.Marshal(ref)
	raw = strings.Replace(raw, `"asset":{"kind":"blob","blob":`+string(encoded)+`}`, `"asset":{"kind":"resource","resource":{"resourceId":"recording"}}`, 1)
	raw = strings.Replace(raw, `"resources":[]`, `"resources":[{"id":"recording","kind":"macro","name":"Recording","macro":{"blob":`+string(encoded)+`,"baseResolution":[1920,1080],"actionCount":1,"durationUs":1000}}]`, 1)
	raw = strings.Replace(raw, `"dependencies":[]`, `"dependencies":[{"publisherNamespace":"https://schemas.yotta.dev/packages","packageId":"https://schemas.yotta.dev/packages/test/v1","packageVersion":"1.0.0","manifestDigest":"sha256:`+strings.Repeat("2", 64)+`","nodeRefs":[{"nodeTypeId":"https://schemas.yotta.dev/nodes/test","version":"1.0.0","semanticDigest":"sha256:`+strings.Repeat("1", 64)+`"}]}]`, 1)
	return []byte(raw)
}

func writeTestZip(t *testing.T, destination string, entries map[string][]byte) {
	t.Helper()
	file, err := os.Create(destination)
	if err != nil {
		t.Fatal(err)
	}
	archive := zip.NewWriter(file)
	for name, payload := range entries {
		writer, err := archive.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := bytes.NewReader(payload).WriteTo(writer); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}
