package schema

import (
	"errors"
	"sort"
	"strings"
)

const MaxTargetDefaults = 64

func TargetDefaultSlot(source WorkflowSource, target string) (string, bool) {
	for _, candidate := range source.TargetDefaults {
		if candidate.Target == target {
			return candidate.Slot, true
		}
	}
	return "", false
}

func SetTargetDefault(source *WorkflowSource, target, slot string) error {
	if source == nil {
		return errors.New("workflow source is required")
	}
	target = strings.TrimSpace(target)
	slot = strings.TrimSpace(slot)
	if !validTargetDefaultName(target) || !validTargetDefaultName(slot) {
		return errors.New("target default names are invalid")
	}
	for index := range source.TargetDefaults {
		if source.TargetDefaults[index].Target == target {
			source.TargetDefaults[index].Slot = slot
			return nil
		}
	}
	if len(source.TargetDefaults) >= MaxTargetDefaults {
		return errors.New("target default budget exceeded")
	}
	source.TargetDefaults = append(source.TargetDefaults, TargetDefault{Target: target, Slot: slot})
	sort.Slice(source.TargetDefaults, func(i, j int) bool {
		return source.TargetDefaults[i].Target < source.TargetDefaults[j].Target
	})
	return nil
}

func ClearTargetDefault(source *WorkflowSource, target string) error {
	if source == nil {
		return errors.New("workflow source is required")
	}
	for index, candidate := range source.TargetDefaults {
		if candidate.Target != target {
			continue
		}
		source.TargetDefaults = append(source.TargetDefaults[:index], source.TargetDefaults[index+1:]...)
		if len(source.TargetDefaults) == 0 {
			source.TargetDefaults = nil
		}
		return nil
	}
	return nil
}

func validateTargetDefaults(defaults []TargetDefault) error {
	if len(defaults) > MaxTargetDefaults {
		return errors.New("target default budget exceeded")
	}
	previous := ""
	for _, candidate := range defaults {
		if !validTargetDefaultName(candidate.Target) || !validTargetDefaultName(candidate.Slot) {
			return errors.New("target default names are invalid")
		}
		if candidate.Target <= previous {
			return errors.New("target defaults must be unique and sorted")
		}
		previous = candidate.Target
	}
	return nil
}

func validTargetDefaultName(value string) bool {
	if value == "" || len(value) > 128 || value[0] < 'a' || value[0] > 'z' {
		return false
	}
	for index := 1; index < len(value); index++ {
		character := value[index]
		if character >= 'a' && character <= 'z' || character >= '0' && character <= '9' {
			continue
		}
		if (character == '-' || character == '_') && index+1 < len(value) && value[index+1] >= 'a' && value[index+1] <= 'z' ||
			(character == '-' || character == '_') && index+1 < len(value) && value[index+1] >= '0' && value[index+1] <= '9' {
			continue
		}
		return false
	}
	return true
}
