package noderuntime_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/noderuntime"
	"github.com/yottaapp/yotta/internal/nodes"
	run "github.com/yottaapp/yotta/internal/run"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
)

func TestStateReadIsBoundFromProgramStateAndJournaledAsAnEffect(t *testing.T) {
	builtins, err := nodes.Build()
	if err != nil {
		t.Fatal(err)
	}
	definition, _ := builtins.Definition(nodes.StateReadNodeID)
	nodeRef, typeRef := definition.Contract.NodeRef(), builtins.StringType.TypeRef()
	source := []byte(fmt.Sprintf(`{
		"format":"yotta.workflow","version":"3.1","workflow":{"id":"wf-state-read","name":"State read"},
		"revision":0,"entryGraph":"main","graphs":[{"id":"main","kind":"main","nodes":[{
			"id":"read","nodeRef":{"nodeTypeId":%q,"version":"1.0.0","semanticDigest":%q},"position":{"x":0,"y":0},
			"config":{"variable":"message"},"bindings":{}
		}],"edges":[],"inputs":[],"outputs":[]}],
		"variables":[{"name":"message","type":{"kind":"ref","ref":{"typeId":%q,"semanticDigest":%q}},"default":"ready"}],"resources":[],"targetProfileDefinitions":[],"credentialRequirements":[],"dependencies":[]
	}`, nodeRef.NodeTypeID, nodeRef.SemanticDigest, typeRef.TypeID, typeRef.SemanticDigest))
	program := compilePrimitiveProgram(t, builtins, source)
	now := time.Date(2026, 7, 15, 14, 0, 0, 0, time.UTC)
	_, owner, journal := admittedExecution(t, builtins, program, nil, now)
	t.Cleanup(func() { _ = owner.Close(context.Background()) })
	adapters, err := noderuntime.Installed(builtins, testDependencies())
	if err != nil {
		t.Fatal(err)
	}
	execution, err := compiler.NewExecutor(builtins.Catalog, adapters, compiler.ExecutorOptions{Now: func() time.Time { return now }}).
		Run(context.Background(), program, owner, journal)
	if err != nil {
		t.Fatal(err)
	}
	var value string
	if err := json.Unmarshal(execution.NodeOutputs["read"]["result"].InlineJSON(), &value); err != nil || value != "ready" {
		t.Fatalf("state value=%q err=%v", value, err)
	}
	found := false
	for _, fact := range journal.Current().Journal() {
		if fact.Kind == run.JournalAdapterAction && fact.EffectID == nodes.StateReadEffectID && fact.ActionOutcome == run.ActionSucceeded {
			found = true
		}
	}
	if !found {
		t.Fatal("state read did not persist its declared effect")
	}
}
