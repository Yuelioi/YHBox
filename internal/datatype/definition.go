package datatype

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strings"

	"github.com/yottaapp/yotta/internal/artifact"
)

const (
	Format            = "yotta.data-type"
	Version           = "3.1"
	JSONSchemaDialect = "https://json-schema.org/draft/2020-12/schema"

	semanticDigestDomain   = "yotta/data-type-semantic/v1"
	MaxDefinitionBytes     = 1 << 20
	MaxDefinitionDepth     = 96
	MaxDefinitionNodes     = 262_144
	MaxSchemaResources     = 256
	MaxSchemaResourceBytes = 256 << 10
	MaxSchemaBundleBytes   = 1 << 20
	MaxSchemaDepth         = 64
	MaxSchemaNodes         = 65_536
	MaxSchemaReferences    = 1_024
	MaxAuthoringExamples   = 64
	MaxExampleBytes        = 64 << 10
	MaxAuthoringBytes      = 256 << 10
	MaxAnnotationBytes     = 4 << 10

	CodecJCSV1       = "yotta.jcs/v1"
	CodecBlobRefV1   = "yotta.blob-ref/v1"
	CodecStreamRefV1 = "yotta.stream-ref/v1"
	CodecHandleRefV1 = "yotta.handle-ref/v1"
)

var builtInEditorAdapters = map[string]struct{}{
	"point": {},
}

type TypeRef struct {
	TypeID         string          `json:"typeId"`
	SemanticDigest artifact.Digest `json:"semanticDigest"`
}

type RepresentationKind string

const (
	RepresentationInlineJSON RepresentationKind = "inline-json"
	RepresentationBlobRef    RepresentationKind = "blob-ref"
	RepresentationStreamRef  RepresentationKind = "stream-ref"
	RepresentationHandleRef  RepresentationKind = "handle-ref"
)

type RepresentationSpec struct {
	Kind  RepresentationKind `json:"kind"`
	Codec string             `json:"codec"`
}

type SchemaResource struct {
	ID     string          `json:"id"`
	Schema json.RawMessage `json:"schema"`
}

type Authoring struct {
	TitleKey       string            `json:"titleKey,omitempty"`
	DescriptionKey string            `json:"descriptionKey,omitempty"`
	Color          string            `json:"color,omitempty"`
	Icon           string            `json:"icon,omitempty"`
	EditorAdapter  string            `json:"editorAdapter,omitempty"`
	Examples       []json.RawMessage `json:"examples,omitempty"`
}

type DefinitionDraft struct {
	TypeID          string
	SchemaDialect   string
	SchemaBundle    []SchemaResource
	Representations []RepresentationSpec
	Authoring       Authoring
}

type semanticDocument struct {
	TypeID          string               `json:"typeId"`
	SchemaDialect   string               `json:"schemaDialect"`
	SchemaBundle    []SchemaResource     `json:"schemaBundle"`
	Representations []RepresentationSpec `json:"representations"`
}

type definitionDocument struct {
	Format    string           `json:"format"`
	Version   string           `json:"version"`
	TypeRef   TypeRef          `json:"typeRef"`
	Semantic  semanticDocument `json:"semantic"`
	Authoring Authoring        `json:"authoring"`
}

type definitionState struct {
	document definitionDocument
	bytes    []byte
}

type Definition struct{ state *definitionState }

func SealDefinition(draft DefinitionDraft) (Definition, error) {
	semantic, err := normalizeSemantic(semanticDocument{
		TypeID:          draft.TypeID,
		SchemaDialect:   draft.SchemaDialect,
		SchemaBundle:    draft.SchemaBundle,
		Representations: draft.Representations,
	})
	if err != nil {
		return Definition{}, err
	}
	authoring, err := normalizeAuthoring(draft.Authoring)
	if err != nil {
		return Definition{}, err
	}
	return sealNormalized(semantic, authoring)
}

