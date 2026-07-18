package datatype

import (
	"errors"
	"fmt"
	"sort"
)

// Trait is a closed semantic capability used by generic node contracts.
// Traits describe operations a type supports; they do not infer compatibility
// from the type's JSON representation.
type Trait string

const (
	TraitDurable    Trait = "durable"
	TraitEquatable  Trait = "equatable"
	TraitNumeric    Trait = "numeric"
	TraitObservable Trait = "observable"
	TraitOrdered    Trait = "ordered"
)

var knownTraits = map[Trait]struct{}{
	TraitDurable: {}, TraitEquatable: {}, TraitNumeric: {}, TraitObservable: {}, TraitOrdered: {},
}

func normalizeTraits(source []Trait) ([]Trait, error) {
	result := append([]Trait(nil), source...)
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	for index, trait := range result {
		if _, ok := knownTraits[trait]; !ok {
			return nil, fmt.Errorf("unknown data type trait %q", trait)
		}
		if index > 0 && trait == result[index-1] {
			return nil, fmt.Errorf("duplicate data type trait %q", trait)
		}
	}
	return result, nil
}

// System is the authoritative relation graph for one sealed Catalog.
type System struct {
	definitions map[string]Definition
}

func NewSystem(definitions []Definition) (*System, error) {
	byID := make(map[string]Definition, len(definitions))
	for _, definition := range definitions {
		if !definition.Valid() {
			return nil, errors.New("type system contains invalid definition")
		}
		ref := definition.TypeRef()
		if _, exists := byID[ref.TypeID]; exists {
			return nil, fmt.Errorf("type system contains duplicate type %q", ref.TypeID)
		}
		byID[ref.TypeID] = definition
	}
	for _, definition := range definitions {
		for _, target := range definition.Machine().AssignableTo {
			resolved, ok := byID[target.TypeID]
			if !ok || resolved.TypeRef() != target {
				return nil, fmt.Errorf("type %q references unknown assignable target %q", definition.TypeRef().TypeID, target.TypeID)
			}
		}
		if structure := definition.Machine().Structure; structure != nil {
			for _, field := range structure.Fields {
				if err := validateCatalogExpression(field.Type, byID, 0); err != nil {
					return nil, fmt.Errorf("type %q structure field %q: %w", definition.TypeRef().TypeID, field.ID, err)
				}
			}
		}
	}
	system := &System{definitions: byID}
	for _, definition := range definitions {
		if _, err := system.reachable(definition.TypeRef(), definition.TypeRef(), true, map[string]bool{}); err != nil {
			return nil, err
		}
	}
	return system, nil
}

func validateCatalogExpression(expression TypeExpression, definitions map[string]Definition, depth int) error {
	if depth > MaxTypeDepth {
		return errors.New("type expression exceeds depth budget")
	}
	switch expression.Kind {
	case TypeExpressionRef:
		definition, ok := definitions[expression.Ref.TypeID]
		if !ok || definition.TypeRef() != *expression.Ref {
			return fmt.Errorf("references unknown type %q", expression.Ref.TypeID)
		}
	case TypeExpressionList:
		return validateCatalogExpression(*expression.Element, definitions, depth+1)
	case TypeExpressionUnion:
		for _, member := range expression.Members {
			if err := validateCatalogExpression(member, definitions, depth+1); err != nil {
				return err
			}
		}
	case TypeExpressionVariable:
		return errors.New("contains an unresolved variable")
	}
	return nil
}

func (s *System) HasTrait(ref TypeRef, trait Trait) bool {
	definition, ok := s.definitions[ref.TypeID]
	if !ok || definition.TypeRef() != ref {
		return false
	}
	for _, candidate := range definition.Machine().Traits {
		if candidate == trait {
			return true
		}
	}
	return false
}

func (s *System) Satisfies(ref TypeRef, constraints []string) (bool, error) {
	for _, raw := range constraints {
		trait := Trait(raw)
		if _, ok := knownTraits[trait]; !ok {
			return false, fmt.Errorf("unknown type constraint %q", raw)
		}
		if !s.HasTrait(ref, trait) {
			return false, nil
		}
	}
	return true, nil
}

func (s *System) AssignableRef(source, target TypeRef) (bool, error) {
	if source == target {
		_, ok := s.definitions[source.TypeID]
		return ok, nil
	}
	return s.reachable(source, target, false, map[string]bool{})
}

