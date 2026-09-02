package compiler

import (
	"bytes"
	"context"
	"testing"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/nodes"
	"github.com/yottaapp/yotta/internal/workflow/schema"
)

func TestCompilerAppliesRunResourceOverrideWithoutChangingSourceHash(t *testing.T) {
	builtins, err := nodes.Build()
	if err != nil {
		t.Fatal(err)
	}
	runStarted, ok := builtins.Catalog.Lookup(nodes.RunStartedNodeID)
	if !ok {
		t.Fatal("run-started node is missing")
	}
	original := blob.BlobRef{MediaType: schema.MacroResourceMediaType, Digest: testDigest(t, "original macro"), Size: 4}
	prepared := blob.BlobRef{MediaType: schema.MacroResourceMediaType, Digest: testDigest(t, "prepared macro"), Size: 5}
	source := schema.WorkflowSource{
		Format: schema.Format, Version: schema.Version,
		Workflow: schema.Workflow{ID: "override", Name: "Override"}, Revision: 0, EntryGraph: "main",
		Graphs: []schema.Graph{{
			ID: "main", Kind: schema.GraphKindMain,
			Nodes: []schema.Node{
				{ID: "run-started", NodeRef: runStarted.Contract.NodeRef(), Position: schema.Position{}, Config: map[string]any{}, Bindings: map[string]schema.InputBinding{}},
				{ID: "play", NodeRef: builtins.PlayMacroContract.NodeRef(), Position: schema.Position{}, Config: map[string]any{"slot": "game"}, Bindings: map[string]schema.InputBinding{
					"macro": {Kind: schema.BindingResource, Resource: &schema.ResourceBinding{ResourceID: "macro"}},
				}},
			},
			Edges:  []schema.Edge{{Channel: schema.EdgeExec, From: schema.Endpoint{NodeID: "run-started", PortID: "started"}, To: schema.Endpoint{NodeID: "play", PortID: "in"}}},
			Inputs: []schema.GraphPort{}, Outputs: []schema.GraphPort{},
		}},
		Resources:                []schema.WorkflowResource{{ID: "macro", Kind: schema.ResourceMacro, Name: "Macro", Macro: &schema.MacroResource{Blob: original, BaseResolution: [2]int{1920, 1080}, ActionCount: 1, DurationUs: 1}}},
		TargetProfileDefinitions: []schema.TargetProfileDefinition{}, CredentialRequirements: []schema.CredentialRequirement{}, Dependencies: []schema.NodePackageDependency{}, Variables: []schema.Variable{},
	}
	raw, err := artifact.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	_, _, sourceHash, diagnostics, err := schema.CanonicalSource(raw)
	if err != nil || schema.HasErrors(diagnostics) {
		t.Fatalf("source diagnostics=%+v err=%v", diagnostics, err)
	}
	verified := []blob.BlobRef{}
	result, err := New(testDigest(t, "override compiler"), builtins.ConfigValidators).CompileDraft(context.Background(), CompileRequest{
		SourceJSON: raw, Catalog: builtins.Catalog,
		BlobVerifier: BlobVerifierFunc(func(_ context.Context, ref blob.BlobRef) error {
			verified = append(verified, ref)
			return nil
		}),
		ResourceOverrides: []ResourceOverride{{ResourceID: "macro", TargetSlot: "game", Blob: prepared}},
	})
	program, ok := result.Program()
	if err != nil || !ok || result.SourceHash != sourceHash || len(verified) != 1 || verified[0] != prepared {
		t.Fatalf("program=%v source=%s verified=%+v diagnostics=%+v err=%v", ok, result.SourceHash, verified, result.Diagnostics, err)
	}
	if !bytes.Contains(program.Artifact(), []byte(prepared.Digest)) || bytes.Contains(program.Artifact(), []byte(original.Digest)) {
		t.Fatal("Program did not freeze the prepared override")
	}
}
