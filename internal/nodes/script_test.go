package nodes_test

import (
	"testing"

	"github.com/yottaapp/yotta/internal/nodeauthoring"
	"github.com/yottaapp/yotta/internal/nodecontract"
	"github.com/yottaapp/yotta/internal/nodes"
	"github.com/yottaapp/yotta/internal/scriptengine"
)

func TestScriptExecuteContractHasExplicitSignalsAndGeneratedCodeEditor(t *testing.T) {
	builtins, err := nodes.Build()
	if err != nil {
		t.Fatal(err)
	}
	contract := builtins.ScriptExecuteContract.Machine()
	if contract.Execution.Class != nodecontract.ExecutionEffect || contract.Execution.Determinism != nodecontract.Recorded ||
		contract.Execution.Cache != nodecontract.CacheNone || contract.Execution.Retry != nodecontract.RetryNever ||
		contract.Execution.Cancellation != nodecontract.CancellationImmediate || contract.Execution.Timeout != nodecontract.TimeoutRequired {
		t.Fatalf("script execution = %#v", contract.Execution)
	}
	if len(contract.Execution.Effects) != 1 || contract.Execution.Effects[0] != nodes.ScriptExecuteEffectID ||
		len(contract.CapabilityRequirements) != 0 {
		t.Fatalf("script effect/capabilities = %#v / %#v", contract.Execution.Effects, contract.CapabilityRequirements)
	}
	if len(contract.HostFeatureRequirements) != 1 || contract.HostFeatureRequirements[0].ID != "isolation" ||
		contract.HostFeatureRequirements[0].FeatureID != scriptengine.IsolationHostFeatureID {
		t.Fatalf("script host features = %#v", contract.HostFeatureRequirements)
	}
	if len(contract.Ports.DataInputs) != 1 || contract.Ports.DataInputs[0].ID != "input" || contract.Ports.DataInputs[0].Default == nil ||
		len(contract.Ports.DataOutputs) != 1 || contract.Ports.DataOutputs[0].ID != "result" ||
		len(contract.Ports.ExecInputs) != 1 || contract.Ports.ExecInputs[0].ID != "in" ||
		len(contract.Ports.ExecOutputs) != 1 || contract.Ports.ExecOutputs[0].ID != "completed" ||
		len(contract.Ports.ErrorOutputs) != 1 || contract.Ports.ErrorOutputs[0].ID != "failed" {
		t.Fatalf("script ports = %#v", contract.Ports)
	}
	wantCodes := map[string]bool{
		scriptengine.CodeSourceInvalid: true, scriptengine.CodeGuestThrown: true,
		scriptengine.CodeDeadlineExceeded: true, scriptengine.CodeStackExceeded: true,
		scriptengine.CodeContractViolation: true, scriptengine.CodeRunnerProtocolViolation: true,
		scriptengine.CodeRunnerCrashed: true, scriptengine.CodeIsolationUnavailable: true,
	}
	for _, declaration := range contract.Errors {
		delete(wantCodes, declaration.Code)
	}
	if len(wantCodes) != 0 || len(contract.Errors) != 8 {
		t.Fatalf("script errors = %#v, missing %#v", contract.Errors, wantCodes)
	}
	projection, err := nodeauthoring.Project(nodeauthoring.Input{
		Catalog: builtins.Catalog, Types: builtins.Types, Capabilities: builtins.Capabilities,
		Contracts: builtins.Contracts, GeneratorVersion: nodes.GeneratorVersion,
	})
	if err != nil {
		t.Fatal(err)
	}
	projected, ok := projection.Node(nodes.ScriptExecuteNodeID)
	if !ok || len(projected.ConfigFields) != 2 || projected.Availability != nodeauthoring.AvailabilityHostRequired ||
		len(projected.HostFeatures) != 1 || projected.HostFeatures[0].FeatureID != scriptengine.IsolationHostFeatureID {
		t.Fatalf("script projection = %#v", projected)
	}
	controls := map[string]nodeauthoring.Control{}
	for _, field := range projected.ConfigFields {
		controls[field.ID] = field.Control
	}
	if controls["source"] != nodeauthoring.ControlCode || controls["timeoutMilliseconds"] != nodeauthoring.ControlInteger {
		t.Fatalf("script controls = %#v", controls)
	}
}
