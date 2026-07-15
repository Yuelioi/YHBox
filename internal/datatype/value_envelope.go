package datatype

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	runtimejsonschema "github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/resource"
	"github.com/yottaapp/yotta/internal/stream"
)

const (
	ValueEnvelopeFormat   = "yotta.value-envelope"
	ValueEnvelopeVersion  = "3.1"
	MaxValueEnvelopeBytes = 4 << 20
	MaxInlineValueBytes   = 1 << 20
	valueDigestDomain     = "yotta/value-envelope/v2"
)

type valueEnvelopeDocument struct {
	Format      string             `json:"format"`
	Version     string             `json:"version"`
	ValueDigest artifact.Digest    `json:"valueDigest"`
	Type        ResolvedType       `json:"type"`
	Repr        RepresentationKind `json:"repr"`
	Codec       string             `json:"codec"`
	Value       json.RawMessage    `json:"value"`
}

type valueEnvelopeState struct {
	document valueEnvelopeDocument
	artifact []byte
}

// ValueEnvelope is the immutable, typed value passed across Program 3.1
// boundaries. External values contain only strict references; raw resources
// remain inside the Blob Store or Resource Broker.
type ValueEnvelope struct{ state *valueEnvelopeState }

type ValueTypeCatalog interface {
	LookupType(string) (Definition, bool)
}

func SealInlineJSON(catalog ValueTypeCatalog, resolved ResolvedType, raw []byte) (ValueEnvelope, error) {
	if err := resolved.Validate(); err != nil {
		return ValueEnvelope{}, fmt.Errorf("invalid value type: %w", err)
	}
	if len(raw) == 0 || len(raw) > MaxInlineValueBytes {
		return ValueEnvelope{}, errors.New("inline value exceeds byte budget")
	}
	if err := artifact.InspectJSONBudget(raw, MaxDefinitionDepth, MaxDefinitionNodes, MaxInlineValueBytes); err != nil {
		return ValueEnvelope{}, fmt.Errorf("inline value exceeds structural budget: %w", err)
	}
	canonical, err := artifact.Canonicalize(raw)
	if err != nil {
		return ValueEnvelope{}, fmt.Errorf("canonicalize inline value: %w", err)
	}
	if err := validateCarrierAgainstType(catalog, resolved, RepresentationInlineJSON, CodecJCSV1, canonical); err != nil {
		return ValueEnvelope{}, err
	}
	return sealValue(resolved, RepresentationInlineJSON, CodecJCSV1, canonical)
}

func SealBlobRef(catalog ValueTypeCatalog, resolved ResolvedType, ref blob.Ref) (ValueEnvelope, error) {
	if err := ref.Validate(); err != nil {
		return ValueEnvelope{}, fmt.Errorf("invalid blob reference: %w", err)
	}
	if err := validateCarrierAgainstType(catalog, resolved, RepresentationBlobRef, CodecBlobRefV1, nil); err != nil {
		return ValueEnvelope{}, err
	}
	return sealCarrier(resolved, RepresentationBlobRef, CodecBlobRefV1, ref)
}

func SealStreamRef(catalog ValueTypeCatalog, resolved ResolvedType, handle resource.Handle) (ValueEnvelope, error) {
	if err := handle.Validate(); err != nil {
		return ValueEnvelope{}, fmt.Errorf("invalid stream reference: %w", err)
	}
	if handle.Kind != stream.Kind {
		return ValueEnvelope{}, fmt.Errorf("stream reference kind %q, require %q", handle.Kind, stream.Kind)
	}
	if err := validateCarrierAgainstType(catalog, resolved, RepresentationStreamRef, CodecStreamRefV1, nil); err != nil {
		return ValueEnvelope{}, err
	}
	return sealCarrier(resolved, RepresentationStreamRef, CodecStreamRefV1, handle)
}

func SealHandleRef(catalog ValueTypeCatalog, resolved ResolvedType, handle resource.Handle) (ValueEnvelope, error) {
	if err := handle.Validate(); err != nil {
		return ValueEnvelope{}, fmt.Errorf("invalid handle reference: %w", err)
	}
	if err := validateCarrierAgainstType(catalog, resolved, RepresentationHandleRef, CodecHandleRefV1, nil); err != nil {
		return ValueEnvelope{}, err
	}
	return sealCarrier(resolved, RepresentationHandleRef, CodecHandleRefV1, handle)
}

func sealCarrier(resolved ResolvedType, representation RepresentationKind, codec string, value any) (ValueEnvelope, error) {
	canonical, err := artifact.Marshal(value)
	if err != nil {
		return ValueEnvelope{}, fmt.Errorf("encode value carrier: %w", err)
	}
	return sealValue(resolved, representation, codec, canonical)
}

