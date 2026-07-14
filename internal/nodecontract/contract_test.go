package nodecontract

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/datatype"
)

func TestSealConcatContractHasOnlyDataPortsAndStableIdentity(t *testing.T) {
	draft := concatContractDraftForTest()
	contract, err := Seal(draft)
	if err != nil {
		t.Fatal(err)
	}
	const want = artifact.Digest("sha256:190cd3358854b3b347b9cddc89065ae9a2f22fa6d1ccd6bc89241e880c353e19")
	if got := contract.NodeRef().SemanticDigest; got != want {
		t.Fatalf("semantic digest = %q, want %q", got, want)
	}

	var document struct {
		Semantic struct {
			Ports PortSet `json:"ports"`
		} `json:"semantic"`
	}
	if err := json.Unmarshal(contract.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	if len(document.Semantic.Ports.DataInputs) != 2 || len(document.Semantic.Ports.DataOutputs) != 1 {
		t.Fatalf("concat data ports = %#v", document.Semantic.Ports)
	}
	if document.Semantic.Ports.ExecInputs == nil || document.Semantic.Ports.ExecOutputs == nil ||
		document.Semantic.Ports.ErrorOutputs == nil || document.Semantic.Ports.StatusOutputs == nil {
		t.Fatal("empty control port arrays were encoded as null or omitted")
	}
	if len(document.Semantic.Ports.ExecInputs)+len(document.Semantic.Ports.ExecOutputs)+
		len(document.Semantic.Ports.ErrorOutputs)+len(document.Semantic.Ports.StatusOutputs) != 0 {
		t.Fatalf("concat gained control ports: %#v", document.Semantic.Ports)
	}

	draft.Authoring.TitleKey = "node.text.concat.renamed"
	renamed, err := Seal(draft)
	if err != nil {
		t.Fatal(err)
	}
	if renamed.NodeRef() != contract.NodeRef() {
		t.Fatal("authoring annotation changed node semantic identity")
	}
	reopened, err := Open(contract.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if reopened.NodeRef() != contract.NodeRef() {
		t.Fatalf("reopened ref = %#v, want %#v", reopened.NodeRef(), contract.NodeRef())
	}
}

func TestSealRejectsPureDataControlPortsAndInvalidTypeExpressions(t *testing.T) {
	draft := concatContractDraftForTest()
	draft.Ports.ExecOutputs = []SignalPort{{ID: "out"}}
	if _, err := Seal(draft); err == nil {
		t.Fatal("accepted pure-data contract with exec output")
	}

	draft = concatContractDraftForTest()
	draft.Ports.DataInputs[0].Type = datatype.TypeExpression{Kind: datatype.TypeExpressionUnion}
	if _, err := Seal(draft); err == nil {
		t.Fatal("accepted invalid data port type expression")
	}
}

func TestSealEnforcesOfflineConfigSchemaClosure(t *testing.T) {
	draft := concatContractDraftForTest()
	draft.ConfigSchemaBundle[0].Schema = json.RawMessage(`{
		"$id":"https://schemas.yotta.dev/nodes/text/concat/v1/config",
		"$schema":"https://json-schema.org/draft/2020-12/schema",
		"$ref":"https://remote.example/missing"
	}`)
	if _, err := Seal(draft); err == nil {
		t.Fatal("accepted unbundled config schema reference")
	}

	draft = concatContractDraftForTest()
	draft.ConfigSchemaBundle[0].Schema = json.RawMessage(`{
		"$id":"https://schemas.yotta.dev/nodes/text/concat/v1/config",
		"$schema":"https://json-schema.org/draft/2020-12/schema",
		"$defs":{
			"container":{"$id":"embedded/","$ref":"value"},
			"value":{"$id":"embedded/value","type":"string"}
		},
		"const":{"$id":"not-a-schema-id","$ref":"literal-instance-data"}
	}`)
	if _, err := Seal(draft); err != nil {
		t.Fatalf("rejected bundled nested reference or interpreted const as schema: %v", err)
	}
}

func TestOpenRejectsUnknownAndOverdeepContracts(t *testing.T) {
	contract, err := Seal(concatContractDraftForTest())
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(contract.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	document["unexpected"] = true
	raw, err := artifact.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Open(raw); err == nil {
		t.Fatal("accepted unknown node contract field")
	}

	overdeep := []byte(strings.Repeat("[", MaxContractDepth+1) + "0" + strings.Repeat("]", MaxContractDepth+1))
	if _, err := Open(overdeep); err == nil {
		t.Fatal("accepted node contract over depth budget")
	}
}

func TestSealEnforcesExecutionCapabilityInvariants(t *testing.T) {
	draft := concatContractDraftForTest()
	draft.Capabilities = []CapabilityID{"https://schemas.yotta.dev/capabilities/network/v1"}
	if _, err := Seal(draft); err == nil {
		t.Fatal("accepted pure-data contract with a capability requirement")
	}

	draft = concatContractDraftForTest()
	draft.Execution.Class = ExecutionEffect
	draft.Execution.Cache = CacheNone
	draft.Execution.Evaluation = EvaluationPush
	draft.Ports.ExecInputs = []SignalPort{{ID: "in"}}
	draft.Ports.ExecOutputs = []SignalPort{{ID: "done"}}
	if _, err := Seal(draft); err == nil {
		t.Fatal("accepted effect contract without an effect or capability")
	}
}

func concatContractDraftForTest() Draft {
	stringRef := datatype.TypeRef{
		TypeID:         "https://schemas.yotta.dev/types/core/string/v1",
		SemanticDigest: artifact.Digest("sha256:" + strings.Repeat("1", 64)),
	}
	stringType := datatype.RefExpression(stringRef)
	const configID = "https://schemas.yotta.dev/nodes/text/concat/v1/config"
	return Draft{
		NodeTypeID:       "https://schemas.yotta.dev/nodes/text/concat/v1",
		ConfigSchemaRoot: configID,
		ConfigSchemaBundle: []datatype.SchemaResource{{
			ID: configID,
			Schema: json.RawMessage(`{
				"$id":"https://schemas.yotta.dev/nodes/text/concat/v1/config",
				"$schema":"https://json-schema.org/draft/2020-12/schema",
				"type":"object",
				"additionalProperties":false
			}`),
		}},
		Ports: PortSet{
			DataInputs: []DataInputPort{
				{ID: "a", Type: stringType, Required: true},
				{ID: "b", Type: stringType, Required: true},
			},
			DataOutputs:   []DataOutputPort{{ID: "result", Type: stringType}},
			ExecInputs:    []SignalPort{},
			ExecOutputs:   []SignalPort{},
			ErrorOutputs:  []SignalPort{},
			StatusOutputs: []SignalPort{},
		},
		Execution: ExecutionSpec{
			Class: ExecutionPureData, Determinism: Deterministic,
			Evaluation: EvaluationPull, Cache: CachePerRun, Retry: RetryNever,
			Cancellation: CancellationCooperative, Timeout: TimeoutNone,
			Effects: []EffectID{},
		},
		Capabilities:      []CapabilityID{},
		Errors:            []ErrorSpec{},
		ImplementationABI: []ABIRequirement{{Kind: ABIBuiltin, Version: "v1"}},
		Authoring:         Authoring{TitleKey: "node.text.concat.title", Category: "text"},
	}
}
