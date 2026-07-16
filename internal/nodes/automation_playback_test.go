package nodes

import (
	"testing"

	"github.com/yottaapp/yotta/internal/automation/installed"
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
	if len(requirements) != 2 || requirements["target"].Capability.CapabilityID != AutomationPlaybackCapabilityID ||
		requirements["blob-read"].Capability.CapabilityID != BlobReadCapabilityID {
		t.Fatalf("requirements = %#v", machine.CapabilityRequirements)
	}
	if got := requirements["target"].Operations; len(got) != 2 || got[0] != installed.OperationPlayEvent || got[1] != installed.OperationReleaseHeld {
		t.Fatalf("playback operations = %v", got)
	}
	playback, ok := builtins.Catalog.LookupCapability(AutomationPlaybackCapabilityID)
	if !ok || playback.Machine().Risk != capability.RiskDangerous || playback.Machine().Consent != capability.ConsentOnce {
		t.Fatalf("playback capability = %#v", playback.Machine())
	}
	for _, capabilityID := range []string{AutomationInputCapabilityID, AutomationWindowCapabilityID, AutomationCaptureCapabilityID} {
		other, _ := builtins.Catalog.LookupCapability(capabilityID)
		for _, operation := range other.Machine().Operations {
			if operation == installed.OperationPlayEvent || operation == installed.OperationReleaseHeld {
				t.Fatalf("%s incorrectly grants playback operation %q", capabilityID, operation)
			}
		}
	}
}
