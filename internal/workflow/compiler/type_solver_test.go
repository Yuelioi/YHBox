package compiler

import (
	"testing"

	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodes"
)

func TestTypeSolverExecutesCatalogConstraints(t *testing.T) {
	builtins, err := nodes.Build()
	if err != nil {
		t.Fatal(err)
	}
	ordered := datatype.VariableExpression("T", string(datatype.TraitOrdered))
	solver := newTypeSolver(builtins.Catalog.TypeSystem())
	if err := solver.unify(
		scopedTypeExpression{scope: "repeat", expression: datatype.RefExpression(builtins.IntegerType.TypeRef())},
		scopedTypeExpression{scope: "comparison", expression: ordered},
	); err != nil {
		t.Fatalf("integer did not satisfy Ordered: %v", err)
	}

	solver = newTypeSolver(builtins.Catalog.TypeSystem())
	if err := solver.unify(
		scopedTypeExpression{scope: "source", expression: datatype.RefExpression(builtins.BooleanType.TypeRef())},
		scopedTypeExpression{scope: "comparison", expression: ordered},
	); err == nil {
		t.Fatal("boolean satisfied Ordered")
	}
}

func TestTypeSolverMergesRepeatedBindingsWithoutEdgeOrderDependence(t *testing.T) {
	builtins, err := nodes.Build()
	if err != nil {
		t.Fatal(err)
	}
	variable := scopedTypeExpression{scope: "select", expression: datatype.VariableExpression("T", string(datatype.TraitNumeric))}
	integer := scopedTypeExpression{scope: "integer", expression: datatype.RefExpression(builtins.IntegerType.TypeRef())}
	number := scopedTypeExpression{scope: "number", expression: datatype.RefExpression(builtins.NumberType.TypeRef())}
	for _, order := range [][2]scopedTypeExpression{{integer, number}, {number, integer}} {
		solver := newTypeSolver(builtins.Catalog.TypeSystem())
		if err := solver.unify(order[0], variable); err != nil {
			t.Fatal(err)
		}
		if err := solver.unify(order[1], variable); err != nil {
			t.Fatal(err)
		}
		resolved, err := solver.resolve(variable)
		if err != nil || resolved.Ref == nil || *resolved.Ref != builtins.NumberType.TypeRef() {
			t.Fatalf("resolved repeated binding = %#v, %v", resolved, err)
		}
	}
}
