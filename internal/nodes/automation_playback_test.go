package nodes

import (
	"testing"

	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

func TestPlayInputClipUsesNominalBlobAndExclusivePlaybackAuthority(t *testing.T) {
	builtins, err := Build()
	if err != nil {
		t.Fatal(err)
	}
	clip, ok := builtins.Catalog.LookupType(InputClipTypeID)
	if !ok || len(clip.Machine().Representations) != 1 || clip.Machine().Representations[0].Kind != datatype.RepresentationBlobRef {
		t.Fatalf("input clip type = %#v", clip)
	}
	definition, ok := builtins.Definition(PlayInputClipNodeID)
	if !ok {
		t.Fatal("play input clip definition is missing")
	}
	machine := definition.Contract.Machine()
	if machine.Execution.Class != nodecontract.ExecutionEffect || machine.Execution.Determinism != nodecontract.Recorded ||
		len(machine.Execution.Effects) != 1 || machine.Execution.Effects[0] != PlayInputClipEffectID {
		t.Fatalf("execution = %#v", machine.Execution)
	}
	if len(machine.Ports.DataInputs) != 1 || machine.Ports.DataInputs[0].ID != "clip" || !machine.Ports.DataInputs[0].Required ||
		!signalIDsEqual(machine.Ports.ExecInputs, []string{"in"}) || !signalIDsEqual(machine.Ports.ExecOutputs, []string{"completed"}) ||
		!signalIDsEqual(machine.Ports.ErrorOutputs, []string{"failed"}) {
		t.Fatalf("ports = %#v", machine.Ports)
	}
	requirements := map[string]capability.Requirement{}
	for _, requirement := range machine.CapabilityRequirements {
		requirements[requirement.ID] = requirement
	}
	if len(requirements) != 1 || requirements["blob-read"].Capability.CapabilityID != BlobReadCapabilityID {
		t.Fatalf("requirements = %#v", machine.CapabilityRequirements)
	}
	if len(machine.ConfiguredTargets) != 1 || machine.ConfiguredTargets[0].ID != "target" || len(machine.ConfiguredTargets[0].TargetKinds) != 2 {
		t.Fatalf("configured targets = %#v", machine.ConfiguredTargets)
	}
}
