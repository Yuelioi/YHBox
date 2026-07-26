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
	Version           = "1"
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
	MaxStructureFields     = 256

	CodecJCSV1       = "yotta.jcs/v1"
	CodecBlobRefV1   = "yotta.blob-ref/v1"
	CodecStreamRefV1 = "yotta.stream-ref/v1"
	CodecHandleRefV1 = "yotta.handle-ref/v1"
)

var builtInEditorAdapters = map[string]struct{}{
	"color-range": {},
	"duration":    {},
	"point":       {},
	"region":      {},
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
	TitleKey            string            `json:"titleKey,omitempty"`
	DescriptionKey      string            `json:"descriptionKey,omitempty"`
	Color               string            `json:"color,omitempty"`
	Icon                string            `json:"icon,omitempty"`
	EditorAdapter       string            `json:"editorAdapter,omitempty"`
	Unit                string            `json:"unit,omitempty"`
	HelpKey             string            `json:"helpKey,omitempty"`
	Importance          string            `json:"importance,omitempty"`
	InlinePriority      int               `json:"inlinePriority,omitempty"`
	Preset              string            `json:"preset,omitempty"`
	Examples            []json.RawMessage `json:"examples,omitempty"`
	BreakTitleKey       string            `json:"breakTitleKey,omitempty"`
	BreakDescriptionKey string            `json:"breakDescriptionKey,omitempty"`
}

// StructureSpec is the semantic, typed decomposition contract for an object
// data type. JSON Schema validates values; this contract names the exact
// TypeExpression of each field and the stable node used to expose it.
type StructureSpec struct {
	BreakNodeTypeID string           `json:"breakNodeTypeId"`
	Fields          []StructureField `json:"fields"`
}

type StructureField struct {
	ID      string         `json:"id"`
	JSONKey string         `json:"jsonKey"`
	Type    TypeExpression `json:"type"`
}

type DefinitionDraft struct {
	TypeID          string
	SchemaDialect   string
	SchemaRoot      string
	SchemaBundle    []SchemaResource
	Representations []RepresentationSpec
	Traits          []Trait
	AssignableTo    []TypeRef
	Structure       *StructureSpec
	Authoring       Authoring
}

// MachineDefinition is the compiler-facing semantic portion of a data type.
type MachineDefinition struct {
	TypeID          string               `json:"typeId"`
	SchemaDialect   string               `json:"schemaDialect"`
	SchemaRoot      string               `json:"schemaRoot"`
	SchemaBundle    []SchemaResource     `json:"schemaBundle"`
	Representations []RepresentationSpec `json:"representations"`
	Traits          []Trait              `json:"traits,omitempty"`
	AssignableTo    []TypeRef            `json:"assignableTo,omitempty"`
	Structure       *StructureSpec       `json:"structure,omitempty"`
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
		Traits:          draft.Traits,
		AssignableTo:    draft.AssignableTo,
		Structure:       draft.Structure,
	})
	if err != nil {
		return Definition{}, err
	}
	authoring, err := normalizeAuthoring(draft.Authoring)
	if err != nil {
		return Definition{}, err
	}
	if err := validateAuthoringExamples(semantic, authoring); err != nil {
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
	if err := validateAuthoringExamples(semantic, authoring); err != nil {
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
	traits, err := normalizeTraits(source.Traits)
	if err != nil {
		return MachineDefinition{}, err
	}
	assignableTo := append([]TypeRef(nil), source.AssignableTo...)
	sort.Slice(assignableTo, func(i, j int) bool {
		if assignableTo[i].TypeID == assignableTo[j].TypeID {
			return assignableTo[i].SemanticDigest < assignableTo[j].SemanticDigest
		}
		return assignableTo[i].TypeID < assignableTo[j].TypeID
	})
	for index, target := range assignableTo {
		if err := target.Validate(); err != nil {
			return MachineDefinition{}, fmt.Errorf("invalid assignable target: %w", err)
		}
		if target.TypeID == source.TypeID {
			return MachineDefinition{}, errors.New("data type cannot declare itself as an assignable target")
		}
		if index > 0 && target == assignableTo[index-1] {
			return MachineDefinition{}, errors.New("data type contains duplicate assignable target")
		}
	}
	structure, err := normalizeStructure(source.Structure, bundle, source.SchemaRoot)
	if err != nil {
		return MachineDefinition{}, err
	}
	return MachineDefinition{
		TypeID:          source.TypeID,
		SchemaDialect:   source.SchemaDialect,
		SchemaRoot:      source.SchemaRoot,
		SchemaBundle:    bundle,
		Representations: representations,
		Traits:          traits,
		AssignableTo:    assignableTo,
		Structure:       structure,
	}, nil
}

func normalizeStructure(source *StructureSpec, bundle []SchemaResource, rootID string) (*StructureSpec, error) {
	if source == nil {
		return nil, nil
	}
	if err := validateAbsoluteURI(source.BreakNodeTypeID); err != nil {
		return nil, fmt.Errorf("invalid structure break node id: %w", err)
	}
	if len(source.Fields) == 0 || len(source.Fields) > MaxStructureFields {
		return nil, errors.New("data type structure field count is outside the supported range")
	}
	fields := append([]StructureField(nil), source.Fields...)
	sort.Slice(fields, func(i, j int) bool { return fields[i].ID < fields[j].ID })
	for index, field := range fields {
		if !validStructureFieldID(field.ID) {
			return nil, fmt.Errorf("invalid structure field id %q", field.ID)
		}
		if index > 0 && fields[index-1].ID == field.ID {
			return nil, fmt.Errorf("duplicate structure field %q", field.ID)
		}
		if field.JSONKey == "" {
			fields[index].JSONKey = field.ID
		} else if strings.TrimSpace(field.JSONKey) != field.JSONKey || len(field.JSONKey) > 256 {
			return nil, fmt.Errorf("invalid JSON key for structure field %q", field.ID)
		}
		if err := field.Type.Validate(); err != nil {
			return nil, fmt.Errorf("structure field %q has invalid type: %w", field.ID, err)
		}
		if expressionContainsVariable(field.Type) {
			return nil, fmt.Errorf("structure field %q must have a concrete type", field.ID)
		}
	}
	var root json.RawMessage
	for _, resource := range bundle {
		if resource.ID == rootID {
			root = resource.Schema
			break
		}
	}
	var schema map[string]any
	if err := json.Unmarshal(root, &schema); err != nil {
		return nil, errors.New("decode structured data type schema root")
	}
	if schema["type"] != "object" {
		return nil, errors.New("structured data type schema root must be an object")
	}
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		return nil, errors.New("structured data type schema must declare properties")
	}
	required := map[string]bool{}
	if values, ok := schema["required"].([]any); ok {
		for _, value := range values {
			if id, ok := value.(string); ok {
				required[id] = true
			}
		}
	}
	if len(properties) != len(fields) || len(required) != len(fields) {
		return nil, errors.New("structure fields must exactly cover required schema properties")
	}
	seenJSONKeys := map[string]bool{}
	for _, field := range fields {
		if seenJSONKeys[field.JSONKey] {
			return nil, fmt.Errorf("duplicate structure JSON key %q", field.JSONKey)
		}
		seenJSONKeys[field.JSONKey] = true
		if _, ok := properties[field.JSONKey]; !ok || !required[field.JSONKey] {
			return nil, fmt.Errorf("structure field %q JSON key %q is not a required schema property", field.ID, field.JSONKey)
		}
	}
	return &StructureSpec{BreakNodeTypeID: source.BreakNodeTypeID, Fields: fields}, nil
}