func OpenDefinition(raw []byte) (Definition, error) {
	if len(raw) == 0 || len(raw) > MaxDefinitionBytes {
		return Definition{}, errors.New("data type definition exceeds byte budget")
	}
	if err := inspectJSONBudget(raw, MaxDefinitionDepth, MaxDefinitionNodes); err != nil {
		return Definition{}, fmt.Errorf("data type definition exceeds structural budget: %w", err)
	}
	canonical, err := artifact.Canonicalize(raw)
	if err != nil {
		return Definition{}, fmt.Errorf("canonicalize data type definition: %w", err)
	}
	if !bytes.Equal(raw, canonical) {
		return Definition{}, errors.New("data type definition is not canonical")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	var document definitionDocument
	if err := decoder.Decode(&document); err != nil {
		return Definition{}, fmt.Errorf("decode data type definition: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return Definition{}, errors.New("data type definition contains trailing values")
		}
		return Definition{}, fmt.Errorf("decode trailing data type definition value: %w", err)
	}
	if document.Format != Format || document.Version != Version {
		return Definition{}, errors.New("unsupported data type definition format")
	}
	semantic, err := normalizeSemantic(document.Semantic)
	if err != nil {
		return Definition{}, err
	}
	if document.TypeRef.TypeID != semantic.TypeID || !document.TypeRef.SemanticDigest.Valid() {
		return Definition{}, errors.New("data type reference does not match semantic definition")
	}
	authoring, err := normalizeAuthoring(document.Authoring)
	if err != nil {
		return Definition{}, err
	}
	sealed, err := sealNormalized(semantic, authoring)
	if err != nil {
		return Definition{}, err
	}
	if sealed.TypeRef() != document.TypeRef {
		return Definition{}, errors.New("data type semantic digest mismatch")
	}
	if !bytes.Equal(sealed.Bytes(), raw) {
		return Definition{}, errors.New("data type definition is not normalized")
	}
	return sealed, nil
}

func (d Definition) Valid() bool {
	return d.state != nil && d.state.document.TypeRef.SemanticDigest.Valid()
}

func (d Definition) TypeRef() TypeRef {
	if !d.Valid() {
		return TypeRef{}
	}
	return d.state.document.TypeRef
}

func (d Definition) Bytes() []byte {
	if !d.Valid() {
		return nil
	}
	return append([]byte(nil), d.state.bytes...)
}

func sealNormalized(semantic semanticDocument, authoring Authoring) (Definition, error) {
	canonicalSemantic, err := artifact.Marshal(semantic)
	if err != nil {
		return Definition{}, fmt.Errorf("encode data type semantics: %w", err)
	}
	digest, err := artifact.Sum(semanticDigestDomain, canonicalSemantic)
	if err != nil {
		return Definition{}, err
	}
	document := definitionDocument{
		Format:    Format,
		Version:   Version,
		TypeRef:   TypeRef{TypeID: semantic.TypeID, SemanticDigest: digest},
		Semantic:  semantic,
		Authoring: authoring,
	}
	canonical, err := artifact.Marshal(document)
	if err != nil {
		return Definition{}, fmt.Errorf("encode data type definition: %w", err)
	}
	if len(canonical) > MaxDefinitionBytes {
		return Definition{}, errors.New("data type definition exceeds byte budget")
	}
	return Definition{state: &definitionState{document: document, bytes: canonical}}, nil
}

func normalizeSemantic(source semanticDocument) (semanticDocument, error) {
	if err := validateTypeID(source.TypeID); err != nil {
		return semanticDocument{}, fmt.Errorf("invalid data type id: %w", err)
	}
	if source.SchemaDialect != JSONSchemaDialect {
		return semanticDocument{}, errors.New("unsupported data type schema dialect")
	}
	if len(source.SchemaBundle) == 0 || len(source.SchemaBundle) > MaxSchemaResources {
		return semanticDocument{}, errors.New("data type schema bundle exceeds resource budget")
	}
	bundle := make([]SchemaResource, len(source.SchemaBundle))
	seenResources := make(map[string]bool, len(bundle))
	totalSchemaBytes := 0
	for _, resource := range source.SchemaBundle {
		if err := validateAbsoluteURI(resource.ID); err != nil {
			return semanticDocument{}, fmt.Errorf("invalid schema resource id: %w", err)
		}
		if seenResources[resource.ID] {
			return semanticDocument{}, fmt.Errorf("duplicate schema resource %q", resource.ID)
		}
		seenResources[resource.ID] = true
		if len(resource.Schema) == 0 || len(resource.Schema) > MaxSchemaResourceBytes || totalSchemaBytes > MaxSchemaBundleBytes-len(resource.Schema) {
			return semanticDocument{}, errors.New("data type schema bundle exceeds byte budget")
		}
		totalSchemaBytes += len(resource.Schema)
		if err := inspectJSONBudget(resource.Schema, MaxSchemaDepth, MaxSchemaNodes); err != nil {
			return semanticDocument{}, fmt.Errorf("schema resource %q exceeds structural budget: %w", resource.ID, err)
		}
	}
	parsedSchemas := make([]any, len(source.SchemaBundle))
	for i, resource := range source.SchemaBundle {
		canonical, err := artifact.Canonicalize(resource.Schema)
		if err != nil {
			return semanticDocument{}, fmt.Errorf("canonicalize schema resource %q: %w", resource.ID, err)
		}
		var schema map[string]json.RawMessage
		if err := json.Unmarshal(canonical, &schema); err != nil || schema == nil {
			return semanticDocument{}, fmt.Errorf("schema resource %q must be a JSON object", resource.ID)
		}
		var schemaID string
		if err := json.Unmarshal(schema["$id"], &schemaID); err != nil || schemaID != resource.ID {
			return semanticDocument{}, fmt.Errorf("schema resource %q has mismatched $id", resource.ID)
		}
		var dialect string
		if err := json.Unmarshal(schema["$schema"], &dialect); err != nil || dialect != source.SchemaDialect {
			return semanticDocument{}, fmt.Errorf("schema resource %q has mismatched $schema", resource.ID)
		}
		parsed, err := decodeJSONValue(canonical)
		if err != nil {
			return semanticDocument{}, fmt.Errorf("decode schema resource %q: %w", resource.ID, err)
		}
		parsedSchemas[i] = parsed
		bundle[i] = SchemaResource{ID: resource.ID, Schema: append(json.RawMessage(nil), canonical...)}
	}
	allResourceIDs := make(map[string]bool, len(seenResources))
	for id := range seenResources {
		allResourceIDs[id] = true
	}
	for i, resource := range source.SchemaBundle {
		base, _ := url.Parse(resource.ID)
		if err := collectBundledResourceIDs(parsedSchemas[i], base, allResourceIDs, 0, true); err != nil {
			return semanticDocument{}, fmt.Errorf("schema resource %q: %w", resource.ID, err)
		}
	}
	referenceCount := 0
	for i, resource := range source.SchemaBundle {
		base, _ := url.Parse(resource.ID)
		if err := validateBundledReferences(parsedSchemas[i], base, allResourceIDs, 0, &referenceCount); err != nil {
			return semanticDocument{}, fmt.Errorf("schema resource %q: %w", resource.ID, err)
		}
	}
	sort.Slice(bundle, func(i, j int) bool { return bundle[i].ID < bundle[j].ID })

	if len(source.Representations) == 0 || len(source.Representations) > 4 {
		return semanticDocument{}, errors.New("data type must declare one to four representations")
	}
	representations := append([]RepresentationSpec(nil), source.Representations...)
	sort.Slice(representations, func(i, j int) bool {
		if representations[i].Kind == representations[j].Kind {
			return representations[i].Codec < representations[j].Codec
		}
		return representations[i].Kind < representations[j].Kind
	})
	seenKinds := map[RepresentationKind]bool{}
	for _, representation := range representations {
		if !validRepresentationCodec(representation.Kind, representation.Codec) {
			return semanticDocument{}, errors.New("data type contains invalid representation")
		}
		if seenKinds[representation.Kind] {
			return semanticDocument{}, fmt.Errorf("duplicate data type representation %q", representation.Kind)
		}
		seenKinds[representation.Kind] = true
	}
	return semanticDocument{
		TypeID:          source.TypeID,
		SchemaDialect:   source.SchemaDialect,
		SchemaBundle:    bundle,
		Representations: representations,
	}, nil
}

func normalizeAuthoring(source Authoring) (Authoring, error) {
	if len(source.Examples) > MaxAuthoringExamples {
		return Authoring{}, errors.New("data type authoring exceeds example budget")
	}
	for _, value := range []string{source.TitleKey, source.DescriptionKey, source.Color, source.Icon, source.EditorAdapter} {
		if len(value) > MaxAnnotationBytes {
			return Authoring{}, errors.New("data type authoring annotation exceeds byte budget")
		}
	}
	if source.EditorAdapter != "" {
		if _, ok := builtInEditorAdapters[source.EditorAdapter]; !ok {
			return Authoring{}, fmt.Errorf("unregistered data type editor adapter %q", source.EditorAdapter)
		}
	}
	examples := make([]json.RawMessage, len(source.Examples))
	totalExampleBytes := 0
	for i, example := range source.Examples {
		if len(example) == 0 || len(example) > MaxExampleBytes || totalExampleBytes > MaxAuthoringBytes-len(example) {
			return Authoring{}, errors.New("data type authoring examples exceed byte budget")
		}
		totalExampleBytes += len(example)
		if err := inspectJSONBudget(example, MaxSchemaDepth, MaxSchemaNodes); err != nil {
			return Authoring{}, fmt.Errorf("data type example exceeds structural budget: %w", err)
		}
		canonical, err := artifact.Canonicalize(example)
		if err != nil {
			return Authoring{}, fmt.Errorf("canonicalize data type example: %w", err)
		}
		examples[i] = append(json.RawMessage(nil), canonical...)
	}
	return Authoring{
		TitleKey: source.TitleKey, DescriptionKey: source.DescriptionKey,
		Color: source.Color, Icon: source.Icon, EditorAdapter: source.EditorAdapter,
		Examples: examples,
	}, nil
}

func validateAbsoluteURI(value string) error {
	parsed, err := url.Parse(value)
	if err != nil || !parsed.IsAbs() || parsed.Host == "" || parsed.Fragment != "" {
		return fmt.Errorf("%q must be an absolute URI without a fragment", value)
	}
	return nil
}

func validateTypeID(value string) error {
	if err := validateAbsoluteURI(value); err != nil {
		return err
	}
	parsed, _ := url.Parse(value)
	segments := strings.Split(strings.Trim(parsed.EscapedPath(), "/"), "/")
	if len(segments) == 0 || !typeVersionPattern.MatchString(segments[len(segments)-1]) {
		return fmt.Errorf("%q must end in an independent /vN version", value)
	}
	return nil
}

func validRepresentationCodec(kind RepresentationKind, codec string) bool {
	if strings.TrimSpace(codec) != codec {
		return false
	}
	switch kind {
	case RepresentationInlineJSON:
		return codec == CodecJCSV1
	case RepresentationBlobRef:
		return codec == CodecBlobRefV1
	case RepresentationStreamRef:
		return codec == CodecStreamRefV1
	case RepresentationHandleRef:
		return codec == CodecHandleRefV1
	default:
		return false
	}
}
