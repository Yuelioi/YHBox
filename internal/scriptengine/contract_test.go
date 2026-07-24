package scriptengine

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"strings"
	"testing"

	"github.com/yottaapp/yotta/internal/artifact"
)

func TestRequestRequiresExactCanonicalTypedInput(t *testing.T) {
	request := testRequest()
	if err := request.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*Request)
	}{
		{name: "protocol", mutate: func(value *Request) { value.Protocol = "legacy" }},
		{name: "attempt", mutate: func(value *Request) { value.AttemptID = "bad attempt" }},
		{name: "source", mutate: func(value *Request) { value.Source = "" }},
		{name: "timeout", mutate: func(value *Request) { value.TimeoutMillis = 0 }},
		{name: "epoch", mutate: func(value *Request) { value.EpochUnixMillis = MaxEpochUnixMillis + 1 }},
		{name: "seed", mutate: func(value *Request) { value.RandomSeed = strings.Repeat("AB", 32) }},
		{name: "non-canonical input", mutate: func(value *Request) { value.Input = json.RawMessage(`{"b":2,"a":1}`) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			invalid := request
			test.mutate(&invalid)
			if err := invalid.Validate(); err == nil {
				t.Fatal("Validate() error = nil")
			}
		})
	}
}

func TestFrameRoundTripIsCanonicalAndRejectsAmbiguity(t *testing.T) {
	request := testRequest()
	var encoded bytes.Buffer
	if err := WriteRequest(&encoded, request); err != nil {
		t.Fatalf("WriteRequest() error = %v", err)
	}
	decoded, err := ReadRequest(bytes.NewReader(encoded.Bytes()))
	if err != nil {
		t.Fatalf("ReadRequest() error = %v", err)
	}
	if decoded.AttemptID != request.AttemptID || !bytes.Equal(decoded.Input, request.Input) {
		t.Fatalf("ReadRequest() = %#v", decoded)
	}

	withTrailing := append(append([]byte(nil), encoded.Bytes()...), 0)
	if _, err := ReadRequest(bytes.NewReader(withTrailing)); err == nil {
		t.Fatal("ReadRequest(trailing data) error = nil")
	}

	payload := []byte(`{"protocol":"yotta.script.worker/0", "attemptId":"attempt-1"}`)
	if _, err := artifact.Canonicalize(payload); err != nil {
		t.Fatalf("test payload is invalid JSON: %v", err)
	}
	frame := make([]byte, 4+len(payload))
	binary.BigEndian.PutUint32(frame[:4], uint32(len(payload)))
	copy(frame[4:], payload)
	if _, err := ReadRequest(bytes.NewReader(frame)); err == nil || !strings.Contains(err.Error(), "canonical") {
		t.Fatalf("ReadRequest(non-canonical) error = %v", err)
	}
}

func TestFrameRejectsUnknownFieldsAndOversizeHeader(t *testing.T) {
	request := testRequest()
	payload, err := artifact.Marshal(struct {
		Request
		Unknown bool `json:"unknown"`
	}{Request: request, Unknown: true})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	frame := make([]byte, 4+len(payload))
	binary.BigEndian.PutUint32(frame[:4], uint32(len(payload)))
	copy(frame[4:], payload)
	if _, err := ReadRequest(bytes.NewReader(frame)); err == nil || !strings.Contains(err.Error(), "unknown") {
		t.Fatalf("ReadRequest(unknown field) error = %v", err)
	}

	var oversized [4]byte
	binary.BigEndian.PutUint32(oversized[:], MaxFrameBytes+1)
	if _, err := ReadRequest(bytes.NewReader(oversized[:])); err == nil {
		t.Fatal("ReadRequest(oversize) error = nil")
	}
}

func TestResponseUnionIsExact(t *testing.T) {
	succeeded := Response{Protocol: Protocol, AttemptID: "attempt-1", Outcome: OutcomeSucceeded, Output: json.RawMessage(`{"ok":true}`)}
	if err := succeeded.Validate(); err != nil {
		t.Fatalf("success Validate() error = %v", err)
	}
	failed := failedResponse("attempt-1", CodeGuestThrown, "script threw an exception")
	if err := failed.Validate(); err != nil {
		t.Fatalf("failure Validate() error = %v", err)
	}

	failed.Output = json.RawMessage(`null`)
	if err := failed.Validate(); err == nil {
		t.Fatal("failed response with output passed validation")
	}
}

func testRequest() Request {
	return Request{
		Protocol:        Protocol,
		AttemptID:       "attempt-1",
		Source:          `return {value: input.a + 1};`,
		Input:           json.RawMessage(`{"a":1,"b":"x"}`),
		EpochUnixMillis: 1_700_000_000_123,
		RandomSeed:      strings.Repeat("01", 32),
		TimeoutMillis:   1_000,
	}
}
