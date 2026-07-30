package nodes

import "testing"

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
	if len(machine.CapabilityRequirements) != 1 || machine.CapabilityRequirements[0].ID != "blob-read" ||
		len(machine.ConfiguredTargets) != 1 || machine.ConfiguredTargets[0].ID != "target" {
		t.Fatalf("macro dependencies = %#v / %#v", machine.CapabilityRequirements, machine.ConfiguredTargets)
	}
}
