package nodes

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodeauthoring"
)

func TestStructuredTypesGenerateTypedBreakNodes(t *testing.T) {
	builtins, err := Build()
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		typeDefinition datatype.Definition
		nodeID         string
		fields         []string
	}{
		{builtins.PointType, BreakPointNodeID, []string{"unit", "x", "y"}},
		{builtins.RegionType, BreakRegionNodeID, []string{"height", "unit", "width", "x", "y"}},
		{builtins.TemplateMatchType, BreakTemplateMatchNodeID, []string{"bounds", "center", "score"}},
		{builtins.QRCodeType, BreakQRCodeNodeID, []string{"points", "text"}},
		{builtins.ColorBlobType, BreakColorBlobNodeID, []string{"area", "bounds", "center"}},
		{builtins.FileMetadataType, BreakFileMetadataNodeID, []string{"extension", "is-directory", "media-type", "modified-unix-millis", "name", "path", "size"}},
	} {
		structure := test.typeDefinition.Machine().Structure
		if structure == nil || structure.BreakNodeTypeID != test.nodeID || len(structure.Fields) != len(test.fields) {
			t.Fatalf("structure %s = %#v", test.typeDefinition.TypeRef().TypeID, structure)
		}
		definition, ok := builtins.Definition(test.nodeID)
		if !ok || definition.EvaluateInline == nil {
			t.Fatalf("missing break node %s", test.nodeID)
		}
		machine := definition.Contract.Machine()
		if len(machine.Ports.DataInputs) != 1 || len(machine.Ports.DataOutputs) != len(test.fields) ||
			machine.Ports.DataInputs[0].Type.Ref == nil || *machine.Ports.DataInputs[0].Type.Ref != test.typeDefinition.TypeRef() {
			t.Fatalf("break node ports %s = %#v", test.nodeID, machine.Ports)
		}
		for index, field := range test.fields {
			if machine.Ports.DataOutputs[index].ID != field || !reflect.DeepEqual(machine.Ports.DataOutputs[index].Type, structure.Fields[index].Type) {
				t.Fatalf("break node %s field %d = %#v", test.nodeID, index, machine.Ports.DataOutputs[index])
			}
		}
	}
}

func TestBreakFileMetadataReturnsEveryTypedField(t *testing.T) {
	builtins, err := Build()
	if err != nil {
		t.Fatal(err)
	}
	definition, _ := builtins.Definition(BreakFileMetadataNodeID)
	input := json.RawMessage(`{"path":"a.txt","name":"a.txt","extension":"txt","mediaType":"text/plain","size":12,"modifiedUnixMillis":42,"isDirectory":false}`)
	outputs, err := definition.EvaluateInline(context.Background(), map[string]json.RawMessage{"value": input}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(outputs["name"]) != `"a.txt"` || string(outputs["size"]) != `12` || string(outputs["is-directory"]) != `false` {
		t.Fatalf("metadata fields = %#v", outputs)
	}
}

func TestStructureContractIsProjectedWithoutSchemaGuessing(t *testing.T) {
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
	point, ok := projection.Type(PointTypeID)
	if !ok || point.Structure == nil || point.Structure.BreakNodeTypeID != BreakPointNodeID || len(point.Structure.Fields) != 3 {
		t.Fatalf("point structure projection = %#v", point.Structure)
	}
	breakPoint, ok := projection.Node(BreakPointNodeID)
	if !ok || len(breakPoint.DataInputs) != 1 || len(breakPoint.DataOutputs) != 3 || breakPoint.DataOutputs[1].ID != "x" {
		t.Fatalf("break point projection = %#v", breakPoint)
	}
}
