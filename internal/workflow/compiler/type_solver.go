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
}

func newTypeSolver() *typeSolver {
	return &typeSolver{bindings: map[scopedTypeVariable]scopedTypeExpression{}}
}

func (s *typeSolver) unify(output, input scopedTypeExpression) error {
	left, err := s.dereference(output, 0)
	if err != nil {
		return err
	}
	right, err := s.dereference(input, 0)
	if err != nil {
		return err
	}
	if left.expression.Kind == datatype.TypeExpressionVariable {
		return s.bind(left, right)
	}
	if right.expression.Kind == datatype.TypeExpressionVariable {
		return s.bind(right, left)
	}
	if left.expression.Kind == datatype.TypeExpressionList && right.expression.Kind == datatype.TypeExpressionList {
		return s.unify(
			scopedTypeExpression{scope: left.scope, expression: *left.expression.Element},
			scopedTypeExpression{scope: right.scope, expression: *right.expression.Element},
		)
	}
	assignable, err := datatype.Assignable(left.expression, right.expression)
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
	if len(variable.expression.Constraints) != 0 {
		return fmt.Errorf("type variable %q uses unsupported constraints", variable.expression.Variable)
	}
	key := scopedTypeVariable{scope: variable.scope, name: variable.expression.Variable}
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
