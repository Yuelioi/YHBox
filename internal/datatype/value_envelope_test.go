package datatype

import (
	"bytes"
	"testing"
)

func TestInlineValueEnvelopeRoundTripsTypeAndCanonicalValue(t *testing.T) {
	definition, err := SealDefinition(DefinitionDraft{
		TypeID: "https://schemas.yotta.dev/types/test/string/v1", SchemaDialect: JSONSchemaDialect,
		SchemaBundle:    []SchemaResource{{ID: "https://schemas.yotta.dev/types/test/string/v1/schema", Schema: []byte(`{"$id":"https://schemas.yotta.dev/types/test/string/v1/schema","$schema":"https://json-schema.org/draft/2020-12/schema","type":"string"}`)}},
		Representations: []RepresentationSpec{{Kind: RepresentationInlineJSON, Codec: CodecJCSV1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	wantType := RefResolvedType(definition.TypeRef())
	sealed, err := SealInlineJSON(wantType, []byte(`"hello"`))
	if err != nil {
		t.Fatal(err)
	}
	opened, err := OpenValueEnvelope(sealed.Artifact())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(opened.InlineJSON(), []byte(`"hello"`)) || opened.Type().Ref == nil || *opened.Type().Ref != definition.TypeRef() {
		t.Fatalf("opened envelope lost type or value: %#v %s", opened.Type(), opened.InlineJSON())
	}
	forged := append([]byte(nil), sealed.Artifact()...)
	forged[len(forged)-2] ^= 1
	if _, err := OpenValueEnvelope(forged); err == nil {
		t.Fatal("accepted forged value envelope")
	}
}
