package compiler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecatalog"
	"github.com/yottaapp/yotta/internal/nodecontract"
	"github.com/yottaapp/yotta/internal/nodes31"
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
		len(nodes[0].Ports.ExecInputs)+len(nodes[0].Ports.ExecOutputs)+len(nodes[0].Ports.ErrorOutputs)+len(nodes[0].Ports.StatusOutputs) != 0 {
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
	if !hasDiagnostic(result.Diagnostics, CodeUnsupportedSourceFeature) {
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
	if got := program.Nodes()[0].Ports.DataInputs[0].ID; got != "a" {
		t.Fatalf("caller mutated immutable program view: %q", got)
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
