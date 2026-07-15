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

func TestInstalledAdaptersExcludeHostLoweredInstructions(t *testing.T) {
	builtins, err := nodes31.Build()
	if err != nil {
		t.Fatal(err)
	}
	installed, err := nodes31runtime.Installed(builtins, testDependencies())
	if err != nil {
		t.Fatal(err)
	}
	for _, nodeTypeID := range []string{
		nodes31.RunStartedNodeID,
		nodes31.RepeatNodeID,
		nodes31.ForEachNodeID,
		nodes31.RetryNodeID,
	} {
		definition, ok := builtins.Definition(nodeTypeID)
		if !ok {
			t.Fatalf("definition %s is missing", nodeTypeID)
		}
		if _, exists := installed[definition.Implementation.Entrypoint]; exists {
			t.Fatalf("host-lowered instruction %s installed an adapter", nodeTypeID)
		}
	}
}

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
	adapters, err := nodes31runtime.Installed(builtins, testDependencies())
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
	adapters, err := nodes31runtime.Installed(builtins, testDependencies())
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

func TestRepeatAndForEachUseIsolatedActivationState(t *testing.T) {
	for _, test := range []struct {
		name         string
		regionNodeID string
		variableType func(nodes31.Builtins) datatype.TypeRef
		bindingPort  string
		bindingValue string
		outputPort   string
		want         any
	}{
		{name: "repeat", regionNodeID: nodes31.RepeatNodeID, variableType: func(b nodes31.Builtins) datatype.TypeRef { return b.IntegerType.TypeRef() }, bindingPort: "count", bindingValue: "3", outputPort: "index", want: int64(2)},
		{name: "for-each", regionNodeID: nodes31.ForEachNodeID, variableType: func(b nodes31.Builtins) datatype.TypeRef { return b.StringType.TypeRef() }, bindingPort: "items", bindingValue: `["a","b","c"]`, outputPort: "item", want: "c"},
	} {
		t.Run(test.name, func(t *testing.T) {
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
			startID, startDigest := ref(nodes31.RunStartedNodeID)
			regionID, regionDigest := ref(test.regionNodeID)
			writeID, writeDigest := ref(nodes31.StateWriteNodeID)
			endID, endDigest := ref(nodes31.EndBranchNodeID)
			typeRef := test.variableType(builtins)
			source := []byte(fmt.Sprintf(`{
				"format":"yotta.workflow","version":"3.1","workflow":{"id":"wf-region","name":"Region"},
				"revision":0,"entryGraph":"main","graphs":[{"id":"main","kind":"main","nodes":[
					{"id":"started","nodeRef":{"nodeTypeId":%q,"semanticDigest":%q},"position":{"x":0,"y":0},"config":{},"bindings":{}},
					{"id":"region","nodeRef":{"nodeTypeId":%q,"semanticDigest":%q},"position":{"x":1,"y":0},"config":{},"bindings":{%q:{"kind":"value","value":%s}}},
					{"id":"write","nodeRef":{"nodeTypeId":%q,"semanticDigest":%q},"position":{"x":2,"y":0},"config":{"variable":"value"},"bindings":{}},
					{"id":"end","nodeRef":{"nodeTypeId":%q,"semanticDigest":%q},"position":{"x":3,"y":0},"config":{},"bindings":{}}
				],"edges":[
					{"channel":"exec","from":{"nodeId":"started","portId":"started"},"to":{"nodeId":"region","portId":"in"}},
					{"channel":"exec","from":{"nodeId":"region","portId":"body"},"to":{"nodeId":"write","portId":"in"}},
					{"channel":"exec","from":{"nodeId":"region","portId":"completed"},"to":{"nodeId":"end","portId":"in"}},
					{"channel":"data","from":{"nodeId":"region","portId":%q},"to":{"nodeId":"write","portId":"value"}}
				],"inputs":[],"outputs":[]}],
				"variables":[{"name":"value","type":{"kind":"ref","ref":{"typeId":%q,"semanticDigest":%q}},"default":%s}],"secretRefs":[]
			}`, startID, startDigest, regionID, regionDigest, test.bindingPort, test.bindingValue, writeID, writeDigest, endID, endDigest,
				test.outputPort, typeRef.TypeID, typeRef.SemanticDigest, initialJSON(test.want)))
			program := compilePrimitiveProgram(t, builtins, source)
			now := time.Date(2026, 7, 15, 16, 0, 0, 0, time.UTC)
			_, owner, journal := admittedExecution(t, builtins, program, nil, now)
			t.Cleanup(func() { _ = owner.Close(context.Background()) })
			adapters, err := nodes31runtime.Installed(builtins, testDependencies())
			if err != nil {
				t.Fatal(err)
			}
			execution, err := compiler.NewExecutor(builtins.Catalog, adapters, compiler.ExecutorOptions{Now: func() time.Time { return now }}).
				Run(context.Background(), program, owner, journal)
			if err != nil {
				t.Fatal(err)
			}
			var got any
			if err := json.Unmarshal(execution.NodeOutputs["write"]["result"].InlineJSON(), &got); err != nil {
				t.Fatal(err)
			}
			if fmt.Sprint(got) != fmt.Sprint(test.want) {
				t.Fatalf("final region value = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestRetryConsumesOnlyExplicitlyRoutedFailuresInsideItsActivation(t *testing.T) {
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
	startID, startDigest := ref(nodes31.RunStartedNodeID)
	retryID, retryDigest := ref(nodes31.RetryNodeID)
	delayID, delayDigest := ref(nodes31.DelayNodeID)
	endID, endDigest := ref(nodes31.EndBranchNodeID)
	source := []byte(fmt.Sprintf(`{
		"format":"yotta.workflow","version":"3.1","workflow":{"id":"wf-retry","name":"Retry"},
		"revision":0,"entryGraph":"main","graphs":[{"id":"main","kind":"main","nodes":[
			{"id":"started","nodeRef":{"nodeTypeId":%q,"semanticDigest":%q},"position":{"x":0,"y":0},"config":{},"bindings":{}},
			{"id":"retry","nodeRef":{"nodeTypeId":%q,"semanticDigest":%q},"position":{"x":1,"y":0},"config":{},"bindings":{"attempts":{"kind":"value","value":3}}},
			{"id":"delay","nodeRef":{"nodeTypeId":%q,"semanticDigest":%q},"position":{"x":2,"y":0},"config":{},"bindings":{"duration-milliseconds":{"kind":"value","value":1}}},
			{"id":"end","nodeRef":{"nodeTypeId":%q,"semanticDigest":%q},"position":{"x":3,"y":0},"config":{},"bindings":{}}
		],"edges":[
			{"channel":"exec","from":{"nodeId":"started","portId":"started"},"to":{"nodeId":"retry","portId":"in"}},
			{"channel":"exec","from":{"nodeId":"retry","portId":"body"},"to":{"nodeId":"delay","portId":"in"}},
			{"channel":"error","from":{"nodeId":"delay","portId":"failed"},"to":{"nodeId":"retry","portId":"retry"}},
			{"channel":"exec","from":{"nodeId":"retry","portId":"completed"},"to":{"nodeId":"end","portId":"in"}},
			{"channel":"exec","from":{"nodeId":"retry","portId":"exhausted"},"to":{"nodeId":"end","portId":"in"}}
		],"inputs":[],"outputs":[]}],"variables":[],"secretRefs":[]
	}`, startID, startDigest, retryID, retryDigest, delayID, delayDigest, endID, endDigest))
	program := compilePrimitiveProgram(t, builtins, source)
	now := time.Date(2026, 7, 15, 16, 30, 0, 0, time.UTC)
	_, owner, journal := admittedExecution(t, builtins, program, nil, now)
	t.Cleanup(func() { _ = owner.Close(context.Background()) })
	adapters, err := nodes31runtime.Installed(builtins, testDependencies())
	if err != nil {
		t.Fatal(err)
	}
	waits := 0
	execution, err := compiler.NewExecutor(builtins.Catalog, adapters, compiler.ExecutorOptions{
		Now: func() time.Time { return now },
		Wait: func(context.Context, time.Duration) error {
			waits++
			if waits < 3 {
				return errors.New("transient wait failure")
			}
			return nil
		},
	}).Run(context.Background(), program, owner, journal)
	if err != nil {
		t.Fatal(err)
	}
	var attempt int64
	if err := json.Unmarshal(execution.NodeOutputs["retry"]["attempt"].InlineJSON(), &attempt); err != nil || waits != 3 || attempt != 3 {
		t.Fatalf("waits=%d attempt=%d err=%v", waits, attempt, err)
	}
}

func TestNestedRegionSignalPropagatesToItsExactOwner(t *testing.T) {
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
	startID, startDigest := ref(nodes31.RunStartedNodeID)
	repeatID, repeatDigest := ref(nodes31.RepeatNodeID)
	branchID, branchDigest := ref(nodes31.BranchNodeID)
	writeID, writeDigest := ref(nodes31.StateWriteNodeID)
	endID, endDigest := ref(nodes31.EndBranchNodeID)
	integerRef := builtins.IntegerType.TypeRef()
	source := []byte(fmt.Sprintf(`{
		"format":"yotta.workflow","version":"3.1","workflow":{"id":"wf-nested-region","name":"Nested region"},
		"revision":0,"entryGraph":"main","graphs":[{"id":"main","kind":"main","nodes":[
			{"id":"started","nodeRef":{"nodeTypeId":%q,"semanticDigest":%q},"position":{"x":0,"y":0},"config":{},"bindings":{}},
			{"id":"outer","nodeRef":{"nodeTypeId":%q,"semanticDigest":%q},"position":{"x":1,"y":0},"config":{},"bindings":{"count":{"kind":"value","value":3}}},
			{"id":"inner","nodeRef":{"nodeTypeId":%q,"semanticDigest":%q},"position":{"x":2,"y":0},"config":{},"bindings":{"count":{"kind":"value","value":2}}},
			{"id":"branch","nodeRef":{"nodeTypeId":%q,"semanticDigest":%q},"position":{"x":3,"y":0},"config":{},"bindings":{"condition":{"kind":"value","value":true}}},
			{"id":"write","nodeRef":{"nodeTypeId":%q,"semanticDigest":%q},"position":{"x":4,"y":0},"config":{"variable":"value"},"bindings":{}},
			{"id":"end","nodeRef":{"nodeTypeId":%q,"semanticDigest":%q},"position":{"x":5,"y":0},"config":{},"bindings":{}}
		],"edges":[
			{"channel":"exec","from":{"nodeId":"started","portId":"started"},"to":{"nodeId":"outer","portId":"in"}},
			{"channel":"exec","from":{"nodeId":"outer","portId":"body"},"to":{"nodeId":"inner","portId":"in"}},
			{"channel":"exec","from":{"nodeId":"inner","portId":"body"},"to":{"nodeId":"branch","portId":"in"}},
			{"channel":"exec","from":{"nodeId":"branch","portId":"true"},"to":{"nodeId":"outer","portId":"break"}},
			{"channel":"exec","from":{"nodeId":"outer","portId":"completed"},"to":{"nodeId":"write","portId":"in"}},
			{"channel":"exec","from":{"nodeId":"write","portId":"done"},"to":{"nodeId":"end","portId":"in"}},
			{"channel":"data","from":{"nodeId":"outer","portId":"index"},"to":{"nodeId":"write","portId":"value"}}
		],"inputs":[],"outputs":[]}],
		"variables":[{"name":"value","type":{"kind":"ref","ref":{"typeId":%q,"semanticDigest":%q}},"default":-1}],"secretRefs":[]
	}`, startID, startDigest, repeatID, repeatDigest, repeatID, repeatDigest, branchID, branchDigest,
		writeID, writeDigest, endID, endDigest, integerRef.TypeID, integerRef.SemanticDigest))
	program := compilePrimitiveProgram(t, builtins, source)
	now := time.Date(2026, 7, 15, 16, 45, 0, 0, time.UTC)
	_, owner, journal := admittedExecution(t, builtins, program, nil, now)
	t.Cleanup(func() { _ = owner.Close(context.Background()) })
	adapters, err := nodes31runtime.Installed(builtins, testDependencies())
	if err != nil {
		t.Fatal(err)
	}
	execution, err := compiler.NewExecutor(builtins.Catalog, adapters, compiler.ExecutorOptions{Now: func() time.Time { return now }}).
		Run(context.Background(), program, owner, journal)
	if err != nil {
		t.Fatal(err)
	}
	var value int64
	if err := json.Unmarshal(execution.NodeOutputs["write"]["result"].InlineJSON(), &value); err != nil || value != 0 {
		t.Fatalf("outer break value=%d err=%v", value, err)
	}
}

func TestCompilerRejectsRetrySignalsOutsideTheirActivationOrOnTheWrongChannel(t *testing.T) {
	builtins, err := nodes31.Build()
	if err != nil {
		t.Fatal(err)
	}
	ref := func(nodeTypeID string) (string, string) {
		definition, _ := builtins.Definition(nodeTypeID)
		value := definition.Contract.NodeRef()
		return value.NodeTypeID, value.SemanticDigest.String()
	}
	startID, startDigest := ref(nodes31.RunStartedNodeID)
	retryID, retryDigest := ref(nodes31.RetryNodeID)
	delayID, delayDigest := ref(nodes31.DelayNodeID)
	compile := func(edge string) compiler.CompileResult {
		t.Helper()
		source := []byte(fmt.Sprintf(`{
			"format":"yotta.workflow","version":"3.1","workflow":{"id":"wf-retry-scope","name":"Retry scope"},
			"revision":0,"entryGraph":"main","graphs":[{"id":"main","kind":"main","nodes":[
				{"id":"started","nodeRef":{"nodeTypeId":%q,"semanticDigest":%q},"position":{"x":0,"y":0},"config":{},"bindings":{}},
				{"id":"retry","nodeRef":{"nodeTypeId":%q,"semanticDigest":%q},"position":{"x":1,"y":0},"config":{},"bindings":{"attempts":{"kind":"value","value":3}}},
				{"id":"delay","nodeRef":{"nodeTypeId":%q,"semanticDigest":%q},"position":{"x":2,"y":0},"config":{},"bindings":{"duration-milliseconds":{"kind":"value","value":1}}}
			],"edges":[
				{"channel":"exec","from":{"nodeId":"started","portId":"started"},"to":{"nodeId":"retry","portId":"in"}},
				{"channel":"exec","from":{"nodeId":"started","portId":"started"},"to":{"nodeId":"delay","portId":"in"}},%s
			],"inputs":[],"outputs":[]}],"variables":[],"secretRefs":[]
		}`, startID, startDigest, retryID, retryDigest, delayID, delayDigest, edge))
		build, err := compiler.BuildDigest()
		if err != nil {
			t.Fatal(err)
		}
		result, err := compiler.New(build, builtins.ConfigValidators).CompileDraft(context.Background(), compiler.CompileRequest{SourceJSON: source, Catalog: builtins.Catalog})
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	outside := compile(`{"channel":"error","from":{"nodeId":"delay","portId":"failed"},"to":{"nodeId":"retry","portId":"retry"}}`)
	if !compileHasDiagnostic(outside, compiler.CodeRegionSignalScope) {
		t.Fatalf("outside diagnostics = %#v", outside.Diagnostics)
	}
	wrongChannel := compile(`{"channel":"exec","from":{"nodeId":"delay","portId":"done"},"to":{"nodeId":"retry","portId":"retry"}}`)
	if !compileHasDiagnostic(wrongChannel, compiler.CodeEdgeChannelMismatch) {
		t.Fatalf("wrong-channel diagnostics = %#v", wrongChannel.Diagnostics)
	}
	ambiguous := compile(`
		{"channel":"exec","from":{"nodeId":"retry","portId":"body"},"to":{"nodeId":"delay","portId":"in"}},
		{"channel":"error","from":{"nodeId":"delay","portId":"failed"},"to":{"nodeId":"retry","portId":"retry"}}
	`)
	if !compileHasDiagnostic(ambiguous, compiler.CodeRegionSignalScope) {
		t.Fatalf("multiply-entered body diagnostics = %#v", ambiguous.Diagnostics)
	}
}

func compileHasDiagnostic(result compiler.CompileResult, code string) bool {
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}

func initialJSON(value any) string {
	switch value.(type) {
	case string:
		return `""`
	default:
		return "0"
	}
}
