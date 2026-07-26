package nodes

import (
	"fmt"
	"testing"

	"github.com/yottaapp/yotta/internal/automation/installed"
)

func TestPlayMacroUsesDistinctTypeAndSafePlaybackAuthority(t *testing.T) {
	builtins, err := Build()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := builtins.Catalog.LookupType(MacroTypeID); !ok {
		t.Fatal("macro type is missing")
	}
	definition, ok := builtins.Definition(PlayMacroNodeID)
	if !ok {
		t.Fatal("play macro definition is missing")
	}
	machine := definition.Contract.Machine()
	if len(machine.Ports.DataInputs) != 1 || machine.Ports.DataInputs[0].ID != "macro" {
		t.Fatalf("macro ports = %+v", machine.Ports)
	}
	if len(machine.Execution.Effects) != 1 || machine.Execution.Effects[0] != PlayMacroEffectID {
		t.Fatalf("macro execution = %+v", machine.Execution)
	}
	requirements := map[string]struct{ Operations []string }{}
	for _, requirement := range machine.CapabilityRequirements {
		requirements[requirement.ID] = struct{ Operations []string }{requirement.Operations}
	}
	if got := fmt.Sprint(requirements["target"].Operations); got != fmt.Sprint(installed.PlaybackOperations()) {
		t.Fatalf("playback operations = %s", got)
	}
}
