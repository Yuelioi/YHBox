package compiler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/node"
	"github.com/yottaapp/yotta/internal/workflow/catalog"
	"github.com/yottaapp/yotta/internal/workflow/schema"
)

type compilerTestNode struct {
	kind         string
	inputType    string
	defaultValue any
	fieldSchema  *node.FieldSchema
	required     bool
	dynamic      bool
	execInput    bool
	capability   node.RuntimeCapability
}

func (n compilerTestNode) Spec() node.Spec {
	inputType := n.inputType
	if inputType == "" {
		inputType = "Number"
	}
	inputs := []node.InputSpec{{Name: "Value", Type: inputType, Required: n.required, Default: n.defaultValue, Schema: n.fieldSchema}}
	if n.execInput {
		inputs = append([]node.InputSpec{{Name: "In", Type: node.TypeExec}}, inputs...)
	}
	capability := n.capability
	if capability == "" {
		capability = node.RuntimeCapabilityLog
	}
	return node.Spec{
		Kind:                n.kind,
		Inputs:              inputs,
		Outputs:             []node.OutputSpec{{Name: "Next", Type: node.TypeExec}},
		RuntimeCapabilities: []node.RuntimeCapability{capability},
		DynamicInputs:       n.dynamic,
	}
}

func TestCompileDraftUsesCanonicalIntegerAndEnumContracts(t *testing.T) {
	tests := []struct {
		name, config string
		node         compilerTestNode
		valid        bool
	}{
		{"integer", `{"Value":2}`, compilerTestNode{kind: "fixture", inputType: "Integer", defaultValue: 1}, true},
		{"fractional integer", `{"Value":1.5}`, compilerTestNode{kind: "fixture", inputType: "Integer", defaultValue: 1}, false},
		{"enum member", `{"Value":2}`, compilerTestNode{kind: "fixture", fieldSchema: node.EnumSchema(node.EnumOption{Value: 1}, node.EnumOption{Value: 2})}, true},
		{"enum outsider", `{"Value":3}`, compilerTestNode{kind: "fixture", fieldSchema: node.EnumSchema(node.EnumOption{Value: 1}, node.EnumOption{Value: 2})}, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			compiler, snapshot := testCompilerWithNode(t, test.node)
			raw := strings.Replace(validSource("1", 0, 0), `"config":{"Value":1}`, `"config":`+test.config, 1)
			result, err := compiler.CompileDraft(context.Background(), CompileRequest{SourceJSON: []byte(raw), Catalog: snapshot})
			if err != nil {
				t.Fatal(err)
			}
			_, ok := result.Program()
			if ok != test.valid {
				t.Fatalf("valid=%t diagnostics=%#v", ok, result.Diagnostics)
			}
		})
	}
}

