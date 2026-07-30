package nodes

import (
	"testing"

	"github.com/yottaapp/yotta/internal/httpegress"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

func TestHTTPGetContractUsesConfiguredTargetAndHasNoGenericOut(t *testing.T) {
	builtins, err := Build()
	if err != nil {
		t.Fatal(err)
	}
	machine := builtins.HTTPGetContract.Machine()
	if machine.Execution.Class != nodecontract.ExecutionEffect || machine.Execution.Determinism != nodecontract.Recorded || machine.Execution.Timeout != nodecontract.TimeoutRequired {
		t.Fatalf("execution = %#v", machine.Execution)
	}
	if len(machine.CapabilityRequirements) != 0 || len(machine.ConfiguredTargets) != 1 ||
		machine.ConfiguredTargets[0].TargetKinds[0] != httpegress.TargetKind || machine.ConfiguredTargets[0].SlotConfigKey != "slot" {
		t.Fatalf("configured targets = %#v", machine.ConfiguredTargets)
	}
	if !signalIDsEqual(machine.Ports.ExecInputs, []string{"in"}) || !signalIDsEqual(machine.Ports.ExecOutputs, []string{"completed"}) || !signalIDsEqual(machine.Ports.ErrorOutputs, []string{"failed"}) {
		t.Fatalf("signals = %#v", machine.Ports)
	}
	for _, port := range machine.Ports.DataOutputs {
		if port.ID == "out" {
			t.Fatal("HTTP node exposed generic out")
		}
	}
}
