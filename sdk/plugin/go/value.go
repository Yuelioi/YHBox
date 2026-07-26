package pluginsdk

import (
	"bytes"
	"encoding/json"
	"errors"

	"github.com/yottaapp/yotta/internal/artifact"
)

const (
	valueEnvelopeFormat  = "yotta.value-envelope"
	valueEnvelopeVersion = "1"
	valueDigestDomain    = "yotta/value-envelope/v2"
)

type envelopeDocument struct {
	Format      string          `json:"format"`
	Version     string          `json:"version"`
	ValueDigest artifact.Digest `json:"valueDigest"`
	Type        json.RawMessage `json:"type"`
	Repr        string          `json:"repr"`
	Codec       string          `json:"codec"`
	Value       json.RawMessage `json:"value"`
}

// InlineJSON opens and integrity-checks the carrier of an inline Value
// Envelope. Schema validation remains authoritative in the Yotta host.
func InlineJSON(envelope []byte) ([]byte, error) {
	document, err := openInlineEnvelope(envelope)
	if err != nil {
		return nil, err
	}
	return append([]byte(nil), document.Value...), nil
}

// ReplaceInlineJSON keeps the exact resolved type of an input envelope and
// seals a new canonical inline carrier. Generated typed bindings should decode
// and validate the value before calling this helper; the host validates again.
func ReplaceInlineJSON(envelope, value []byte) ([]byte, error) {
	document, err := openInlineEnvelope(envelope)
	if err != nil {
		return nil, err
	}
	canonical, err := artifact.Canonicalize(value)
	if err != nil {
		return nil, err
	}
	document.Value = canonical
	document.ValueDigest, err = envelopeDigest(document)
	if err != nil {
		return nil, err
	}
	return artifact.Marshal(document)
}

func openInlineEnvelope(raw []byte) (envelopeDocument, error) {
	canonical, err := artifact.Canonicalize(raw)
	if err != nil || !bytes.Equal(canonical, raw) {
		return envelopeDocument{}, errors.New("value envelope is not canonical JSON")
	}
	var document envelopeDocument
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&document); err != nil {
		return envelopeDocument{}, err
	}
	if document.Format != valueEnvelopeFormat || document.Version != valueEnvelopeVersion || document.Repr != "inline-json" || document.Codec != "yotta.jcs/v1" || !document.ValueDigest.Valid() {
		return envelopeDocument{}, errors.New("value envelope is not a supported inline JCS value")
	}
	digest, err := envelopeDigest(document)
	if err != nil || digest != document.ValueDigest {
		return envelopeDocument{}, errors.New("value envelope digest mismatch")
	}
	return document, nil
}

func envelopeDigest(document envelopeDocument) (artifact.Digest, error) {
	body := struct {
		Version string          `json:"version"`
		Type    json.RawMessage `json:"type"`
		Repr    string          `json:"repr"`
		Codec   string          `json:"codec"`
		Value   json.RawMessage `json:"value"`
	}{document.Version, document.Type, document.Repr, document.Codec, document.Value}
	raw, err := artifact.Marshal(body)
	if err != nil {
		return "", err
	}
	return artifact.Sum(valueDigestDomain, raw)
}
