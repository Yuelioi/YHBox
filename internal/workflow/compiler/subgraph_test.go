package compiler

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/yottaapp/yotta/internal/node"
	"github.com/yottaapp/yotta/internal/workflow/catalog"
	"github.com/yottaapp/yotta/internal/workflow/schema"
)

func TestCompileDraftCompilesTypedSubgraphClosure(t *testing.T) {
	compiler, snapshot := testCompilerWithNode(t, compilerTestNode{kind: "fixture", required: true, execInput: true})
	result, err := compiler.CompileDraft(context.Background(), CompileRequest{SourceJSON: []byte(validSubgraphSource()), Catalog: snapshot})
	if err != nil {
		t.Fatal(err)
	}
	program, ok := result.Program()
	if !ok || len(result.Diagnostics) != 0 {
		t.Fatalf("result = %#v", result)
	}
	opened, err := OpenProgram(program.Artifact(), snapshot, compiler.build)
	if err != nil {
		t.Fatal(err)
	}
	if opened.Hash() != program.Hash() {
		t.Fatal("opened subgraph program changed identity")
	}
	var envelope programEnvelope
	if err := json.Unmarshal(program.Artifact(), &envelope); err != nil {
		t.Fatal(err)
	}
	if len(envelope.Program.Graphs) != 2 || envelope.Program.Graphs[0].ID != "main" || envelope.Program.Graphs[1].ID != "worker" {
		t.Fatalf("compiled closure = %#v", envelope.Program.Graphs)
	}
	call := envelope.Program.Graphs[0].Nodes[0].Call
	if call == nil || call.GraphID != "worker" || call.Entry.ID != "start" || call.Entry.NodeID != "$entry" || len(call.Inputs) != 1 || call.Inputs[0].ID != "amount" || call.Inputs[0].NodeID != "$amount" || len(call.Outputs) != 1 || call.Outputs[0].ID != "done" || call.Outputs[0].NodeID != "$out" {
		t.Fatalf("program did not freeze call plan: %#v", call)
	}
}

func TestCompileDraftRejectsInvalidSubgraphReferences(t *testing.T) {
	compiler, snapshot := testCompilerWithNode(t, compilerTestNode{kind: "fixture", required: true, execInput: true})
	tests := []struct {
		name, source, code string
	}{
		{"missing target", strings.Replace(validSubgraphSource(), `"graphId":"worker"`, `"graphId":"missing"`, 1), "UNKNOWN_CALLEE_GRAPH"},
		{"self recursion", recursiveSubgraphSource(), "SUBGRAPH_CALL_CYCLE"},
		{"main target", strings.Replace(validSubgraphSource(), `"graphId":"worker"`, `"graphId":"main"`, 1), "INVALID_CALLEE_GRAPH_KIND"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := compiler.CompileDraft(context.Background(), CompileRequest{SourceJSON: []byte(test.source), Catalog: snapshot})
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := result.Program(); ok || !hasDiagnosticCode(result.Diagnostics, test.code) {
				t.Fatalf("result = %#v", result)
			}
		})
	}
}

func TestCompileDraftRejectsInvalidSubgraphBoundary(t *testing.T) {
	compiler, snapshot := testCompilerWithNode(t, compilerTestNode{kind: "fixture", required: true, execInput: true})
	tests := []struct {
		name, old, replacement, code string
	}{
		{"edge into input boundary", `"from":"$entry.start","to":"n1.In"`, `"from":"n1.Next","to":"$entry.start"`, "INVALID_GRAPH_BOUNDARY_EDGE"},
		{"edge out of output boundary", `"from":"n1.Next","to":"$out.done"`, `"from":"$out.done","to":"n1.In"`, "INVALID_GRAPH_BOUNDARY_EDGE"},
		{"wrong boundary type", `"id":"amount","name":"Amount","type":"Number"`, `"id":"amount","name":"Amount","type":"String"`, "CALL_PIN_TYPE_MISMATCH"},
		{"missing exec entry", `{"id":"start","name":"Start","type":"Exec","nodeId":"$entry"},`, ``, "INVALID_GRAPH_ENTRY"},
		{"unbound data boundary", `,{"from":"$amount.amount","to":"n1.Value"}`, ``, "INVALID_GRAPH_BOUNDARY_EDGE"},
		{"ambiguous boundary identity", `"id":"amount","name":"Amount"`, `"id":"amount.bad","name":"Amount"`, schema.CodeInvalidField},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			raw := strings.Replace(validSubgraphSource(), test.old, test.replacement, 1)
			result, err := compiler.CompileDraft(context.Background(), CompileRequest{SourceJSON: []byte(raw), Catalog: snapshot})
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := result.Program(); ok || !hasDiagnosticCode(result.Diagnostics, test.code) {
				t.Fatalf("result = %#v", result)
			}
		})
	}
}

