package datatype

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/yottaapp/yotta/internal/artifact"
)

const (
	ValueEnvelopeFormat   = "yotta.value-envelope"
	ValueEnvelopeVersion  = "3.1"
	MaxValueEnvelopeBytes = 4 << 20
	MaxInlineValueBytes   = 2 << 20
	valueDigestDomain     = "yotta/value-envelope/v1"
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
// boundaries. The first tracer supports only canonical inline JSON; the
// representation tag makes later blob/stream/handle carriers explicit.
type ValueEnvelope struct{ state *valueEnvelopeState }

func SealInlineJSON(resolved ResolvedType, raw []byte) (ValueEnvelope, error) {
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
	body := struct {
		Type  ResolvedType       `json:"type"`
		Repr  RepresentationKind `json:"repr"`
		Codec string             `json:"codec"`
		Value json.RawMessage    `json:"value"`
	}{resolved, RepresentationInlineJSON, CodecJCSV1, canonical}
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
		Type: resolved, Repr: RepresentationInlineJSON, Codec: CodecJCSV1, Value: canonical,
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

func OpenValueEnvelope(raw []byte) (ValueEnvelope, error) {
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
		document.Repr != RepresentationInlineJSON || document.Codec != CodecJCSV1 {
		return ValueEnvelope{}, errors.New("unsupported value envelope")
	}
	sealed, err := SealInlineJSON(document.Type, document.Value)
	if err != nil {
		return ValueEnvelope{}, err
	}
	if sealed.Digest() != document.ValueDigest || !bytes.Equal(sealed.Artifact(), raw) {
		return ValueEnvelope{}, errors.New("value envelope digest mismatch")
	}
	return sealed, nil
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
	if !v.Valid() {
		return nil
	}
	return append([]byte(nil), v.state.document.Value...)
}

func (v ValueEnvelope) Artifact() []byte {
	if !v.Valid() {
		return nil
	}
	return append([]byte(nil), v.state.artifact...)
}
