package datatype

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/yottaapp/yotta/internal/artifact"
)

const (
	MaxTypeDepth       = 64
	MaxTypeNodes       = 1_024
	MaxUnionMembers    = 64
	MaxTypeConstraints = 32
	MaxTypeStringBytes = 1_024
)

var ErrUnresolvedTypeVariable = errors.New("type expression contains an unresolved variable")

func (r TypeRef) Validate() error {
	if err := validateTypeID(r.TypeID); err != nil {
		return err
	}
	if !r.SemanticDigest.Valid() {
		return errors.New("type reference semantic digest is invalid")
	}
	return nil
}

type ResolvedTypeKind string

const (
	ResolvedTypeRef  ResolvedTypeKind = "ref"
	ResolvedTypeList ResolvedTypeKind = "list"
)

type ResolvedType struct {
	Kind    ResolvedTypeKind `json:"kind"`
	Ref     *TypeRef         `json:"ref,omitempty"`
	Element *ResolvedType    `json:"element,omitempty"`
}

func RefResolvedType(ref TypeRef) ResolvedType {
	copy := ref
	return ResolvedType{Kind: ResolvedTypeRef, Ref: &copy}
}

func ListResolvedType(element ResolvedType) ResolvedType {
	copy := element
	return ResolvedType{Kind: ResolvedTypeList, Element: &copy}
}

func (t ResolvedType) Validate() error {
	budget := typeValidationBudget{}
	return t.validateAt(0, &budget)
}

func (t ResolvedType) validateAt(depth int, budget *typeValidationBudget) error {
	if depth > MaxTypeDepth {
		return errors.New("resolved type exceeds depth budget")
	}
	if err := budget.consume(); err != nil {
		return err
	}
	switch t.Kind {
	case ResolvedTypeRef:
		if t.Ref == nil || t.Element != nil {
			return errors.New("resolved ref type has invalid members")
		}
		return t.Ref.Validate()
	case ResolvedTypeList:
		if t.Ref != nil || t.Element == nil {
			return errors.New("resolved list type has invalid members")
		}
		return t.Element.validateAt(depth+1, budget)
	default:
		return fmt.Errorf("unknown resolved type kind %q", t.Kind)
	}
}

type TypeExpressionKind string

const (
	TypeExpressionRef      TypeExpressionKind = "ref"
	TypeExpressionList     TypeExpressionKind = "list"
	TypeExpressionUnion    TypeExpressionKind = "union"
	TypeExpressionVariable TypeExpressionKind = "variable"
)

type TypeExpression struct {
	Kind        TypeExpressionKind `json:"kind"`
	Ref         *TypeRef           `json:"ref,omitempty"`
	Element     *TypeExpression    `json:"element,omitempty"`
	Members     []TypeExpression   `json:"members,omitempty"`
	Variable    string             `json:"variable,omitempty"`
	Constraints []string           `json:"constraints,omitempty"`
}

func RefExpression(ref TypeRef) TypeExpression {
	copy := ref
	return TypeExpression{Kind: TypeExpressionRef, Ref: &copy}
}

func ListExpression(element TypeExpression) TypeExpression {
	copy := element
	return TypeExpression{Kind: TypeExpressionList, Element: &copy}
}

func UnionExpression(members ...TypeExpression) (TypeExpression, error) {
	budget := typeValidationBudget{}
	flattened := make([]TypeExpression, 0, len(members))
	for _, member := range members {
		if err := collectUnionMembers(member, 0, &budget, &flattened); err != nil {
			return TypeExpression{}, fmt.Errorf("normalize union expression: %w", err)
		}
	}
	if len(flattened) == 0 || len(flattened) > MaxUnionMembers {
		return TypeExpression{}, errors.New("union expression exceeds member budget")
	}
	type keyedExpression struct {
		expression TypeExpression
		key        string
	}
	keyed := make([]keyedExpression, len(flattened))
	for i, member := range flattened {
		key, err := expressionKey(member)
		if err != nil {
			return TypeExpression{}, fmt.Errorf("encode union member: %w", err)
		}
		keyed[i] = keyedExpression{expression: member, key: key}
	}
	sort.SliceStable(keyed, func(i, j int) bool { return keyed[i].key < keyed[j].key })
	deduplicated := make([]TypeExpression, 0, len(keyed))
	previous := ""
	for _, member := range keyed {
		if len(deduplicated) == 0 || member.key != previous {
			deduplicated = append(deduplicated, member.expression)
			previous = member.key
		}
	}
	if len(deduplicated) == 1 {
		return deduplicated[0], nil
	}
	return TypeExpression{Kind: TypeExpressionUnion, Members: deduplicated}, nil
}

func VariableExpression(name string, constraints ...string) TypeExpression {
	copy := append([]string(nil), constraints...)
	sort.Strings(copy)
	return TypeExpression{Kind: TypeExpressionVariable, Variable: name, Constraints: copy}
}

