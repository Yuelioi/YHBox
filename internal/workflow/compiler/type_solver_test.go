package compiler

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/yottaapp/yotta/internal/artifact"
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

func TestTypeSolverMatchesSharedConnectionPlanFixture(t *testing.T) {
	raw, err := os.ReadFile("testdata/connection_plan_parity.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture connectionPlanParityFixture
	if err := json.Unmarshal(raw, &fixture); err != nil || fixture.Version != 1 || len(fixture.Cases) == 0 || len(fixture.Cases) > 64 {
		t.Fatalf("invalid connection parity fixture: %#v, %v", fixture, err)
	}
	builtins, err := nodes.Build()
	if err != nil {
		t.Fatal(err)
	}
	named := map[string]datatype.TypeRef{
		"string":  builtins.StringType.TypeRef(),
		"number":  builtins.NumberType.TypeRef(),
		"integer": builtins.IntegerType.TypeRef(),
		"boolean": builtins.BooleanType.TypeRef(),
	}
	seen := make(map[string]bool, len(fixture.Cases))
	for _, test := range fixture.Cases {
		if test.ID == "" || seen[test.ID] {
			t.Fatalf("connection parity fixture contains invalid case ID %q", test.ID)
		}
		seen[test.ID] = true
		if test.Expected != "exact" && test.Expected != "assignable" && test.Expected != "generic-bind" && test.Expected != "invalid" {
			t.Fatalf("connection parity case %q has invalid expectation %q", test.ID, test.Expected)
		}
		t.Run(test.ID, func(t *testing.T) {
			output := resolveParityExpression(t, test.Output, named, 0)
			input := resolveParityExpression(t, test.Input, named, 0)
			solver := newTypeSolver(builtins.Catalog.TypeSystem())
			err := solver.unify(
				scopedTypeExpression{scope: "output", expression: output},
				scopedTypeExpression{scope: "input", expression: input},
			)
			if got, want := err == nil, test.Expected != "invalid"; got != want {
				t.Fatalf("Compiler direct compatibility = %v, want %v (error: %v)", got, want, err)
			}
		})
	}
}

type connectionPlanParityFixture struct {
	Version int                        `json:"version"`
	Cases   []connectionPlanParityCase `json:"cases"`
}

type connectionPlanParityCase struct {
	ID       string                     `json:"id"`
	Output   connectionParityExpression `json:"output"`
	Input    connectionParityExpression `json:"input"`
	Expected string                     `json:"expected"`
}

type connectionParityExpression struct {
	Kind        string                       `json:"kind"`
	Name        string                       `json:"name"`
	Variable    string                       `json:"variable"`
	Constraints []string                     `json:"constraints"`
	StaleDigest bool                         `json:"staleDigest"`
	Element     *connectionParityExpression  `json:"element"`
	Members     []connectionParityExpression `json:"members"`
}

func resolveParityExpression(t *testing.T, source connectionParityExpression, named map[string]datatype.TypeRef, depth int) datatype.TypeExpression {
	t.Helper()
	if depth > datatype.MaxTypeDepth {
		t.Fatal("connection parity fixture exceeds type depth budget")
	}
	switch source.Kind {
	case "named":
		ref, ok := named[source.Name]
		if !ok {
			t.Fatalf("unknown fixture type %q", source.Name)
		}
		if source.StaleDigest {
			ref.SemanticDigest = artifact.Digest("sha256:0000000000000000000000000000000000000000000000000000000000000000")
		}
		return datatype.RefExpression(ref)
	case "variable":
		return datatype.VariableExpression(source.Variable, source.Constraints...)
	case "list":
		if source.Element == nil {
			t.Fatal("list fixture expression omitted element")
		}
		return datatype.ListExpression(resolveParityExpression(t, *source.Element, named, depth+1))
	case "union":
		members := make([]datatype.TypeExpression, len(source.Members))
		for index, member := range source.Members {
			members[index] = resolveParityExpression(t, member, named, depth+1)
		}
		expression, err := datatype.UnionExpression(members...)
		if err != nil {
			t.Fatalf("invalid union fixture: %v", err)
		}
		return expression
	default:
		t.Fatalf("unknown fixture expression kind %q", source.Kind)
		return datatype.TypeExpression{}
	}
}