// LeastUpperBoundRef returns the unique most-specific named type that can
// accept both inputs. It never guesses from JSON Schema shape: only sealed,
// explicit assignability edges participate.
func (s *System) LeastUpperBoundRef(left, right TypeRef) (TypeRef, error) {
	if _, ok := s.definitions[left.TypeID]; !ok || s.definitions[left.TypeID].TypeRef() != left {
		return TypeRef{}, fmt.Errorf("unknown left type %q", left.TypeID)
	}
	if _, ok := s.definitions[right.TypeID]; !ok || s.definitions[right.TypeID].TypeRef() != right {
		return TypeRef{}, fmt.Errorf("unknown right type %q", right.TypeID)
	}
	common := make([]TypeRef, 0)
	for _, definition := range s.definitions {
		candidate := definition.TypeRef()
		leftOK, err := s.AssignableRef(left, candidate)
		if err != nil {
			return TypeRef{}, err
		}
		rightOK, err := s.AssignableRef(right, candidate)
		if err != nil {
			return TypeRef{}, err
		}
		if leftOK && rightOK {
			common = append(common, candidate)
		}
	}
	minimal := make([]TypeRef, 0, len(common))
	for _, candidate := range common {
		moreSpecific := false
		for _, other := range common {
			if other == candidate {
				continue
			}
			assignable, err := s.AssignableRef(other, candidate)
			if err != nil {
				return TypeRef{}, err
			}
			if assignable {
				moreSpecific = true
				break
			}
		}
		if !moreSpecific {
			minimal = append(minimal, candidate)
		}
	}
	if len(minimal) == 0 {
		return TypeRef{}, fmt.Errorf("types %q and %q have no common assignable type", left.TypeID, right.TypeID)
	}
	if len(minimal) != 1 {
		sort.Slice(minimal, func(i, j int) bool { return minimal[i].TypeID < minimal[j].TypeID })
		ids := make([]string, len(minimal))
		for index, ref := range minimal {
			ids[index] = ref.TypeID
		}
		return TypeRef{}, fmt.Errorf("types %q and %q have ambiguous common assignable types %v", left.TypeID, right.TypeID, ids)
	}
	return minimal[0], nil
}

func (s *System) reachable(source, target TypeRef, detectingCycle bool, visiting map[string]bool) (bool, error) {
	definition, ok := s.definitions[source.TypeID]
	if !ok || definition.TypeRef() != source {
		return false, fmt.Errorf("unknown source type %q", source.TypeID)
	}
	if visiting[source.TypeID] {
		return false, fmt.Errorf("assignable relation contains a cycle at %q", source.TypeID)
	}
	visiting[source.TypeID] = true
	defer delete(visiting, source.TypeID)
	for _, next := range definition.Machine().AssignableTo {
		if next == target && !detectingCycle {
			return true, nil
		}
		found, err := s.reachable(next, target, detectingCycle, visiting)
		if err != nil || found {
			return found, err
		}
	}
	return false, nil
}

func (s *System) Assignable(source, target TypeExpression) (bool, error) {
	if err := source.Validate(); err != nil {
		return false, fmt.Errorf("invalid source type: %w", err)
	}
	if err := target.Validate(); err != nil {
		return false, fmt.Errorf("invalid target type: %w", err)
	}
	return s.assignable(source, target, 0)
}

func (s *System) assignable(source, target TypeExpression, depth int) (bool, error) {
	if depth > MaxTypeDepth {
		return false, errors.New("type relation exceeds depth budget")
	}
	if source.Kind == TypeExpressionVariable || target.Kind == TypeExpressionVariable {
		return false, errors.New("type variable must be resolved before assignability")
	}
	if source.Kind == TypeExpressionUnion {
		for _, member := range source.Members {
			ok, err := s.assignable(member, target, depth+1)
			if err != nil || !ok {
				return ok, err
			}
		}
		return true, nil
	}
	if target.Kind == TypeExpressionUnion {
		for _, member := range target.Members {
			ok, err := s.assignable(source, member, depth+1)
			if err != nil {
				return false, err
			}
			if ok {
				return true, nil
			}
		}
		return false, nil
	}
	if source.Kind != target.Kind {
		return false, nil
	}
	switch source.Kind {
	case TypeExpressionRef:
		return s.AssignableRef(*source.Ref, *target.Ref)
	case TypeExpressionList:
		// Lists are invariant until mutation safety is represented in the contract.
		sourceKey, err := expressionKey(*source.Element)
		if err != nil {
			return false, err
		}
		targetKey, err := expressionKey(*target.Element)
		if err != nil {
			return false, err
		}
		return sourceKey == targetKey, nil
	default:
		return false, errors.New("unknown type expression")
	}
}
