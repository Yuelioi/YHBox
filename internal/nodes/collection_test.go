package nodes

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodeauthoring"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

func TestCollectionDefinitionsDeclareExactPolymorphicRelationships(t *testing.T) {
	builtins, err := Build()
	if err != nil {
		t.Fatal(err)
	}
	for _, nodeID := range []string{SplitNodeID, JoinNodeID, ListLengthNodeID, ListGetNodeID, ListContainsNodeID, ListAppendNodeID, ListSliceNodeID} {
		definition, ok := builtins.Definition(nodeID)
		if !ok || definition.EvaluateInline == nil {
			t.Fatalf("missing collection definition %q", nodeID)
		}
		machine := definition.Contract.Machine()
		if machine.Execution.Class != nodecontract.ExecutionPureData || len(machine.Ports.ExecInputs)+len(machine.Ports.ExecOutputs)+len(machine.Ports.ErrorOutputs) != 0 {
			t.Fatalf("collection node invented control semantics: %#v", machine.Ports)
		}
	}
	get, _ := builtins.Definition(ListGetNodeID)
	getMachine := get.Contract.Machine()
	if getMachine.Ports.DataInputs[0].Type.Kind != datatype.TypeExpressionList ||
		getMachine.Ports.DataInputs[0].Type.Element.Kind != datatype.TypeExpressionVariable ||
		getMachine.Ports.DataOutputs[0].Type.Kind != datatype.TypeExpressionVariable {
		t.Fatalf("ListGet lost T relationship: %#v", getMachine.Ports)
	}
	if len(getMachine.Errors) != 1 || getMachine.Errors[0].Code != collectionIndexOutOfRangeCode {
		t.Fatalf("ListGet errors = %#v", getMachine.Errors)
	}
	projection, err := nodeauthoring.Project(nodeauthoring.Input{
		Catalog: builtins.Catalog, Types: builtins.Types, Capabilities: builtins.Capabilities,
		Contracts: builtins.Contracts, GeneratorVersion: GeneratorVersion,
	})
	if err != nil {
		t.Fatal(err)
	}
	split, _ := projection.Node(SplitNodeID)
	getProjection, _ := projection.Node(ListGetNodeID)
	if split.DataOutputs[0].Type.Control != nodeauthoring.ControlList || len(split.DataOutputs[0].Type.Representations) != 1 {
		t.Fatalf("Split list authoring = %#v", split.DataOutputs[0].Type)
	}
	if getProjection.DataInputs[0].Type.Control != nodeauthoring.ControlList || len(getProjection.DataInputs[0].Type.Representations) != 0 {
		t.Fatalf("generic list authoring should require a typed edge: %#v", getProjection.DataInputs[0].Type)
	}
}

func TestCollectionEvaluatorsAreStrictAndImmutable(t *testing.T) {
	builtins, err := Build()
	if err != nil {
		t.Fatal(err)
	}
	evaluate := func(nodeID string, inputs map[string]json.RawMessage) (map[string]json.RawMessage, error) {
		t.Helper()
		definition, _ := builtins.Definition(nodeID)
		return definition.EvaluateInline(context.Background(), inputs, nil)
	}
	split, err := evaluate(SplitNodeID, map[string]json.RawMessage{"text": json.RawMessage(`"a,节点"`), "separator": json.RawMessage(`","`)})
	if err != nil || string(split["result"]) != `["a","节点"]` {
		t.Fatalf("split = %s, %v", split["result"], err)
	}
	appended, err := evaluate(ListAppendNodeID, map[string]json.RawMessage{"list": json.RawMessage(`[1,2]`), "item": json.RawMessage(`3`)})
	if err != nil || string(appended["result"]) != `[1,2,3]` {
		t.Fatalf("append = %s, %v", appended["result"], err)
	}
	contains, err := evaluate(ListContainsNodeID, map[string]json.RawMessage{"list": json.RawMessage(`[{"b":2,"a":1}]`), "value": json.RawMessage(`{"a":1,"b":2}`)})
	if err != nil || string(contains["result"]) != `true` {
		t.Fatalf("contains = %s, %v", contains["result"], err)
	}
	_, err = evaluate(ListGetNodeID, map[string]json.RawMessage{"list": json.RawMessage(`[1]`), "index": json.RawMessage(`2`)})
	var failure *InlineFailure
	if !errors.As(err, &failure) || failure.Code != collectionIndexOutOfRangeCode || failure.Output != "" {
		t.Fatalf("ListGet failure = %#v, %v", failure, err)
	}
}
