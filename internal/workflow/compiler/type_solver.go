package compiler

import (
	"errors"
	"fmt"

	"github.com/yottaapp/yotta/internal/datatype"
)

type scopedTypeExpression struct {
	scope      string
	expression datatype.TypeExpression
}

type scopedTypeVariable struct {
	scope string
	name  string
}

// typeSolver unifies contract variables across data edges while preserving
// node-local variable scope. Its result is frozen into Program input/output
// types; runtime adapters never infer types from JSON values.
type typeSolver struct {
	bindings map[scopedTypeVariable]scopedTypeExpression
	types    *datatype.System
}

func newTypeSolver(types *datatype.System) *typeSolver {
	return &typeSolver{bindings: map[scopedTypeVariable]scopedTypeExpression{}, types: types}
}

func (s *typeSolver) unify(output, input scopedTypeExpression) error {
	left, err := s.dereference(output, 0)
	if err != nil {
		return err
	}
	// Preserve the declared variable and its constraints even when it already
	// has a binding. bind merges repeated evidence deterministically.
	if input.expression.Kind == datatype.TypeExpressionVariable {
		right, err := s.dereference(input, 0)
		if err != nil {
			return err
		}
		if left.expression.Kind == datatype.TypeExpressionVariable && right.expression.Kind != datatype.TypeExpressionVariable {
			return s.bind(output, right)
		}
		return s.bind(input, left)
	}
	right, err := s.dereference(input, 0)
	if err != nil {
		return err
	}
	if output.expression.Kind == datatype.TypeExpressionVariable {
		return s.bind(output, right)
	}
	if left.expression.Kind == datatype.TypeExpressionList && right.expression.Kind == datatype.TypeExpressionList {
		return s.unify(
			scopedTypeExpression{scope: left.scope, expression: *left.expression.Element},
			scopedTypeExpression{scope: right.scope, expression: *right.expression.Element},
		)
	}
	if s.types == nil {
		return errors.New("catalog type system is unavailable")
	}
	assignable, err := s.types.Assignable(left.expression, right.expression)
	if err != nil {
		return err
	}
	if !assignable {
		return errors.New("type expressions are not assignable")
	}
	return nil
}

func (s *typeSolver) resolve(value scopedTypeExpression) (datatype.ResolvedType, error) {
	return s.resolveAt(value, 0)
}

func (s *typeSolver) resolveAt(value scopedTypeExpression, depth int) (datatype.ResolvedType, error) {
	if depth > datatype.MaxTypeDepth {
		return datatype.ResolvedType{}, errors.New("type resolution exceeds depth budget")
	}
	value, err := s.dereference(value, depth)
	if err != nil {
		return datatype.ResolvedType{}, err
	}
	switch value.expression.Kind {
	case datatype.TypeExpressionRef:
		return datatype.RefResolvedType(*value.expression.Ref), nil
	case datatype.TypeExpressionList:
		element, err := s.resolveAt(scopedTypeExpression{scope: value.scope, expression: *value.expression.Element}, depth+1)
		if err != nil {
			return datatype.ResolvedType{}, err
		}
		return datatype.ListResolvedType(element), nil
	case datatype.TypeExpressionVariable:
		return datatype.ResolvedType{}, fmt.Errorf("type variable %q is unresolved", value.expression.Variable)
	case datatype.TypeExpressionUnion:
		return datatype.ResolvedType{}, errors.New("union port requires a concrete data-edge type")
	default:
		return datatype.ResolvedType{}, errors.New("unknown type expression")
	}
}

