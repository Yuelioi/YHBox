package nodes31

import (
	"testing"

	"github.com/yottaapp/yotta/internal/automation/installed"
	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

func TestActivateWindowUsesDedicatedExactCapability(t *testing.T) {
	builtins, err := Build()
	if err != nil {
		t.Fatal(err)
	}
	definition, ok := builtins.Definition(ActivateWindowNodeID)
	if !ok {
		t.Fatal("activate window definition is missing")
	}
	machine := definition.Contract.Machine()
	if machine.Execution.Class != nodecontract.ExecutionEffect || machine.Execution.Determinism != nodecontract.Recorded ||
		len(machine.Execution.Effects) != 1 || machine.Execution.Effects[0] != ActivateWindowEffectID {
		t.Fatalf("execution = %#v", machine.Execution)
	}
	if len(machine.Ports.DataInputs)+len(machine.Ports.DataOutputs) != 0 ||
		!signalIDsEqual(machine.Ports.ExecInputs, []string{"in"}) || !signalIDsEqual(machine.Ports.ExecOutputs, []string{"completed"}) ||
		!signalIDsEqual(machine.Ports.ErrorOutputs, []string{"failed"}) {
		t.Fatalf("ports = %#v", machine.Ports)
	}
	if len(machine.CapabilityRequirements) != 1 || machine.CapabilityRequirements[0].Capability.CapabilityID != AutomationWindowCapabilityID ||
		len(machine.CapabilityRequirements[0].Operations) != 1 || machine.CapabilityRequirements[0].Operations[0] != installed.OperationActivate {
		t.Fatalf("requirements = %#v", machine.CapabilityRequirements)
	}
	windowCapability, ok := builtins.Catalog.LookupCapability(AutomationWindowCapabilityID)
	if !ok || windowCapability.Machine().Risk != capability.RiskDangerous || windowCapability.Machine().Consent != capability.ConsentOnce {
		t.Fatalf("window capability = %#v", windowCapability.Machine())
	}
	inputCapability, ok := builtins.Catalog.LookupCapability(AutomationInputCapabilityID)
	if !ok {
		t.Fatal("automation input capability is missing")
	}
	for _, operation := range inputCapability.Machine().Operations {
		if operation == installed.OperationActivate {
			t.Fatal("input capability incorrectly grants window activation")
		}
	}
}
