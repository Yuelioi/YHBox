package compiler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecatalog"
	"github.com/yottaapp/yotta/internal/nodecontract"
	"github.com/yottaapp/yotta/internal/nodes31"
	"github.com/yottaapp/yotta/internal/workflow/schema"
)

func TestConcatTracerCompilesOpensAndRunsWithoutExecOut(t *testing.T) {
	catalog, contract := concatCatalogForTest(t)
	build := testDigest(t, "compiler")
	result, err := New(build).CompileDraft(context.Background(), CompileRequest{
		SourceJSON: concatSourceForTest(contract.NodeRef(), "hello", " world", nil), Catalog: catalog,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v", result.Diagnostics)
	}
	program, ok := result.Program()
	if !ok {
		t.Fatal("missing program")
	}
	if plan := program.CapabilityPlan(); !plan.Valid() || len(plan.Entries()) != 0 {
		t.Fatalf("concat capability plan = %#v", plan.Entries())
	}
	nodes := program.Nodes()
	if len(nodes) != 1 || len(nodes[0].Ports.DataInputs) != 2 || len(nodes[0].Ports.DataOutputs) != 1 ||
		len(nodes[0].Ports.ExecInputs)+len(nodes[0].Ports.ExecOutputs)+len(nodes[0].Ports.ErrorOutputs) != 0 {
		t.Fatalf("program ports = %#v", nodes)
	}
	opened, err := OpenProgram(program.Artifact(), catalog, build)
	if err != nil {
		t.Fatal(err)
	}
	entry, _ := catalog.Lookup(nodes31.ConcatNodeID)
	run, err := NewInterpreter(catalog, map[string]InstalledBuiltin{
		"text.concat": {Implementation: entry.Implementation, Run: nodes31.Concat},
	}).Run(context.Background(), opened)
	if err != nil {
		t.Fatal(err)
	}
	envelope, err := datatype.OpenValueEnvelope(catalog, run.NodeOutputs["concat-1"]["result"])
	if err != nil {
		t.Fatal(err)
	}
	if got := string(envelope.InlineJSON()); got != `"hello world"` {
		t.Fatalf("concat result = %s", got)
	}
}

func TestConcatTracerRejectsInventedOutAndContractMismatch(t *testing.T) {
	catalog, contract := concatCatalogForTest(t)
	build := testDigest(t, "compiler")
	outEdge := `{"channel":"exec","from":{"nodeId":"concat-1","portId":"out"},"to":{"nodeId":"concat-1","portId":"in"}}`
	result, err := New(build).CompileDraft(context.Background(), CompileRequest{
		SourceJSON: concatSourceForTest(contract.NodeRef(), "a", "b", &outEdge), Catalog: catalog,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !hasDiagnostic(result.Diagnostics, CodeUnknownPort) {
		t.Fatalf("invented out diagnostics = %#v", result.Diagnostics)
	}

	ref := contract.NodeRef()
	ref.SemanticDigest = artifact.Digest("sha256:" + strings.Repeat("0", 64))
	result, err = New(build).CompileDraft(context.Background(), CompileRequest{
		SourceJSON: concatSourceForTest(ref, "a", "b", nil), Catalog: catalog,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !hasDiagnostic(result.Diagnostics, CodeNodeContractMismatch) {
		t.Fatalf("contract mismatch diagnostics = %#v", result.Diagnostics)
	}
}

func TestCompilerLowersExecAndErrorEdgesIntoOrderedSignalRoutes(t *testing.T) {
	catalog, source, target := signalCatalogForTest(t)
	build := testDigest(t, "signal-compiler")
	raw := signalSourceForTest(source.NodeRef(), target.NodeRef(), []string{
		`{"channel":"exec","from":{"nodeId":"source","portId":"next"},"to":{"nodeId":"target","portId":"in"}}`,
		`{"channel":"error","from":{"nodeId":"source","portId":"failed"},"to":{"nodeId":"target","portId":"in"}}`,
	})
	compiled, err := New(build).CompileDraft(context.Background(), CompileRequest{SourceJSON: raw, Catalog: catalog})
	if err != nil || len(compiled.Diagnostics) != 0 {
		t.Fatalf("compile diagnostics=%#v err=%v", compiled.Diagnostics, err)
	}
	program, ok := compiled.Program()
	if !ok {
		t.Fatal("missing signal Program")
	}
	var document programDocument
	if err := json.Unmarshal(program.Artifact(), &document); err != nil {
		t.Fatal(err)
	}
	graph := document.Body.Graphs[0]
	if len(graph.SignalRoutes) != 2 || graph.SignalRoutes[0].Channel != schema.EdgeExec || graph.SignalRoutes[1].Channel != schema.EdgeError {
		t.Fatalf("signal routes = %#v", graph.SignalRoutes)
	}
	if !slices.Equal(graph.DataOrder, []string{"source", "target"}) {
		t.Fatalf("data order = %#v", graph.DataOrder)
	}
	if _, err := OpenProgram(program.Artifact(), catalog, build); err != nil {
		t.Fatalf("strict-open signal Program: %v", err)
	}

	document.Body.Graphs[0].SignalRoutes[0].To.PortID = "missing"
	forged, err := sealProgram(document.Body)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := OpenProgram(forged.Artifact(), catalog, build); err == nil {
		t.Fatal("accepted a forged signal route")
	}

	document.Body.Graphs[0].SignalRoutes[0].To.PortID = "in"
	slices.Reverse(document.Body.Graphs[0].DataOrder)
	forged, err = sealProgram(document.Body)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := OpenProgram(forged.Artifact(), catalog, build); err == nil {
		t.Fatal("accepted a forged data order")
	}
}

func TestCompilerRejectsDuplicateSignalRoutesAndRootlessControlCycles(t *testing.T) {
	catalog, source, target := signalCatalogForTest(t)
	duplicate := `{"channel":"exec","from":{"nodeId":"source","portId":"next"},"to":{"nodeId":"target","portId":"in"}}`
	compiled, err := New(testDigest(t, "duplicate-route")).CompileDraft(context.Background(), CompileRequest{
		SourceJSON: signalSourceForTest(source.NodeRef(), target.NodeRef(), []string{duplicate, duplicate}), Catalog: catalog,
	})
	if err != nil || !hasDiagnostic(compiled.Diagnostics, CodeDuplicateSignalRoute) {
		t.Fatalf("duplicate route diagnostics=%#v err=%v", compiled.Diagnostics, err)
	}

	cycleCatalog, left, right := signalCycleCatalogForTest(t)
	cycle := signalSourceForTest(left.NodeRef(), right.NodeRef(), []string{
		`{"channel":"exec","from":{"nodeId":"source","portId":"next"},"to":{"nodeId":"target","portId":"in"}}`,
		`{"channel":"exec","from":{"nodeId":"target","portId":"next"},"to":{"nodeId":"source","portId":"in"}}`,
	})
	compiled, err = New(testDigest(t, "control-cycle")).CompileDraft(context.Background(), CompileRequest{SourceJSON: cycle, Catalog: cycleCatalog})
	if err != nil || !hasDiagnostic(compiled.Diagnostics, CodeNoExecutionRoot) {
		t.Fatalf("control cycle diagnostics=%#v err=%v", compiled.Diagnostics, err)
	}
}

func TestCompilerDistinguishesWrongChannelFromUnknownPort(t *testing.T) {
	catalog, source, target := signalCatalogForTest(t)
	compiled, err := New(testDigest(t, "channel-mismatch")).CompileDraft(context.Background(), CompileRequest{
		SourceJSON: signalSourceForTest(source.NodeRef(), target.NodeRef(), []string{
			`{"channel":"error","from":{"nodeId":"source","portId":"next"},"to":{"nodeId":"target","portId":"in"}}`,
		}), Catalog: catalog,
	})
	if err != nil || !hasDiagnostic(compiled.Diagnostics, CodeEdgeChannelMismatch) {
		t.Fatalf("channel mismatch diagnostics=%#v err=%v", compiled.Diagnostics, err)
	}
}

func TestConcatTracerFreezesTypedDataEdgesIndependentOfSourceOrder(t *testing.T) {
	catalog, contract := concatCatalogForTest(t)
	ref := contract.NodeRef()
	raw := []byte(fmt.Sprintf(`{
		"format":"yotta.workflow","version":"3.1","workflow":{"id":"wf-2","name":"Chain"},
		"revision":0,"entryGraph":"main","graphs":[{"id":"main","kind":"main","nodes":[
			{"id":"second","nodeRef":{"nodeTypeId":%q,"semanticDigest":%q},"position":{"x":1,"y":0},"config":{},"bindings":{"b":{"kind":"value","value":"c"}}},
			{"id":"first","nodeRef":{"nodeTypeId":%q,"semanticDigest":%q},"position":{"x":0,"y":0},"config":{},"bindings":{"a":{"kind":"value","value":"a"},"b":{"kind":"value","value":"b"}}}
		],"edges":[{"channel":"data","from":{"nodeId":"first","portId":"result"},"to":{"nodeId":"second","portId":"a"}}],"inputs":[],"outputs":[]}],
		"variables":[],"secretRefs":[]
	}`, ref.NodeTypeID, ref.SemanticDigest, ref.NodeTypeID, ref.SemanticDigest))
	result, err := New(testDigest(t, "compiler")).CompileDraft(context.Background(), CompileRequest{SourceJSON: raw, Catalog: catalog})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v", result.Diagnostics)
	}
	program, ok := result.Program()
	if !ok {
		t.Fatal("missing data-edge Program")
	}
	var document programDocument
	if err := json.Unmarshal(program.Artifact(), &document); err != nil {
		t.Fatal(err)
	}
	if got := document.Body.Graphs[0].Nodes[0].Inputs["a"]; got.Kind != inputEdge || got.From.NodeID != "first" || got.From.PortID != "result" {
		t.Fatalf("frozen data input = %#v", got)
	}
}

func TestOpenProgramRevalidatesConfigAndLiteralBindings(t *testing.T) {
	catalog, contract := concatCatalogForTest(t)
	build := testDigest(t, "compiler")
	compiled, err := New(build).CompileDraft(context.Background(), CompileRequest{
		SourceJSON: concatSourceForTest(contract.NodeRef(), "a", "b", nil), Catalog: catalog,
	})
	if err != nil || len(compiled.Diagnostics) != 0 {
		t.Fatalf("compile diagnostics=%#v err=%v", compiled.Diagnostics, err)
	}
	program, _ := compiled.Program()

	var document programDocument
	if err := json.Unmarshal(program.Artifact(), &document); err != nil {
		t.Fatal(err)
	}
	document.Body.Graphs[0].Nodes[0].Config["unexpected"] = true
	forged, err := sealProgram(document.Body)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := OpenProgram(forged.Artifact(), catalog, build); err == nil {
		t.Fatal("accepted rehashed program with config outside the Node Contract schema")
	}

	document.Body.Graphs[0].Nodes[0].Config = map[string]any{}
	document.Body.Graphs[0].Nodes[0].Inputs["a"] = inputPlan{Kind: inputLiteral, Value: json.RawMessage(`1`)}
	forged, err = sealProgram(document.Body)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := OpenProgram(forged.Artifact(), catalog, build); err == nil {
		t.Fatal("accepted rehashed program with a literal violating the pinned Data Type")
	}

	if err := json.Unmarshal(program.Artifact(), &document); err != nil {
		t.Fatal(err)
	}
	number, ok := catalog.LookupType(nodes31.NumberTypeID)
	if !ok {
		t.Fatal("number type is missing")
	}
	document.Body.Graphs[0].Nodes[0].OutputTypes["result"] = datatype.RefResolvedType(number.TypeRef())
	forged, err = sealProgram(document.Body)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := OpenProgram(forged.Artifact(), catalog, build); err == nil {
		t.Fatal("accepted rehashed Program with an effective type outside the Node Contract")
	}
}

func TestProgramNodeViewsAreDefensive(t *testing.T) {
	catalog, contract := concatCatalogForTest(t)
	compiled, err := New(testDigest(t, "compiler")).CompileDraft(context.Background(), CompileRequest{
		SourceJSON: concatSourceForTest(contract.NodeRef(), "a", "b", nil), Catalog: catalog,
	})
	if err != nil || len(compiled.Diagnostics) != 0 {
		t.Fatalf("compile diagnostics=%#v err=%v", compiled.Diagnostics, err)
	}
	program, _ := compiled.Program()
	view := program.Nodes()
	view[0].Ports.DataInputs[0].ID = "mutated"
	delete(view[0].InputTypes, "a")
	if got := program.Nodes()[0].Ports.DataInputs[0].ID; got != "a" {
		t.Fatalf("caller mutated immutable program view: %q", got)
	}
	if got := program.Nodes()[0].InputTypes["a"]; got.Ref == nil || got.Ref.TypeID != nodes31.StringTypeID {
		t.Fatalf("caller mutated immutable effective type view: %#v", got)
	}
}

func TestInterpreterRejectsUnpinnedBuiltinAndIllTypedOutput(t *testing.T) {
	catalog, contract := concatCatalogForTest(t)
	compiled, err := New(testDigest(t, "compiler")).CompileDraft(context.Background(), CompileRequest{
		SourceJSON: concatSourceForTest(contract.NodeRef(), "a", "b", nil), Catalog: catalog,
	})
	if err != nil || len(compiled.Diagnostics) != 0 {
		t.Fatalf("compile diagnostics=%#v err=%v", compiled.Diagnostics, err)
	}
	program, _ := compiled.Program()
	entry, _ := catalog.Lookup(nodes31.ConcatNodeID)
	wrong := entry.Implementation
	wrong.ArtifactDigest = testDigest(t, "wrong implementation")
	if _, err := NewInterpreter(catalog, map[string]InstalledBuiltin{
		"text.concat": {Implementation: wrong, Run: nodes31.Concat},
	}).Run(context.Background(), program); err == nil {
		t.Fatal("interpreter dispatched a builtin that did not match the Program lock")
	}
	badOutput := func(context.Context, map[string]json.RawMessage) (map[string]json.RawMessage, error) {
		return map[string]json.RawMessage{"result": json.RawMessage(`1`)}, nil
	}
	if _, err := NewInterpreter(catalog, map[string]InstalledBuiltin{
		"text.concat": {Implementation: entry.Implementation, Run: badOutput},
	}).Run(context.Background(), program); err == nil {
		t.Fatal("interpreter accepted output outside the pinned Data Type")
	}
}

func TestCompilerFailsClosedForDisabledNodes(t *testing.T) {
	catalog, contract := concatCatalogForTest(t)
	raw := bytes.Replace(concatSourceForTest(contract.NodeRef(), "a", "b", nil), []byte(`"config":{}`), []byte(`"config":{},"disabled":true`), 1)
	result, err := New(testDigest(t, "compiler")).CompileDraft(context.Background(), CompileRequest{SourceJSON: raw, Catalog: catalog})
	if err != nil {
		t.Fatal(err)
	}
	if !hasDiagnostic(result.Diagnostics, CodeUnsupportedSourceFeature) {
		t.Fatalf("diagnostics = %#v", result.Diagnostics)
	}
}

func TestCompilerFreezesConcreteTypedStateAndStrictOpenRevalidatesInitialValues(t *testing.T) {
	builtins, err := nodes31.Build()
	if err != nil {
		t.Fatal(err)
	}
	ref := builtins.ConcatContract.NodeRef()
	typeRef := builtins.StringType.TypeRef()
	raw := []byte(fmt.Sprintf(`{
		"format":"yotta.workflow","version":"3.1","workflow":{"id":"wf-state","name":"State"},
		"revision":0,"entryGraph":"main","graphs":[{"id":"main","kind":"main","nodes":[{
			"id":"concat","nodeRef":{"nodeTypeId":%q,"semanticDigest":%q},"position":{"x":0,"y":0},"config":{},
			"bindings":{"a":{"kind":"value","value":"a"},"b":{"kind":"value","value":"b"}}
		}],"edges":[],"inputs":[],"outputs":[]}],
		"variables":[{"name":"message","type":{"kind":"ref","ref":{"typeId":%q,"semanticDigest":%q}},"default":"ready"}],"secretRefs":[]
	}`, ref.NodeTypeID, ref.SemanticDigest, typeRef.TypeID, typeRef.SemanticDigest))
	build := testDigest(t, "compiler-state")
	compiled, err := New(build).CompileDraft(context.Background(), CompileRequest{SourceJSON: raw, Catalog: builtins.Catalog})
	if err != nil || len(compiled.Diagnostics) != 0 {
		t.Fatalf("compile=%v diagnostics=%#v", err, compiled.Diagnostics)
	}
	program, ok := compiled.Program()
	if !ok || len(program.State()) != 1 || program.State()[0].Name != "message" || !reflect.DeepEqual(program.State()[0].Type, datatype.RefResolvedType(typeRef)) {
		t.Fatalf("program state = %#v", program.State())
	}
	initial, err := datatype.OpenValueEnvelope(builtins.Catalog, program.State()[0].InitialArtifact)
	if err != nil || string(initial.InlineJSON()) != `"ready"` {
		t.Fatalf("initial=%s err=%v", initial.InlineJSON(), err)
	}
	if _, err := OpenProgram(program.Artifact(), builtins.Catalog, build); err != nil {
		t.Fatalf("strict-open state Program: %v", err)
	}
	view := program.State()
	view[0].Type.Ref.TypeID = "https://attacker.invalid/types/forged/v1"
	view[0].InitialArtifact[0] = '['
	if program.State()[0].Type.Ref.TypeID != typeRef.TypeID || program.State()[0].InitialArtifact[0] != '{' {
		t.Fatal("Program State view leaked mutable snapshot storage")
	}

	var document programDocument
	if err := json.Unmarshal(program.Artifact(), &document); err != nil {
		t.Fatal(err)
	}
	wrong, err := datatype.SealInlineJSON(builtins.Catalog, datatype.RefResolvedType(builtins.BooleanType.TypeRef()), []byte(`true`))
	if err != nil {
		t.Fatal(err)
	}
	document.Body.State[0].Initial = wrong.Artifact()
	forged, err := sealProgram(document.Body)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := OpenProgram(forged.Artifact(), builtins.Catalog, build); err == nil {
		t.Fatal("strict opener accepted a state initial value with a forged type")
	}
}

func TestCompilerRejectsUnresolvedUnknownAndNonInlineStateDeclarations(t *testing.T) {
	builtins, err := nodes31.Build()
	if err != nil {
		t.Fatal(err)
	}
	ref := builtins.ConcatContract.NodeRef()
	base := fmt.Sprintf(`{
		"format":"yotta.workflow","version":"3.1","workflow":{"id":"wf-state-invalid","name":"State invalid"},
		"revision":0,"entryGraph":"main","graphs":[{"id":"main","kind":"main","nodes":[{
			"id":"concat","nodeRef":{"nodeTypeId":%q,"semanticDigest":%q},"position":{"x":0,"y":0},"config":{},
			"bindings":{"a":{"kind":"value","value":"a"},"b":{"kind":"value","value":"b"}}
		}],"edges":[],"inputs":[],"outputs":[]}],"variables":[%s],"secretRefs":[]
	}`, ref.NodeTypeID, ref.SemanticDigest, "%s")
	unknown := artifact.Digest("sha256:" + strings.Repeat("2", 64))
	tests := []string{
		`{"name":"generic","type":{"kind":"variable","variable":"T"},"default":null}`,
		fmt.Sprintf(`{"name":"unknown","type":{"kind":"ref","ref":{"typeId":"https://schemas.yotta.dev/types/unknown/v1","semanticDigest":%q}},"default":null}`, unknown),
		fmt.Sprintf(`{"name":"binary","type":{"kind":"ref","ref":{"typeId":%q,"semanticDigest":%q}},"default":null}`, builtins.BinaryType.TypeRef().TypeID, builtins.BinaryType.TypeRef().SemanticDigest),
	}
	for index, declaration := range tests {
		compiled, err := New(testDigest(t, fmt.Sprintf("invalid-state-%d", index))).CompileDraft(context.Background(), CompileRequest{
			SourceJSON: []byte(fmt.Sprintf(base, declaration)), Catalog: builtins.Catalog,
		})
		if err != nil || !hasDiagnostic(compiled.Diagnostics, CodeInvalidStateVariable) {
			t.Fatalf("case %d compile=%v diagnostics=%#v", index, err, compiled.Diagnostics)
		}
	}
}

func TestBlobStreamConversionTracerCompilesExactEffectPlanAndStaysOutOfPreview(t *testing.T) {
	builtins, err := nodes31.Build()
	if err != nil {
		t.Fatal(err)
	}
	build := testDigest(t, "compiler-conversion")
	blobRef := blob.BlobRef{MediaType: "application/octet-stream", Digest: testDigest(t, "blob-value"), Size: 4}
	result, err := New(build).CompileDraft(context.Background(), CompileRequest{
		SourceJSON: conversionSourceForTest(builtins.BlobToStreamContract.NodeRef(), builtins.StreamToBlobContract.NodeRef(), blobRef),
		Catalog:    builtins.Catalog,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v", result.Diagnostics)
	}
	program, ok := result.Program()
	if !ok {
		t.Fatal("missing conversion Program")
	}
	if entries := program.CapabilityPlan().Entries(); len(entries) != 4 {
		t.Fatalf("capability plan = %#v", entries)
	}
	opened, err := OpenProgram(program.Artifact(), builtins.Catalog, build)
	if err != nil {
		t.Fatal(err)
	}
	nodes := opened.Nodes()
	if len(nodes) != 2 || nodes[0].Execution.Class != nodecontract.ExecutionEffect || nodes[1].Execution.Class != nodecontract.ExecutionEffect {
		t.Fatalf("effect nodes = %#v", nodes)
	}
	if _, err := NewInterpreter(builtins.Catalog, nil).Run(context.Background(), opened); err == nil {
		t.Fatal("pure-data preview executed an effect Program")
	}
}

func TestCompilerRejectsBlobLiteralForResourceLeasedInput(t *testing.T) {
	builtins, err := nodes31.Build()
	if err != nil {
		t.Fatal(err)
	}
	ref := builtins.StreamToBlobContract.NodeRef()
	blobRef := blob.BlobRef{MediaType: "application/octet-stream", Digest: testDigest(t, "wrong carrier"), Size: 4}
	source := []byte(fmt.Sprintf(`{
		"format":"yotta.workflow","version":"3.1","workflow":{"id":"wf-invalid-carrier","name":"Invalid"},
		"revision":0,"entryGraph":"main","graphs":[{"id":"main","kind":"main","nodes":[{
			"id":"to-blob","nodeRef":{"nodeTypeId":%q,"semanticDigest":%q},"position":{"x":0,"y":0},
			"config":{"mediaType":"application/octet-stream"},"bindings":{"stream":{"kind":"blob","blob":{"mediaType":%q,"digest":%q,"size":%d}}}
		}],"edges":[],"inputs":[],"outputs":[]}],"variables":[],"secretRefs":[]
	}`, ref.NodeTypeID, ref.SemanticDigest, blobRef.MediaType, blobRef.Digest, blobRef.Size))
	result, err := New(testDigest(t, "compiler-invalid-carrier")).CompileDraft(context.Background(), CompileRequest{
		SourceJSON: source, Catalog: builtins.Catalog,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !hasDiagnostic(result.Diagnostics, CodeInvalidBinding) {
		t.Fatalf("diagnostics = %#v", result.Diagnostics)
	}
}

func TestResourceLeaseAssignmentNeverWidensOrChangesCarrierClass(t *testing.T) {
	lease := func(operations ...string) *nodecontract.ResourceLeaseBinding {
		return &nodecontract.ResourceLeaseBinding{RequirementID: "stream", Operations: operations}
	}
	if !resourceLeaseAssignable(lease("cancel", "receive"), lease("receive")) {
		t.Fatal("narrowed resource lease was rejected")
	}
	if resourceLeaseAssignable(lease("receive"), lease("receive", "send")) {
		t.Fatal("resource lease widened across a data edge")
	}
	if resourceLeaseAssignable(nil, lease("receive")) || resourceLeaseAssignable(lease("receive"), nil) {
		t.Fatal("durable/runtime carrier classes were mixed across a data edge")
	}
}

func TestOpenProgramRejectsRehashedEntryAndCapabilityForgery(t *testing.T) {
	catalog, contract := concatCatalogForTest(t)
	build := testDigest(t, "compiler")
	compiled, err := New(build).CompileDraft(context.Background(), CompileRequest{SourceJSON: concatSourceForTest(contract.NodeRef(), "a", "b", nil), Catalog: catalog})
	if err != nil || len(compiled.Diagnostics) != 0 {
		t.Fatalf("compile diagnostics=%#v err=%v", compiled.Diagnostics, err)
	}
	program, _ := compiled.Program()
	var document programDocument
	if err := json.Unmarshal(program.Artifact(), &document); err != nil {
		t.Fatal(err)
	}
	document.Body.EntryGraph = "missing"
	forged, err := sealProgram(document.Body)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := OpenProgram(forged.Artifact(), catalog, build); err == nil {
		t.Fatal("accepted missing entry graph")
	}
	document.Body.EntryGraph = "main"
	forgedPlan, err := capability.SealPlan([]capability.PlanEntry{{
		GraphID: "main", NodeID: "concat-1", Requirement: capability.Requirement{
			ID: "forged", Capability: capability.Ref{
				CapabilityID: "https://schemas.yotta.dev/capabilities/forged/v1", SemanticDigest: testDigest(t, "forged capability"),
			}, Operations: []string{"read"}, TargetSlot: "target", Scope: json.RawMessage(`{}`),
		},
	}})
	if err != nil {
		t.Fatal(err)
	}
	document.Body.CapabilityPlan = forgedPlan.Bytes()
	forged, err = sealProgram(document.Body)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := OpenProgram(forged.Artifact(), catalog, build); err == nil {
		t.Fatal("accepted forged capability manifest")
	}
}

func concatSourceForTest(ref nodecontract.NodeRef, a, b string, edge *string) []byte {
	edges := ""
	if edge != nil {
		edges = *edge
	}
	return []byte(fmt.Sprintf(`{
		"format":"yotta.workflow","version":"3.1","workflow":{"id":"wf-1","name":"Concat"},
		"revision":0,"entryGraph":"main","graphs":[{"id":"main","kind":"main","nodes":[{
			"id":"concat-1","nodeRef":{"nodeTypeId":%q,"semanticDigest":%q},"position":{"x":0,"y":0},
			"config":{},"bindings":{"a":{"kind":"value","value":%q},"b":{"kind":"value","value":%q}}
		}],"edges":[%s],"inputs":[],"outputs":[]}],"variables":[],"secretRefs":[]
	}`, ref.NodeTypeID, ref.SemanticDigest, a, b, edges))
}

func conversionSourceForTest(toStream, toBlob nodecontract.NodeRef, ref blob.BlobRef) []byte {
	return []byte(fmt.Sprintf(`{
		"format":"yotta.workflow","version":"3.1","workflow":{"id":"wf-convert","name":"Convert"},
		"revision":0,"entryGraph":"main","graphs":[{"id":"main","kind":"main","nodes":[
			{"id":"to-blob","nodeRef":{"nodeTypeId":%q,"semanticDigest":%q},"position":{"x":1,"y":0},"config":{"mediaType":"application/octet-stream"},"bindings":{}},
			{"id":"to-stream","nodeRef":{"nodeTypeId":%q,"semanticDigest":%q},"position":{"x":0,"y":0},"config":{},
			 "bindings":{"blob":{"kind":"blob","blob":{"mediaType":%q,"digest":%q,"size":%d}}}}
		],"edges":[{"channel":"data","from":{"nodeId":"to-stream","portId":"stream"},"to":{"nodeId":"to-blob","portId":"stream"}}],"inputs":[],"outputs":[]}],
		"variables":[],"secretRefs":[]
	}`, toBlob.NodeTypeID, toBlob.SemanticDigest, toStream.NodeTypeID, toStream.SemanticDigest, ref.MediaType, ref.Digest, ref.Size))
}

func signalSourceForTest(source, target nodecontract.NodeRef, edges []string) []byte {
	return []byte(fmt.Sprintf(`{
		"format":"yotta.workflow","version":"3.1","workflow":{"id":"wf-signals","name":"Signals"},
		"revision":0,"entryGraph":"main","graphs":[{"id":"main","kind":"main","nodes":[
			{"id":"source","nodeRef":{"nodeTypeId":%q,"semanticDigest":%q},"position":{"x":0,"y":0},"config":{},"bindings":{}},
			{"id":"target","nodeRef":{"nodeTypeId":%q,"semanticDigest":%q},"position":{"x":1,"y":0},"config":{},"bindings":{}}
		],"edges":[%s],"inputs":[],"outputs":[]}],"variables":[],"secretRefs":[]
	}`, source.NodeTypeID, source.SemanticDigest, target.NodeTypeID, target.SemanticDigest, strings.Join(edges, ",")))
}

func signalCatalogForTest(t *testing.T) (nodecatalog.Snapshot, nodecontract.Contract, nodecontract.Contract) {
	t.Helper()
	source := signalContractForTest(t, "source", []string{}, []string{"next"}, []string{"failed"})
	target := signalContractForTest(t, "target", []string{"in"}, []string{}, []string{})
	return sealSignalCatalogForTest(t, source, target), source, target
}

func signalCycleCatalogForTest(t *testing.T) (nodecatalog.Snapshot, nodecontract.Contract, nodecontract.Contract) {
	t.Helper()
	left := signalContractForTest(t, "source", []string{"in"}, []string{"next"}, []string{})
	right := signalContractForTest(t, "target", []string{"in"}, []string{"next"}, []string{})
	return sealSignalCatalogForTest(t, left, right), left, right
}

func signalContractForTest(t *testing.T, name string, execInputs, execOutputs, errorOutputs []string) nodecontract.Contract {
	t.Helper()
	nodeID := "https://schemas.yotta.dev/nodes/test/" + name + "/v1"
	configID := nodeID + "/config"
	ports := func(values []string) []nodecontract.SignalPort {
		result := make([]nodecontract.SignalPort, len(values))
		for index, value := range values {
			result[index] = nodecontract.SignalPort{ID: value}
		}
		return result
	}
	class := nodecontract.ExecutionEvent
	if len(execInputs) != 0 {
		class = nodecontract.ExecutionControl
	}
	contract, err := nodecontract.Seal(nodecontract.Draft{
		NodeTypeID: nodeID, ConfigSchemaRoot: configID,
		ConfigSchemaBundle: []datatype.SchemaResource{{ID: configID, Schema: json.RawMessage(fmt.Sprintf(`{"$id":%q,"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","additionalProperties":false}`, configID))}},
		Ports: nodecontract.PortSet{
			DataInputs: []nodecontract.DataInputPort{}, DataOutputs: []nodecontract.DataOutputPort{},
			ExecInputs: ports(execInputs), ExecOutputs: ports(execOutputs), ErrorOutputs: ports(errorOutputs),
		},
		Execution: nodecontract.ExecutionSpec{
			Class: class, Effects: []nodecontract.EffectID{},
			Determinism: nodecontract.Deterministic, Evaluation: nodecontract.EvaluationPush, Cache: nodecontract.CacheNone,
			Retry: nodecontract.RetryNever, Cancellation: nodecontract.CancellationCooperative, Timeout: nodecontract.TimeoutNone,
		},
		CapabilityRequirements: []capability.Requirement{},
		Errors:                 []nodecontract.ErrorSpec{{Code: "test." + name + "_failed", Category: "test", RetryHint: false}},
		StatusEvents:           []nodecontract.StatusEventSpec{},
		ImplementationABI:      []nodecontract.ABIRequirement{{Kind: nodecontract.ABIBuiltin, Version: "v1"}},
		Authoring:              nodecontract.Authoring{Tags: []string{"test"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return contract
}

func sealSignalCatalogForTest(t *testing.T, contracts ...nodecontract.Contract) nodecatalog.Snapshot {
	t.Helper()
	bindings := make([]nodecatalog.Binding, len(contracts))
	for index, contract := range contracts {
		bindings[index] = nodecatalog.Binding{Contract: contract, Implementation: nodecatalog.ImplementationLock{
			PackageID: "https://schemas.yotta.dev/packages/test/v1", ArtifactDigest: testDigest(t, contract.NodeRef().NodeTypeID),
			ABI: nodecontract.ABIRequirement{Kind: nodecontract.ABIBuiltin, Version: "v1"}, Entrypoint: "test." + fmt.Sprint(index),
		}}
	}
	catalog, err := nodecatalog.Seal([]datatype.Definition{}, []capability.Definition{}, bindings, "v1")
	if err != nil {
		t.Fatal(err)
	}
	return catalog
}

func concatCatalogForTest(t *testing.T) (nodecatalog.Snapshot, nodecontract.Contract) {
	t.Helper()
	builtins, err := nodes31.Build()
	if err != nil {
		t.Fatal(err)
	}
	return builtins.Catalog, builtins.ConcatContract
}

func testDigest(t *testing.T, value string) artifact.Digest {
	t.Helper()
	digest, err := artifact.Sum("yotta/test/v1", []byte(value))
	if err != nil {
		t.Fatal(err)
	}
	return digest
}

func hasDiagnostic(values []Diagnostic, code string) bool {
	for _, diagnostic := range values {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}
