package datatype

import (
	"strings"
	"testing"

	"github.com/yottaapp/yotta/internal/artifact"
)

func TestAssignableUsesNominalUnionAndListSemantics(t *testing.T) {
	stringType := RefExpression(typeRefForTest("https://schemas.yotta.dev/types/core/string/v1", "1"))
	numberType := RefExpression(typeRefForTest("https://schemas.yotta.dev/types/core/number/v1", "2"))
	stringWithOtherDigest := RefExpression(typeRefForTest("https://schemas.yotta.dev/types/core/string/v1", "3"))
	stringOrNumber, err := UnionExpression(stringType, numberType)
	if err != nil {
		t.Fatal(err)
	}

	assertAssignable(t, stringType, stringOrNumber, true)
	assertAssignable(t, stringOrNumber, stringType, false)
	assertAssignable(t, ListExpression(stringType), ListExpression(stringOrNumber), true)
	assertAssignable(t, stringType, stringWithOtherDigest, false)
}

func TestResolvedTypeRejectsNonConcreteRuntimeTypes(t *testing.T) {
	stringType := RefResolvedType(typeRefForTest("https://schemas.yotta.dev/types/core/string/v1", "1"))
	if err := ListResolvedType(ListResolvedType(stringType)).Validate(); err != nil {
		t.Fatalf("valid nested resolved type rejected: %v", err)
	}
	for _, invalid := range []ResolvedType{
		{},
		{Kind: ResolvedTypeKind("union")},
		{Kind: ResolvedTypeList},
	} {
		if err := invalid.Validate(); err == nil {
			t.Fatalf("accepted invalid runtime type %#v", invalid)
		}
	}
}

func TestMatchResolvedBindsOneVariableConsistentlyAcrossPorts(t *testing.T) {
	stringRef := typeRefForTest("https://schemas.yotta.dev/types/core/string/v1", "1")
	numberRef := typeRefForTest("https://schemas.yotta.dev/types/core/number/v1", "2")
	variable := VariableExpression("T")
	bindings := map[string]ResolvedType{}
	matched, err := MatchResolved(ListExpression(variable), ListResolvedType(RefResolvedType(stringRef)), bindings)
	if err != nil || !matched {
		t.Fatalf("list variable did not bind: matched=%t err=%v", matched, err)
	}
	matched, err = MatchResolved(variable, RefResolvedType(stringRef), bindings)
	if err != nil || !matched {
		t.Fatalf("matching output variable rejected: matched=%t err=%v", matched, err)
	}
	matched, err = MatchResolved(variable, RefResolvedType(numberRef), bindings)
	if err != nil || matched {
		t.Fatalf("inconsistent variable accepted: matched=%t err=%v", matched, err)
	}
}

func TestUnionExpressionFlattensAndRejectsCyclicInput(t *testing.T) {
	stringType := RefExpression(typeRefForTest("https://schemas.yotta.dev/types/core/string/v1", "1"))
	numberType := RefExpression(typeRefForTest("https://schemas.yotta.dev/types/core/number/v1", "2"))
	inner, err := UnionExpression(numberType, stringType)
	if err != nil {
		t.Fatal(err)
	}
	flattened, err := UnionExpression(stringType, inner)
	if err != nil {
		t.Fatal(err)
	}
	if flattened.Kind != TypeExpressionUnion || len(flattened.Members) != 2 {
		t.Fatalf("flattened union = %#v", flattened)
	}

	cyclic := TypeExpression{Kind: TypeExpressionList}
	cyclic.Element = &cyclic
	if _, err := UnionExpression(stringType, cyclic); err == nil {
		t.Fatal("accepted cyclic expression")
	}
}

func TestTypeRefRequiresIndependentVersionedIdentity(t *testing.T) {
	ref := typeRefForTest("https://schemas.yotta.dev/types/core/string", "1")
	if err := ref.Validate(); err == nil {
		t.Fatal("accepted unversioned type id")
	}
	ref.TypeID = "https://schemas.yotta.dev/types/core/string/not-versioned"
	if err := ref.Validate(); err == nil {
		t.Fatal("accepted product version as type identity version")
	}
}

func assertAssignable(t *testing.T, output, input TypeExpression, want bool) {
	t.Helper()
	got, err := Assignable(output, input)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("Assignable(%#v, %#v) = %v, want %v", output, input, got, want)
	}
}

func typeRefForTest(typeID, digit string) TypeRef {
	return TypeRef{TypeID: typeID, SemanticDigest: artifact.Digest("sha256:" + strings.Repeat(digit, 64))}
}
