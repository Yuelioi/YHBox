package nodes

import (
	"testing"

	"github.com/yottaapp/yotta/internal/nodecontract"
)

func TestAutomationTemplateContractsUseExplicitBlobAndExactTargetCapabilities(t *testing.T) {
	builtins, err := Build()
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		id           string
		execOutputs  []string
		requirements int
	}{
		{WaitTemplateNodeID, []string{"found", "timeout"}, 2},
		{ClickTemplateNodeID, []string{"completed", "timeout"}, 3},
		{WaitTemplateGoneNodeID, []string{"gone", "timeout"}, 2},
	}
	for _, test := range tests {
		definition, ok := builtins.Definition(test.id)
		if !ok {
			t.Fatalf("template definition %q is missing", test.id)
		}
		machine := definition.Contract.Machine()
		if machine.Execution.Class != nodecontract.ExecutionEffect || machine.Execution.Timeout != nodecontract.TimeoutRequired ||
			machine.Instruction.Kind != nodecontract.InstructionInvoke || !signalIDsEqual(machine.Ports.ExecOutputs, test.execOutputs) ||
			!signalIDsEqual(machine.Ports.ErrorOutputs, []string{"failed"}) || len(machine.CapabilityRequirements) != test.requirements {
			t.Fatalf("template contract %q = %#v", test.id, machine)
		}
		inputs := make(map[string]bool, len(machine.Ports.DataInputs))
		for _, input := range machine.Ports.DataInputs {
			inputs[input.ID] = input.Required
		}
		for _, id := range []string{"template", "region", "threshold", "timeout", "poll-interval"} {
			if !inputs[id] {
				t.Fatalf("template contract %q omitted required input %q", test.id, id)
			}
		}
		for _, requirement := range machine.CapabilityRequirements {
			if requirement.ID == "capture-target" || requirement.ID == "input-target" {
				if requirement.TargetSlot != "target" {
					t.Fatalf("template target requirement %q uses slot %q", requirement.ID, requirement.TargetSlot)
				}
			}
		}
		statuses := make(map[string]nodecontract.StatusCategory, len(machine.StatusEvents))
		for _, status := range machine.StatusEvents {
			statuses[status.Code] = status.Category
		}
		if statuses[AutomationTemplateWaitingStatus] != nodecontract.StatusWaiting ||
			statuses[AutomationTemplateMatchedStatus] != nodecontract.StatusProgress ||
			statuses[AutomationTemplateTimeoutStatus] != nodecontract.StatusProgress {
			t.Fatalf("template contract %q statuses = %#v", test.id, statuses)
		}
	}
}