func (s *typeSolver) bind(variable, value scopedTypeExpression) error {
	key := scopedTypeVariable{scope: variable.scope, name: variable.expression.Variable}
	value, err := s.dereference(value, 0)
	if err != nil {
		return err
	}
	if existing, ok := s.bindings[key]; ok {
		existing, err = s.dereference(existing, 0)
		if err != nil {
			return err
		}
		if existing.expression.Kind == datatype.TypeExpressionRef && value.expression.Kind == datatype.TypeExpressionRef {
			if s.types == nil {
				return errors.New("catalog type system is unavailable")
			}
			merged, err := s.types.LeastUpperBoundRef(*existing.expression.Ref, *value.expression.Ref)
			if err != nil {
				return fmt.Errorf("conflicting bindings for type variable %q: %w", variable.expression.Variable, err)
			}
			value = scopedTypeExpression{scope: variable.scope, expression: datatype.RefExpression(merged)}
		} else {
			existingKey, existingErr := expressionIdentity(existing.expression)
			valueKey, valueErr := expressionIdentity(value.expression)
			if existingErr != nil || valueErr != nil || existingKey != valueKey {
				return fmt.Errorf("conflicting bindings for type variable %q", variable.expression.Variable)
			}
			value = existing
		}
	}
	if len(variable.expression.Constraints) != 0 && value.expression.Kind == datatype.TypeExpressionRef {
		if s.types == nil {
			return errors.New("catalog type system is unavailable")
		}
		satisfied, err := s.types.Satisfies(*value.expression.Ref, variable.expression.Constraints)
		if err != nil {
			return err
		}
		if !satisfied {
			return fmt.Errorf("type does not satisfy constraints for variable %q", variable.expression.Variable)
		}
	} else if len(variable.expression.Constraints) != 0 && value.expression.Kind != datatype.TypeExpressionVariable {
		return fmt.Errorf("constraints for type variable %q require a concrete named type", variable.expression.Variable)
	}
	if value.expression.Kind == datatype.TypeExpressionVariable {
		other := scopedTypeVariable{scope: value.scope, name: value.expression.Variable}
		if key == other {
			return nil
		}
	}
	occurs, err := s.occurs(key, value, 0)
	if err != nil {
		return err
	}
	if occurs {
		return errors.New("recursive type variable binding")
	}
	s.bindings[key] = value
	return nil
}

func expressionIdentity(value datatype.TypeExpression) (string, error) {
	switch value.Kind {
	case datatype.TypeExpressionRef:
		return "ref:" + value.Ref.TypeID + "@" + value.Ref.SemanticDigest.String(), nil
	case datatype.TypeExpressionList:
		element, err := expressionIdentity(*value.Element)
		return "list<" + element + ">", err
	case datatype.TypeExpressionVariable:
		return "variable:" + value.Variable, nil
	case datatype.TypeExpressionUnion:
		parts := make([]string, len(value.Members))
		for index, member := range value.Members {
			part, err := expressionIdentity(member)
			if err != nil {
				return "", err
			}
			parts[index] = part
		}
		return fmt.Sprintf("union:%q", parts), nil
	default:
		return "", errors.New("unknown type expression")
	}
}

func (s *typeSolver) dereference(value scopedTypeExpression, depth int) (scopedTypeExpression, error) {
	if depth > datatype.MaxTypeDepth {
		return scopedTypeExpression{}, errors.New("type variable chain exceeds depth budget")
	}
	if value.expression.Kind != datatype.TypeExpressionVariable {
		return value, nil
	}
	bound, ok := s.bindings[scopedTypeVariable{scope: value.scope, name: value.expression.Variable}]
	if !ok {
		return value, nil
	}
	return s.dereference(bound, depth+1)
}

func (s *typeSolver) occurs(variable scopedTypeVariable, value scopedTypeExpression, depth int) (bool, error) {
	if depth > datatype.MaxTypeDepth {
		return false, errors.New("type occurrence check exceeds depth budget")
	}
	value, err := s.dereference(value, depth)
	if err != nil {
		return false, err
	}
	switch value.expression.Kind {
	case datatype.TypeExpressionVariable:
		return variable == (scopedTypeVariable{scope: value.scope, name: value.expression.Variable}), nil
	case datatype.TypeExpressionList:
		return s.occurs(variable, scopedTypeExpression{scope: value.scope, expression: *value.expression.Element}, depth+1)
	case datatype.TypeExpressionUnion:
		for _, member := range value.expression.Members {
			found, err := s.occurs(variable, scopedTypeExpression{scope: value.scope, expression: member}, depth+1)
			if err != nil || found {
				return found, err
			}
		}
	}
	return false, nil
}
