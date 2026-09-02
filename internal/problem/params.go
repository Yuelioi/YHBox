// Package problem owns locale-free, bounded problem parameters shared by
// transports, Run evidence, MCP, and AI diagnostics. It deliberately does not
// carry rendered messages or process-local causes.
package problem

import (
	"bytes"
	"encoding/json"
	"errors"
	"regexp"

	"github.com/yottaapp/yotta/internal/artifact"
)

const (
	MaxParamsBytes = 8 << 10
	MaxParamsDepth = 8
	MaxParamsNodes = 256
)

var paramNamePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9._-]{0,127}$`)

// Params is an immutable canonical JSON object. An empty Params represents no
// parameters and serializes as an omitted field at its carriers.
type Params struct{ raw []byte }

func New(values map[string]any) (Params, error) {
	if len(values) == 0 {
		return Params{}, nil
	}
	for name := range values {
		if !paramNamePattern.MatchString(name) {
			return Params{}, errors.New("problem parameter name is invalid")
		}
	}
	raw, err := artifact.Marshal(values)
	if err != nil {
		return Params{}, err
	}
	return Open(raw)
}

func Must(values map[string]any) Params {
	params, err := New(values)
	if err != nil {
		panic(err)
	}
	return params
}

func Open(raw []byte) (Params, error) {
	if len(raw) == 0 || bytes.Equal(raw, []byte("{}")) {
		return Params{}, nil
	}
	if len(raw) > MaxParamsBytes {
		return Params{}, errors.New("problem parameters exceed byte budget")
	}
	if err := artifact.InspectJSONBudget(raw, MaxParamsDepth, MaxParamsNodes, MaxParamsBytes); err != nil {
		return Params{}, err
	}
	canonical, err := artifact.Canonicalize(raw)
	if err != nil || !bytes.Equal(canonical, raw) {
		return Params{}, errors.New("problem parameters are not canonical")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var values map[string]any
	if err := decoder.Decode(&values); err != nil || values == nil {
		return Params{}, errors.New("problem parameters must be a JSON object")
	}
	for name := range values {
		if !paramNamePattern.MatchString(name) {
			return Params{}, errors.New("problem parameter name is invalid")
		}
	}
	return Params{raw: append([]byte(nil), raw...)}, nil
}

func (p Params) Empty() bool { return len(p.raw) == 0 }

func (p Params) Bytes() json.RawMessage {
	if p.Empty() {
		return nil
	}
	return append(json.RawMessage(nil), p.raw...)
}

func (p Params) Values() map[string]any {
	if p.Empty() {
		return map[string]any{}
	}
	decoder := json.NewDecoder(bytes.NewReader(p.raw))
	decoder.UseNumber()
	var values map[string]any
	if err := decoder.Decode(&values); err != nil {
		panic("problem Params invariant: " + err.Error())
	}
	return values
}
