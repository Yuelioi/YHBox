package compiler

import (
	"context"
	"fmt"
	"testing"

	"github.com/yottaapp/yotta/internal/nodes"
	"github.com/yottaapp/yotta/internal/workflow/schema"
)

func TestCompileAllowsDisconnectedDraftNodesBesideReachableExecution(t *testing.T) {
	builtins, err := nodes.Build()
	if err != nil {
		t.Fatal(err)
	}
	started, _ := builtins.Definition(nodes.RunStartedNodeID)
	end, _ := builtins.Definition(nodes.EndBranchNodeID)
	concat, _ := builtins.Definition(nodes.ConcatNodeID)
	source := []byte(fmt.Sprintf(`{
		"format":"yotta.workflow","version":"1","workflow":{"id":"draft","name":"Draft"},"revision":0,"entryGraph":"main",
		"graphs":[{"id":"main","kind":"main","name":"Main","nodes":[
			{"id":"start","nodeRef":{"nodeTypeId":%q,"version":%q,"semanticDigest":%q},"config":{},"bindings":{},"position":{"x":0,"y":0}},
			{"id":"end","nodeRef":{"nodeTypeId":%q,"version":%q,"semanticDigest":%q},"config":{},"bindings":{},"position":{"x":200,"y":0}},
			{"id":"draft-node","nodeRef":{"nodeTypeId":%q,"version":%q,"semanticDigest":%q},"config":{},"bindings":{},"position":{"x":0,"y":200}}
		],"edges":[{"channel":"exec","from":{"nodeId":"start","portId":"started"},"to":{"nodeId":"end","portId":"in"}}],"inputs":[],"outputs":[]}],
		"variables":[],"resources":[],"targetProfileDefinitions":[],"credentialRequirements":[],"dependencies":[]
	}`,
		started.Contract.NodeRef().NodeTypeID, started.Contract.NodeRef().Version, started.Contract.NodeRef().SemanticDigest,
		end.Contract.NodeRef().NodeTypeID, end.Contract.NodeRef().Version, end.Contract.NodeRef().SemanticDigest,
		concat.Contract.NodeRef().NodeTypeID, concat.Contract.NodeRef().Version, concat.Contract.NodeRef().SemanticDigest,
	))

	result, err := New(testDigest(t, "disconnected draft"), builtins.ConfigValidators).CompileDraft(context.Background(), CompileRequest{
		SourceJSON: source,
		Catalog:    builtins.Catalog,
	})
	if err != nil {
		t.Fatal(err)
	}
	program, ok := result.Program()
	if !ok {
		t.Fatalf("disconnected draft blocked Program: %#v", result.Diagnostics)
	}
	if schema.HasErrors(result.Diagnostics) {
		t.Fatalf("disconnected draft emitted blocking diagnostics: %#v", result.Diagnostics)
	}
	if len(result.Diagnostics) != 2 {
		t.Fatalf("disconnected draft diagnostics = %#v", result.Diagnostics)
	}
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code != CodeMissingInputBinding ||
			diagnostic.Severity != schema.SeverityWarning ||
			diagnostic.NodeID != "draft-node" {
			t.Fatalf("disconnected draft diagnostic = %#v", diagnostic)
		}
	}
	nodes := program.Nodes()
	if len(nodes) != 2 {
		t.Fatalf("Program nodes = %#v", nodes)
	}
	for _, node := range nodes {
		if node.ID == "draft-node" {
			t.Fatalf("disconnected draft leaked into Program: %#v", node)
		}
	}
}
