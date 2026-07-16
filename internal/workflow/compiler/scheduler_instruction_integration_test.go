package compiler_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/nodecontract"
	"github.com/yottaapp/yotta/internal/nodes31"
	"github.com/yottaapp/yotta/internal/nodes31runtime"
	"github.com/yottaapp/yotta/internal/resource"
	run31 "github.com/yottaapp/yotta/internal/run"
	"github.com/yottaapp/yotta/internal/runid"
	"github.com/yottaapp/yotta/internal/scriptengine"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
)

func TestSchedulerExecutesCountedAndCollectionRegions(t *testing.T) {
	for _, test := range []struct {
		name         string
		nodeTypeID   string
		bindingPort  string
		bindingValue string
		outputPort   string
		want         string
	}{
		{name: "counted loop", nodeTypeID: nodes31.RepeatNodeID, bindingPort: "count", bindingValue: `3`, outputPort: "index", want: `2`},
		{name: "for each", nodeTypeID: nodes31.ForEachNodeID, bindingPort: "items", bindingValue: `["a","b","c"]`, outputPort: "item", want: `"c"`},
	} {
		t.Run(test.name, func(t *testing.T) {
			builtins := schedulerBuiltins(t)
			started := schedulerNodeRef(t, builtins, nodes31.RunStartedNodeID)
			region := schedulerNodeRef(t, builtins, test.nodeTypeID)
			write := schedulerNodeRef(t, builtins, nodes31.StateWriteNodeID)
			end := schedulerNodeRef(t, builtins, nodes31.EndBranchNodeID)
			valueType := builtins.IntegerType.TypeRef()
			defaultValue := `0`
			if test.nodeTypeID == nodes31.ForEachNodeID {
				valueType = builtins.StringType.TypeRef()
				defaultValue = `""`
			}
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
					{"channel":"exec","from":{"nodeId":"write","portId":"done"},"to":{"nodeId":"end","portId":"in"}},
					{"channel":"data","from":{"nodeId":"region","portId":%q},"to":{"nodeId":"write","portId":"value"}}
				],"inputs":[],"outputs":[]}],
				"variables":[{"name":"value","type":{"kind":"ref","ref":{"typeId":%q,"semanticDigest":%q}},"default":%s}],"secretRefs":[]
			}`, started.NodeTypeID, started.SemanticDigest, region.NodeTypeID, region.SemanticDigest,
				test.bindingPort, test.bindingValue, write.NodeTypeID, write.SemanticDigest,
				end.NodeTypeID, end.SemanticDigest, test.outputPort, valueType.TypeID, valueType.SemanticDigest, defaultValue))
			program := compileSchedulerInstructionProgram(t, builtins, source)
			execution, journal := runSchedulerInstructionProgram(t, builtins, program, compiler.ExecutorOptions{})
			got := execution.NodeOutputs["region"][test.outputPort]
			if !got.Valid() || string(got.InlineJSON()) != test.want {
				t.Fatalf("%s output = %s, want %s", test.outputPort, got.InlineJSON(), test.want)
			}
			if journal.Current().Status() != run31.StatusSucceeded {
				t.Fatalf("Run status = %s", journal.Current().Status())
			}
		})
	}
}

func TestSchedulerRetriesOnlyExplicitRoutedFailure(t *testing.T) {
	builtins := schedulerBuiltins(t)
	started := schedulerNodeRef(t, builtins, nodes31.RunStartedNodeID)
	retry := schedulerNodeRef(t, builtins, nodes31.RetryNodeID)
	delay := schedulerNodeRef(t, builtins, nodes31.DelayNodeID)
	end := schedulerNodeRef(t, builtins, nodes31.EndBranchNodeID)
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
	}`, started.NodeTypeID, started.SemanticDigest, retry.NodeTypeID, retry.SemanticDigest,
		delay.NodeTypeID, delay.SemanticDigest, end.NodeTypeID, end.SemanticDigest))
	program := compileSchedulerInstructionProgram(t, builtins, source)
	waits := 0
	execution, journal := runSchedulerInstructionProgram(t, builtins, program, compiler.ExecutorOptions{
		Wait: func(context.Context, time.Duration) error {
			waits++
			if waits < 3 {
				return errors.New("transient wait failure")
			}
			return nil
		},
	})
	var attempt int64
	if err := json.Unmarshal(execution.NodeOutputs["retry"]["attempt"].InlineJSON(), &attempt); err != nil || waits != 3 || attempt != 3 {
		t.Fatalf("waits=%d attempt=%d error=%v", waits, attempt, err)
	}
	if journal.Current().Status() != run31.StatusSucceeded {
		t.Fatalf("Run status = %s", journal.Current().Status())
	}
}