func sealValue(resolved ResolvedType, representation RepresentationKind, codec string, canonical json.RawMessage) (ValueEnvelope, error) {
	if err := resolved.Validate(); err != nil {
		return ValueEnvelope{}, fmt.Errorf("invalid value type: %w", err)
	}
	if !validRepresentationCodec(representation, codec) {
		return ValueEnvelope{}, errors.New("invalid value representation codec")
	}
	body := struct {
		Version string             `json:"version"`
		Type    ResolvedType       `json:"type"`
		Repr    RepresentationKind `json:"repr"`
		Codec   string             `json:"codec"`
		Value   json.RawMessage    `json:"value"`
	}{ValueEnvelopeVersion, resolved, representation, codec, canonical}
	bodyBytes, err := artifact.Marshal(body)
	if err != nil {
		return ValueEnvelope{}, err
	}
	digest, err := artifact.Sum(valueDigestDomain, bodyBytes)
	if err != nil {
		return ValueEnvelope{}, err
	}
	document := valueEnvelopeDocument{
		Format: ValueEnvelopeFormat, Version: ValueEnvelopeVersion, ValueDigest: digest,
		Type: resolved, Repr: representation, Codec: codec, Value: canonical,
	}
	artifactBytes, err := artifact.Marshal(document)
	if err != nil {
		return ValueEnvelope{}, err
	}
	if len(artifactBytes) > MaxValueEnvelopeBytes {
		return ValueEnvelope{}, errors.New("value envelope exceeds byte budget")
	}
	return ValueEnvelope{state: &valueEnvelopeState{document: document, artifact: artifactBytes}}, nil
}

