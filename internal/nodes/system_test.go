package nodes_test

import (
	"testing"

	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecontract"
	"github.com/yottaapp/yotta/internal/nodes"
)

func TestLogAndThrowUseExplicitEffectAndTerminalFailureSemantics(t *testing.T) {
	builtins, err := nodes.Build()
	if err != nil {
		t.Fatal(err)
	}
	logDefinition, ok := builtins.Definition(nodes.LogNodeID)
	if !ok {
		t.Fatal("log definition is missing")
	}
	logContract := logDefinition.Contract.Machine()
	if logContract.Execution.Class != nodecontract.ExecutionEffect || logContract.Execution.Determinism != nodecontract.Recorded ||
		len(logContract.Execution.Effects) != 1 || logContract.Execution.Effects[0] != nodes.LogWriteEffectID ||
		len(logContract.Ports.ExecInputs) != 1 || logContract.Ports.ExecInputs[0].ID != "in" ||
		len(logContract.Ports.ExecOutputs) != 1 || logContract.Ports.ExecOutputs[0].ID != "completed" ||
		len(logContract.Ports.ErrorOutputs) != 1 || logContract.Ports.ErrorOutputs[0].ID != "failed" ||
		len(logContract.CapabilityRequirements) != 0 {
		t.Fatalf("log contract = %#v", logContract)
	}
	logInput := logContract.Ports.DataInputs[0].Type
	if logInput.Kind != datatype.TypeExpressionVariable || len(logInput.Constraints) != 1 || logInput.Constraints[0] != string(datatype.TraitObservable) {
		t.Fatalf("log input is not constrained observable: %#v", logInput)
	}
	throwDefinition, ok := builtins.Definition(nodes.ThrowNodeID)
	if !ok {
		t.Fatal("throw definition is missing")
	}
	throwContract := throwDefinition.Contract.Machine()
	if throwContract.Execution.Class != nodecontract.ExecutionControl || len(throwContract.Execution.Effects) != 0 ||
		len(throwContract.Ports.ExecInputs) != 1 || len(throwContract.Ports.ExecOutputs) != 0 || len(throwContract.Ports.ErrorOutputs) != 0 ||
		len(throwContract.Errors) != 1 || throwContract.Errors[0].Code != nodes.ControlThrown {
		t.Fatalf("throw contract = %#v", throwContract)
	}
	if builtins.ObservabilityMessageType.TypeRef().TypeID != nodes.ObservabilityMessageTypeID {
		t.Fatalf("observability message type = %#v", builtins.ObservabilityMessageType.TypeRef())
	}
}