func expressionContainsVariable(expression TypeExpression) bool {
	switch expression.Kind {
	case TypeExpressionVariable:
		return true
	case TypeExpressionList:
		return expressionContainsVariable(*expression.Element)
	case TypeExpressionUnion:
		for _, member := range expression.Members {
			if expressionContainsVariable(member) {
				return true
			}
		}
	}
	return false
}

func validStructureFieldID(value string) bool {
	if value == "" || len(value) > 128 {
		return false
	}
	for index, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || char == '_' ||
			(index > 0 && ((char >= '0' && char <= '9') || char == '-' || char == '.')) {
			continue
		}
		return false
	}
	return true
}

func normalizeAuthoring(source Authoring) (Authoring, error) {
	if len(source.Examples) > MaxAuthoringExamples {
		return Authoring{}, errors.New("data type authoring exceeds example budget")
	}
	for _, value := range []string{source.TitleKey, source.DescriptionKey, source.Color, source.Icon, source.EditorAdapter, source.Unit, source.HelpKey, source.Importance, source.Preset, source.BreakTitleKey, source.BreakDescriptionKey} {
		if len(value) > MaxAnnotationBytes {
			return Authoring{}, errors.New("data type authoring annotation exceeds byte budget")
		}
	}
	if source.EditorAdapter != "" {
		if _, ok := builtInEditorAdapters[source.EditorAdapter]; !ok {
			return Authoring{}, fmt.Errorf("unregistered data type editor adapter %q", source.EditorAdapter)
		}
	}
	if source.Importance != "" && source.Importance != "primary" && source.Importance != "common" && source.Importance != "advanced" {
		return Authoring{}, fmt.Errorf("invalid data type authoring importance %q", source.Importance)
	}
	if source.InlinePriority < 0 || source.InlinePriority > 1000 {
		return Authoring{}, errors.New("data type inline priority must be between 0 and 1000")
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
		Color: source.Color, Icon: source.Icon, EditorAdapter: source.EditorAdapter, Unit: source.Unit,
		HelpKey: source.HelpKey, Importance: source.Importance, InlinePriority: source.InlinePriority, Preset: source.Preset,
		Examples: examples, BreakTitleKey: source.BreakTitleKey, BreakDescriptionKey: source.BreakDescriptionKey,
	}, nil
}

func validateAuthoringExamples(machine MachineDefinition, authoring Authoring) error {
	for index, raw := range authoring.Examples {
		var value any
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.UseNumber()
		if err := decoder.Decode(&value); err != nil {
			return fmt.Errorf("decode data type example %d: %w", index, err)
		}
		if err := validateDefinitionInline(machine, value); err != nil {
			return fmt.Errorf("data type example %d violates its pinned schema: %w", index, err)
		}
	}
	return nil
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