func OpenValueEnvelope(catalog ValueTypeCatalog, raw []byte) (ValueEnvelope, error) {
	if len(raw) == 0 || len(raw) > MaxValueEnvelopeBytes {
		return ValueEnvelope{}, errors.New("value envelope exceeds byte budget")
	}
	if err := artifact.InspectJSONBudget(raw, MaxDefinitionDepth, MaxDefinitionNodes, MaxInlineValueBytes); err != nil {
		return ValueEnvelope{}, fmt.Errorf("value envelope exceeds structural budget: %w", err)
	}
	canonical, err := artifact.Canonicalize(raw)
	if err != nil || !bytes.Equal(canonical, raw) {
		return ValueEnvelope{}, errors.New("value envelope is not canonical")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	var document valueEnvelopeDocument
	if err := decoder.Decode(&document); err != nil {
		return ValueEnvelope{}, fmt.Errorf("decode value envelope: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return ValueEnvelope{}, errors.New("value envelope contains trailing JSON values")
	}
	if document.Format != ValueEnvelopeFormat || document.Version != ValueEnvelopeVersion ||
		!validRepresentationCodec(document.Repr, document.Codec) {
		return ValueEnvelope{}, errors.New("unsupported value envelope")
	}
	sealed, err := resealDocument(catalog, document)
	if err != nil {
		return ValueEnvelope{}, err
	}
	if sealed.Digest() != document.ValueDigest || !bytes.Equal(sealed.RuntimeArtifact(), raw) {
		return ValueEnvelope{}, errors.New("value envelope digest mismatch")
	}
	return sealed, nil
}

func resealDocument(catalog ValueTypeCatalog, document valueEnvelopeDocument) (ValueEnvelope, error) {
	switch document.Repr {
	case RepresentationInlineJSON:
		return SealInlineJSON(catalog, document.Type, document.Value)
	case RepresentationBlobRef:
		var ref blob.Ref
		if err := decodeCarrier(document.Value, &ref); err != nil {
			return ValueEnvelope{}, fmt.Errorf("decode blob reference: %w", err)
		}
		return SealBlobRef(catalog, document.Type, ref)
	case RepresentationStreamRef:
		var handle resource.Handle
		if err := decodeCarrier(document.Value, &handle); err != nil {
			return ValueEnvelope{}, fmt.Errorf("decode stream reference: %w", err)
		}
		return SealStreamRef(catalog, document.Type, handle)
	case RepresentationHandleRef:
		var handle resource.Handle
		if err := decodeCarrier(document.Value, &handle); err != nil {
			return ValueEnvelope{}, fmt.Errorf("decode handle reference: %w", err)
		}
		return SealHandleRef(catalog, document.Type, handle)
	default:
		return ValueEnvelope{}, errors.New("unsupported value representation")
	}
}

func validateCarrierAgainstType(catalog ValueTypeCatalog, resolved ResolvedType, representation RepresentationKind, codec string, inline []byte) error {
	if catalog == nil {
		return errors.New("trusted value type catalog is required")
	}
	if resolved.Kind != ResolvedTypeRef || resolved.Ref == nil {
		return errors.New("this value envelope generation requires an exact ref type")
	}
	definition, ok := catalog.LookupType(resolved.Ref.TypeID)
	if !ok || definition.TypeRef() != *resolved.Ref {
		return errors.New("value type is not pinned in the trusted catalog")
	}
	machine := definition.Machine()
	allowed := false
	for _, candidate := range machine.Representations {
		if candidate.Kind == representation && candidate.Codec == codec {
			allowed = true
			break
		}
	}
	if !allowed {
		return fmt.Errorf("data type does not allow %s with codec %s", representation, codec)
	}
	if representation != RepresentationInlineJSON {
		return nil
	}
	if len(machine.SchemaBundle) != 1 {
		return errors.New("inline value type requires one explicit schema root in this generation")
	}
	var value any
	decoder := json.NewDecoder(bytes.NewReader(inline))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	compiler := runtimejsonschema.NewCompiler()
	for _, schemaResource := range machine.SchemaBundle {
		var schemaDocument any
		decoder := json.NewDecoder(bytes.NewReader(schemaResource.Schema))
		decoder.UseNumber()
		if err := decoder.Decode(&schemaDocument); err != nil {
			return err
		}
		if err := compiler.AddResource(schemaResource.ID, schemaDocument); err != nil {
			return err
		}
	}
	validator, err := compiler.Compile(machine.SchemaBundle[0].ID)
	if err != nil {
		return err
	}
	if err := validator.Validate(value); err != nil {
		return fmt.Errorf("inline value violates its pinned data type: %w", err)
	}
	return nil
}

func (v ValueEnvelope) Valid() bool { return v.state != nil && v.state.document.ValueDigest.Valid() }

func (v ValueEnvelope) Digest() artifact.Digest {
	if !v.Valid() {
		return ""
	}
	return v.state.document.ValueDigest
}

func (v ValueEnvelope) Type() ResolvedType {
	if !v.Valid() {
		return ResolvedType{}
	}
	raw, _ := json.Marshal(v.state.document.Type)
	var clone ResolvedType
	_ = json.Unmarshal(raw, &clone)
	return clone
}

func (v ValueEnvelope) InlineJSON() []byte {
	if !v.Valid() || v.state.document.Repr != RepresentationInlineJSON {
		return nil
	}
	return append([]byte(nil), v.state.document.Value...)
}

func (v ValueEnvelope) Representation() RepresentationKind {
	if !v.Valid() {
		return ""
	}
	return v.state.document.Repr
}

// Durable reports whether this envelope may enter Program, trace, clipboard,
// or cache artifacts. Runtime resource authority must never be persisted.
func (v ValueEnvelope) Durable() bool {
	if !v.Valid() {
		return false
	}
	return v.state.document.Repr == RepresentationInlineJSON || v.state.document.Repr == RepresentationBlobRef
}

func (v ValueEnvelope) BlobRef() (blob.Ref, bool) {
	if !v.Valid() || v.state.document.Repr != RepresentationBlobRef {
		return blob.Ref{}, false
	}
	var ref blob.Ref
	if decodeCarrier(v.state.document.Value, &ref) != nil {
		return blob.Ref{}, false
	}
	return ref, true
}

func (v ValueEnvelope) StreamRef() (resource.Handle, bool) {
	return v.resourceRef(RepresentationStreamRef)
}

func (v ValueEnvelope) HandleRef() (resource.Handle, bool) {
	return v.resourceRef(RepresentationHandleRef)
}

func (v ValueEnvelope) resourceRef(representation RepresentationKind) (resource.Handle, bool) {
	if !v.Valid() || v.state.document.Repr != representation {
		return resource.Handle{}, false
	}
	var handle resource.Handle
	if decodeCarrier(v.state.document.Value, &handle) != nil {
		return resource.Handle{}, false
	}
	return handle, true
}

func (v ValueEnvelope) Artifact() []byte {
	if !v.Durable() {
		return nil
	}
	return append([]byte(nil), v.state.artifact...)
}

// RuntimeArtifact serializes a runtime-only envelope for an authorized,
// bounded transport. It must not be written to durable storage or logs.
func (v ValueEnvelope) RuntimeArtifact() []byte {
	if !v.Valid() {
		return nil
	}
	return append([]byte(nil), v.state.artifact...)
}

func decodeCarrier(raw []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return errors.New("carrier contains trailing JSON")
	}
	return nil
}