func TestCompileDraftFreezesTypedSubgraphDataOutput(t *testing.T) {
	compiler, snapshot := testCompilerWithNode(t, compilerTestNode{kind: "fixture", required: true, execInput: true, dataOutput: true})
	raw := validTypedSubgraphSource()
	result, err := compiler.CompileDraft(context.Background(), CompileRequest{SourceJSON: []byte(raw), Catalog: snapshot})
	if err != nil {
		t.Fatal(err)
	}
	program, ok := result.Program()
	if !ok {
		t.Fatalf("diagnostics = %#v", result.Diagnostics)
	}
	var envelope programEnvelope
	if err := json.Unmarshal(program.Artifact(), &envelope); err != nil {
		t.Fatal(err)
	}
	outputs := envelope.Program.Graphs[0].Nodes[0].Call.Outputs
	if len(outputs) != 2 || outputs[1] != (programCallPort{ID: "result", Type: "Number", NodeID: "$result"}) {
		t.Fatalf("typed call outputs = %#v", outputs)
	}
}

func TestCompileDraftRejectsMultipleTypedGraphOutputSources(t *testing.T) {
	compiler, snapshot := testCompilerWithNode(t, compilerTestNode{kind: "fixture", required: true, execInput: true, dataOutput: true})
	raw := strings.Replace(validTypedSubgraphSource(),
		`{"from":"n1.Result","to":"$result.result"}`,
		`{"from":"n1.Result","to":"$result.result"},{"from":"n1.Result","to":"$result.result"}`, 1)
	result, err := compiler.CompileDraft(context.Background(), CompileRequest{SourceJSON: []byte(raw), Catalog: snapshot})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := result.Program(); ok || len(result.Diagnostics) != 1 || result.Diagnostics[0].Params["keyword"] != "multipleGraphOutputSources" {
		t.Fatalf("result = %#v", result)
	}
}

func TestCompileDraftRejectsExpandedCallPlanBudget(t *testing.T) {
	compiler, snapshot := testCompilerWithNode(t, compilerTestNode{kind: "fixture", required: true, execInput: true})
	nodes := make([]string, MaxNodesPerGraph)
	for index := range nodes {
		nodes[index] = `{"id":"call-` + stringInt(index) + `","kind":"core.call-subgraph","position":{"x":0,"y":0},"config":{"graphId":"worker","amount":1}}`
	}
	raw := strings.Replace(validSubgraphSource(),
		`{"id":"call","kind":"core.call-subgraph","position":{"x":0,"y":0},"config":{"graphId":"worker","amount":1}}`,
		strings.Join(nodes, ","), 1)
	if len(raw) >= MaxSourceBytes {
		t.Fatalf("budget fixture unexpectedly exceeds source bytes: %d", len(raw))
	}
	if _, err := compiler.CompileDraft(context.Background(), CompileRequest{SourceJSON: []byte(raw), Catalog: snapshot}); !errors.Is(err, ErrSourceBudgetExceeded) {
		t.Fatalf("expanded call plan err = %v", err)
	}
}

func TestCompileDraftRejectsMultipleSourcesForOneInput(t *testing.T) {
	compiler, snapshot := testCompilerWithNode(t, compilerTestNode{kind: "fixture", required: true, execInput: true})
	raw := strings.Replace(validSubgraphSource(), `{"from":"$amount.amount","to":"n1.Value"}`, `{"from":"$amount.amount","to":"n1.Value"},{"from":"$amount.amount","to":"n1.Value"}`, 1)
	result, err := compiler.CompileDraft(context.Background(), CompileRequest{SourceJSON: []byte(raw), Catalog: snapshot})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := result.Program(); ok || len(result.Diagnostics) != 1 || result.Diagnostics[0].Params["keyword"] != "multipleInputSources" {
		t.Fatalf("result = %#v", result)
	}
}

