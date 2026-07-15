package nodes31

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/yottaapp/yotta/internal/nodeauthoring"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

func TestPrimitiveDefinitionsAreExactPortablePureData(t *testing.T) {
	builtins, err := Build()
	if err != nil {
		t.Fatal(err)
	}
	if len(builtins.Definitions()) != len(builtins.Contracts) || len(builtins.Definitions()) != 15 {
		t.Fatalf("definitions=%d contracts=%d", len(builtins.Definitions()), len(builtins.Contracts))
	}
	seenEntrypoints := map[string]bool{}
	for _, nodeID := range []string{
		AddNodeID, SubtractNodeID, MultiplyNodeID, LessThanNodeID, LessOrEqualNodeID,
		GreaterThanNodeID, GreaterOrEqualNodeID, AndNodeID, OrNodeID, NotNodeID, ContainsNodeID, LengthNodeID,
	} {
		definition, ok := builtins.Definition(nodeID)
		if !ok || definition.EvaluateInline == nil {
			t.Fatalf("missing inline definition %q", nodeID)
		}
		entry, ok := builtins.Catalog.Lookup(nodeID)
		if !ok || entry.Contract.NodeRef() != definition.Contract.NodeRef() || entry.Implementation != definition.Implementation {
			t.Fatalf("Catalog definition mismatch for %q", nodeID)
		}
		if seenEntrypoints[definition.Implementation.Entrypoint] {
			t.Fatalf("duplicate entrypoint %q", definition.Implementation.Entrypoint)
		}
		seenEntrypoints[definition.Implementation.Entrypoint] = true
		machine := definition.Contract.Machine()
		if machine.Execution.Class != nodecontract.ExecutionPureData || machine.Execution.Cache != nodecontract.CachePerRun ||
			len(machine.Execution.Effects)+len(machine.CapabilityRequirements)+len(machine.Ports.ExecInputs)+len(machine.Ports.ExecOutputs)+len(machine.Ports.ErrorOutputs) != 0 {
			t.Fatalf("primitive contract contains non-pure semantics: %#v", machine)
		}
	}

	projection, err := nodeauthoring.Project(nodeauthoring.Input{
		Catalog: builtins.Catalog, Types: builtins.Types, Capabilities: builtins.Capabilities,
		Contracts: builtins.Contracts, GeneratorVersion: GeneratorVersion,
	})
	if err != nil {
		t.Fatal(err)
	}
	for typeID, want := range map[string]nodeauthoring.Control{
		StringTypeID: nodeauthoring.ControlText, NumberTypeID: nodeauthoring.ControlNumber,
		IntegerTypeID: nodeauthoring.ControlInteger, BooleanTypeID: nodeauthoring.ControlToggle,
	} {
		projected, ok := projection.Type(typeID)
		if !ok || projected.Control != want {
			t.Fatalf("type %s control = %q, present=%t", typeID, projected.Control, ok)
		}
	}
	add, ok := projection.Node(AddNodeID)
	if !ok || len(add.DataInputs) != 2 || add.DataInputs[0].Type.Control != nodeauthoring.ControlNumber || !add.DataInputs[0].HasDefault {
		t.Fatalf("add authoring projection = %#v", add)
	}
}

func TestPrimitiveEvaluatorsAreStrictAndUnicodeAware(t *testing.T) {
	builtins, err := Build()
	if err != nil {
		t.Fatal(err)
	}
	evaluate := func(nodeID string, inputs map[string]json.RawMessage) map[string]json.RawMessage {
		t.Helper()
		definition, ok := builtins.Definition(nodeID)
		if !ok {
			t.Fatalf("missing definition %q", nodeID)
		}
		outputs, err := definition.EvaluateInline(context.Background(), inputs, map[string]any{})
		if err != nil {
			t.Fatal(err)
		}
		return outputs
	}

	if got := evaluate(AddNodeID, map[string]json.RawMessage{"a": json.RawMessage(`2.5`), "b": json.RawMessage(`3.25`)}); string(got["result"]) != "5.75" {
		t.Fatalf("add = %s", got["result"])
	}
	if got := evaluate(LengthNodeID, map[string]json.RawMessage{"text": json.RawMessage(`"Yotta节点"`)}); string(got["result"]) != "7" {
		t.Fatalf("length = %s", got["result"])
	}
	if got := evaluate(ContainsNodeID, map[string]json.RawMessage{"text": json.RawMessage(`"Yotta节点"`), "search": json.RawMessage(`"节点"`)}); string(got["result"]) != "true" {
		t.Fatalf("contains = %s", got["result"])
	}

	add, _ := builtins.Definition(AddNodeID)
	if _, err := add.EvaluateInline(context.Background(), map[string]json.RawMessage{"a": json.RawMessage(`"2"`), "b": json.RawMessage(`3`)}, nil); err == nil {
		t.Fatal("add accepted an implicit string-to-number coercion")
	}
	_, err = add.EvaluateInline(context.Background(), map[string]json.RawMessage{
		"a": json.RawMessage(`1.7976931348623157e308`), "b": json.RawMessage(`1.7976931348623157e308`),
	}, nil)
	var failure *InlineFailure
	if !errors.As(err, &failure) || failure.Code != unrepresentableResultCode || failure.Output != "" {
		t.Fatalf("overflow failure = %#v, %v", failure, err)
	}
}