func TestCompileDraftRejectsMalformedPointAndGeometry(t *testing.T) {
	tests := []struct {
		name, typ string
		field     *node.FieldSchema
		config    string
	}{
		{"point", "Point", node.PointSchema(), `{"Value":{"garbage":true}}`},
		{"geometry", "Geometry", node.GeometrySchema(), `{"Value":{"garbage":true}}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			compiler, snapshot := testCompilerWithNode(t, compilerTestNode{kind: "fixture", inputType: test.typ, fieldSchema: test.field})
			raw := strings.Replace(validSource("1", 0, 0), `"config":{"Value":1}`, `"config":`+test.config, 1)
			result, err := compiler.CompileDraft(context.Background(), CompileRequest{SourceJSON: []byte(raw), Catalog: snapshot})
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := result.Program(); ok || len(result.Diagnostics) == 0 {
				t.Fatalf("result = %#v", result)
			}
		})
	}
}
func (compilerTestNode) Run(node.Ctx, node.Inputs) (node.Outputs, error) { return nil, nil }

func TestCompileDraftProducesDeterministicOpaqueProgram(t *testing.T) {
	compiler, catalogSnapshot := testCompiler(t)
	left, err := compiler.CompileDraft(context.Background(), CompileRequest{SourceJSON: []byte(validSource("1.0", 1, 2)), Catalog: catalogSnapshot})
	if err != nil {
		t.Fatal(err)
	}
	rightRaw := strings.ReplaceAll(validSource("1e0", 1, 2), `,"`, ",\n  \"")
	right, err := compiler.CompileDraft(context.Background(), CompileRequest{SourceJSON: []byte(rightRaw), Catalog: catalogSnapshot})
	if err != nil {
		t.Fatal(err)
	}
	leftProgram, ok := left.Program()
	if !ok || len(left.Diagnostics) != 0 {
		t.Fatalf("left = %#v", left)
	}
	rightProgram, ok := right.Program()
	if !ok || len(right.Diagnostics) != 0 {
		t.Fatalf("right = %#v", right)
	}
	if left.SourceHash != right.SourceHash || leftProgram.Hash() != rightProgram.Hash() || !bytes.Equal(leftProgram.Artifact(), rightProgram.Artifact()) {
		t.Fatal("semantic JSON equivalents produced different program identities")
	}
	opened, err := OpenProgram(leftProgram.Artifact(), catalogSnapshot, compiler.build)
	if err != nil {
		t.Fatal(err)
	}
	if opened.Hash() != leftProgram.Hash() || opened.SourceHash() != left.SourceHash || opened.CatalogHash() != catalogSnapshot.Hash() {
		t.Fatal("opened snapshot metadata drifted")
	}
	locks := opened.NodeLocks()
	if len(locks) != 1 || locks[0].Kind != "fixture" || !locks[0].ContractHash.Valid() {
		t.Fatalf("locks = %#v", locks)
	}
	if got := opened.RequestedCapabilities(); len(got) != 1 || got[0] != "runtime:log" {
		t.Fatalf("requested capabilities = %#v", got)
	}
	if got := opened.RequiredCapabilities(); len(got) != 1 || got[0] != "runtime:log" {
		t.Fatalf("capabilities = %#v", got)
	}
}

func TestCompileDraftRejectsEpochAndUnknownNodeKind(t *testing.T) {
	compiler, catalogSnapshot := testCompiler(t)
	result, err := compiler.CompileDraft(context.Background(), CompileRequest{SourceJSON: []byte(`{"format":"yotta.workflow","version":2}`), Catalog: catalogSnapshot})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := result.Program(); ok || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != schema.CodeUnsupportedWorkflowFormat {
		t.Fatalf("result = %#v", result)
	}
	result, err = compiler.CompileDraft(context.Background(), CompileRequest{SourceJSON: []byte(`{"format":"yotta.workflow","version":2}`)})
	if err != nil || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != schema.CodeUnsupportedWorkflowFormat {
		t.Fatalf("epoch rejection depended on catalog: result=%#v err=%v", result, err)
	}
	raw := strings.Replace(validSource("1", 0, 0), `"kind":"fixture"`, `"kind":"missing"`, 1)
	result, err = compiler.CompileDraft(context.Background(), CompileRequest{SourceJSON: []byte(raw), Catalog: catalogSnapshot})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := result.Program(); ok || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != schema.CodeUnknownNodeKind {
		t.Fatalf("result = %#v", result)
	}
	diagnostic := result.Diagnostics[0]
	if diagnostic.NodeID != "n1" || strings.Join(diagnostic.GraphPath, "/") != "main" || strings.Join(diagnostic.FieldPath, "/") != "graphs/0/nodes/0/kind" {
		t.Fatalf("diagnostic = %#v", diagnostic)
	}
}

func TestCompileDraftRequiresExactCapabilityDeclaration(t *testing.T) {
	compiler, snapshot := testCompiler(t)
	raw := strings.Replace(validSource("1", 0, 0), `"runtime:log"`, `"network"`, 1)
	result, err := compiler.CompileDraft(context.Background(), CompileRequest{SourceJSON: []byte(raw), Catalog: snapshot})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := result.Program(); ok || len(result.Diagnostics) != 2 {
		t.Fatalf("result = %#v", result)
	}
	codes := map[string]bool{}
	for _, diagnostic := range result.Diagnostics {
		codes[diagnostic.Code] = true
	}
	if !codes[schema.CodeMissingCapabilityDeclaration] || !codes[schema.CodeUnusedCapabilityDeclaration] {
		t.Fatalf("diagnostics = %#v", result.Diagnostics)
	}
}

func TestCompileDraftRejectsDanglingNodesAndUnknownPins(t *testing.T) {
	compiler, catalogSnapshot := testCompiler(t)
	tests := []struct {
		name, edge, keyword string
	}{
		{"dangling node", `{"from":"missing.Next","to":"n1.Value"}`, "edgeEndpoint"},
		{"unknown pin", `{"from":"n1.Missing","to":"n1.Value"}`, "edgePin"},
		{"pin type", `{"from":"n1.Next","to":"n1.Value"}`, "pinType"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			raw := strings.Replace(validSource("1", 0, 0), `"edges":[]`, `"edges":[`+test.edge+`]`, 1)
			result, err := compiler.CompileDraft(context.Background(), CompileRequest{SourceJSON: []byte(raw), Catalog: catalogSnapshot})
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := result.Program(); ok || len(result.Diagnostics) != 1 || result.Diagnostics[0].Params["keyword"] != test.keyword {
				t.Fatalf("result = %#v", result)
			}
		})
	}
}

