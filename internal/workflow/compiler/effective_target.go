package compiler

import (
	"fmt"

	"github.com/yottaapp/yotta/internal/nodecontract"
	"github.com/yottaapp/yotta/internal/workflow/schema"
)

func effectiveNodeConfig(defaults []schema.TargetDefault, node schema.Node, machine nodecontract.MachineContract) (map[string]any, error) {
	config := make(map[string]any, len(node.Config)+len(machine.RequirementBindings))
	for key, value := range node.Config {
		config[key] = value
	}
	bindings := make(map[string]nodecontract.RequirementBindingSpec, len(machine.RequirementBindings))
	inheritedByConfigKey := make(map[string]string)
	for _, binding := range machine.RequirementBindings {
		bindings[binding.RequirementID] = binding
	}
	for _, requirement := range machine.CapabilityRequirements {
		binding, ok := bindings[requirement.ID]
		if !ok || binding.TargetSlotConfigKey == "" {
			continue
		}
		if _, explicit := node.Config[binding.TargetSlotConfigKey]; explicit {
			continue
		}
		slot, inherited := targetDefaultSlot(defaults, requirement.TargetSlot)
		if !inherited {
			continue
		}
		if existing, assigned := inheritedByConfigKey[binding.TargetSlotConfigKey]; assigned && existing != slot {
			return nil, fmt.Errorf("target config %q resolves to multiple workflow defaults", binding.TargetSlotConfigKey)
		}
		inheritedByConfigKey[binding.TargetSlotConfigKey] = slot
		config[binding.TargetSlotConfigKey] = slot
	}
	return config, nil
}

func targetDefaultSlot(defaults []schema.TargetDefault, target string) (string, bool) {
	for _, candidate := range defaults {
		if candidate.Target == target {
			return candidate.Slot, true
		}
	}
	return "", false
}
