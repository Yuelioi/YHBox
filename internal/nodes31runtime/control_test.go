package nodes31runtime_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodes31"
	"github.com/yottaapp/yotta/internal/nodes31runtime"
	run31 "github.com/yottaapp/yotta/internal/run"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
)

func TestRunStartedBranchDelayAndStateWriteFormOneExplicitSignalFlow(t *testing.T) {
	builtins, err := nodes31.Build()
	if err != nil {
		t.Fatal(err)
	}
	ref := func(nodeTypeID string) (string, string) {
		definition, ok := builtins.Definition(nodeTypeID)
		if !ok {
			t.Fatalf("missing definition %s", nodeTypeID)
		}
		value := definition.Contract.NodeRef()
		return value.NodeTypeID, value.SemanticDigest.String()
	}
	startedID, startedDigest := ref(nodes31.RunStartedNodeID)
	branchID, branchDigest := ref(nodes31.BranchNodeID)
	delayID, delayDigest := ref(nodes31.DelayNodeID)
	writeID, writeDigest := ref(nodes31.StateWriteNodeID)
	endID, endDigest := ref(nodes31.EndBranchNodeID)
	booleanRef := builtins.BooleanType.TypeRef()
	source := []byte(fmt.Sprintf(`{
		"format":"yotta.workflow","version":"3.1","workflow":{"id":"wf-control","name":"Control"},
		"revision":0,"entryGraph":"main","graphs":[{"id":"main","kind":"main","nodes":[
			{"id":"started","nodeRef":{"nodeTypeId":%q,"semanticDigest":%q},"position":{"x":0,"y":0},"config":{},"bindings":{}},
			{"id":"branch","nodeRef":{"nodeTypeId":%q,"semanticDigest":%q},"position":{"x":1,"y":0},"config":{},"bindings":{"condition":{"kind":"value","value":true}}},
			{"id":"delay","nodeRef":{"nodeTypeId":%q,"semanticDigest":%q},"position":{"x":2,"y":0},"config":{},"bindings":{"duration-milliseconds":{"kind":"value","value":25}}},
			{"id":"write","nodeRef":{"nodeTypeId":%q,"semanticDigest":%q},"position":{"x":3,"y":0},"config":{"variable":"flag"},"bindings":{"value":{"kind":"value","value":true}}},
			{"id":"end","nodeRef":{"nodeTypeId":%q,"semanticDigest":%q},"position":{"x":4,"y":0},"config":{},"bindings":{}}
		],"edges":[
			{"channel":"exec","from":{"nodeId":"started","portId":"started"},"to":{"nodeId":"branch","portId":"in"}},
			{"channel":"exec","from":{"nodeId":"branch","portId":"true"},"to":{"nodeId":"delay","portId":"in"}},
			{"channel":"exec","from":{"nodeId":"delay","portId":"done"},"to":{"nodeId":"write","portId":"in"}},
			{"channel":"exec","from":{"nodeId":"write","portId":"done"},"to":{"nodeId":"end","portId":"in"}}
		],"inputs":[],"outputs":[]}],
		"variables":[{"name":"flag","type":{"kind":"ref","ref":{"typeId":%q,"semanticDigest":%q}},"default":false}],"secretRefs":[]
	}`, startedID, startedDigest, branchID, branchDigest, delayID, delayDigest, writeID, writeDigest, endID, endDigest,
		booleanRef.TypeID, booleanRef.SemanticDigest))
	program := compilePrimitiveProgram(t, builtins, source)
	now := time.Date(2026, 7, 15, 15, 30, 0, 0, time.UTC)
	_, owner, journal := admittedExecution(t, builtins, program, nil, now)
	t.Cleanup(func() { _ = owner.Close(context.Background()) })
	adapters, err := nodes31runtime.Installed(builtins)
	if err != nil {
		t.Fatal(err)
	}
	var waited time.Duration
	execution, err := compiler.NewExecutor(builtins.Catalog, adapters, compiler.ExecutorOptions{
		Now: func() time.Time { return now },
		Wait: func(ctx context.Context, duration time.Duration) error {
			waited = duration
			return ctx.Err()
		},
	}).Run(context.Background(), program, owner, journal)
	if err != nil {
		t.Fatal(err)
	}
	var value bool
	if waited != 25*time.Millisecond || json.Unmarshal(execution.NodeOutputs["write"]["result"].InlineJSON(), &value) != nil || !value {
		t.Fatalf("waited=%s value=%t", waited, value)
	}
	actions, statuses := map[string]bool{}, 0
	for _, fact := range journal.Current().Journal() {
		if fact.Kind == run31.JournalAdapterAction {
			actions[fact.EffectID] = fact.ActionOutcome == run31.ActionSucceeded
		}
		if fact.Kind == run31.JournalNodeStatus && fact.StatusCode == nodes31.DelayWaitingStatus {
			statuses++
		}
	}
	if !actions[nodes31.DelayWaitEffectID] || !actions[nodes31.StateWriteEffectID] || len(actions) != 2 || statuses != 1 {
		t.Fatalf("actions=%#v statuses=%d", actions, statuses)
	}
}

func TestDelayRecordsCooperativeCancellation(t *testing.T) {
	builtins, err := nodes31.Build()
	if err != nil {
		t.Fatal(err)
	}
	adapters, err := nodes31runtime.Installed(builtins)
	if err != nil {
		t.Fatal(err)
	}
	definition, _ := builtins.Definition(nodes31.DelayNodeID)
	adapter := adapters[definition.Implementation.Entrypoint].Run
	duration, err := datatype.SealInlineJSON(
		builtins.Catalog,
		datatype.RefResolvedType(builtins.DurationMillisecondsType.TypeRef()),
		[]byte("25"),
	)
	if err != nil {
		t.Fatal(err)
	}
	var recorded compiler.AdapterAction
	_, runErr := adapter(context.Background(), compiler.Invocation{
		Inputs:     map[string]datatype.ValueEnvelope{"duration-milliseconds": duration},
		Wait:       func(context.Context, time.Duration) error { return context.Canceled },
		EmitStatus: func(context.Context, string, map[string]int64) error { return nil },
		RecordAction: func(_ context.Context, action compiler.AdapterAction) error {
			recorded = action
			return nil
		},
	})
	if !errors.Is(runErr, context.Canceled) || recorded.EffectID != nodes31.DelayWaitEffectID || recorded.Outcome != run31.ActionCancelled {
		t.Fatalf("runErr=%v action=%#v", runErr, recorded)
	}
}
