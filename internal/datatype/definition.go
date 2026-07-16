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
	"github.com/yottaapp/yotta/internal/contractschema"
)

const (
	Format            = "yotta.data-type"
	Version           = "3.1"
	JSONSchemaDialect = "https://json-schema.org/draft/2020-12/schema"

	semanticDigestDomain   = "yotta/data-type-semantic/v1"
	MaxDefinitionBytes     = 1 << 20
	MaxDefinitionDepth     = 96
	MaxDefinitionNodes     = 262_144
	MaxSchemaResources     = contractschema.MaxResources
	MaxSchemaResourceBytes = contractschema.MaxResourceBytes
	MaxSchemaBundleBytes   = contractschema.MaxBundleBytes
	MaxSchemaDepth         = contractschema.MaxDepth
	MaxSchemaNodes         = contractschema.MaxNodes
	MaxSchemaReferences    = contractschema.MaxReferences
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
	"color-range": {},
	"point":       {},
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
	Kind  RepresentationKind `json:"kind" jsonschema:"required,enum=inline-json,enum=blob-ref,enum=stream-ref,enum=handle-ref"`
	Codec string             `json:"codec"`
}

type SchemaResource = contractschema.Resource

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
	SchemaRoot      string
	SchemaBundle    []SchemaResource
	Representations []RepresentationSpec
	Authoring       Authoring
}

// MachineDefinition is the compiler-facing semantic portion of a data type.
type MachineDefinition struct {
	TypeID          string               `json:"typeId"`
	SchemaDialect   string               `json:"schemaDialect"`
	SchemaRoot      string               `json:"schemaRoot"`
	SchemaBundle    []SchemaResource     `json:"schemaBundle"`
	Representations []RepresentationSpec `json:"representations"`
}

type definitionDocument struct {
	Format    string            `json:"format"`
	Version   string            `json:"version"`
	TypeRef   TypeRef           `json:"typeRef"`
	Semantic  MachineDefinition `json:"semantic"`
	Authoring Authoring         `json:"authoring"`
}

type definitionState struct {
	document definitionDocument
	bytes    []byte
}

type Definition struct{ state *definitionState }

func SealDefinition(draft DefinitionDraft) (Definition, error) {
	semantic, err := normalizeSemantic(MachineDefinition{
		TypeID:          draft.TypeID,
		SchemaDialect:   draft.SchemaDialect,
		SchemaRoot:      draft.SchemaRoot,
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

// OpenSemanticDefinition validates the machine-only portion of a data type
// against its pinned TypeRef. Presentation metadata is intentionally absent.
func OpenSemanticDefinition(ref TypeRef, raw []byte) (Definition, error) {
	if err := ref.Validate(); err != nil {
		return Definition{}, fmt.Errorf("invalid data type reference: %w", err)
	}
	if len(raw) == 0 || len(raw) > MaxDefinitionBytes {
		return Definition{}, errors.New("data type semantic artifact exceeds byte budget")
	}
	if err := inspectJSONBudget(raw, MaxDefinitionDepth, MaxDefinitionNodes); err != nil {
		return Definition{}, fmt.Errorf("data type semantic artifact exceeds structural budget: %w", err)
	}
	canonical, err := artifact.Canonicalize(raw)
	if err != nil {
		return Definition{}, fmt.Errorf("canonicalize data type semantic artifact: %w", err)
	}
	if !bytes.Equal(raw, canonical) {
		return Definition{}, errors.New("data type semantic artifact is not canonical")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	var semantic MachineDefinition
	if err := decoder.Decode(&semantic); err != nil {
		return Definition{}, fmt.Errorf("decode data type semantic artifact: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return Definition{}, errors.New("data type semantic artifact contains trailing JSON values")
	}
	normalized, err := normalizeSemantic(semantic)
	if err != nil {
		return Definition{}, err
	}
	sealed, err := sealNormalized(normalized, Authoring{})
	if err != nil {
		return Definition{}, err
	}
	if sealed.TypeRef() != ref || !bytes.Equal(sealed.SemanticBytes(), raw) {
		return Definition{}, errors.New("data type semantic digest mismatch")
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

func (d Definition) SemanticBytes() []byte {
	if !d.Valid() {
		return nil
	}
	canonical, err := artifact.Marshal(d.state.document.Semantic)
	if err != nil {
		panic("data type definition invariant: " + err.Error())
	}
	return canonical
}

func (d Definition) Machine() MachineDefinition {
	if !d.Valid() {
		return MachineDefinition{}
	}
	raw, err := json.Marshal(d.state.document.Semantic)
	if err != nil {
		panic("data type definition invariant: " + err.Error())
	}
	var machine MachineDefinition
	if err := json.Unmarshal(raw, &machine); err != nil {
		panic("data type definition invariant: " + err.Error())
	}
	return machine
}

func (d Definition) Authoring() Authoring {
	if !d.Valid() {
		return Authoring{}
	}
	raw, err := json.Marshal(d.state.document.Authoring)
	if err != nil {
		panic("data type definition invariant: " + err.Error())
	}
	var authoring Authoring
	if err := json.Unmarshal(raw, &authoring); err != nil {
		panic("data type definition invariant: " + err.Error())
	}
	return authoring
}

func sealNormalized(semantic MachineDefinition, authoring Authoring) (Definition, error) {
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

func normalizeSemantic(source MachineDefinition) (MachineDefinition, error) {
	if err := validateTypeID(source.TypeID); err != nil {
		return MachineDefinition{}, fmt.Errorf("invalid data type id: %w", err)
	}
	if source.SchemaDialect != JSONSchemaDialect {
		return MachineDefinition{}, errors.New("unsupported data type schema dialect")
	}
	bundle, err := contractschema.Normalize(source.SchemaDialect, "", source.SchemaBundle)
	if err != nil {
		return MachineDefinition{}, fmt.Errorf("normalize data type schema bundle: %w", err)
	}
	rootFound := false
	for _, resource := range bundle {
		if resource.ID == source.SchemaRoot {
			rootFound = true
			break
		}
	}
	if !rootFound {
		return MachineDefinition{}, errors.New("data type schema root is not present in its bundle")
	}

	if len(source.Representations) == 0 || len(source.Representations) > 4 {
		return MachineDefinition{}, errors.New("data type must declare one to four representations")
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
			return MachineDefinition{}, errors.New("data type contains invalid representation")
		}
		if seenKinds[representation.Kind] {
			return MachineDefinition{}, fmt.Errorf("duplicate data type representation %q", representation.Kind)
		}
		seenKinds[representation.Kind] = true
	}
	return MachineDefinition{
		TypeID:          source.TypeID,
		SchemaDialect:   source.SchemaDialect,
		SchemaRoot:      source.SchemaRoot,
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
