package nodes31_test

import (
	"testing"

	"github.com/yottaapp/yotta/internal/nodecontract"
	"github.com/yottaapp/yotta/internal/nodes31"
)

func TestLogAndThrowUseExplicitEffectAndTerminalFailureSemantics(t *testing.T) {
	builtins, err := nodes31.Build()
	if err != nil {
		t.Fatal(err)
	}
	logDefinition, ok := builtins.Definition(nodes31.LogNodeID)
	if !ok {
		t.Fatal("log definition is missing")
	}
	logContract := logDefinition.Contract.Machine()
	if logContract.Execution.Class != nodecontract.ExecutionEffect || logContract.Execution.Determinism != nodecontract.Recorded ||
		len(logContract.Execution.Effects) != 1 || logContract.Execution.Effects[0] != nodes31.LogWriteEffectID ||
		len(logContract.Ports.ExecInputs) != 1 || logContract.Ports.ExecInputs[0].ID != "in" ||
		len(logContract.Ports.ExecOutputs) != 1 || logContract.Ports.ExecOutputs[0].ID != "completed" ||
		len(logContract.Ports.ErrorOutputs) != 1 || logContract.Ports.ErrorOutputs[0].ID != "failed" ||
		len(logContract.CapabilityRequirements) != 0 {
		t.Fatalf("log contract = %#v", logContract)
	}
	throwDefinition, ok := builtins.Definition(nodes31.ThrowNodeID)
	if !ok {
		t.Fatal("throw definition is missing")
	}
	throwContract := throwDefinition.Contract.Machine()
	if throwContract.Execution.Class != nodecontract.ExecutionControl || len(throwContract.Execution.Effects) != 0 ||
		len(throwContract.Ports.ExecInputs) != 1 || len(throwContract.Ports.ExecOutputs) != 0 || len(throwContract.Ports.ErrorOutputs) != 0 ||
		len(throwContract.Errors) != 1 || throwContract.Errors[0].Code != nodes31.ControlThrown {
		t.Fatalf("throw contract = %#v", throwContract)
	}
	if builtins.ObservabilityMessageType.TypeRef().TypeID != nodes31.ObservabilityMessageTypeID {
		t.Fatalf("observability message type = %#v", builtins.ObservabilityMessageType.TypeRef())
	}
}
