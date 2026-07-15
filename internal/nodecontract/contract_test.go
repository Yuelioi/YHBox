package nodecontract

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/datatype"
)

func TestSealConcatContractHasOnlyDataPortsAndStableIdentity(t *testing.T) {
	draft := concatContractDraftForTest()
	contract, err := Seal(draft)
	if err != nil {
		t.Fatal(err)
	}
	const want = artifact.Digest("sha256:f9d3ba7e507746f26df951a2d8f3b99b330ad978486f298c4654fddccf13a573")
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
		document.Semantic.Ports.ErrorOutputs == nil {
		t.Fatal("empty control port arrays were encoded as null or omitted")
	}
	if len(document.Semantic.Ports.ExecInputs)+len(document.Semantic.Ports.ExecOutputs)+
		len(document.Semantic.Ports.ErrorOutputs) != 0 {
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

func TestSemanticArtifactExcludesAuthoringAndCanBeReopened(t *testing.T) {
	left := concatContractDraftForTest()
	left.Authoring = Authoring{TitleKey: "node.concat.title", Tags: []string{"text"}}
	right := concatContractDraftForTest()
	right.Authoring = Authoring{TitleKey: "node.concat.renamed", Tags: []string{"utility"}}

	leftContract, err := Seal(left)
	if err != nil {
		t.Fatal(err)
	}
	rightContract, err := Seal(right)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(leftContract.SemanticBytes(), rightContract.SemanticBytes()) {
		t.Fatal("authoring changed semantic artifact")
	}
	reopened, err := OpenSemantic(leftContract.NodeRef(), leftContract.SemanticBytes())
	if err != nil {
		t.Fatal(err)
	}
	if reopened.NodeRef() != leftContract.NodeRef() || !bytes.Equal(reopened.SemanticBytes(), leftContract.SemanticBytes()) {
		t.Fatal("semantic artifact round trip changed identity")
	}
	if reopened.Authoring().TitleKey != "" || len(reopened.Authoring().Tags) != 0 {
		t.Fatalf("machine-only contract leaked authoring: %#v", reopened.Authoring())
	}
}

func TestSealRejectsPureDataControlPortsAndInvalidTypeExpressions(t *testing.T) {
	draft := concatContractDraftForTest()
	draft.Ports.ExecOutputs = []SignalPort{{ID: "out"}}
	if _, err := Seal(draft); err == nil {
		t.Fatal("accepted pure-data contract with exec output")
	}

	draft = concatContractDraftForTest()
	draft.StatusEvents = []StatusEventSpec{{Code: "node.progress", Category: StatusProgress}}
	if _, err := Seal(draft); err == nil {
		t.Fatal("accepted pure-data contract with a status event")
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
	draft.CapabilityRequirements = []capability.Requirement{{
		ID: "network", Capability: capability.Ref{
			CapabilityID:   "https://schemas.yotta.dev/capabilities/network/v1",
			SemanticDigest: artifact.Digest("sha256:" + strings.Repeat("2", 64)),
		}, Operations: []string{"read"}, TargetSlot: "network", Scope: json.RawMessage(`{}`),
	}}
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

func TestSealBindsResourcePortsToExactCapabilityOperations(t *testing.T) {
	draft := concatContractDraftForTest()
	draft.Execution.Class = ExecutionEffect
	draft.Execution.Cache = CacheNone
	draft.Execution.Effects = []EffectID{"https://schemas.yotta.dev/effects/blob/read/v1"}
	draft.CapabilityRequirements = []capability.Requirement{{
		ID: "blob", Capability: capability.Ref{
			CapabilityID:   "https://schemas.yotta.dev/capabilities/blob/read/v1",
			SemanticDigest: artifact.Digest("sha256:" + strings.Repeat("2", 64)),
		}, Operations: []string{"read-range"}, TargetSlot: "workspace", Scope: json.RawMessage(`{}`),
	}}
	draft.Ports.DataOutputs[0].ResourceLease = &ResourceLeaseBinding{RequirementID: "blob", Operations: []string{"read-range"}}
	contract, err := Seal(draft)
	if err != nil {
		t.Fatal(err)
	}
	lease := contract.Machine().Ports.DataOutputs[0].ResourceLease
	if lease == nil || lease.RequirementID != "blob" || len(lease.Operations) != 1 {
		t.Fatalf("resource lease = %#v", lease)
	}
	draft.Ports.DataOutputs[0].ResourceLease.Operations = []string{"delete"}
	if _, err := Seal(draft); err == nil {
		t.Fatal("accepted a resource port that widened capability operations")
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
			DataOutputs:  []DataOutputPort{{ID: "result", Type: stringType}},
			ExecInputs:   []SignalPort{},
			ExecOutputs:  []SignalPort{},
			ErrorOutputs: []SignalPort{},
		},
		Execution: ExecutionSpec{
			Class: ExecutionPureData, Determinism: Deterministic,
			Evaluation: EvaluationPull, Cache: CachePerRun, Retry: RetryNever,
			Cancellation: CancellationCooperative, Timeout: TimeoutNone,
			Effects: []EffectID{},
		},
		CapabilityRequirements: []capability.Requirement{},
		Errors:                 []ErrorSpec{},
		StatusEvents:           []StatusEventSpec{},
		ImplementationABI:      []ABIRequirement{{Kind: ABIBuiltin, Version: "v1"}},
		Authoring:              Authoring{TitleKey: "node.text.concat.title", Category: "text"},
	}
}
