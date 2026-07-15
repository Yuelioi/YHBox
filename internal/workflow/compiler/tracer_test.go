package compiler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/yottaapp/yotta/internal/artifact"
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
	run, err := NewInterpreter(catalog, nil, map[string]InstalledBuiltin{
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
		"variables":[],"secretRefs":[],"requestedCapabilities":[]
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
	if _, err := NewInterpreter(catalog, nil, map[string]InstalledBuiltin{
		"text.concat": {Implementation: wrong, Run: nodes31.Concat},
	}).Run(context.Background(), program); err == nil {
		t.Fatal("interpreter dispatched a builtin that did not match the Program lock")
	}
	badOutput := func(context.Context, map[string]json.RawMessage) (map[string]json.RawMessage, error) {
		return map[string]json.RawMessage{"result": json.RawMessage(`1`)}, nil
	}
	if _, err := NewInterpreter(catalog, nil, map[string]InstalledBuiltin{
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
	document.Body.RequiredCapabilities = []string{"https://schemas.yotta.dev/capabilities/forged/v1"}
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
		}],"edges":[%s],"inputs":[],"outputs":[]}],"variables":[],"secretRefs":[],"requestedCapabilities":[]
	}`, ref.NodeTypeID, ref.SemanticDigest, a, b, edges))
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
