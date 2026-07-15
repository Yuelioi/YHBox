package nodes31

import (
	"testing"

	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/httpegress"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

func TestHTTPGetContractBindsOneInstalledOriginAndHasNoGenericOut(t *testing.T) {
	builtins, err := Build()
	if err != nil {
		t.Fatal(err)
	}
	machine := builtins.HTTPGetContract.Machine()
	if machine.Execution.Class != nodecontract.ExecutionEffect || machine.Execution.Determinism != nodecontract.Recorded || machine.Execution.Timeout != nodecontract.TimeoutRequired {
		t.Fatalf("execution = %#v", machine.Execution)
	}
	if len(machine.CapabilityRequirements) != 1 || machine.CapabilityRequirements[0].ID != "origin" || machine.CapabilityRequirements[0].Operations[0] != httpegress.OperationGet || len(machine.RequirementBindings) != 1 || machine.RequirementBindings[0].TargetSlotConfigKey != "slot" {
		t.Fatalf("requirements = %#v, bindings = %#v", machine.CapabilityRequirements, machine.RequirementBindings)
	}
	definition, ok := builtins.Catalog.LookupCapability(HTTPGetCapabilityID)
	if !ok || definition.Machine().Consent != capability.ConsentOnce || definition.Machine().Risk != capability.RiskSensitive || definition.Machine().Credential != capability.CredentialNone {
		t.Fatalf("capability = %#v", definition.Machine())
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