func TestCompileDraftClosureExcludesUnreachableLocksAndCapabilities(t *testing.T) {
	registry := node.NewRegistry()
	registry.Register(compilerTestNode{kind: "fixture", required: true, execInput: true})
	registry.Register(compilerTestNode{kind: "shadow", execInput: true, capability: node.RuntimeCapabilityAI})
	snapshot, err := catalog.NewSnapshot(registry.Snapshot(), testDigest(t, "implementation"))
	if err != nil {
		t.Fatal(err)
	}
	compiler, err := New(testDigest(t, "compiler"))
	if err != nil {
		t.Fatal(err)
	}
	unreachable := `,{"id":"unused","kind":"subgraph","nodes":[{"id":"n2","kind":"shadow","position":{"x":0,"y":0},"config":{"Value":1}}],"edges":[{"from":"$entry.start","to":"n2.In"},{"from":"n2.Next","to":"$out.done"}],"inputs":[{"id":"start","name":"Start","type":"Exec","nodeId":"$entry"}],"outputs":[{"id":"done","name":"Done","type":"Exec","nodeId":"$out"}]}`
	raw := strings.Replace(validSubgraphSource(), `}],"variables"`, `}`+unreachable+`],"variables"`, 1)
	result, err := compiler.CompileDraft(context.Background(), CompileRequest{SourceJSON: []byte(raw), Catalog: snapshot})
	if err != nil {
		t.Fatal(err)
	}
	program, ok := result.Program()
	if !ok || len(result.Diagnostics) != 0 {
		t.Fatalf("result = %#v", result)
	}
	if locks := program.NodeLocks(); len(locks) != 1 || locks[0].Kind != "fixture" {
		t.Fatalf("reachable locks = %#v", locks)
	}
	if caps := program.RequiredCapabilities(); len(caps) != 1 || caps[0] != "runtime:log" {
		t.Fatalf("reachable capabilities = %#v", caps)
	}
	var envelope programEnvelope
	if err := json.Unmarshal(program.Artifact(), &envelope); err != nil {
		t.Fatal(err)
	}
	if len(envelope.Program.Graphs) != 2 {
		t.Fatalf("program retained unreachable graph: %#v", envelope.Program.Graphs)
	}
}

func TestOpenProgramRejectsForgedSubgraphCallPlan(t *testing.T) {
	compiler, snapshot := testCompilerWithNode(t, compilerTestNode{kind: "fixture", required: true, execInput: true})
	result, err := compiler.CompileDraft(context.Background(), CompileRequest{SourceJSON: []byte(validSubgraphSource()), Catalog: snapshot})
	if err != nil {
		t.Fatal(err)
	}
	program, ok := result.Program()
	if !ok {
		t.Fatalf("diagnostics = %#v", result.Diagnostics)
	}
	var envelope programEnvelope
	if err := json.Unmarshal(program.Artifact(), &envelope); err != nil {
		t.Fatal(err)
	}
	envelope.Program.Graphs[0].Nodes[0].Call.GraphID = "forged"
	forged := rehashEnvelope(t, envelope)
	if _, err := OpenProgram(forged, snapshot, compiler.build); !errors.Is(err, ErrInvalidProgramArtifact) {
		t.Fatalf("forged call plan err = %v", err)
	}
}

func hasDiagnosticCode(diagnostics []schema.Diagnostic, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}

func validSubgraphSource() string {
	return `{"format":"yotta.workflow","version":3,"workflow":{"id":"w","name":"Workflow"},"revision":0,"entryGraph":"main","graphs":[` +
		`{"id":"main","kind":"main","nodes":[{"id":"call","kind":"core.call-subgraph","position":{"x":0,"y":0},"config":{"graphId":"worker","amount":1}}],"edges":[],"inputs":[],"outputs":[]},` +
		`{"id":"worker","kind":"subgraph","nodes":[{"id":"n1","kind":"fixture","position":{"x":0,"y":0},"config":{}}],"edges":[{"from":"$entry.start","to":"n1.In"},{"from":"$amount.amount","to":"n1.Value"},{"from":"n1.Next","to":"$out.done"}],` +
		`"inputs":[{"id":"start","name":"Start","type":"Exec","nodeId":"$entry"},{"id":"amount","name":"Amount","type":"Number","nodeId":"$amount"}],"outputs":[{"id":"done","name":"Done","type":"Exec","nodeId":"$out"}]}` +
		`],"variables":[],"secretRefs":[],"requestedCapabilities":["runtime:log"]}`
}

func validTypedSubgraphSource() string {
	raw := strings.Replace(validSubgraphSource(),
		`{"from":"n1.Next","to":"$out.done"}],`,
		`{"from":"n1.Next","to":"$out.done"},{"from":"n1.Result","to":"$result.result"}],`, 1)
	return strings.Replace(raw,
		`"outputs":[{"id":"done","name":"Done","type":"Exec","nodeId":"$out"}]}`,
		`"outputs":[{"id":"done","name":"Done","type":"Exec","nodeId":"$out"},{"id":"result","name":"Result","type":"Number","nodeId":"$result"}]}`, 1)
}

func recursiveSubgraphSource() string {
	return strings.Replace(validSubgraphSource(),
		`"nodes":[{"id":"n1","kind":"fixture","position":{"x":0,"y":0},"config":{}}],"edges":[{"from":"$entry.start","to":"n1.In"},{"from":"$amount.amount","to":"n1.Value"},{"from":"n1.Next","to":"$out.done"}]`,
		`"nodes":[{"id":"again","kind":"core.call-subgraph","position":{"x":0,"y":0},"config":{"graphId":"worker","amount":1}}],"edges":[]`, 1)
}
