package noderuntime_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/noderuntime"
	"github.com/yottaapp/yotta/internal/nodes"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
)

func TestTypedSelectResolvesFromItsConsumerAndExecutesWithoutCoercion(t *testing.T) {
	builtins, err := nodes.Build()
	if err != nil {
		t.Fatal(err)
	}
	selectDefinition, _ := builtins.Definition(nodes.SelectNodeID)
	concatDefinition, _ := builtins.Definition(nodes.ConcatNodeID)
	source := []byte(fmt.Sprintf(`{
		"format":"yotta.workflow","version":"1","workflow":{"id":"wf-select","name":"Select"},
		"revision":0,"entryGraph":"main","graphs":[{"id":"main","kind":"main","nodes":[
			{"id":"select","nodeRef":{"nodeTypeId":%q,"version":"1.0.0","semanticDigest":%q},"position":{"x":0,"y":0},"config":{},
			 "bindings":{"condition":{"kind":"value","value":true},"when_true":{"kind":"value","value":"typed"},"when_false":{"kind":"value","value":"wrong"}}},
			{"id":"concat","nodeRef":{"nodeTypeId":%q,"version":"1.0.0","semanticDigest":%q},"position":{"x":1,"y":0},"config":{},
			 "bindings":{"b":{"kind":"value","value":"!"}}}
		],"edges":[{"channel":"data","from":{"nodeId":"select","portId":"result"},"to":{"nodeId":"concat","portId":"a"}}],
		"inputs":[],"outputs":[]}],"variables":[],"resources":[],"targetProfileDefinitions":[],"credentialRequirements":[],"dependencies":[]
	}`, selectDefinition.Contract.NodeRef().NodeTypeID, selectDefinition.Contract.NodeRef().SemanticDigest,
		concatDefinition.Contract.NodeRef().NodeTypeID, concatDefinition.Contract.NodeRef().SemanticDigest))
	program := compilePrimitiveProgram(t, builtins, source)
	for _, view := range program.Nodes() {
		if view.ID != "select" {
			continue
		}
		for _, portID := range []string{"when_true", "when_false"} {
			resolved := view.InputTypes[portID]
			if resolved.Kind != datatype.ResolvedTypeRef || resolved.Ref == nil || *resolved.Ref != builtins.StringType.TypeRef() {
				t.Fatalf("%s effective type = %#v", portID, resolved)
			}
		}
	}
	now := time.Date(2026, 7, 15, 13, 0, 0, 0, time.UTC)
	_, owner, journal := admittedExecution(t, builtins, program, nil, now)
	t.Cleanup(func() { _ = owner.Close(context.Background()) })
	adapters, err := noderuntime.Installed(builtins, testDependencies())
	if err != nil {
		t.Fatal(err)
	}
	execution, err := compiler.NewExecutor(builtins.Catalog, adapters, compiler.ExecutorOptions{Now: func() time.Time { return now.Add(time.Second) }}).Run(context.Background(), program, owner, journal)
	if err != nil {
		t.Fatal(err)
	}
	var result string
	if err := json.Unmarshal(execution.NodeOutputs["concat"]["result"].InlineJSON(), &result); err != nil || result != "typed!" {
		t.Fatalf("result = %q, %v", result, err)
	}
}