func TestCompatibleTypesKeepsConcreteDomainFamiliesDistinct(t *testing.T) {
	for _, pair := range [][2]string{{"Geometry", "String"}, {"Image", "Window"}, {"Point", "Rect"}} {
		if compatibleTypes(pair[0], pair[1]) {
			t.Fatalf("accepted %s -> %s", pair[0], pair[1])
		}
	}
	for _, pair := range [][2]string{{"Number", "Integer"}, {"Duration", "Number"}, {"Geometry", "JSON"}} {
		if !compatibleTypes(pair[0], pair[1]) {
			t.Fatalf("rejected %s -> %s", pair[0], pair[1])
		}
	}
}

func TestCompileDraftValidatesStaticConfigAndRejectsUnmodeledContracts(t *testing.T) {
	tests := []struct {
		name     string
		node     compilerTestNode
		config   string
		wantCode string
		keyword  string
	}{
		{"missing required", compilerTestNode{kind: "fixture", required: true}, `{}`, schema.CodeInvalidField, "requiredInput"},
		{"wrong literal type", compilerTestNode{kind: "fixture"}, `{"Value":"bad"}`, schema.CodeInvalidField, "configType"},
		{"unknown config", compilerTestNode{kind: "fixture"}, `{"Other":1}`, schema.CodeInvalidField, "configField"},
		{"dynamic contract", compilerTestNode{kind: "fixture", dynamic: true}, `{"Value":1}`, schema.CodeUnsupportedNodeContract, ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			compiler, snapshot := testCompilerWithNode(t, test.node)
			raw := strings.Replace(validSource("1", 0, 0), `"config":{"Value":1}`, `"config":`+test.config, 1)
			result, err := compiler.CompileDraft(context.Background(), CompileRequest{SourceJSON: []byte(raw), Catalog: snapshot})
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := result.Program(); ok || len(result.Diagnostics) == 0 || result.Diagnostics[0].Code != test.wantCode {
				t.Fatalf("result = %#v", result)
			}
			if test.keyword != "" && result.Diagnostics[0].Params["keyword"] != test.keyword {
				t.Fatalf("diagnostic = %#v", result.Diagnostics[0])
			}
		})
	}
}

