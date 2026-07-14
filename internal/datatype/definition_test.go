package datatype

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/yottaapp/yotta/internal/artifact"
)

func TestSealDefinitionUsesStableSemanticDigest(t *testing.T) {
	draft := DefinitionDraft{
		TypeID:        "https://schemas.yotta.dev/types/core/string/v1",
		SchemaDialect: JSONSchemaDialect,
		SchemaBundle: []SchemaResource{{
			ID: "https://schemas.yotta.dev/types/core/string/v1/schema",
			Schema: json.RawMessage(`{
				"$schema":"https://json-schema.org/draft/2020-12/schema",
				"type":"string",
				"$id":"https://schemas.yotta.dev/types/core/string/v1/schema"
			}`),
		}},
		Representations: []RepresentationSpec{{
			Kind:  RepresentationInlineJSON,
			Codec: "yotta.jcs/v1",
		}},
		Authoring: Authoring{TitleKey: "type.core.string.title"},
	}

	definition, err := SealDefinition(draft)
	if err != nil {
		t.Fatal(err)
	}
	const want = artifact.Digest("sha256:b72d7af7a8647481d7602af40e672b7ed1f216a96b264e956bb50aa37cf57305")
	if got := definition.TypeRef().SemanticDigest; got != want {
		t.Fatalf("semantic digest = %q, want %q", got, want)
	}

	draft.Authoring.TitleKey = "type.core.string.renamed"
	renamed, err := SealDefinition(draft)
	if err != nil {
		t.Fatal(err)
	}
	if got := renamed.TypeRef().SemanticDigest; got != want {
		t.Fatalf("authoring changed semantic digest to %q", got)
	}

	reopened, err := OpenDefinition(definition.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if reopened.TypeRef() != definition.TypeRef() {
		t.Fatalf("reopened ref = %#v, want %#v", reopened.TypeRef(), definition.TypeRef())
	}
}

func TestSealDefinitionRejectsDuplicateSchemaKeys(t *testing.T) {
	const schemaID = "https://schemas.yotta.dev/types/core/broken/v1/schema"
	_, err := SealDefinition(DefinitionDraft{
		TypeID:        "https://schemas.yotta.dev/types/core/broken/v1",
		SchemaDialect: JSONSchemaDialect,
		SchemaBundle: []SchemaResource{{
			ID: schemaID,
			Schema: json.RawMessage(`{
				"$id":"https://schemas.yotta.dev/types/core/broken/v1/schema",
				"$schema":"https://json-schema.org/draft/2020-12/schema",
				"type":"string",
				"type":"number"
			}`),
		}},
		Representations: []RepresentationSpec{{Kind: RepresentationInlineJSON, Codec: "yotta.jcs/v1"}},
	})
	if err == nil {
		t.Fatal("accepted schema with duplicate object keys")
	}
}

func TestSealDefinitionRejectsUnregisteredPresentationAndCodec(t *testing.T) {
	draft := definitionDraftForTest()
	draft.Authoring.EditorAdapter = "https://plugin.example/editor.js"
	if _, err := SealDefinition(draft); err == nil {
		t.Fatal("accepted plugin editor adapter")
	}

	draft = definitionDraftForTest()
	draft.Representations[0].Codec = " yotta.jcs/v1 "
	if _, err := SealDefinition(draft); err == nil {
		t.Fatal("accepted non-canonical codec identity")
	}
}

func TestSealDefinitionRejectsUnbundledRefAndOversizedInput(t *testing.T) {
	draft := definitionDraftForTest()
	draft.SchemaBundle[0].Schema = json.RawMessage(`{
		"$id":"https://schemas.yotta.dev/types/core/string/v1/schema",
		"$schema":"https://json-schema.org/draft/2020-12/schema",
		"$ref":"https://remote.example/missing/schema"
	}`)
	if _, err := SealDefinition(draft); err == nil {
		t.Fatal("accepted unbundled schema reference")
	}

	draft = definitionDraftForTest()
	draft.SchemaBundle[0].Schema = json.RawMessage(`{"padding":"` + strings.Repeat("x", MaxSchemaResourceBytes) + `"}`)
	if _, err := SealDefinition(draft); err == nil {
		t.Fatal("accepted schema resource over byte budget")
	}
}

func TestSealDefinitionResolvesRelativeRefAgainstNestedID(t *testing.T) {
	draft := definitionDraftForTest()
	draft.SchemaBundle[0].Schema = json.RawMessage(`{
		"$id":"https://schemas.yotta.dev/types/core/string/v1/schema",
		"$schema":"https://json-schema.org/draft/2020-12/schema",
		"$defs":{
			"container":{"$id":"embedded/","$ref":"value"},
			"value":{"$id":"embedded/value","type":"string"}
		}
	}`)
	if _, err := SealDefinition(draft); err != nil {
		t.Fatalf("rejected bundled relative ref with nested base: %v", err)
	}
}

func TestSealDefinitionDoesNotInterpretInstanceDataAsSchema(t *testing.T) {
	draft := definitionDraftForTest()
	literals := strings.TrimSuffix(strings.Repeat(`"$ref",`, MaxSchemaReferences+1), ",")
	draft.SchemaBundle[0].Schema = json.RawMessage(`{
		"$id":"https://schemas.yotta.dev/types/core/string/v1/schema",
		"$schema":"https://json-schema.org/draft/2020-12/schema",
		"const":{"$id":"not-a-schema-id","$ref":"literal-instance-data","values":[` + literals + `]}
	}`)
	if _, err := SealDefinition(draft); err != nil {
		t.Fatalf("interpreted const instance as a subschema: %v", err)
	}
}

func FuzzOpenDefinitionNeverPanics(f *testing.F) {
	definition, err := SealDefinition(definitionDraftForTest())
	if err != nil {
		f.Fatal(err)
	}
	f.Add(definition.Bytes())
	f.Add([]byte(`{"format":"yotta.data-type"}`))
	f.Add([]byte(`{"x":[[[[[null]]]]]}`))
	f.Add([]byte(strings.Repeat("[", MaxDefinitionDepth+1) + "0" + strings.Repeat("]", MaxDefinitionDepth+1)))

	f.Fuzz(func(t *testing.T, raw []byte) {
		_, _ = OpenDefinition(raw)
	})
}

func definitionDraftForTest() DefinitionDraft {
	const schemaID = "https://schemas.yotta.dev/types/core/string/v1/schema"
	return DefinitionDraft{
		TypeID:        "https://schemas.yotta.dev/types/core/string/v1",
		SchemaDialect: JSONSchemaDialect,
		SchemaBundle: []SchemaResource{{
			ID: schemaID,
			Schema: json.RawMessage(`{
				"$id":"https://schemas.yotta.dev/types/core/string/v1/schema",
				"$schema":"https://json-schema.org/draft/2020-12/schema",
				"type":"string"
			}`),
		}},
		Representations: []RepresentationSpec{{Kind: RepresentationInlineJSON, Codec: CodecJCSV1}},
	}
}
