package nodepackage

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

const testNamespace = "https://packages.example.test/acme"

func TestManifestIsDeterministicContentAddressedAndReopens(t *testing.T) {
	typeDefinition := testType(t)
	capabilityDefinition := testCapability(t)
	contract := testNode(t, typeDefinition.TypeRef(), capability.Ref{}, nodecontract.ABIWIT)
	node := NodeDraft{Contract: contract, Implementation: testImplementation(t, nodecontract.ABIWIT)}
	documentation := testPayload(t, "docs/node.md", "text/markdown", "documentation")

	left, err := Seal(Draft{
		PublisherNamespace: testNamespace,
		PackageID:          testNamespace + "/packages/transform/v1",
		PackageVersion:     "1.2.3-rc.1+build.7",
		HostAPI:            HostAPIRange{Min: "3.1", MaxExclusive: "4.0"},
		Types:              []datatype.Definition{typeDefinition},
		Capabilities:       []capability.Definition{capabilityDefinition},
		Nodes:              []NodeDraft{node},
		Documentation:      []Payload{documentation},
	})
	if err != nil {
		t.Fatal(err)
	}
	right, err := Seal(Draft{
		PublisherNamespace: testNamespace,
		PackageID:          testNamespace + "/packages/transform/v1",
		PackageVersion:     "1.2.3-rc.1+build.7",
		HostAPI:            HostAPIRange{Min: "3.1", MaxExclusive: "4.0"},
		Nodes:              []NodeDraft{node},
		Capabilities:       []capability.Definition{capabilityDefinition},
		Types:              []datatype.Definition{typeDefinition},
		Documentation:      []Payload{documentation},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !left.Valid() || left.Digest() != right.Digest() || !bytes.Equal(left.Bytes(), right.Bytes()) {
		t.Fatal("equivalent package drafts did not produce one content identity")
	}
	reopened, err := Open(left.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if reopened.Digest() != left.Digest() || reopened.PackageID() != left.PackageID() ||
		reopened.PackageVersion() != "1.2.3-rc.1+build.7" || reopened.HostAPI() != left.HostAPI() {
		t.Fatal("reopened manifest changed package identity")
	}
	if !reopened.SupportsHostAPI("3.1") || !reopened.SupportsHostAPI("3.99") || reopened.SupportsHostAPI("4.0") {
		t.Fatal("half-open host API range was not enforced")
	}
	if len(reopened.Types()) != 1 || len(reopened.Capabilities()) != 1 ||
		len(reopened.Nodes()) != 1 || len(reopened.Documentation()) != 1 {
		t.Fatal("reopened manifest lost contributions")
	}
	if got := reopened.Nodes()[0].Implementation.Platforms; strings.Join(got.OperatingSystems, ",") != "linux,windows" ||
		strings.Join(got.Architectures, ",") != "amd64,arm64" {
		t.Fatalf("platform sets were not normalized: %#v", got)
	}
	nodes := reopened.Nodes()
	nodes[0].Implementation.Platforms.OperatingSystems[0] = "tampered"
	if reopened.Nodes()[0].Implementation.Platforms.OperatingSystems[0] != "linux" {
		t.Fatal("node projection mutated immutable manifest state")
	}
}

func TestManifestRejectsNamespaceVersionHostRangeAndGoABI(t *testing.T) {
	base := testDraft(t, nodecontract.ABIWIT)
	cases := []struct {
		name   string
		mutate func(*Draft)
	}{
		{"namespace", func(d *Draft) { d.Nodes[0].Contract = testForeignNode(t, d.Types[0].TypeRef()) }},
		{"version", func(d *Draft) { d.PackageVersion = "01.2.3" }},
		{"prerelease version", func(d *Draft) { d.PackageVersion = "1.2.3-01" }},
		{"host range", func(d *Draft) { d.HostAPI.MaxExclusive = "3.1" }},
		{"builtin ABI", func(d *Draft) {
			d.Nodes[0].Contract = testNode(t, d.Types[0].TypeRef(), capability.Ref{}, nodecontract.ABIBuiltin)
			d.Nodes[0].Implementation.ABI = nodecontract.ABIRequirement{Kind: nodecontract.ABIBuiltin, Version: "v1"}
		}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			draft := base
			draft.Types = append([]datatype.Definition(nil), base.Types...)
			draft.Nodes = append([]NodeDraft(nil), base.Nodes...)
			testCase.mutate(&draft)
			if _, err := Seal(draft); err == nil {
				t.Fatal("accepted invalid package manifest")
			}
		})
	}
}

func TestManifestRejectsTraversalConflictsAndTamperedSemantics(t *testing.T) {
	draft := testDraft(t, nodecontract.ABIProcess)
	draft.Nodes[0].Implementation.Payload.Path = "../escape.exe"
	if _, err := Seal(draft); err == nil {
		t.Fatal("accepted path traversal")
	}
	draft = testDraft(t, nodecontract.ABIProcess)
	draft.Nodes[0].Implementation.Payload.Path = "bin/CON.exe"
	if _, err := Seal(draft); err == nil {
		t.Fatal("accepted a Windows reserved payload path")
	}
	draft = testDraft(t, nodecontract.ABIProcess)
	draft.Nodes[0].Implementation.Payload.Path = strings.ToUpper(ArchiveManifestPath)
	if _, err := Seal(draft); err == nil {
		t.Fatal("accepted the reserved archive manifest path as a payload")
	}
	draft = testDraft(t, nodecontract.ABIProcess)
	draft.Documentation = []Payload{
		testPayload(t, "docs/Guide.md", "text/markdown", "upper"),
		testPayload(t, "docs/guide.md", "text/markdown", "lower"),
	}
	if _, err := Seal(draft); err == nil {
		t.Fatal("accepted case-folding payload path collision")
	}

	draft = testDraft(t, nodecontract.ABIProcess)
	draft.Documentation = []Payload{draft.Nodes[0].Implementation.Payload}
	if _, err := Seal(draft); err == nil {
		t.Fatal("accepted executable payload as documentation")
	}

	manifest, err := Seal(testDraft(t, nodecontract.ABIProcess))
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(manifest.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	nodes := document["nodes"].([]any)
	node := nodes[0].(map[string]any)
	ref := node["nodeRef"].(map[string]any)
	ref["semanticDigest"] = "sha256:" + strings.Repeat("0", 64)
	tampered, err := artifact.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Open(tampered); err == nil {
		t.Fatal("accepted node semantic digest tampering")
	}

	pretty, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Open(pretty); err == nil {
		t.Fatal("accepted non-canonical manifest")
	}
}

func testDraft(t *testing.T, kind nodecontract.ABIKind) Draft {
	t.Helper()
	typeDefinition := testType(t)
	contract := testNode(t, typeDefinition.TypeRef(), capability.Ref{}, kind)
	return Draft{
		PublisherNamespace: testNamespace,
		PackageID:          testNamespace + "/packages/transform/v1",
		PackageVersion:     "1.0.0",
		HostAPI:            HostAPIRange{Min: "3.1", MaxExclusive: "4.0"},
		Types:              []datatype.Definition{typeDefinition},
		Nodes: []NodeDraft{{
			Contract: contract, Implementation: testImplementation(t, kind),
		}},
	}
}

func testType(t *testing.T) datatype.Definition {
	t.Helper()
	const id = testNamespace + "/types/word/v1"
	definition, err := datatype.SealDefinition(datatype.DefinitionDraft{
		TypeID: id, SchemaDialect: datatype.JSONSchemaDialect, SchemaRoot: id + "/schema",
		SchemaBundle: []datatype.SchemaResource{{ID: id + "/schema", Schema: json.RawMessage(
			`{"$id":"https://packages.example.test/acme/types/word/v1/schema","$schema":"https://json-schema.org/draft/2020-12/schema","type":"string"}`,
		)}},
		Representations: []datatype.RepresentationSpec{{Kind: datatype.RepresentationInlineJSON, Codec: datatype.CodecJCSV1}},
		Authoring:       datatype.Authoring{TitleKey: "acme.word.title"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return definition
}

func testCapability(t *testing.T) capability.Definition {
	t.Helper()
	const id = testNamespace + "/capabilities/dictionary/read/v1"
	definition, err := capability.SealDefinition(capability.DefinitionDraft{
		CapabilityID: id, Operations: []string{"read"}, TargetKinds: []string{"dictionary"},
		ScopeSchemaRoot: id + "/scope",
		ScopeSchemaBundle: []datatype.SchemaResource{{ID: id + "/scope", Schema: json.RawMessage(
			`{"$id":"https://packages.example.test/acme/capabilities/dictionary/read/v1/scope","$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","additionalProperties":false}`,
		)}},
		Credential: capability.CredentialNone, Risk: capability.RiskLow, Consent: capability.ConsentNone,
		ProviderABI: "https://schemas.yotta.dev/provider-abi/resource/v1",
	})
	if err != nil {
		t.Fatal(err)
	}
	return definition
}

func testNode(t *testing.T, typeRef datatype.TypeRef, capabilityRef capability.Ref, kind nodecontract.ABIKind) nodecontract.Contract {
	t.Helper()
	const id = testNamespace + "/nodes/uppercase/v1"
	return testNodeAtID(t, id, typeRef, capabilityRef, kind)
}

func testNodeAtID(t *testing.T, id string, typeRef datatype.TypeRef, capabilityRef capability.Ref, kind nodecontract.ABIKind) nodecontract.Contract {
	t.Helper()
	requirements := []capability.Requirement{}
	if capabilityRef.CapabilityID != "" {
		requirements = append(requirements, capability.Requirement{
			ID: "dictionary", Capability: capabilityRef, Operations: []string{"read"},
			TargetSlot: "dictionary", Scope: json.RawMessage(`{}`),
		})
	}
	contract, err := nodecontract.Seal(nodecontract.Draft{
		NodeTypeID: id, ConfigSchemaRoot: id + "/config",
		ConfigSchemaBundle: []datatype.SchemaResource{{ID: id + "/config", Schema: json.RawMessage(fmt.Sprintf(
			`{"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","additionalProperties":false}`,
			id+"/config",
		))}},
		Ports: nodecontract.PortSet{
			DataInputs:  []nodecontract.DataInputPort{{ID: "value", Type: datatype.RefExpression(typeRef), Required: true}},
			DataOutputs: []nodecontract.DataOutputPort{{ID: "result", Type: datatype.RefExpression(typeRef)}},
			ExecInputs:  []nodecontract.SignalPort{}, ExecOutputs: []nodecontract.SignalPort{}, ErrorOutputs: []nodecontract.SignalPort{},
		},
		Execution: nodecontract.ExecutionSpec{
			Class: nodecontract.ExecutionPureData, Effects: []nodecontract.EffectID{}, Determinism: nodecontract.Deterministic,
			Evaluation: nodecontract.EvaluationPull, Cache: nodecontract.CachePerRun, Retry: nodecontract.RetryNever,
			Cancellation: nodecontract.CancellationCooperative, Timeout: nodecontract.TimeoutNone,
		},
		Instruction: nodecontract.Invoke(), CapabilityRequirements: requirements,
		Errors: []nodecontract.ErrorSpec{}, StatusEvents: []nodecontract.StatusEventSpec{},
		ImplementationABI: []nodecontract.ABIRequirement{{Kind: kind, Version: "v1"}},
		Authoring:         nodecontract.Authoring{TitleKey: "acme.uppercase.title", Tags: []string{"text"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return contract
}

func testForeignNode(t *testing.T, typeRef datatype.TypeRef) nodecontract.Contract {
	t.Helper()
	return testNodeAtID(t, "https://foreign.example.test/nodes/uppercase/v1", typeRef, capability.Ref{}, nodecontract.ABIWIT)
}

func testImplementation(t *testing.T, kind nodecontract.ABIKind) Implementation {
	t.Helper()
	path, mediaType, label := "bin/plugin.wasm", "application/wasm", "wasm"
	if kind == nodecontract.ABIProcess {
		path, mediaType, label = "bin/plugin.exe", "application/vnd.microsoft.portable-executable", "process"
	}
	return Implementation{
		ABI: nodecontract.ABIRequirement{Kind: kind, Version: "v1"}, Entrypoint: "acme:transform/uppercase#run",
		Payload: testPayload(t, path, mediaType, label),
		Platforms: PlatformSupport{
			OperatingSystems: []string{"windows", "linux"}, Architectures: []string{"arm64", "amd64"},
		},
	}
}

func testPayload(t *testing.T, path, mediaType, label string) Payload {
	t.Helper()
	hash := sha256.Sum256([]byte(label))
	digest := artifact.Digest("sha256:" + hex.EncodeToString(hash[:]))
	return Payload{Path: path, Digest: digest, Size: int64(len(label)), MediaType: mediaType}
}
