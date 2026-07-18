package datatype

import (
	"encoding/json"
	"testing"
)

func TestSystemUsesExplicitRelationsAndTraits(t *testing.T) {
	number := testSystemDefinition(t, "number", nil, []Trait{TraitNumeric, TraitOrdered})
	integer := testSystemDefinition(t, "integer", []TypeRef{number.TypeRef()}, []Trait{TraitNumeric, TraitOrdered})
	duration := testSystemDefinition(t, "duration", nil, []Trait{TraitOrdered})
	system, err := NewSystem([]Definition{number, integer, duration})
	if err != nil {
		t.Fatal(err)
	}
	ok, err := system.Assignable(RefExpression(integer.TypeRef()), RefExpression(number.TypeRef()))
	if err != nil || !ok {
		t.Fatalf("integer -> number = %v, %v", ok, err)
	}
	ok, err = system.Assignable(RefExpression(duration.TypeRef()), RefExpression(number.TypeRef()))
	if err != nil || ok {
		t.Fatalf("duration -> number = %v, %v", ok, err)
	}
	if ok, err = system.Satisfies(integer.TypeRef(), []string{string(TraitNumeric), string(TraitOrdered)}); err != nil || !ok {
		t.Fatalf("integer traits = %v, %v", ok, err)
	}
	if ok, err = system.Satisfies(duration.TypeRef(), []string{string(TraitNumeric)}); err != nil || ok {
		t.Fatalf("duration numeric = %v, %v", ok, err)
	}
}

func TestSystemKeepsListsInvariant(t *testing.T) {
	number := testSystemDefinition(t, "number", nil, nil)
	integer := testSystemDefinition(t, "integer", []TypeRef{number.TypeRef()}, nil)
	system, err := NewSystem([]Definition{number, integer})
	if err != nil {
		t.Fatal(err)
	}
	ok, err := system.Assignable(ListExpression(RefExpression(integer.TypeRef())), ListExpression(RefExpression(number.TypeRef())))
	if err != nil || ok {
		t.Fatalf("list<integer> -> list<number> = %v, %v", ok, err)
	}
}

func TestSystemFindsOrderIndependentLeastUpperBound(t *testing.T) {
	number := testSystemDefinition(t, "number", nil, []Trait{TraitNumeric})
	integer := testSystemDefinition(t, "integer", []TypeRef{number.TypeRef()}, []Trait{TraitNumeric})
	system, err := NewSystem([]Definition{number, integer})
	if err != nil {
		t.Fatal(err)
	}
	for _, pair := range [][2]TypeRef{{integer.TypeRef(), number.TypeRef()}, {number.TypeRef(), integer.TypeRef()}} {
		got, err := system.LeastUpperBoundRef(pair[0], pair[1])
		if err != nil || got != number.TypeRef() {
			t.Fatalf("least upper bound %q + %q = %#v, %v", pair[0].TypeID, pair[1].TypeID, got, err)
		}
	}
}

func TestSystemRejectsAmbiguousLeastUpperBound(t *testing.T) {
	topA := testSystemDefinition(t, "top-a", nil, nil)
	topB := testSystemDefinition(t, "top-b", nil, nil)
	left := testSystemDefinition(t, "left", []TypeRef{topA.TypeRef(), topB.TypeRef()}, nil)
	right := testSystemDefinition(t, "right", []TypeRef{topA.TypeRef(), topB.TypeRef()}, nil)
	system, err := NewSystem([]Definition{topA, topB, left, right})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := system.LeastUpperBoundRef(left.TypeRef(), right.TypeRef()); err == nil {
		t.Fatal("accepted an ambiguous least upper bound")
	}
}

func testSystemDefinition(t *testing.T, name string, targets []TypeRef, traits []Trait) Definition {
	t.Helper()
	id := "https://schemas.yotta.dev/types/test/" + name + "/v1"
	definition, err := SealDefinition(DefinitionDraft{
		TypeID: id, SchemaDialect: JSONSchemaDialect, SchemaRoot: id + "/schema",
		SchemaBundle:    []SchemaResource{{ID: id + "/schema", Schema: json.RawMessage(`{"$id":"` + id + `/schema","$schema":"https://json-schema.org/draft/2020-12/schema","type":"number"}`)}},
		Representations: []RepresentationSpec{{Kind: RepresentationInlineJSON, Codec: CodecJCSV1}},
		AssignableTo:    targets, Traits: traits,
	})
	if err != nil {
		t.Fatal(err)
	}
	return definition
}
