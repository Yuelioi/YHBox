package nodes31

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/yottaapp/yotta/internal/nodeauthoring"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

func TestExtendedPureDefinitionsAreStrictAndGeneratedFromTheCatalog(t *testing.T) {
	builtins, err := Build()
	if err != nil {
		t.Fatal(err)
	}
	nodeIDs := []string{
		DivideNodeID, ModuloNodeID, NegateNodeID, AbsoluteNodeID, MinimumNodeID, MaximumNodeID,
		FloorNodeID, CeilingNodeID, RoundNodeID, ClampNodeID, PowerNodeID, SquareRootNodeID,
		EqualNodeID, NotEqualNodeID, ReplaceNodeID, SubstringNodeID, TrimNodeID, UppercaseNodeID,
		LowercaseNodeID, IndexOfNodeID, StartsWithNodeID, EndsWithNodeID, RegexMatchNodeID,
		RegexExtractNodeID, ToStringNodeID, StringToNumberNodeID, StringToBooleanNodeID,
		ParseJSONNodeID, ToJSONNodeID, JSONPathNodeID, SelectNodeID, MakePointNodeID,
		OffsetPointNodeID, PointDistanceNodeID, RegionAroundPointNodeID,
	}
	if len(nodeIDs) != 35 || len(builtins.Definitions()) != 57 {
		t.Fatalf("extended=%d total=%d", len(nodeIDs), len(builtins.Definitions()))
	}
	for _, nodeID := range nodeIDs {
		definition, ok := builtins.Definition(nodeID)
		if !ok || definition.EvaluateInline == nil {
			t.Fatalf("missing extended definition %q", nodeID)
		}
		machine := definition.Contract.Machine()
		if machine.Execution.Class != nodecontract.ExecutionPureData || machine.Execution.Determinism != nodecontract.Deterministic ||
			len(machine.Execution.Effects)+len(machine.CapabilityRequirements)+len(machine.Ports.ExecInputs)+len(machine.Ports.ExecOutputs)+len(machine.Ports.ErrorOutputs) != 0 {
			t.Fatalf("extended node %q contains non-pure semantics", nodeID)
		}
	}
	projection, err := nodeauthoring.Project(nodeauthoring.Input{
		Catalog: builtins.Catalog, Types: builtins.Types, Capabilities: builtins.Capabilities,
		Contracts: builtins.Contracts, GeneratorVersion: GeneratorVersion,
	})
	if err != nil {
		t.Fatal(err)
	}
	unit, _ := projection.Type(PointUnitTypeID)
	point, _ := projection.Type(PointTypeID)
	jsonType, _ := projection.Type(JSONTypeID)
	if unit.Control != nodeauthoring.ControlSelect || len(unit.Constraints.Enum) != 2 {
		t.Fatalf("point unit authoring = %#v", unit)
	}
	if point.EditorAdapter != "point" || jsonType.Control != nodeauthoring.ControlJSON {
		t.Fatalf("point/json authoring = %#v / %#v", point, jsonType)
	}
}

func TestExtendedEvaluatorsReportDeclaredTerminalFailures(t *testing.T) {
	builtins, err := Build()
	if err != nil {
		t.Fatal(err)
	}
	evaluate := func(nodeID string, inputs map[string]json.RawMessage) (map[string]json.RawMessage, error) {
		t.Helper()
		definition, _ := builtins.Definition(nodeID)
		return definition.EvaluateInline(context.Background(), inputs, nil)
	}
	if result, err := evaluate(StringToNumberNodeID, map[string]json.RawMessage{"text": json.RawMessage(`"12.5"`)}); err != nil || string(result["result"]) != "12.5" {
		t.Fatalf("string-to-number = %s, %v", result["result"], err)
	}
	if result, err := evaluate(ParseJSONNodeID, map[string]json.RawMessage{"text": json.RawMessage(`"{\"b\":2,\"a\":1}"`)}); err != nil || string(result["result"]) != `{"a":1,"b":2}` {
		t.Fatalf("parse JSON = %s, %v", result["result"], err)
	}
	if result, err := evaluate(JSONPathNodeID, map[string]json.RawMessage{"json": json.RawMessage(`{"a":["节点"]}`), "path": json.RawMessage(`"$.a[0]"`)}); err != nil || string(result["result"]) != `"节点"` {
		t.Fatalf("JSON path = %s, %v", result["result"], err)
	}
	for _, failureCase := range []struct {
		nodeID string
		inputs map[string]json.RawMessage
		code   string
	}{
		{DivideNodeID, map[string]json.RawMessage{"a": json.RawMessage(`1`), "b": json.RawMessage(`0`)}, divisionByZeroCode},
		{SquareRootNodeID, map[string]json.RawMessage{"value": json.RawMessage(`-1`)}, mathDomainErrorCode},
		{RegexMatchNodeID, map[string]json.RawMessage{"text": json.RawMessage(`"x"`), "pattern": json.RawMessage(`"["`)}, invalidRegexCode},
		{StringToBooleanNodeID, map[string]json.RawMessage{"text": json.RawMessage(`"yes"`)}, invalidBooleanCode},
	} {
		_, err := evaluate(failureCase.nodeID, failureCase.inputs)
		var failure *InlineFailure
		if !errors.As(err, &failure) || failure.Code != failureCase.code || failure.Output != "" {
			t.Fatalf("%s failure = %#v, %v", failureCase.nodeID, failure, err)
		}
	}
}
