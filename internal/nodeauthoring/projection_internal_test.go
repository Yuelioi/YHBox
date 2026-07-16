package nodeauthoring

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

func TestConfigProjectionResolvesLocalDefsReferences(t *testing.T) {
	const root = "https://schemas.yotta.dev/test/config"
	fields, err := projectConfigFields([]datatype.SchemaResource{{ID: root, Schema: json.RawMessage(`{
		"$id":"https://schemas.yotta.dev/test/config",
		"type":"object",
		"$defs":{"mime":{"type":"string","title":"Media type","minLength":3}},
		"properties":{"mediaType":{"$ref":"#/$defs/mime"}},
		"required":["mediaType"]
	}`)}}, root)
	if err != nil {
		t.Fatal(err)
	}
	if len(fields) != 1 || fields[0].Control != ControlText || fields[0].Title != "Media type" ||
		fields[0].Constraints.MinLength == nil || *fields[0].Constraints.MinLength != 3 {
		t.Fatalf("fields = %#v", fields)
	}
}

func TestConfigProjectionRejectsWriteOnlySourceFields(t *testing.T) {
	const root = "https://schemas.yotta.dev/test/config"
	_, err := projectConfigFields([]datatype.SchemaResource{{ID: root, Schema: json.RawMessage(`{
		"$id":"https://schemas.yotta.dev/test/config",
		"type":"object",
		"properties":{"apiKey":{"type":"string","writeOnly":true}}
	}`)}}, root)
	if err == nil || !strings.Contains(err.Error(), "capability credential slot") {
		t.Fatalf("writeOnly error = %v", err)
	}
}

func TestConfigProjectionUsesExplicitCodeControlOnlyForStrings(t *testing.T) {
	const root = "https://schemas.yotta.dev/test/config"
	fields, err := projectConfigFields([]datatype.SchemaResource{{ID: root, Schema: json.RawMessage(`{
		"$id":"https://schemas.yotta.dev/test/config",
		"type":"object",
		"properties":{"source":{"type":"string","x-yotta-control":"code"}}
	}`)}}, root)
	if err != nil {
		t.Fatal(err)
	}
	if len(fields) != 1 || fields[0].Control != ControlCode {
		t.Fatalf("fields = %#v", fields)
	}
	_, err = projectConfigFields([]datatype.SchemaResource{{ID: root, Schema: json.RawMessage(`{
		"$id":"https://schemas.yotta.dev/test/config",
		"type":"object",
		"properties":{"source":{"type":"number","x-yotta-control":"code"}}
	}`)}}, root)
	if err == nil || !strings.Contains(err.Error(), "unsupported generated control") {
		t.Fatalf("invalid code control error = %v", err)
	}
}

func TestConfigProjectionBoundsReferenceDAGExpansion(t *testing.T) {
	const root = "https://schemas.yotta.dev/test/config"
	defs := []string{`"leaf":{"type":"string"}`}
	previous := "leaf"
	for level := 0; level < 13; level++ {
		name := fmt.Sprintf("level%d", level)
		defs = append(defs, fmt.Sprintf(`%q:{"type":"object","properties":{"a":{"$ref":"#/$defs/%s"},"b":{"$ref":"#/$defs/%s"}}}`, name, previous, previous))
		previous = name
	}
	raw := fmt.Sprintf(`{"$id":%q,"type":"object","$defs":{%s},"properties":{"root":{"$ref":"#/$defs/%s"}}}`, root, strings.Join(defs, ","), previous)
	_, err := projectConfigFields([]datatype.SchemaResource{{ID: root, Schema: json.RawMessage(raw)}}, root)
	if err == nil || (!strings.Contains(err.Error(), "field expansion budget") && !strings.Contains(err.Error(), "reference expansion budget")) {
		t.Fatalf("expansion error = %v", err)
	}
}

func TestNodeProjectionCarriesInputDefaultAnnotation(t *testing.T) {
	const typeID = "https://schemas.yotta.dev/types/test/string/v1"
	typeRef := datatype.TypeRef{TypeID: typeID, SemanticDigest: artifact.Digest("sha256:1111111111111111111111111111111111111111111111111111111111111111")}
	defaultValue := json.RawMessage(`"suffix"`)
	contract, err := nodecontract.Seal(nodecontract.Draft{Version: "1.0.0",
		NodeTypeID:       "https://schemas.yotta.dev/nodes/test/default",
		ConfigSchemaRoot: "https://schemas.yotta.dev/nodes/test/default/config",
		ConfigSchemaBundle: []datatype.SchemaResource{{ID: "https://schemas.yotta.dev/nodes/test/default/config", Schema: json.RawMessage(`{
			"$id":"https://schemas.yotta.dev/nodes/test/default/config",
			"$schema":"https://json-schema.org/draft/2020-12/schema",
			"type":"object"
		}`)}},
		Ports: nodecontract.PortSet{DataInputs: []nodecontract.DataInputPort{{ID: "value", Type: datatype.RefExpression(typeRef), Default: &defaultValue}}},
		Execution: nodecontract.ExecutionSpec{
			Class: nodecontract.ExecutionPureData, Determinism: nodecontract.Deterministic,
			Evaluation: nodecontract.EvaluationPull, Cache: nodecontract.CachePerRun,
			Retry: nodecontract.RetryNever, Cancellation: nodecontract.CancellationCooperative,
			Timeout: nodecontract.TimeoutNone,
		},
		Instruction:       nodecontract.Invoke(),
		ImplementationABI: []nodecontract.ABIRequirement{{Kind: nodecontract.ABIBuiltin, Version: "v1"}},
		Authoring:         nodecontract.Authoring{TitleKey: "node.test.default.title"},
	})
	if err != nil {
		t.Fatal(err)
	}
	projected, err := projectNode(contract, map[string]TypeProjection{
		typeID: {TypeRef: typeRef, Representations: []datatype.RepresentationSpec{{Kind: datatype.RepresentationInlineJSON, Codec: datatype.CodecJCSV1}}, Lifecycle: LifecycleDurable},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(projected.DataInputs) != 1 || !projected.DataInputs[0].HasDefault || string(projected.DataInputs[0].Default) != `"suffix"` {
		t.Fatalf("projected default = %#v", projected.DataInputs)
	}
}