func Assignable(output, input TypeExpression) (bool, error) {
	if err := output.Validate(); err != nil {
		return false, fmt.Errorf("invalid output type expression: %w", err)
	}
	if err := input.Validate(); err != nil {
		return false, fmt.Errorf("invalid input type expression: %w", err)
	}
	return assignableAt(output, input, 0)
}

func (t TypeExpression) Validate() error {
	budget := typeValidationBudget{}
	return t.validateAt(0, &budget)
}

func (t TypeExpression) validateAt(depth int, budget *typeValidationBudget) error {
	if depth > MaxTypeDepth {
		return errors.New("type expression exceeds depth budget")
	}
	if err := budget.consume(); err != nil {
		return err
	}
	switch t.Kind {
	case TypeExpressionRef:
		if t.Ref == nil || t.Element != nil || len(t.Members) != 0 || t.Variable != "" || len(t.Constraints) != 0 {
			return errors.New("ref expression has invalid members")
		}
		return t.Ref.Validate()
	case TypeExpressionList:
		if t.Ref != nil || t.Element == nil || len(t.Members) != 0 || t.Variable != "" || len(t.Constraints) != 0 {
			return errors.New("list expression has invalid members")
		}
		return t.Element.validateAt(depth+1, budget)
	case TypeExpressionUnion:
		if t.Ref != nil || t.Element != nil || len(t.Members) < 2 || len(t.Members) > MaxUnionMembers || t.Variable != "" || len(t.Constraints) != 0 {
			return errors.New("union expression has invalid members")
		}
		previous := ""
		for _, member := range t.Members {
			if member.Kind == TypeExpressionUnion {
				return errors.New("nested union expression is not normalized")
			}
			if err := member.validateAt(depth+1, budget); err != nil {
				return err
			}
			key, err := expressionKey(member)
			if err != nil {
				return err
			}
			if key <= previous {
				return errors.New("union expression is not normalized")
			}
			previous = key
		}
		return nil
	case TypeExpressionVariable:
		if t.Ref != nil || t.Element != nil || len(t.Members) != 0 || t.Variable == "" || len(t.Variable) > MaxTypeStringBytes || len(t.Constraints) > MaxTypeConstraints {
			return errors.New("variable expression has invalid members")
		}
		previous := ""
		for _, constraint := range t.Constraints {
			if strings.TrimSpace(constraint) != constraint || constraint == "" || len(constraint) > MaxTypeStringBytes || constraint <= previous {
				return errors.New("variable constraints are not normalized")
			}
			previous = constraint
		}
		return nil
	default:
		return fmt.Errorf("unknown type expression kind %q", t.Kind)
	}
}

func assignableAt(output, input TypeExpression, depth int) (bool, error) {
	if depth > MaxTypeDepth {
		return false, errors.New("assignability exceeds depth budget")
	}
	if output.Kind == TypeExpressionVariable || input.Kind == TypeExpressionVariable {
		return false, ErrUnresolvedTypeVariable
	}
	if output.Kind == TypeExpressionUnion {
		for _, member := range output.Members {
			ok, err := assignableAt(member, input, depth+1)
			if err != nil || !ok {
				return ok, err
			}
		}
		return true, nil
	}
	if input.Kind == TypeExpressionUnion {
		for _, member := range input.Members {
			ok, err := assignableAt(output, member, depth+1)
			if err != nil {
				return false, err
			}
			if ok {
				return true, nil
			}
		}
		return false, nil
	}
	if output.Kind != input.Kind {
		return false, nil
	}
	switch output.Kind {
	case TypeExpressionRef:
		return *output.Ref == *input.Ref, nil
	case TypeExpressionList:
		return assignableAt(*output.Element, *input.Element, depth+1)
	default:
		return false, fmt.Errorf("unsupported assignability kind %q", output.Kind)
	}
}

func expressionKey(expression TypeExpression) (string, error) {
	canonical, err := artifact.Marshal(expression)
	if err != nil {
		return "", err
	}
	return string(canonical), nil
}

type typeValidationBudget struct{ nodes int }

func (b *typeValidationBudget) consume() error {
	b.nodes++
	if b.nodes > MaxTypeNodes {
		return errors.New("type expression exceeds node budget")
	}
	return nil
}

func collectUnionMembers(expression TypeExpression, depth int, budget *typeValidationBudget, out *[]TypeExpression) error {
	if depth > MaxTypeDepth {
		return errors.New("type expression exceeds depth budget")
	}
	if expression.Kind != TypeExpressionUnion {
		if err := expression.validateAt(depth, budget); err != nil {
			return err
		}
		*out = append(*out, expression)
		if len(*out) > MaxUnionMembers {
			return errors.New("union expression exceeds member budget")
		}
		return nil
	}
	if err := budget.consume(); err != nil {
		return err
	}
	if expression.Ref != nil || expression.Element != nil || len(expression.Members) < 2 || expression.Variable != "" || len(expression.Constraints) != 0 {
		return errors.New("union expression has invalid members")
	}
	for _, member := range expression.Members {
		if err := collectUnionMembers(member, depth+1, budget, out); err != nil {
			return err
		}
	}
	return nil
}
