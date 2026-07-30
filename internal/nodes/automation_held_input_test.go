package nodes

import (
	"slices"
	"testing"

	"github.com/yottaapp/yotta/internal/automation/installed"
	"github.com/yottaapp/yotta/internal/datatype"
)

func TestHeldInputUsesRunOwnedHandleLeaseAndExplicitRelease(t *testing.T) {
	builtins, err := Build()
	if err != nil {
		t.Fatal(err)
	}
	representations := builtins.HeldInputType.Machine().Representations
	if len(representations) != 1 || representations[0].Kind != datatype.RepresentationHandleRef || representations[0].Codec != datatype.CodecHandleRefV1 {
		t.Fatalf("held input representations = %#v", representations)
	}
	for _, nodeID := range []string{HoldKeysNodeID, HoldPointerButtonNodeID} {
		definition, ok := builtins.Definition(nodeID)
		if !ok {
			t.Fatalf("held input definition %q is missing", nodeID)
		}
		machine := definition.Contract.Machine()
		if len(machine.Ports.DataOutputs) != 1 || machine.Ports.DataOutputs[0].ID != "held" || machine.Ports.DataOutputs[0].ResourceLease == nil ||
			!slices.Equal(machine.Ports.DataOutputs[0].ResourceLease.Operations, []string{installed.OperationReleaseHeld}) {
			t.Fatalf("held output for %q = %#v", nodeID, machine.Ports.DataOutputs)
		}
		if len(machine.CapabilityRequirements) != 0 || len(machine.ConfiguredTargets) != 1 ||
			machine.Ports.DataOutputs[0].ResourceLease.TargetID != "target" {
			t.Fatalf("held target for %q = %#v", nodeID, machine)
		}
	}
	release, ok := builtins.Definition(ReleaseHeldInputNodeID)
	if !ok {
		t.Fatal("release held input definition is missing")
	}
	machine := release.Contract.Machine()
	if len(machine.Ports.DataInputs) != 1 || machine.Ports.DataInputs[0].ResourceLease == nil ||
		!slices.Equal(machine.Ports.DataInputs[0].ResourceLease.Operations, []string{installed.OperationReleaseHeld}) ||
		machine.Ports.DataInputs[0].ResourceLease.TargetID != "target" || len(machine.CapabilityRequirements) != 0 {
		t.Fatalf("release held input contract = %#v", machine)
	}
}