func TestCompileDraftRejectsMalformedSubgraphInterface(t *testing.T) {
	compiler, snapshot := testCompiler(t)
	subgraph := `,{"id":"sub","kind":"subgraph","nodes":[],"edges":[],"inputs":[],"outputs":[]}`
	raw := strings.Replace(validSource("1", 0, 0), `}],"variables"`, `}`+subgraph+`],"variables"`, 1)
	result, err := compiler.CompileDraft(context.Background(), CompileRequest{SourceJSON: []byte(raw), Catalog: snapshot})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := result.Program(); ok || !hasDiagnosticCode(result.Diagnostics, schema.CodeInvalidGraphEntry) || !hasDiagnosticCode(result.Diagnostics, schema.CodeMissingGraphOutput) {
		t.Fatalf("result = %#v", result)
	}
}

func TestCompileDraftFailsClosedForCatalogAndCancellation(t *testing.T) {
	compiler, _ := testCompiler(t)
	if _, err := compiler.CompileDraft(context.Background(), CompileRequest{SourceJSON: []byte(validSource("1", 0, 0))}); !errors.Is(err, ErrInvalidCatalog) {
		t.Fatalf("err = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := compiler.CompileDraft(ctx, CompileRequest{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
}

func TestProgramSnapshotRejectsMutationAndNonCanonicalArtifacts(t *testing.T) {
	compiler, catalogSnapshot := testCompiler(t)
	result, err := compiler.CompileDraft(context.Background(), CompileRequest{SourceJSON: []byte(validSource("1", 0, 0)), Catalog: catalogSnapshot})
	if err != nil {
		t.Fatal(err)
	}
	program, ok := result.Program()
	if !ok {
		t.Fatalf("diagnostics = %#v", result.Diagnostics)
	}
	original := program.Artifact()
	copyOut := program.Artifact()
	copyOut[0] = 'x'
	if !bytes.Equal(original, program.Artifact()) {
		t.Fatal("artifact accessor leaked mutable storage")
	}
	if _, err := OpenProgram(append([]byte(" "), original...), catalogSnapshot, compiler.build); !errors.Is(err, ErrNonCanonicalProgram) {
		t.Fatalf("noncanonical err = %v", err)
	}
	tampered := bytes.Replace(original, []byte(`"workflowId":"w"`), []byte(`"workflowId":"x"`), 1)
	if _, err := OpenProgram(tampered, catalogSnapshot, compiler.build); !errors.Is(err, ErrProgramHashMismatch) {
		t.Fatalf("tamper err = %v", err)
	}
	var zero ProgramSnapshot
	if zero.Valid() || zero.Artifact() != nil {
		t.Fatal("zero snapshot is executable")
	}
}

func TestOpenProgramRequiresAllFieldsAndTrustedBindings(t *testing.T) {
	compiler, catalogSnapshot := testCompiler(t)
	result, err := compiler.CompileDraft(context.Background(), CompileRequest{SourceJSON: []byte(validSource("1", 0, 0)), Catalog: catalogSnapshot})
	if err != nil {
		t.Fatal(err)
	}
	program, _ := result.Program()
	missingRevision := bytes.Replace(program.Artifact(), []byte(`,"revision":0`), nil, 1)
	if bytes.Equal(missingRevision, program.Artifact()) {
		t.Fatal("fixture did not contain revision")
	}
	if _, err := OpenProgram(missingRevision, catalogSnapshot, compiler.build); !errors.Is(err, ErrInvalidProgramArtifact) {
		t.Fatalf("missing field err = %v", err)
	}
	if _, err := OpenProgram(program.Artifact(), catalogSnapshot, testDigest(t, "other compiler")); !errors.Is(err, ErrCompilerBuildMismatch) {
		t.Fatalf("build err = %v", err)
	}
	otherCatalog, err := catalog.NewSnapshot(node.NewRegistry().Snapshot(), testDigest(t, "other implementation"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := OpenProgram(program.Artifact(), otherCatalog, compiler.build); !errors.Is(err, ErrCatalogHashMismatch) {
		t.Fatalf("catalog err = %v", err)
	}

	var forged programEnvelope
	if err := json.Unmarshal(program.Artifact(), &forged); err != nil {
		t.Fatal(err)
	}
	forged.Program.Graphs[0].Nodes[0].Config["Value"] = "bad"
	forgedRaw := rehashEnvelope(t, forged)
	if _, err := OpenProgram(forgedRaw, catalogSnapshot, compiler.build); !errors.Is(err, ErrInvalidProgramArtifact) {
		t.Fatalf("forged binding err = %v", err)
	}
	if err := json.Unmarshal(program.Artifact(), &forged); err != nil {
		t.Fatal(err)
	}
	forged.Program.RequiredCapabilities = append(forged.Program.RequiredCapabilities, "runtime:vision")
	forgedRaw = rehashEnvelope(t, forged)
	if _, err := OpenProgram(forgedRaw, catalogSnapshot, compiler.build); !errors.Is(err, ErrCapabilityMismatch) {
		t.Fatalf("forged capabilities err = %v", err)
	}
	if err := json.Unmarshal(program.Artifact(), &forged); err != nil {
		t.Fatal(err)
	}
	forged.Program.NodeLocks[0].ContractHash = testDigest(t, "forged lock")
	forgedRaw = rehashEnvelope(t, forged)
	if _, err := OpenProgram(forgedRaw, catalogSnapshot, compiler.build); !errors.Is(err, ErrNodeLockMismatch) {
		t.Fatalf("forged lock err = %v", err)
	}
}

func TestCompileAndOpenHonorResourceBudgets(t *testing.T) {
	compiler, catalogSnapshot := testCompiler(t)
	if _, err := compiler.CompileDraft(context.Background(), CompileRequest{SourceJSON: make([]byte, MaxSourceBytes+1), Catalog: catalogSnapshot}); !errors.Is(err, ErrSourceBudgetExceeded) {
		t.Fatalf("source size err = %v", err)
	}
	deep := []byte(strings.Repeat("[", MaxJSONDepth+1) + strings.Repeat("]", MaxJSONDepth+1))
	if _, err := compiler.CompileDraft(context.Background(), CompileRequest{SourceJSON: deep, Catalog: catalogSnapshot}); !errors.Is(err, ErrSourceBudgetExceeded) {
		t.Fatalf("depth err = %v", err)
	}
	if _, err := OpenProgram(make([]byte, MaxProgramBytes+1), catalogSnapshot, compiler.build); !errors.Is(err, ErrProgramTooLarge) {
		t.Fatalf("program size err = %v", err)
	}
	if _, err := OpenProgram(deep, catalogSnapshot, compiler.build); !errors.Is(err, ErrProgramTooDeep) {
		t.Fatalf("program depth err = %v", err)
	}
}

func TestSealProgramRunsTheSameStructuralValidationAsOpen(t *testing.T) {
	if program, err := sealProgram(programBody{}); err == nil || program.Valid() {
		t.Fatalf("invalid internal body sealed: program=%#v err=%v", program, err)
	}
}

func TestSourceEditsProduceNewImmutableProgramIdentity(t *testing.T) {
	compiler, catalogSnapshot := testCompiler(t)
	left, err := compiler.CompileDraft(context.Background(), CompileRequest{SourceJSON: []byte(validSource("1", 0, 0)), Catalog: catalogSnapshot})
	if err != nil {
		t.Fatal(err)
	}
	right, err := compiler.CompileDraft(context.Background(), CompileRequest{SourceJSON: []byte(validSource("1", 50, 80)), Catalog: catalogSnapshot})
	if err != nil {
		t.Fatal(err)
	}
	leftProgram, _ := left.Program()
	rightProgram, _ := right.Program()
	if left.SourceHash == right.SourceHash || leftProgram.Hash() == rightProgram.Hash() {
		t.Fatal("source provenance edit did not create a new sealed snapshot")
	}
}

func TestCompileDraftRejectsUnsafeJSONIntegerBeforeHashing(t *testing.T) {
	compiler, catalogSnapshot := testCompiler(t)
	raw := validSource("9007199254740992", 0, 0)
	result, err := compiler.CompileDraft(context.Background(), CompileRequest{SourceJSON: []byte(raw), Catalog: catalogSnapshot})
	if err != nil {
		t.Fatal(err)
	}
	if result.SourceHash != "" {
		t.Fatalf("unsafe source received hash %s", result.SourceHash)
	}
	if _, ok := result.Program(); ok || len(result.Diagnostics) != 1 || result.Diagnostics[0].Params["keyword"] != "rfc8785" {
		t.Fatalf("result = %#v", result)
	}
}

func TestProgramIdentityGolden(t *testing.T) {
	compiler, catalogSnapshot := testCompiler(t)
	result, err := compiler.CompileDraft(context.Background(), CompileRequest{SourceJSON: []byte(validSource("1", 0, 0)), Catalog: catalogSnapshot})
	if err != nil {
		t.Fatal(err)
	}
	program, ok := result.Program()
	if !ok {
		t.Fatalf("diagnostics = %#v", result.Diagnostics)
	}
	const wantSource = "sha256:cfbe4d5ba8d9a4105d551e797d2c1e3c212ad87bc6344be83ce62d486bf8729c"
	const wantCatalog = "sha256:98f3593c845500d660bb7fe92af1b9d602d0fb7254c122f1b3cfc19546f126dd"
	const wantProgram = "sha256:186aa0a1e003a891fcc5e004b2d0144d22e90b8966694d485767a1efa0592cd6"
	if result.SourceHash.String() != wantSource || catalogSnapshot.Hash().String() != wantCatalog || program.Hash().String() != wantProgram {
		t.Fatalf("golden drift:\nsource  %s\ncatalog %s\nprogram %s", result.SourceHash, catalogSnapshot.Hash(), program.Hash())
	}
}

func testCompiler(t testing.TB) (*Compiler, catalog.Snapshot) {
	return testCompilerWithNode(t, compilerTestNode{kind: "fixture"})
}

func testCompilerWithNode(t testing.TB, fixture compilerTestNode) (*Compiler, catalog.Snapshot) {
	t.Helper()
	registry := node.NewRegistry()
	registry.Register(fixture)
	implementation := testDigest(t, "implementation")
	snapshot, err := catalog.NewSnapshot(registry.Snapshot(), implementation)
	if err != nil {
		t.Fatal(err)
	}
	compiler, err := New(testDigest(t, "compiler"))
	if err != nil {
		t.Fatal(err)
	}
	return compiler, snapshot
}

func rehashEnvelope(t testing.TB, envelope programEnvelope) []byte {
	t.Helper()
	body, err := artifact.Marshal(envelope.Program)
	if err != nil {
		t.Fatal(err)
	}
	envelope.ProgramHash, err = artifact.Sum(programHashDomain, body)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := artifact.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func testDigest(t testing.TB, value string) artifact.Digest {
	t.Helper()
	digest, err := artifact.Sum("yotta/compiler-test/v1", []byte(value))
	if err != nil {
		t.Fatal(err)
	}
	return digest
}

func validSource(number string, x, y int) string {
	return `{"format":"yotta.workflow","version":3,"workflow":{"id":"w","name":"Workflow"},"revision":0,"entryGraph":"main","graphs":[{"id":"main","kind":"main","nodes":[{"id":"n1","kind":"fixture","position":{"x":` + stringInt(x) + `,"y":` + stringInt(y) + `},"config":{"Value":` + number + `}}],"edges":[],"inputs":[],"outputs":[]}],"variables":[],"secretRefs":[],"requestedCapabilities":["runtime:log"]}`
}

func stringInt(value int) string {
	return strconv.Itoa(value)
}
