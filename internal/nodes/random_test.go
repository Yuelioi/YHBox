package nodes

import (
	"encoding/json"
	"testing"

	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodeauthoring"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

func TestRecordedObservationDefinitionsExposeDataWithoutFakeControlPorts(t *testing.T) {
	builtins, err := Build()
	if err != nil {
		t.Fatal(err)
	}
	for _, nodeID := range []string{
		RandomIntegerNodeID, RandomNumberNodeID, RandomBooleanNodeID, RandomChoiceNodeID, ObserveTimeNodeID,
	} {
		definition, ok := builtins.Definition(nodeID)
		if !ok || definition.EvaluateInline != nil {
			t.Fatalf("recorded definition %q = %#v", nodeID, definition)
		}
		machine := definition.Contract.Machine()
		if machine.Execution.Class != nodecontract.ExecutionEffect || machine.Execution.Determinism != nodecontract.Recorded ||
			machine.Execution.Evaluation != nodecontract.EvaluationPull || machine.Execution.Cache != nodecontract.CacheNone ||
			len(machine.Execution.Effects) != 1 || len(machine.Ports.DataOutputs) != 1 || machine.Ports.DataOutputs[0].ID != "result" ||
			len(machine.Ports.ExecInputs)+len(machine.Ports.ExecOutputs)+len(machine.Ports.ErrorOutputs) != 0 {
			t.Fatalf("recorded contract %q = %#v", nodeID, machine)
		}
	}
}

func TestRandomDistributionAndIntegerAuthoringAreSchemaDerived(t *testing.T) {
	builtins, err := Build()
	if err != nil {
		t.Fatal(err)
	}
	projection, err := nodeauthoring.Project(nodeauthoring.Input{
		Catalog: builtins.Catalog, Types: builtins.Types, Capabilities: builtins.Capabilities,
		Contracts: builtins.Contracts, GeneratorVersion: GeneratorVersion,
	})
	if err != nil {
		t.Fatal(err)
	}
	distribution, ok := projection.Type(RandomDistributionTypeID)
	if !ok || distribution.Control != nodeauthoring.ControlSelect || len(distribution.Constraints.Enum) != 2 {
		t.Fatalf("random distribution authoring = %#v", distribution)
	}
	integer, ok := projection.Type(IntegerTypeID)
	if !ok || integer.Control != nodeauthoring.ControlInteger || integer.Constraints.Minimum == nil || integer.Constraints.Maximum == nil {
		t.Fatalf("safe integer authoring = %#v", integer)
	}
	for _, raw := range []json.RawMessage{json.RawMessage(`-9007199254740991`), json.RawMessage(`9007199254740991`)} {
		if _, err := datatype.SealInlineJSON(builtins.Catalog, datatype.RefResolvedType(builtins.IntegerType.TypeRef()), raw); err != nil {
			t.Fatalf("safe integer boundary %s: %v", raw, err)
		}
	}
	if _, err := datatype.SealInlineJSON(builtins.Catalog, datatype.RefResolvedType(builtins.IntegerType.TypeRef()), json.RawMessage(`9007199254740992`)); err == nil {
		t.Fatal("integer outside the portable JSON range was accepted")
	}
}
