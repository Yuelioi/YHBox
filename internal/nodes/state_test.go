package nodes

import (
	"testing"

	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodeauthoring"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

func TestStateDefinitionsDeclareTypedAttenuatedAccessAndOnlyWriteHasControlFlow(t *testing.T) {
	builtins, err := Build()
	if err != nil {
		t.Fatal(err)
	}
	for _, nodeID := range []string{StateReadNodeID, StateWriteNodeID, StateMetadataNodeID, StateLastChangeNodeID} {
		definition, ok := builtins.Definition(nodeID)
		if !ok || definition.EvaluateInline != nil {
			t.Fatalf("state definition %q = %#v", nodeID, definition)
		}
		machine := definition.Contract.Machine()
		if machine.Execution.Class != nodecontract.ExecutionEffect || machine.Execution.Determinism != nodecontract.Recorded ||
			machine.Execution.Cache != nodecontract.CacheNone || len(machine.StateAccesses) != 1 || machine.StateAccesses[0].ID != "state" {
			t.Fatalf("state contract %q = %#v", nodeID, machine)
		}
		if len(machine.StateAccesses[0].Type.Constraints) != 0 {
			t.Fatalf("state contract %q redundantly constrains an already durable slot = %#v", nodeID, machine.StateAccesses[0])
		}
		controlPorts := len(machine.Ports.ExecInputs) + len(machine.Ports.ExecOutputs)
		if nodeID == StateWriteNodeID {
			if controlPorts != 2 || machine.Ports.ExecInputs[0].ID != "in" || machine.Ports.ExecOutputs[0].ID != "done" {
				t.Fatalf("state write control ports = %#v", machine.Ports)
			}
		} else if controlPorts != 0 {
			t.Fatalf("state read node invented control flow: %#v", machine.Ports)
		}
	}
	increment, ok := builtins.Definition(StateIncrementNodeID)
	if !ok {
		t.Fatal("state increment definition is missing")
	}
	constraints := increment.Contract.Machine().StateAccesses[0].Type.Constraints
	if len(constraints) != 2 || constraints[0] != string(datatype.TraitDurable) ||
		constraints[1] != string(datatype.TraitNumeric) {
		t.Fatalf("state increment constraints = %#v", constraints)
	}
	projection, err := nodeauthoring.Project(nodeauthoring.Input{
		Catalog: builtins.Catalog, Types: builtins.Types, Capabilities: builtins.Capabilities,
		Contracts: builtins.Contracts, GeneratorVersion: GeneratorVersion,
	})
	if err != nil {
		t.Fatal(err)
	}
	read, ok := projection.Node(StateReadNodeID)
	if !ok || len(read.ConfigFields) != 1 || read.ConfigFields[0].Control != nodeauthoring.ControlStateVariable {
		t.Fatalf("state variable authoring = %#v", read.ConfigFields)
	}
}
