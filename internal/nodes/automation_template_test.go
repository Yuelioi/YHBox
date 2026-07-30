package nodes

import (
	"testing"

	"github.com/yottaapp/yotta/internal/nodecontract"
)

func TestAutomationTemplateContractsUseBlobCapabilityAndConfiguredTargets(t *testing.T) {
	builtins, err := Build()
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		id           string
		execOutputs  []string
		requirements int
		targets      int
	}{
		{WaitTemplateNodeID, []string{"found", "timeout"}, 1, 1},
		{ClickTemplateNodeID, []string{"completed", "timeout"}, 1, 2},
		{WaitTemplateGoneNodeID, []string{"gone", "timeout"}, 1, 1},
	}
	for _, test := range tests {
		definition, ok := builtins.Definition(test.id)
		if !ok {
			t.Fatalf("template definition %q is missing", test.id)
		}
		machine := definition.Contract.Machine()
		if machine.Execution.Class != nodecontract.ExecutionEffect || machine.Execution.Timeout != nodecontract.TimeoutRequired ||
			machine.Instruction.Kind != nodecontract.InstructionInvoke || !signalIDsEqual(machine.Ports.ExecOutputs, test.execOutputs) ||
			!signalIDsEqual(machine.Ports.ErrorOutputs, []string{"failed"}) || len(machine.CapabilityRequirements) != test.requirements ||
			len(machine.ConfiguredTargets) != test.targets {
			t.Fatalf("template contract %q = %#v", test.id, machine)
		}
		inputs := make(map[string]bool, len(machine.Ports.DataInputs))
		defaults := make(map[string]string, len(machine.Ports.DataInputs))
		for _, input := range machine.Ports.DataInputs {
			inputs[input.ID] = input.Required
			if input.Default != nil {
				defaults[input.ID] = string(*input.Default)
			}
		}
		for _, id := range []string{"template", "region", "threshold", "timeout", "poll-interval"} {
			if !inputs[id] {
				t.Fatalf("template contract %q omitted required input %q", test.id, id)
			}
		}
		if test.id == ClickTemplateNodeID {
			required, exists := inputs["image"]
			if !exists || required {
				t.Fatalf("click template source image = exists %v required %v, want optional", exists, required)
			}
		} else if _, exists := inputs["image"]; exists {
			t.Fatalf("template contract %q unexpectedly exposes a fixed source image", test.id)
		}
		if test.id != WaitTemplateGoneNodeID && defaults["settle-duration"] != "0" {
			t.Fatalf("template contract %q settle default = %q, want 0", test.id, defaults["settle-duration"])
		}
		for _, target := range machine.ConfiguredTargets {
			if target.TargetSlot != "target" || target.SlotConfigKey != "slot" {
				t.Fatalf("template configured target %q = %#v", target.ID, target)
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
