package nodes

import (
	"testing"

	"github.com/yottaapp/yotta/internal/appcontrol"
	"github.com/yottaapp/yotta/internal/capability"
)

func TestApplicationNodesUseExactDangerousLifecycleCapability(t *testing.T) {
	builtins, err := Build()
	if err != nil {
		t.Fatal(err)
	}
	definition, ok := builtins.Catalog.LookupCapability(ApplicationLifecycleCapabilityID)
	if !ok {
		t.Fatal("application lifecycle capability is missing")
	}
	machine := definition.Machine()
	if machine.Risk != capability.RiskDangerous || machine.Consent != capability.ConsentNone || machine.TargetKinds[0] != appcontrol.TargetKind {
		t.Fatalf("application lifecycle capability = %#v", machine)
	}
	launch := builtins.LaunchApplicationContract.Machine()
	if len(launch.Ports.DataInputs) != 0 || len(launch.Ports.DataOutputs) != 0 || launch.Ports.ExecOutputs[0].ID != "completed" || launch.Ports.ErrorOutputs[0].ID != "failed" {
		t.Fatalf("launch ports = %#v", launch.Ports)
	}
	terminate := builtins.TerminateApplicationContract.Machine()
	if len(terminate.Ports.DataOutputs) != 1 || terminate.Ports.DataOutputs[0].ID != "terminated-count" {
		t.Fatalf("terminate ports = %#v", terminate.Ports)
	}
	for _, candidate := range []struct{ contractID, operation string }{{LaunchApplicationNodeID, appcontrol.OperationLaunch}, {TerminateApplicationNodeID, appcontrol.OperationTerminate}} {
		entry, ok := builtins.Catalog.Lookup(candidate.contractID)
		if !ok {
			t.Fatalf("missing node %s", candidate.contractID)
		}
		requirement := entry.Contract.Machine().CapabilityRequirements[0]
		if len(requirement.Operations) != 1 || requirement.Operations[0] != candidate.operation || requirement.TargetSlot != "application" {
			t.Fatalf("requirement for %s = %#v", candidate.contractID, requirement)
		}
	}
}
