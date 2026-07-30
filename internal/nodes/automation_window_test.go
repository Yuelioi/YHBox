package nodes

import (
	"slices"
	"testing"

	"github.com/yottaapp/yotta/internal/automation/installed"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

func TestActivateWindowUsesConfiguredTarget(t *testing.T) {
	builtins, err := Build()
	if err != nil {
		t.Fatal(err)
	}
	definition, ok := builtins.Definition(ActivateWindowNodeID)
	if !ok {
		t.Fatal("activate window definition is missing")
	}
	machine := definition.Contract.Machine()
	if machine.Execution.Class != nodecontract.ExecutionEffect || machine.Execution.Determinism != nodecontract.Recorded ||
		len(machine.Execution.Effects) != 1 || machine.Execution.Effects[0] != ActivateWindowEffectID {
		t.Fatalf("execution = %#v", machine.Execution)
	}
	if len(machine.Ports.DataInputs)+len(machine.Ports.DataOutputs) != 0 ||
		!signalIDsEqual(machine.Ports.ExecInputs, []string{"in"}) || !signalIDsEqual(machine.Ports.ExecOutputs, []string{"completed"}) ||
		!signalIDsEqual(machine.Ports.ErrorOutputs, []string{"failed"}) {
		t.Fatalf("ports = %#v", machine.Ports)
	}
	if len(machine.CapabilityRequirements) != 0 || len(machine.ConfiguredTargets) != 1 ||
		len(machine.ConfiguredTargets[0].TargetKinds) != 2 {
		t.Fatalf("configured target = %#v", machine.ConfiguredTargets)
	}
}

func TestDesktopWindowOperationsRemainTargetBoundAndTyped(t *testing.T) {
	builtins, err := Build()
	if err != nil {
		t.Fatal(err)
	}
	wants := map[string]struct {
		operation string
		exec      []string
	}{
		CloseWindowNodeID:      {installed.OperationCloseWindow, []string{"completed"}},
		MoveResizeWindowNodeID: {installed.OperationMoveResizeWindow, []string{"completed"}},
		MaximizeWindowNodeID:   {installed.OperationSetWindowState, []string{"completed"}},
		MinimizeWindowNodeID:   {installed.OperationSetWindowState, []string{"completed"}},
		RestoreWindowNodeID:    {installed.OperationSetWindowState, []string{"completed"}},
		GetWindowStateNodeID:   {installed.OperationGetWindowState, []string{"completed"}},
		WaitWindowNodeID:       {installed.OperationWaitWindow, []string{"found", "timeout"}},
		WaitWindowGoneNodeID:   {installed.OperationWaitWindowGone, []string{"gone", "timeout"}},
	}
	for nodeID, want := range wants {
		definition, ok := builtins.Definition(nodeID)
		if !ok {
			t.Fatalf("window definition %q is missing", nodeID)
		}
		machine := definition.Contract.Machine()
		if len(machine.CapabilityRequirements) != 0 || len(machine.ConfiguredTargets) != 1 ||
			!slices.Equal(machine.ConfiguredTargets[0].TargetKinds, []string{installed.TargetKindDesktopWindow}) || !signalIDsEqual(machine.Ports.ExecOutputs, want.exec) {
			t.Fatalf("window contract %q = %#v", nodeID, machine)
		}
	}
}
