package nodes_test

import (
	"testing"

	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/nodecontract"
	"github.com/yottaapp/yotta/internal/nodes"
	"github.com/yottaapp/yotta/internal/workspacefs"
)

func TestFilesystemNodesUseExactWorkspaceCapabilityAndExplicitSignals(t *testing.T) {
	builtins, err := nodes.Build()
	if err != nil {
		t.Fatal(err)
	}
	capabilityDefinition, ok := builtins.Catalog.LookupCapability(nodes.FilesystemReadCapabilityID)
	if !ok {
		t.Fatal("filesystem capability is missing")
	}
	machineCapability := capabilityDefinition.Machine()
	if machineCapability.ProviderABI != workspacefs.ProviderABI || machineCapability.Risk != capability.RiskLow || machineCapability.Consent != capability.ConsentNone {
		t.Fatalf("filesystem capability = %#v", machineCapability)
	}

	for _, nodeID := range []string{nodes.FileReadTextNodeID, nodes.FileReadJSONNodeID, nodes.FileStatNodeID} {
		definition, ok := builtins.Definition(nodeID)
		if !ok {
			t.Fatalf("filesystem node %q is missing", nodeID)
		}
		machine := definition.Contract.Machine()
		if machine.Execution.Class != nodecontract.ExecutionEffect || machine.Execution.Determinism != nodecontract.Recorded ||
			machine.Execution.Cache != nodecontract.CacheNone || machine.Execution.Retry != nodecontract.RetryNever ||
			len(machine.Ports.ExecInputs) != 1 || machine.Ports.ExecInputs[0].ID != "in" ||
			len(machine.Ports.ExecOutputs) != 1 || machine.Ports.ExecOutputs[0].ID != "completed" ||
			len(machine.Ports.ErrorOutputs) != 1 || machine.Ports.ErrorOutputs[0].ID != "failed" ||
			len(machine.CapabilityRequirements) != 1 {
			t.Fatalf("filesystem contract %q = %#v", nodeID, machine)
		}
		requirement := machine.CapabilityRequirements[0]
		if requirement.ID != "workspace-files" || requirement.Capability != capabilityDefinition.Ref() || requirement.TargetSlot != "workspace-files" || string(requirement.Scope) != `{"root":"workflow-files"}` {
			t.Fatalf("filesystem requirement %q = %#v", nodeID, requirement)
		}
	}

	if builtins.FileMetadataType.TypeRef().TypeID != nodes.FileMetadataTypeID {
		t.Fatalf("file metadata type = %#v", builtins.FileMetadataType.TypeRef())
	}
}
