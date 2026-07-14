package nodecatalog

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

func TestSnapshot31IsDeterministicMachineOnlyAndRoundTrips(t *testing.T) {
	stringType := stringDefinition(t, "type.string.title")
	concat := concatContract(t, stringType.TypeRef(), "node.concat.title")
	lock := builtinLock(t)

	left, err := Seal([]datatype.Definition{stringType}, []Binding{{Contract: concat, Implementation: lock}}, "v1")
	if err != nil {
		t.Fatal(err)
	}
	renamedType := stringDefinition(t, "type.string.renamed")
	renamedConcat := concatContract(t, renamedType.TypeRef(), "node.concat.renamed")
	right, err := Seal([]datatype.Definition{renamedType}, []Binding{{Contract: renamedConcat, Implementation: lock}}, "v1")
	if err != nil {
		t.Fatal(err)
	}
	if left.Hash() != right.Hash() || !bytes.Equal(left.Bytes(), right.Bytes()) {
		t.Fatal("presentation metadata changed machine catalog")
	}

	entry, ok := left.Lookup(concat.NodeRef().NodeTypeID)
	if !ok {
		t.Fatal("missing concat binding")
	}
	ports := entry.Contract.Machine().Ports
	if len(ports.DataInputs) != 2 || len(ports.DataOutputs) != 1 ||
		len(ports.ExecInputs)+len(ports.ExecOutputs)+len(ports.ErrorOutputs)+len(ports.StatusOutputs) != 0 {
		t.Fatalf("concat catalog ports = %#v", ports)
	}
	reopened, err := Open(left.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if reopened.Hash() != left.Hash() || !bytes.Equal(reopened.Bytes(), left.Bytes()) {
		t.Fatal("catalog round trip changed identity")
	}

	var document map[string]any
	if err := json.Unmarshal(left.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	if document["format"] != Format || document["version"] != Version {
		t.Fatalf("catalog epoch = %#v", document)
	}
}

func TestSnapshotRejectsMissingTypesAndABIMismatch(t *testing.T) {
	stringType := stringDefinition(t, "type.string.title")
	concat := concatContract(t, stringType.TypeRef(), "node.concat.title")
	if _, err := Seal(nil, []Binding{{Contract: concat, Implementation: builtinLock(t)}}, "v1"); err == nil {
		t.Fatal("accepted binding whose port type is absent from the catalog")
	}
	lock := builtinLock(t)
	lock.ABI = nodecontract.ABIRequirement{Kind: nodecontract.ABIWIT, Version: "v1"}
	if _, err := Seal([]datatype.Definition{stringType}, []Binding{{Contract: concat, Implementation: lock}}, "v1"); err == nil {
		t.Fatal("accepted implementation ABI not allowed by the node contract")
	}
}

func stringDefinition(t *testing.T, title string) datatype.Definition {
	t.Helper()
	const schemaID = "https://schemas.yotta.dev/types/core/string/v1/schema"
	definition, err := datatype.SealDefinition(datatype.DefinitionDraft{
		TypeID: "https://schemas.yotta.dev/types/core/string/v1", SchemaDialect: datatype.JSONSchemaDialect,
		SchemaBundle: []datatype.SchemaResource{{ID: schemaID, Schema: json.RawMessage(`{
			"$id":"https://schemas.yotta.dev/types/core/string/v1/schema",
			"$schema":"https://json-schema.org/draft/2020-12/schema",
			"type":"string"
		}`)}},
		Representations: []datatype.RepresentationSpec{{Kind: datatype.RepresentationInlineJSON, Codec: datatype.CodecJCSV1}},
		Authoring:       datatype.Authoring{TitleKey: title},
	})
	if err != nil {
		t.Fatal(err)
	}
	return definition
}

func concatContract(t *testing.T, stringRef datatype.TypeRef, title string) nodecontract.Contract {
	t.Helper()
	const schemaID = "https://schemas.yotta.dev/nodes/text/concat/v1/config"
	stringType := datatype.RefExpression(stringRef)
	contract, err := nodecontract.Seal(nodecontract.Draft{
		NodeTypeID: "https://schemas.yotta.dev/nodes/text/concat/v1", ConfigSchemaRoot: schemaID,
		ConfigSchemaBundle: []datatype.SchemaResource{{ID: schemaID, Schema: json.RawMessage(`{
			"$id":"https://schemas.yotta.dev/nodes/text/concat/v1/config",
			"$schema":"https://json-schema.org/draft/2020-12/schema",
			"type":"object","additionalProperties":false
		}`)}},
		Ports: nodecontract.PortSet{
			DataInputs:  []nodecontract.DataInputPort{{ID: "a", Type: stringType, Required: true}, {ID: "b", Type: stringType, Required: true}},
			DataOutputs: []nodecontract.DataOutputPort{{ID: "result", Type: stringType}},
			ExecInputs:  []nodecontract.SignalPort{}, ExecOutputs: []nodecontract.SignalPort{},
			ErrorOutputs: []nodecontract.SignalPort{}, StatusOutputs: []nodecontract.SignalPort{},
		},
		Execution: nodecontract.ExecutionSpec{
			Class: nodecontract.ExecutionPureData, Effects: []nodecontract.EffectID{}, Determinism: nodecontract.Deterministic,
			Evaluation: nodecontract.EvaluationPull, Cache: nodecontract.CachePerRun, Retry: nodecontract.RetryNever,
			Cancellation: nodecontract.CancellationCooperative, Timeout: nodecontract.TimeoutNone,
		},
		Capabilities: []nodecontract.CapabilityID{}, Errors: []nodecontract.ErrorSpec{},
		ImplementationABI: []nodecontract.ABIRequirement{{Kind: nodecontract.ABIBuiltin, Version: "v1"}},
		Authoring:         nodecontract.Authoring{TitleKey: title, Tags: []string{"text"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return contract
}

func builtinLock(t *testing.T) ImplementationLock {
	t.Helper()
	digest, err := artifact.Sum("test/implementation/v1", []byte("concat"))
	if err != nil {
		t.Fatal(err)
	}
	return ImplementationLock{
		PackageID: "https://schemas.yotta.dev/packages/builtin/v1", ArtifactDigest: digest,
		ABI: nodecontract.ABIRequirement{Kind: nodecontract.ABIBuiltin, Version: "v1"}, Entrypoint: "text.concat",
	}
}

func TestOpenRejectsTamperedCatalog(t *testing.T) {
	typeDefinition := stringDefinition(t, "type.string.title")
	contract := concatContract(t, typeDefinition.TypeRef(), "node.concat.title")
	snapshot, err := Seal([]datatype.Definition{typeDefinition}, []Binding{{Contract: contract, Implementation: builtinLock(t)}}, "v1")
	if err != nil {
		t.Fatal(err)
	}
	tampered := bytes.Replace(snapshot.Bytes(), []byte(contract.NodeRef().SemanticDigest), []byte("sha256:"+strings.Repeat("0", 64)), 1)
	if bytes.Equal(tampered, snapshot.Bytes()) {
		t.Fatal("test did not tamper catalog")
	}
	if _, err := Open(tampered); err == nil {
		t.Fatal("accepted tampered catalog")
	}
}
