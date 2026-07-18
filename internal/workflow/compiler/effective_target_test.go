package compiler

import (
	"testing"

	"github.com/yottaapp/yotta/internal/nodecontract"
	"github.com/yottaapp/yotta/internal/nodes"
	"github.com/yottaapp/yotta/internal/workflow/schema"
)

func TestEffectiveNodeConfigResolvesWorkflowDefaultAndNodeOverride(t *testing.T) {
	builtins, err := nodes.Build()
	if err != nil {
		t.Fatal(err)
	}
	entry, ok := builtins.Catalog.Lookup(nodes.ClickPointerNodeID)
	if !ok {
		t.Fatal("click pointer node is missing")
	}
	machine := entry.Contract.Machine()
	defaults := []schema.TargetDefault{{Target: "target", Slot: "workflow-target"}}

	inherited, err := effectiveNodeConfig(defaults, schema.Node{Config: map[string]any{}}, machine)
	if err != nil || inherited["slot"] != "workflow-target" {
		t.Fatalf("inherited=%+v err=%v", inherited, err)
	}
	requirements, err := nodecontract.ResolveCapabilityRequirements(machine, inherited)
	if err != nil || len(requirements) == 0 || requirements[0].TargetSlot != "workflow-target" {
		t.Fatalf("requirements=%+v err=%v", requirements, err)
	}

	overridden, err := effectiveNodeConfig(defaults, schema.Node{Config: map[string]any{"slot": "node-target"}}, machine)
	if err != nil || overridden["slot"] != "node-target" {
		t.Fatalf("overridden=%+v err=%v", overridden, err)
	}

	missing, err := effectiveNodeConfig(nil, schema.Node{Config: map[string]any{}}, machine)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := nodecontract.ResolveCapabilityRequirements(machine, missing); err == nil {
		t.Fatalf("missing config resolved: %+v", missing)
	}
}
