package datatype

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/yottaapp/yotta/internal/artifact"
)

func TestSealDefinitionUsesStableSemanticDigest(t *testing.T) {
	draft := DefinitionDraft{
		TypeID:        "https://schemas.yotta.dev/types/core/string/v1",
		SchemaDialect: JSONSchemaDialect,
		SchemaRoot:    "https://schemas.yotta.dev/types/core/string/v1/schema",
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
	const want = artifact.Digest("sha256:74c21cbe095673e47d76098c609362563684e099d33fdff73a10275768ad98eb")
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

func TestSemanticDefinitionArtifactExcludesAuthoring(t *testing.T) {
	draft := definitionDraftForTest()
	draft.Authoring = Authoring{TitleKey: "type.string.title"}
	definition, err := SealDefinition(draft)
	if err != nil {
		t.Fatal(err)
	}

	draft.Authoring = Authoring{TitleKey: "type.string.renamed", Color: "#fff"}
	renamed, err := SealDefinition(draft)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(definition.SemanticBytes(), renamed.SemanticBytes()) {
		t.Fatal("authoring changed machine type artifact")
	}
	reopened, err := OpenSemanticDefinition(definition.TypeRef(), definition.SemanticBytes())
	if err != nil {
		t.Fatal(err)
	}
	if reopened.TypeRef() != definition.TypeRef() || !bytes.Equal(reopened.SemanticBytes(), definition.SemanticBytes()) {
		t.Fatal("machine type artifact changed during round trip")
	}
}

func TestSealDefinitionRejectsDuplicateSchemaKeys(t *testing.T) {
	const schemaID = "https://schemas.yotta.dev/types/core/broken/v1/schema"
	_, err := SealDefinition(DefinitionDraft{
		TypeID:        "https://schemas.yotta.dev/types/core/broken/v1",
		SchemaDialect: JSONSchemaDialect,
		SchemaRoot:    schemaID,
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

func TestDefinitionPinsSchemaRootInsteadOfDependingOnBundleOrder(t *testing.T) {
	rootID := "https://schemas.yotta.dev/types/test/rooted/v1/z-root"
	draft := DefinitionDraft{
		TypeID: "https://schemas.yotta.dev/types/test/rooted/v1", SchemaDialect: JSONSchemaDialect, SchemaRoot: rootID,
		SchemaBundle: []SchemaResource{
			{ID: "https://schemas.yotta.dev/types/test/rooted/v1/a-helper", Schema: json.RawMessage(`{"$id":"https://schemas.yotta.dev/types/test/rooted/v1/a-helper","$schema":"https://json-schema.org/draft/2020-12/schema","type":"number"}`)},
			{ID: rootID, Schema: json.RawMessage(`{"$id":"https://schemas.yotta.dev/types/test/rooted/v1/z-root","$schema":"https://json-schema.org/draft/2020-12/schema","type":"string"}`)},
		},
		Representations: []RepresentationSpec{{Kind: RepresentationInlineJSON, Codec: CodecJCSV1}},
	}
	definition, err := SealDefinition(draft)
	if err != nil {
		t.Fatal(err)
	}
	if definition.Machine().SchemaRoot != rootID {
		t.Fatalf("schema root = %q", definition.Machine().SchemaRoot)
	}
	if _, err := SealInlineJSON(valueTypes{definition.TypeRef().TypeID: definition}, RefResolvedType(definition.TypeRef()), []byte(`"ok"`)); err != nil {
		t.Fatalf("root schema rejected string: %v", err)
	}
	if _, err := SealInlineJSON(valueTypes{definition.TypeRef().TypeID: definition}, RefResolvedType(definition.TypeRef()), []byte(`42`)); err == nil {
		t.Fatal("validator used the lexicographically first helper schema instead of the pinned root")
	}
}

func TestOpenDefinitionRejectsPreSchemaRootArtifact(t *testing.T) {
	definition, err := SealDefinition(definitionDraftForTest())
	if err != nil {
		t.Fatal(err)
	}
	legacy := bytes.Replace(definition.Bytes(), []byte(`"schemaRoot":"https://schemas.yotta.dev/types/core/string/v1/schema",`), nil, 1)
	if bytes.Equal(legacy, definition.Bytes()) {
		t.Fatal("test did not construct a pre-schemaRoot artifact")
	}
	if _, err := OpenDefinition(legacy); err == nil {
		t.Fatal("accepted a pre-schemaRoot Data Type artifact")
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
		SchemaRoot:    schemaID,
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
