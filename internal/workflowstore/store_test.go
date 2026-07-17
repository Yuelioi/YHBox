package workflowstore_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/nodes"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
	"github.com/yottaapp/yotta/internal/workflowstore"
)

func TestSourceStoreRequiresExplicitRevisionCASAndReopensCanonicalArtifact(t *testing.T) {
	root := t.TempDir()
	store, err := workflowstore.OpenSourceStore(root, workflowstore.SourceStoreOptions{MaxSources: 2})
	if err != nil {
		t.Fatal(err)
	}
	created, err := store.Save(context.Background(), concatSource(t, 0, "a", "b"), -1)
	if err != nil {
		t.Fatal(err)
	}
	if created.WorkflowID() != "wf-store" || created.Revision() != 0 || !created.Hash().Valid() || strings.Contains(string(created.Artifact()), "\n") {
		t.Fatalf("created Source = %#v, %s", created, created.Artifact())
	}
	if _, err := store.Save(context.Background(), concatSource(t, 1, "x", "y"), -1); !errors.Is(err, workflowstore.ErrSourceConflict) {
		t.Fatalf("stale create error = %v", err)
	}
	updated, err := store.Save(context.Background(), concatSource(t, 1, "x", "y"), 0)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Revision() != 1 || updated.Hash() == created.Hash() {
		t.Fatalf("updated Source = %#v", updated)
	}
	loaded, err := store.Load("wf-store")
	if err != nil || loaded.Hash() != updated.Hash() {
		t.Fatalf("Load = %#v, %v", loaded, err)
	}
	reopened, err := workflowstore.OpenSourceStore(root, workflowstore.SourceStoreOptions{MaxSources: 2})
	if err != nil {
		t.Fatal(err)
	}
	loaded, err = reopened.Load("wf-store")
	if err != nil || loaded.Revision() != 1 {
		t.Fatalf("reopened Load = %#v, %v", loaded, err)
	}
}

func TestSourceStoreRejectsInvalidAndExternallyChangedSources(t *testing.T) {
	root := t.TempDir()
	store, err := workflowstore.OpenSourceStore(root, workflowstore.SourceStoreOptions{MaxSources: 2})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Save(context.Background(), []byte(`{"format":"yotta.workflow","version":"3"}`), -1); err == nil {
		t.Fatal("legacy Source was accepted")
	}
	if _, err := store.Save(context.Background(), concatSource(t, 0, "a", "b"), -1); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "wf-store.json"), concatSource(t, 1, "x", "y"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load("wf-store"); !errors.Is(err, workflowstore.ErrSourceChanged) {
		t.Fatalf("external change error = %v", err)
	}
}

func TestSourceStoreDeleteRequiresExactRevisionAndHash(t *testing.T) {
	root := t.TempDir()
	store, err := workflowstore.OpenSourceStore(root, workflowstore.SourceStoreOptions{MaxSources: 2})
	if err != nil {
		t.Fatal(err)
	}
	created, err := store.Save(context.Background(), concatSource(t, 0, "a", "b"), -1)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(context.Background(), created.WorkflowID(), created.Revision()+1, created.Hash()); !errors.Is(err, workflowstore.ErrSourceConflict) {
		t.Fatalf("stale revision delete = %v", err)
	}
	wrongHash, err := artifact.Sum("yotta/test/wrong-source/v1", []byte("wrong"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(context.Background(), created.WorkflowID(), created.Revision(), wrongHash); !errors.Is(err, workflowstore.ErrSourceConflict) {
		t.Fatalf("stale hash delete = %v", err)
	}
	if err := store.Delete(context.Background(), created.WorkflowID(), created.Revision(), created.Hash()); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(created.WorkflowID()); !errors.Is(err, workflowstore.ErrSourceNotFound) {
		t.Fatalf("Load after delete = %v", err)
	}
	reopened, err := workflowstore.OpenSourceStore(root, workflowstore.SourceStoreOptions{MaxSources: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(reopened.List()) != 0 {
		t.Fatalf("reopened sources = %#v", reopened.List())
	}
}

func TestProgramStorePersistsOnlyStrictContentAddressedPrograms(t *testing.T) {
	builtins, err := nodes.Build()
	if err != nil {
		t.Fatal(err)
	}
	build := testDigest(t, "workflowstore compiler")
	compiled, err := compiler.New(build, builtins.ConfigValidators).CompileDraft(context.Background(), compiler.CompileRequest{SourceJSON: concatSource(t, 0, "a", "b"), Catalog: builtins.Catalog})
	if err != nil || len(compiled.Diagnostics) != 0 {
		t.Fatalf("compile = %v, %#v", err, compiled.Diagnostics)
	}
	program, ok := compiled.Program()
	if !ok {
		t.Fatal("missing Program")
	}
	root := t.TempDir()
	store, err := workflowstore.OpenProgramStore(root, builtins.Catalog, builtins.ConfigValidators, build, workflowstore.ProgramStoreOptions{MaxPrograms: 2})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Put(context.Background(), program); err != nil {
		t.Fatal(err)
	}
	if err := store.Put(context.Background(), program); err != nil {
		t.Fatalf("idempotent Put = %v", err)
	}
	loaded, err := store.Load(program.Hash())
	if err != nil || loaded.Hash() != program.Hash() || string(loaded.Artifact()) != string(program.Artifact()) {
		t.Fatalf("Load = %s, %v", loaded.Hash(), err)
	}
	reopened, err := workflowstore.OpenProgramStore(root, builtins.Catalog, builtins.ConfigValidators, build, workflowstore.ProgramStoreOptions{MaxPrograms: 2})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reopened.Load(program.Hash()); err != nil {
		t.Fatal(err)
	}
}

func concatSource(t *testing.T, revision int, a, b string) []byte {
	t.Helper()
	builtins, err := nodes.Build()
	if err != nil {
		t.Fatal(err)
	}
	ref := builtins.ConcatContract.NodeRef()
	return []byte(fmt.Sprintf(`{
		"format":"yotta.workflow","version":"3.1","workflow":{"id":"wf-store","name":"Store"},
		"revision":%d,"entryGraph":"main","graphs":[{"id":"main","kind":"main","nodes":[
			{"id":"concat","nodeRef":{"nodeTypeId":%q,"version":"1.0.0","semanticDigest":%q},"position":{"x":0,"y":0},"config":{},
			 "bindings":{"a":{"kind":"value","value":%q},"b":{"kind":"value","value":%q}}}
		],"edges":[],"inputs":[],"outputs":[]}],"variables":[],"secretRefs":[]
	}`, revision, ref.NodeTypeID, ref.SemanticDigest, a, b))
}

func testDigest(t *testing.T, label string) artifact.Digest {
	t.Helper()
	digest, err := artifact.Sum("yotta/test/workflowstore/v1", []byte(label))
	if err != nil {
		t.Fatal(err)
	}
	return digest
}
