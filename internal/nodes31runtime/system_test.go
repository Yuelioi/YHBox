package nodes31runtime_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodes31"
	"github.com/yottaapp/yotta/internal/nodes31runtime"
	run31 "github.com/yottaapp/yotta/internal/run"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
)

func TestLogAdapterEmitsAttributedMessageAndJournalsOnlyDigest(t *testing.T) {
	builtins, err := nodes31.Build()
	if err != nil {
		t.Fatal(err)
	}
	var emitted nodes31runtime.LogEntry
	installed, err := nodes31runtime.Installed(builtins, nodes31runtime.Dependencies{
		Script: unusedScriptRuntime{},
		Log: nodes31runtime.LogEmitterFunc(func(_ context.Context, entry nodes31runtime.LogEntry) error {
			emitted = entry
			return nil
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	definition, _ := builtins.Definition(nodes31.LogNodeID)
	message, err := datatype.SealInlineJSON(
		builtins.Catalog, datatype.RefResolvedType(builtins.ObservabilityMessageType.TypeRef()), json.RawMessage(`"private message"`),
	)
	if err != nil {
		t.Fatal(err)
	}
	var action compiler.AdapterAction
	result, err := installed[definition.Implementation.Entrypoint].Run(context.Background(), compiler.Invocation{
		InvocationID: "invocation-1", Attempt: 2, GraphID: "main", NodeID: "log",
		Config: map[string]any{"level": "warn"}, Inputs: map[string]datatype.ValueEnvelope{"message": message},
		RecordAction: func(_ context.Context, recorded compiler.AdapterAction) error { action = recorded; return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	if emitted.Level != "warn" || emitted.Message != "private message" || emitted.GraphID != "main" || emitted.NodeID != "log" || emitted.InvocationID != "invocation-1" || emitted.Attempt != 2 {
		t.Fatalf("emitted log = %#v", emitted)
	}
	if len(result.ExecOutputs) != 1 || result.ExecOutputs[0] != "completed" || action.Outcome != run31.ActionSucceeded ||
		action.EffectID != nodes31.LogWriteEffectID || action.SummaryCode != "observability.log" ||
		action.Counters["message_bytes"] != 15 || action.Facts["level"] != "warn" || action.Facts["message_digest"] == "" {
		t.Fatalf("log result/action = %#v / %#v", result, action)
	}
	encoded, err := json.Marshal(action)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encoded, []byte("private message")) {
		t.Fatalf("log journal leaked message: %s", encoded)
	}
}

func TestLogAndThrowReturnStableNodeFailures(t *testing.T) {
	builtins, err := nodes31.Build()
	if err != nil {
		t.Fatal(err)
	}
	installed, err := nodes31runtime.Installed(builtins, nodes31runtime.Dependencies{
		Script: unusedScriptRuntime{},
		Log:    nodes31runtime.LogEmitterFunc(func(context.Context, nodes31runtime.LogEntry) error { return errors.New("sink unavailable") }),
	})
	if err != nil {
		t.Fatal(err)
	}
	message, err := datatype.SealInlineJSON(
		builtins.Catalog, datatype.RefResolvedType(builtins.ObservabilityMessageType.TypeRef()), json.RawMessage(`"stop here"`),
	)
	if err != nil {
		t.Fatal(err)
	}
	logDefinition, _ := builtins.Definition(nodes31.LogNodeID)
	_, logErr := installed[logDefinition.Implementation.Entrypoint].Run(context.Background(), compiler.Invocation{
		InvocationID: "invocation-2", Attempt: 1, GraphID: "main", NodeID: "log",
		Inputs:       map[string]datatype.ValueEnvelope{"message": message},
		RecordAction: func(context.Context, compiler.AdapterAction) error { return nil },
	})
	var failure *compiler.NodeFailure
	if !errors.As(logErr, &failure) || failure.Code != nodes31.LogWriteFailed || failure.Output != "failed" {
		t.Fatalf("log failure = %#v / %v", failure, logErr)
	}
	throwDefinition, _ := builtins.Definition(nodes31.ThrowNodeID)
	_, throwErr := installed[throwDefinition.Implementation.Entrypoint].Run(context.Background(), compiler.Invocation{
		InvocationID: "invocation-3", Attempt: 1, GraphID: "main", NodeID: "throw",
		Inputs: map[string]datatype.ValueEnvelope{"message": message},
	})
	failure = nil
	if !errors.As(throwErr, &failure) || failure.Code != nodes31.ControlThrown || failure.Output != "" || !bytes.Contains([]byte(throwErr.Error()), []byte("stop here")) {
		t.Fatalf("throw failure = %#v / %v", failure, throwErr)
	}
}
