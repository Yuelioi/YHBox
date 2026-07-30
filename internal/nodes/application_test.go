package nodes

import (
	"testing"

	"github.com/yottaapp/yotta/internal/appcontrol"
)

func TestApplicationNodesUseConfiguredTargetsWithoutCapabilities(t *testing.T) {
	builtins, err := Build()
	if err != nil {
		t.Fatal(err)
	}
	launch := builtins.LaunchApplicationContract.Machine()
	if len(launch.Ports.DataInputs) != 0 || len(launch.Ports.DataOutputs) != 0 || launch.Ports.ExecOutputs[0].ID != "completed" || launch.Ports.ErrorOutputs[0].ID != "failed" {
		t.Fatalf("launch ports = %#v", launch.Ports)
	}
	terminate := builtins.TerminateApplicationContract.Machine()
	if len(terminate.Ports.DataOutputs) != 1 || terminate.Ports.DataOutputs[0].ID != "terminated-count" {
		t.Fatalf("terminate ports = %#v", terminate.Ports)
	}
	for _, candidate := range []string{LaunchApplicationNodeID, TerminateApplicationNodeID} {
		entry, ok := builtins.Catalog.Lookup(candidate)
		if !ok {
			t.Fatalf("missing node %s", candidate)
		}
		machine := entry.Contract.Machine()
		if len(machine.CapabilityRequirements) != 0 || len(machine.ConfiguredTargets) != 1 ||
			machine.ConfiguredTargets[0].TargetKinds[0] != appcontrol.TargetKind || machine.ConfiguredTargets[0].SlotConfigKey != "slot" {
			t.Fatalf("configured target for %s = %#v", candidate, machine.ConfiguredTargets)
		}
	}
}
