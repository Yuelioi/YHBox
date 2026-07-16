package noderuntime_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/noderuntime"
	"github.com/yottaapp/yotta/internal/nodes"
	run "github.com/yottaapp/yotta/internal/run"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
	"github.com/yottaapp/yotta/internal/workspacefs"
)

func TestFilesystemReadJSONUsesGrantedWorkspaceProviderAndRedactedJournal(t *testing.T) {
	builtins, err := nodes.Build()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "fixtures"), 0o700); err != nil {
		t.Fatal(err)
	}
	const relative = "fixtures/private.json"
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(relative)), []byte("{\n  \"answer\": 42\n}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	provider, err := workspacefs.NewProvider(root, workspacefs.Limits{MaxReadBytes: nodes.DefaultFileReadBytes})
	if err != nil {
		t.Fatal(err)
	}
	program := compilePrimitiveProgram(t, builtins, filesystemSource(t, builtins, nodes.FileReadJSONNodeID, relative))
	now := time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC)
	providers := map[string]run.InstalledProvider{
		workspacefs.ProviderID: {ArtifactDigest: workspaceFSProviderDigest(t), ABI: workspacefs.ProviderABI, Provider: provider},
	}
	_, owner, journal := admittedExecution(t, builtins, program, providers, now)
	t.Cleanup(func() { _ = owner.Close(context.Background()) })
	adapters, err := noderuntime.Installed(builtins, testDependencies())
	if err != nil {
		t.Fatal(err)
	}
	result, err := compiler.NewExecutor(builtins.Catalog, adapters, compiler.ExecutorOptions{Now: func() time.Time { return now }}).
		Run(context.Background(), program, owner, journal)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(result.NodeOutputs["file"]["value"].InlineJSON()); got != `{"answer":42}` {
		t.Fatalf("JSON value = %s", got)
	}
	if got := string(result.NodeOutputs["file"]["text"].InlineJSON()); got != `"{\n  \"answer\": 42\n}\n"` {
		t.Fatalf("text value = %s", got)
	}
	metadata := result.NodeOutputs["file"]["metadata"].InlineJSON()
	if !bytes.Contains(metadata, []byte(`"path":"fixtures/private.json"`)) || !bytes.Contains(metadata, []byte(`"size":19`)) {
		t.Fatalf("metadata = %s", metadata)
	}
	var action *run.JournalEntry
	for _, entry := range journal.Current().Journal() {
		if entry.Kind == run.JournalAdapterAction && entry.NodeID == "file" {
			copy := entry
			action = &copy
		}
	}
	if action == nil || action.EffectID != nodes.FileReadEffectID || action.ActionOutcome != run.ActionSucceeded ||
		action.Summary.Counters["bytes"] != 19 || action.Summary.Facts["path_digest"] == "" {
		t.Fatalf("filesystem action = %#v", action)
	}
	encoded, err := json.Marshal(action)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encoded, []byte(relative)) || bytes.Contains(encoded, []byte("answer")) {
		t.Fatalf("filesystem journal leaked path or content: %s", encoded)
	}
}

func TestFilesystemReadFailsClosedOnTraversalBeforeHostRead(t *testing.T) {
	builtins, err := nodes.Build()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	provider, err := workspacefs.NewProvider(root, workspacefs.Limits{MaxReadBytes: nodes.DefaultFileReadBytes})
	if err != nil {
		t.Fatal(err)
	}
	program := compilePrimitiveProgram(t, builtins, filesystemSource(t, builtins, nodes.FileReadTextNodeID, "../secret.txt"))
	now := time.Date(2026, 7, 16, 10, 30, 0, 0, time.UTC)
	_, owner, journal := admittedExecution(t, builtins, program, map[string]run.InstalledProvider{
		workspacefs.ProviderID: {ArtifactDigest: workspaceFSProviderDigest(t), ABI: workspacefs.ProviderABI, Provider: provider},
	}, now)
	t.Cleanup(func() { _ = owner.Close(context.Background()) })
	adapters, err := noderuntime.Installed(builtins, testDependencies())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := compiler.NewExecutor(builtins.Catalog, adapters, compiler.ExecutorOptions{Now: func() time.Time { return now }}).
		Run(context.Background(), program, owner, journal); err == nil {
		t.Fatal("filesystem traversal unexpectedly succeeded")
	}
	for _, entry := range journal.Current().Journal() {
		if entry.Kind == run.JournalAdapterAction && entry.NodeID == "file" {
			if entry.ActionOutcome != run.ActionFailed || entry.ErrorCode != workspacefs.CodeInvalidPath {
				t.Fatalf("filesystem failure action = %#v", entry)
			}
			return
		}
	}
	t.Fatal("filesystem failure action was not journaled")
}

func filesystemSource(t *testing.T, builtins nodes.Builtins, nodeID, path string) []byte {
	t.Helper()
	started, ok := builtins.Definition(nodes.RunStartedNodeID)
	if !ok {
		t.Fatal("run-started definition is missing")
	}
	file, ok := builtins.Definition(nodeID)
	if !ok {
		t.Fatalf("filesystem definition %q is missing", nodeID)
	}
	config := `{"maxBytes":1048576}`
	if nodeID == nodes.FileReadTextNodeID {
		config = `{"encoding":"utf-8","maxBytes":1048576}`
	} else if nodeID == nodes.FileStatNodeID {
		config = `{}`
	}
	return []byte(fmt.Sprintf(`{
		"format":"yotta.workflow","version":"3.1","workflow":{"id":"wf-filesystem","name":"Filesystem"},
		"revision":0,"entryGraph":"main","graphs":[{"id":"main","kind":"main","nodes":[
			{"id":"start","nodeRef":{"nodeTypeId":%q,"version":"1.0.0","semanticDigest":%q},"position":{"x":0,"y":0},"config":{},"bindings":{}},
			{"id":"file","nodeRef":{"nodeTypeId":%q,"version":"1.0.0","semanticDigest":%q},"position":{"x":1,"y":0},
			 "config":%s,"bindings":{"path":{"kind":"value","value":%q}}}
		],"edges":[{"channel":"exec","from":{"nodeId":"start","portId":"started"},"to":{"nodeId":"file","portId":"in"}}],
		"inputs":[],"outputs":[]}],"variables":[],"secretRefs":[]
	}`, started.Contract.NodeRef().NodeTypeID, started.Contract.NodeRef().SemanticDigest,
		file.Contract.NodeRef().NodeTypeID, file.Contract.NodeRef().SemanticDigest, config, path))
}
