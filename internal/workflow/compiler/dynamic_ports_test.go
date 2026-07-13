package compiler

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/yottaapp/yotta/internal/node"
	"github.com/yottaapp/yotta/internal/nodes/control"
	"github.com/yottaapp/yotta/internal/workflow/catalog"
	"github.com/yottaapp/yotta/internal/workflow/schema"
)

func TestCompileDraftFreezesSwitchDynamicOutputs(t *testing.T) {
	compiler, snapshot := switchCompiler(t)
	result, err := compiler.CompileDraft(context.Background(), CompileRequest{
		SourceJSON: []byte(switchSource(`["Ready","恢复","🎣"]`, "[]")), Catalog: snapshot,
	})
	if err != nil {
		t.Fatal(err)
	}
	program, ok := result.Program()
	if !ok || len(result.Diagnostics) != 0 {
		t.Fatalf("result = %#v", result)
	}
	var envelope programEnvelope
	if err := json.Unmarshal(program.Artifact(), &envelope); err != nil {
		t.Fatal(err)
	}
	want := []programPort{
		{Role: node.DynamicPortOutput, Name: "Ready", Type: node.TypeExec},
		{Role: node.DynamicPortOutput, Name: "恢复", Type: node.TypeExec},
		{Role: node.DynamicPortOutput, Name: "🎣", Type: node.TypeExec},
	}
	got := envelope.Program.Graphs[0].Nodes[0].DynamicPorts
	if !equalProgramPorts(got, want) {
		t.Fatalf("dynamic ports = %#v", got)
	}
	if _, err := OpenProgram(program.Artifact(), snapshot, compiler.build); err != nil {
		t.Fatal(err)
	}
}

