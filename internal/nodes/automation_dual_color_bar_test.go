package nodes

import (
	"testing"

	"github.com/yottaapp/yotta/internal/nodecontract"
)

func TestControlDualColorBarIsOneRecordedPushEffect(t *testing.T) {
	builtins, err := Build()
	if err != nil {
		t.Fatal(err)
	}
	definition, ok := builtins.Definition(ControlDualColorBarNodeID)
	if !ok {
		t.Fatal("dual color bar control definition is missing")
	}
	machine := definition.Contract.Machine()
	if machine.Execution.Class != nodecontract.ExecutionEffect || machine.Execution.Evaluation != nodecontract.EvaluationPush ||
		machine.Execution.Determinism != nodecontract.Recorded || len(machine.Execution.Effects) != 1 || machine.Execution.Effects[0] != ControlDualColorBarEffectID {
		t.Fatalf("execution = %#v", machine.Execution)
	}
	if len(machine.ConfiguredTargets) != 1 || machine.ConfiguredTargets[0].TargetSlot != "target" {
		t.Fatalf("configured targets = %#v", machine.ConfiguredTargets)
	}
	if len(machine.Ports.ExecInputs) != 1 || len(machine.Ports.ExecOutputs) != 1 || len(machine.Ports.DataOutputs) != 5 {
		t.Fatalf("ports = %#v", machine.Ports)
	}
	inputs := make(map[string]struct{}, len(machine.Ports.DataInputs))
	for _, input := range machine.Ports.DataInputs {
		inputs[input.ID] = struct{}{}
	}
	for _, id := range []string{"cycle-duration", "activation-keys", "activation-hold-duration", "appearance-poll-duration", "activation-retry-duration", "appearance-timeout"} {
		if _, ok := inputs[id]; !ok {
			t.Fatalf("activation input %q is missing", id)
		}
	}
}