type schedulerUnusedScriptRuntime struct{}

func (schedulerUnusedScriptRuntime) Execute(context.Context, scriptengine.Request) (scriptengine.Response, error) {
	return scriptengine.Response{}, errors.New("unexpected script execution")
}

type schedulerUnusedLogEmitter struct{}

func (schedulerUnusedLogEmitter) EmitWorkflowLog(context.Context, nodes31runtime.LogEntry) error {
	return errors.New("unexpected log emission")
}

func schedulerBuiltins(t *testing.T) nodes31.Builtins {
	t.Helper()
	builtins, err := nodes31.Build()
	if err != nil {
		t.Fatal(err)
	}
	return builtins
}

func schedulerNodeRef(t *testing.T, builtins nodes31.Builtins, nodeTypeID string) nodecontract.NodeRef {
	t.Helper()
	definition, ok := builtins.Definition(nodeTypeID)
	if !ok {
		t.Fatalf("node definition %q is missing", nodeTypeID)
	}
	return definition.Contract.NodeRef()
}

func compileSchedulerInstructionProgram(t *testing.T, builtins nodes31.Builtins, source []byte) compiler.ProgramSnapshot {
	t.Helper()
	build, err := artifact.Sum("yotta/test/compiler-build/v1", []byte(t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	result, err := compiler.New(build, builtins.ConfigValidators).CompileDraft(context.Background(), compiler.CompileRequest{
		SourceJSON: source,
		Catalog:    builtins.Catalog,
	})
	if err != nil || len(result.Diagnostics) != 0 {
		t.Fatalf("CompileDraft() error=%v diagnostics=%#v", err, result.Diagnostics)
	}
	program, ok := result.Program()
	if !ok {
		t.Fatal("CompileDraft did not produce a Program")
	}
	return program
}

func runSchedulerInstructionProgram(t *testing.T, builtins nodes31.Builtins, program compiler.ProgramSnapshot, options compiler.ExecutorOptions) (compiler.ExecutionResult, *run31.JournalWriter) {
	t.Helper()
	now := time.Date(2026, 7, 17, 2, 0, 0, 0, time.UTC)
	id, err := runid.New()
	if err != nil {
		t.Fatal(err)
	}
	grant, err := capability.SealRunGrant(capability.GrantRequest{
		ProgramHash: program.Hash(), Plan: program.CapabilityPlan(), RunID: id, Principal: "test-user",
		PolicyGeneration: "policy-1", IssuedAt: now, ExpiresAt: now.Add(time.Minute), Bindings: []capability.Binding{},
	}, builtins.Catalog)
	if err != nil {
		t.Fatal(err)
	}
	queued, err := run31.NewQueuedRecord(run31.QueueRequest{
		ProgramHash: program.Hash(), CatalogHash: builtins.Catalog.Hash(), CapabilityPlanDigest: program.CapabilityPlan().Digest(),
		Grant: grant, QueuedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	store, err := run31.OpenStore(t.TempDir(), builtins.Catalog, run31.StoreOptions{MaxRecords: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Create(context.Background(), queued); err != nil {
		t.Fatal(err)
	}
	running, err := queued.Start(now)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Update(context.Background(), queued.Digest(), running); err != nil {
		t.Fatal(err)
	}
	journal, err := store.OpenJournal(grant.RunID())
	if err != nil {
		t.Fatal(err)
	}
	owner, err := run31.NewOwner(context.Background(), grant, map[string]run31.InstalledProvider{}, resource.Options{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = owner.Close(context.Background()) })
	adapters, err := nodes31runtime.Installed(builtins, nodes31runtime.Dependencies{
		Script: schedulerUnusedScriptRuntime{},
		Log:    schedulerUnusedLogEmitter{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if options.Now == nil {
		options.Now = func() time.Time { return now }
	}
	execution, err := compiler.NewExecutor(builtins.Catalog, adapters, options).Run(context.Background(), program, owner, journal)
	if err != nil {
		t.Fatal(err)
	}
	return execution, journal
}