func TestCompileDraftUsesResolvedSwitchPinsForEdges(t *testing.T) {
	compiler, snapshot := switchCompiler(t)
	edges := `[{"from":"switch.Ready","to":"sink.In"}]`
	result, err := compiler.CompileDraft(context.Background(), CompileRequest{
		SourceJSON: []byte(switchSource(`["Ready"]`, edges)), Catalog: snapshot,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := result.Program(); !ok || len(result.Diagnostics) != 0 {
		t.Fatalf("result = %#v", result)
	}
	raw := switchSource(`["Waiting"]`, edges)
	result, err = compiler.CompileDraft(context.Background(), CompileRequest{SourceJSON: []byte(raw), Catalog: snapshot})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := result.Program(); ok || !diagnosticReason(result.Diagnostics, "edgePin") {
		t.Fatalf("result = %#v", result)
	}
}

func TestCompileDraftKeepsDynamicPinIndexesPerNode(t *testing.T) {
	compiler, snapshot := switchCompiler(t)
	raw := `{"format":"yotta.workflow","version":3,"workflow":{"id":"w","name":"Workflow"},"revision":0,"entryGraph":"main","graphs":[{"id":"main","kind":"main","nodes":[{"id":"left","kind":"Switch","position":{"x":0,"y":0},"config":{"Value":"A","cases":["A"]}},{"id":"right","kind":"Switch","position":{"x":1,"y":0},"config":{"Value":"B","cases":["B"]}},{"id":"sink","kind":"sink","position":{"x":2,"y":0},"config":{"Value":1}}],"edges":[{"from":"left.A","to":"sink.In"},{"from":"right.B","to":"sink.In"}],"inputs":[],"outputs":[]}],"variables":[],"secretRefs":[],"requestedCapabilities":["runtime:log"]}`
	result, err := compiler.CompileDraft(context.Background(), CompileRequest{SourceJSON: []byte(raw), Catalog: snapshot})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := result.Program(); !ok || len(result.Diagnostics) != 0 {
		t.Fatalf("result = %#v", result)
	}
	invalid := strings.Replace(raw, `"from":"left.A"`, `"from":"left.B"`, 1)
	result, err = compiler.CompileDraft(context.Background(), CompileRequest{SourceJSON: []byte(invalid), Catalog: snapshot})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := result.Program(); ok || !diagnosticReason(result.Diagnostics, "edgePin") {
		t.Fatalf("result = %#v", result)
	}
}

func TestCompileDraftRejectsInvalidSwitchDynamicPorts(t *testing.T) {
	compiler, snapshot := switchCompiler(t)
	tests := []struct {
		name, cases, reason string
	}{
		{"missing", `null`, "wrong_shape"},
		{"empty list", `[]`, "too_few"},
		{"non string", `["A",1]`, "wrong_item_type"},
		{"empty name", `[""]`, "empty"},
		{"whitespace", `[" A"]`, "whitespace"},
		{"dot", `["A.B"]`, "contains_dot"},
		{"control", `["A\nB"]`, "control_character"},
		{"bidi control", `["A\u202eB"]`, "control_character"},
		{"static conflict", `["default"]`, "static_conflict"},
		{"duplicate", `["A","A"]`, "duplicate"},
		{"too long", `["` + strings.Repeat("a", MaxDynamicPortNameBytes+1) + `"]`, "too_long"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := compiler.CompileDraft(context.Background(), CompileRequest{
				SourceJSON: []byte(switchSource(test.cases, "[]")), Catalog: snapshot,
			})
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := result.Program(); ok || !hasDiagnosticCode(result.Diagnostics, schema.CodeInvalidDynamicPortDeclaration) || !diagnosticReason(result.Diagnostics, test.reason) {
				t.Fatalf("result = %#v", result)
			}
		})
	}
}

func TestCompileDraftRejectsDynamicPortBudget(t *testing.T) {
	compiler, snapshot := switchCompiler(t)
	items := make([]string, MaxDynamicPortsPerNode+1)
	for index := range items {
		items[index] = `"case-` + stringInt(index) + `"`
	}
	result, err := compiler.CompileDraft(context.Background(), CompileRequest{
		SourceJSON: []byte(switchSource("["+strings.Join(items, ",")+"]", "[]")), Catalog: snapshot,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := result.Program(); ok || !hasDiagnosticCode(result.Diagnostics, schema.CodeDynamicPortBudgetExceeded) {
		t.Fatalf("result = %#v", result)
	}
}

func TestCompileDraftRejectsTotalDynamicPortBudget(t *testing.T) {
	compiler, snapshot := switchCompiler(t)
	cases := make([]string, MaxDynamicPortsPerNode)
	for index := range cases {
		cases[index] = `"case-` + stringInt(index) + `"`
	}
	nodeCount := MaxTotalDynamicPorts/MaxDynamicPortsPerNode + 1
	nodes := make([]string, nodeCount)
	for index := range nodes {
		nodes[index] = `{"id":"switch-` + stringInt(index) + `","kind":"Switch","position":{"x":0,"y":0},"config":{"Value":"case-0","cases":[` + strings.Join(cases, ",") + `]}}`
	}
	raw := `{"format":"yotta.workflow","version":3,"workflow":{"id":"w","name":"Workflow"},"revision":0,"entryGraph":"main","graphs":[{"id":"main","kind":"main","nodes":[` + strings.Join(nodes, ",") + `],"edges":[],"inputs":[],"outputs":[]}],"variables":[],"secretRefs":[],"requestedCapabilities":[]}`
	result, err := compiler.CompileDraft(context.Background(), CompileRequest{SourceJSON: []byte(raw), Catalog: snapshot})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := result.Program(); ok || !hasDiagnosticCode(result.Diagnostics, schema.CodeDynamicPortBudgetExceeded) {
		t.Fatalf("result = %#v", result)
	}
}

func TestOpenProgramRejectsForgedSwitchDynamicPorts(t *testing.T) {
	compiler, snapshot := switchCompiler(t)
	result, err := compiler.CompileDraft(context.Background(), CompileRequest{
		SourceJSON: []byte(switchSource(`["Ready"]`, "[]")), Catalog: snapshot,
	})
	if err != nil {
		t.Fatal(err)
	}
	program, ok := result.Program()
	if !ok {
		t.Fatalf("diagnostics = %#v", result.Diagnostics)
	}
	var forged programEnvelope
	if err := json.Unmarshal(program.Artifact(), &forged); err != nil {
		t.Fatal(err)
	}
	forged.Program.Graphs[0].Nodes[0].DynamicPorts[0].Name = "Forged"
	if _, err := OpenProgram(rehashEnvelope(t, forged), snapshot, compiler.build); !errors.Is(err, ErrInvalidProgramArtifact) {
		t.Fatalf("forged dynamic ports err = %v", err)
	}
	if err := json.Unmarshal(program.Artifact(), &forged); err != nil {
		t.Fatal(err)
	}
	forged.Program.Graphs[0].Nodes[0].Config["cases"] = []any{"Waiting"}
	if _, err := OpenProgram(rehashEnvelope(t, forged), snapshot, compiler.build); !errors.Is(err, ErrInvalidProgramArtifact) {
		t.Fatalf("forged dynamic config err = %v", err)
	}
	if err := json.Unmarshal(program.Artifact(), &forged); err != nil {
		t.Fatal(err)
	}
	forged.Program.Graphs[0].Nodes[0].DynamicPorts = nil
	if _, err := OpenProgram(rehashEnvelope(t, forged), snapshot, compiler.build); !errors.Is(err, ErrInvalidProgramArtifact) {
		t.Fatalf("nil dynamic ports err = %v", err)
	}
}

func switchCompiler(t testing.TB) (*Compiler, catalog.Snapshot) {
	t.Helper()
	registry := node.NewRegistry()
	registry.Register(control.Switch{})
	registry.Register(compilerTestNode{kind: "sink", execInput: true})
	snapshot, err := catalog.NewSnapshot(registry.Snapshot(), testDigest(t, "switch implementation"))
	if err != nil {
		t.Fatal(err)
	}
	compiler, err := New(testDigest(t, "compiler"))
	if err != nil {
		t.Fatal(err)
	}
	return compiler, snapshot
}

func switchSource(cases, edges string) string {
	return `{"format":"yotta.workflow","version":3,"workflow":{"id":"w","name":"Workflow"},"revision":0,"entryGraph":"main","graphs":[{"id":"main","kind":"main","nodes":[{"id":"switch","kind":"Switch","position":{"x":0,"y":0},"config":{"Value":"Ready","cases":` + cases + `}},{"id":"sink","kind":"sink","position":{"x":1,"y":0},"config":{"Value":1}}],"edges":` + edges + `,"inputs":[],"outputs":[]}],"variables":[],"secretRefs":[],"requestedCapabilities":["runtime:log"]}`
}

func diagnosticReason(diagnostics []schema.Diagnostic, reason string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Params["reason"] == reason || diagnostic.Params["keyword"] == reason {
			return true
		}
	}
	return false
}

func equalProgramPorts(left, right []programPort) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
