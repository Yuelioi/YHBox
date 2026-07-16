package authoring_test

import (
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodeauthoring"
	"github.com/yottaapp/yotta/internal/nodes31"
	"github.com/yottaapp/yotta/internal/workflow/authoring"
	"github.com/yottaapp/yotta/internal/workflow/schema"
)

func TestEngineAppliesAtomicTypedPatchWithHostOwnedNodeIDs(t *testing.T) {
	builtins, projection := testContracts(t)
	ids := []string{"node-left", "node-right"}
	engine, err := authoring.New(builtins.Catalog, projection, func() string {
		id := ids[0]
		ids = ids[1:]
		return id
	})
	if err != nil {
		t.Fatal(err)
	}
	source := emptySource()
	result, err := engine.Apply(source, []authoring.Command{
		{Kind: authoring.CommandAddNode, AddNode: &authoring.AddNodeCommand{
			GraphID: "main", NodeTypeID: nodes31.ConcatNodeID, Handle: "left", Position: schema.Position{X: 10, Y: 20},
		}},
		{Kind: authoring.CommandAddNode, AddNode: &authoring.AddNodeCommand{
			GraphID: "main", NodeTypeID: nodes31.ConcatNodeID, Handle: "right", Position: schema.Position{X: 30, Y: 40},
		}},
		{Kind: authoring.CommandBindValue, BindValue: &authoring.BindValueCommand{GraphID: "main", NodeID: "$left", PortID: "a", Value: "hello"}},
		{Kind: authoring.CommandBindValue, BindValue: &authoring.BindValueCommand{GraphID: "main", NodeID: "$left", PortID: "b", Value: " world"}},
		{Kind: authoring.CommandClearBinding, ClearBinding: &authoring.PortCommand{GraphID: "main", NodeID: "$right", PortID: "a"}},
		{Kind: authoring.CommandConnect, Connect: &authoring.EdgeCommand{GraphID: "main", Edge: schema.Edge{
			Channel: schema.EdgeData,
			From:    schema.Endpoint{NodeID: "$left", PortID: "result"},
			To:      schema.Endpoint{NodeID: "$right", PortID: "a"},
		}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Source.Revision != 1 || len(result.Source.Graphs[0].Nodes) != 2 || len(result.Source.Graphs[0].Edges) != 1 {
		t.Fatalf("result = %#v", result.Source)
	}
	if got := result.Source.Graphs[0].Edges[0]; got.From.NodeID != "node-left" || got.To.NodeID != "node-right" {
		t.Fatalf("resolved edge = %#v", got)
	}
	if len(result.GeneratedNodes) != 2 || result.GeneratedNodes[0].Handle != "left" || result.GeneratedNodes[1].NodeID != "node-right" {
		t.Fatalf("generated nodes = %#v", result.GeneratedNodes)
	}
	if source.Revision != 0 || len(source.Graphs[0].Nodes) != 0 {
		t.Fatalf("input source was mutated: %#v", source)
	}
}

func TestEngineRejectsMismatchedUnionAndPublishesNothing(t *testing.T) {
	builtins, projection := testContracts(t)
	engine, err := authoring.New(builtins.Catalog, projection, func() string { return "node-one" })
	if err != nil {
		t.Fatal(err)
	}
	source := emptySource()
	_, err = engine.Apply(source, []authoring.Command{{
		Kind:     authoring.CommandAddNode,
		MoveNode: &authoring.MoveNodeCommand{GraphID: "main", NodeID: "missing", Position: schema.Position{}},
	}})
	var patchErr *authoring.PatchError
	if !errors.As(err, &patchErr) || patchErr.Code != "INVALID_COMMAND" || patchErr.CommandIndex != 0 {
		t.Fatalf("error = %#v", err)
	}
	if source.Revision != 0 || len(source.Graphs[0].Nodes) != 0 {
		t.Fatalf("failed patch mutated input: %#v", source)
	}
}

func TestEngineUsesInstructionSignalChannels(t *testing.T) {
	builtins, projection := testContracts(t)
	ids := []string{"delay", "retry"}
	engine, err := authoring.New(builtins.Catalog, projection, func() string {
		id := ids[0]
		ids = ids[1:]
		return id
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := engine.Apply(emptySource(), []authoring.Command{
		{Kind: authoring.CommandAddNode, AddNode: &authoring.AddNodeCommand{GraphID: "main", NodeTypeID: nodes31.DelayNodeID, Handle: "delay"}},
		{Kind: authoring.CommandAddNode, AddNode: &authoring.AddNodeCommand{GraphID: "main", NodeTypeID: nodes31.RetryNodeID, Handle: "retry"}},
		{Kind: authoring.CommandConnect, Connect: &authoring.EdgeCommand{GraphID: "main", Edge: schema.Edge{
			Channel: schema.EdgeError,
			From:    schema.Endpoint{NodeID: "$delay", PortID: "failed"},
			To:      schema.Endpoint{NodeID: "$retry", PortID: "retry"},
		}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := result.Source.Graphs[0].Edges; len(got) != 1 || got[0].Channel != schema.EdgeError {
		t.Fatalf("error route = %#v", got)
	}

	_, err = engine.Apply(result.Source, []authoring.Command{{
		Kind: authoring.CommandConnect,
		Connect: &authoring.EdgeCommand{GraphID: "main", Edge: schema.Edge{
			Channel: schema.EdgeExec,
			From:    schema.Endpoint{NodeID: "delay", PortID: "done"},
			To:      schema.Endpoint{NodeID: "retry", PortID: "retry"},
		}},
	}})
	var patchErr *authoring.PatchError
	if !errors.As(err, &patchErr) || patchErr.Code != "INVALID_EDGE" {
		t.Fatalf("wrong-channel error = %#v", err)
	}
}

func TestEngineEditsStateNodesAndDisconnectsAtomically(t *testing.T) {
	builtins, projection := testContracts(t)
	ids := []string{"node-read", "node-concat", "node-delay"}
	engine, err := authoring.New(builtins.Catalog, projection, func() string {
		id := ids[0]
		ids = ids[1:]
		return id
	})
	if err != nil {
		t.Fatal(err)
	}
	edge := schema.Edge{
		Channel: schema.EdgeData,
		From:    schema.Endpoint{NodeID: "$read", PortID: "result"},
		To:      schema.Endpoint{NodeID: "$concat", PortID: "a"},
	}
	result, err := engine.Apply(emptySource(), []authoring.Command{
		{Kind: authoring.CommandRenameWorkflow, RenameWorkflow: &authoring.RenameWorkflowCommand{Name: "  Stateful workflow  "}},
		{Kind: authoring.CommandAddStateVariable, AddStateVariable: &authoring.AddStateVariableCommand{
			Name: "message", Type: datatype.RefExpression(builtins.StringType.TypeRef()), Default: "hello",
		}},
		{Kind: authoring.CommandAddNode, AddNode: &authoring.AddNodeCommand{GraphID: "main", NodeTypeID: nodes31.StateReadNodeID, Handle: "read"}},
		{Kind: authoring.CommandSetConfig, SetConfig: &authoring.SetConfigCommand{GraphID: "main", NodeID: "$read", FieldID: "variable", Value: "message"}},
		{Kind: authoring.CommandMoveNode, MoveNode: &authoring.MoveNodeCommand{GraphID: "main", NodeID: "$read", Position: schema.Position{X: 12, Y: 34}}},
		{Kind: authoring.CommandSetNodeLabel, SetNodeLabel: &authoring.SetNodeLabelCommand{GraphID: "main", NodeID: "$read", Label: "Read message"}},
		{Kind: authoring.CommandSetNodeDisabled, SetNodeDisabled: &authoring.SetNodeDisabledCommand{GraphID: "main", NodeID: "$read", Disabled: true}},
		{Kind: authoring.CommandAddNode, AddNode: &authoring.AddNodeCommand{GraphID: "main", NodeTypeID: nodes31.ConcatNodeID, Handle: "concat"}},
		{Kind: authoring.CommandAddNode, AddNode: &authoring.AddNodeCommand{GraphID: "main", NodeTypeID: nodes31.DelayNodeID, Handle: "delay"}},
		{Kind: authoring.CommandBindDefault, BindDefault: &authoring.PortCommand{GraphID: "main", NodeID: "$delay", PortID: "duration-milliseconds"}},
		{Kind: authoring.CommandConnect, Connect: &authoring.EdgeCommand{GraphID: "main", Edge: edge}},
		{Kind: authoring.CommandDisconnect, Disconnect: &authoring.EdgeCommand{GraphID: "main", Edge: edge}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Source.Workflow.Name != "Stateful workflow" || len(result.Source.Variables) != 1 || len(result.Source.Graphs[0].Edges) != 0 {
		t.Fatalf("patched source = %#v", result.Source)
	}
	read := result.Source.Graphs[0].Nodes[0]
	if read.ID != "node-read" || read.Position.X != 12 || read.Label != "Read message" || !read.Disabled || read.Config["variable"] != "message" {
		t.Fatalf("state read node = %#v", read)
	}

	removed, err := engine.Apply(result.Source, []authoring.Command{
		{Kind: authoring.CommandRemoveNode, RemoveNode: &authoring.NodeCommand{GraphID: "main", NodeID: "node-read"}},
		{Kind: authoring.CommandRemoveNode, RemoveNode: &authoring.NodeCommand{GraphID: "main", NodeID: "node-concat"}},
		{Kind: authoring.CommandRemoveNode, RemoveNode: &authoring.NodeCommand{GraphID: "main", NodeID: "node-delay"}},
		{Kind: authoring.CommandRemoveStateVariable, RemoveStateVariable: &authoring.RemoveStateVariableCommand{Name: "message"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(removed.Source.Graphs[0].Nodes) != 0 || len(removed.Source.Variables) != 0 || removed.Source.Revision != 2 {
		t.Fatalf("removed source = %#v", removed.Source)
	}
}

func TestEngineRejectsMutationThatBreaksStateContract(t *testing.T) {
	builtins, projection := testContracts(t)
	engine, err := authoring.New(builtins.Catalog, projection, func() string { return "node-read" })
	if err != nil {
		t.Fatal(err)
	}
	result, err := engine.Apply(emptySource(), []authoring.Command{
		{Kind: authoring.CommandAddStateVariable, AddStateVariable: &authoring.AddStateVariableCommand{
			Name: "message", Type: datatype.RefExpression(builtins.StringType.TypeRef()), Default: "hello",
		}},
		{Kind: authoring.CommandAddNode, AddNode: &authoring.AddNodeCommand{GraphID: "main", NodeTypeID: nodes31.StateReadNodeID}},
		{Kind: authoring.CommandSetConfig, SetConfig: &authoring.SetConfigCommand{GraphID: "main", NodeID: "node-read", FieldID: "variable", Value: "message"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, command := range []authoring.Command{
		{Kind: authoring.CommandClearConfig, ClearConfig: &authoring.FieldCommand{GraphID: "main", NodeID: "node-read", FieldID: "variable"}},
		{Kind: authoring.CommandRemoveStateVariable, RemoveStateVariable: &authoring.RemoveStateVariableCommand{Name: "message"}},
	} {
		_, err := engine.Apply(result.Source, []authoring.Command{command})
		var patchErr *authoring.PatchError
		if !errors.As(err, &patchErr) {
			t.Fatalf("command %#v error = %v", command, err)
		}
	}
	if result.Source.Revision != 1 || len(result.Source.Variables) != 1 {
		t.Fatalf("failed mutations changed source = %#v", result.Source)
	}
}

func TestEngineRejectsInvalidCommandBoundariesWithoutPublishing(t *testing.T) {
	builtins, projection := testContracts(t)
	ids := []string{"left", "right"}
	engine, err := authoring.New(builtins.Catalog, projection, func() string {
		id := ids[0]
		ids = ids[1:]
		return id
	})
	if err != nil {
		t.Fatal(err)
	}
	base, err := engine.Apply(emptySource(), []authoring.Command{
		{Kind: authoring.CommandAddNode, AddNode: &authoring.AddNodeCommand{GraphID: "main", NodeTypeID: nodes31.ConcatNodeID, Handle: "left"}},
		{Kind: authoring.CommandAddNode, AddNode: &authoring.AddNodeCommand{GraphID: "main", NodeTypeID: nodes31.ConcatNodeID, Handle: "right"}},
		{Kind: authoring.CommandConnect, Connect: &authoring.EdgeCommand{GraphID: "main", Edge: schema.Edge{
			Channel: schema.EdgeData, From: schema.Endpoint{NodeID: "$left", PortID: "result"}, To: schema.Endpoint{NodeID: "$right", PortID: "a"},
		}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	edge := base.Source.Graphs[0].Edges[0]
	for _, command := range []authoring.Command{
		{Kind: authoring.CommandRenameWorkflow, RenameWorkflow: &authoring.RenameWorkflowCommand{Name: " "}},
		{Kind: authoring.CommandRemoveStateVariable, RemoveStateVariable: &authoring.RemoveStateVariableCommand{Name: "missing"}},
		{Kind: authoring.CommandAddNode, AddNode: &authoring.AddNodeCommand{GraphID: "missing", NodeTypeID: nodes31.ConcatNodeID}},
		{Kind: authoring.CommandAddNode, AddNode: &authoring.AddNodeCommand{GraphID: "main", NodeTypeID: "https://schemas.example.test/missing", Handle: "node"}},
		{Kind: authoring.CommandAddNode, AddNode: &authoring.AddNodeCommand{GraphID: "main", NodeTypeID: nodes31.ConcatNodeID, Handle: "bad handle"}},
		{Kind: authoring.CommandMoveNode, MoveNode: &authoring.MoveNodeCommand{GraphID: "main", NodeID: "left", Position: schema.Position{X: math.NaN()}}},
		{Kind: authoring.CommandSetNodeLabel, SetNodeLabel: &authoring.SetNodeLabelCommand{GraphID: "main", NodeID: "left", Label: strings.Repeat("x", 1025)}},
		{Kind: authoring.CommandSetConfig, SetConfig: &authoring.SetConfigCommand{GraphID: "main", NodeID: "left", FieldID: "missing", Value: true}},
		{Kind: authoring.CommandBindValue, BindValue: &authoring.BindValueCommand{GraphID: "main", NodeID: "left", PortID: "missing", Value: true}},
		{Kind: authoring.CommandBindDefault, BindDefault: &authoring.PortCommand{GraphID: "main", NodeID: "left", PortID: "a"}},
		{Kind: authoring.CommandClearBinding, ClearBinding: &authoring.PortCommand{GraphID: "main", NodeID: "left", PortID: "missing"}},
		{Kind: authoring.CommandConnect, Connect: &authoring.EdgeCommand{GraphID: "main", Edge: edge}},
		{Kind: authoring.CommandDisconnect, Disconnect: &authoring.EdgeCommand{GraphID: "main", Edge: schema.Edge{
			Channel: schema.EdgeData, From: schema.Endpoint{NodeID: "left", PortID: "result"}, To: schema.Endpoint{NodeID: "right", PortID: "b"},
		}}},
		{Kind: authoring.CommandRemoveNode, RemoveNode: &authoring.NodeCommand{GraphID: "main", NodeID: "missing"}},
	} {
		_, err := engine.Apply(base.Source, []authoring.Command{command})
		var patchErr *authoring.PatchError
		if !errors.As(err, &patchErr) {
			t.Fatalf("command %#v error = %v", command, err)
		}
	}
	if base.Source.Revision != 1 || len(base.Source.Graphs[0].Nodes) != 2 || len(base.Source.Graphs[0].Edges) != 1 {
		t.Fatalf("rejected commands mutated base = %#v", base.Source)
	}
}

func testContracts(t *testing.T) (nodes31.Builtins, nodeauthoring.Snapshot) {
	t.Helper()
	builtins, err := nodes31.Build()
	if err != nil {
		t.Fatal(err)
	}
	projection, err := nodeauthoring.Project(nodeauthoring.Input{
		Catalog: builtins.Catalog, Types: builtins.Types, Capabilities: builtins.Capabilities,
		Contracts: builtins.Contracts, GeneratorVersion: nodes31.GeneratorVersion,
	})
	if err != nil {
		t.Fatal(err)
	}
	return builtins, projection
}

func emptySource() schema.WorkflowSource {
	return schema.WorkflowSource{
		Format: schema.Format, Version: schema.Version,
		Workflow: schema.Workflow{ID: "workflow-authoring", Name: "Authoring"},
		Revision: 0, EntryGraph: "main",
		Graphs: []schema.Graph{{
			ID: "main", Kind: schema.GraphKindMain, Nodes: []schema.Node{}, Edges: []schema.Edge{}, Inputs: []schema.GraphPort{}, Outputs: []schema.GraphPort{},
		}},
		Variables: []schema.Variable{}, SecretRefs: []schema.SecretRef{},
	}
}
