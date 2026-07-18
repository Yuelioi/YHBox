package apperr

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func TestError_ErrorReturnsCodeOnly(t *testing.T) {
	e := New("CONTAINER_ID_REQUIRED", map[string]any{"id": "x"})
	if got := e.Error(); got != "CONTAINER_ID_REQUIRED" {
		t.Fatalf("Error() = %q, want code only", got)
	}
}

func TestError_AsRecoversType(t *testing.T) {
	var wrapped error = New("WAILS_NOT_READY", nil)
	var ae *Error
	if !errors.As(wrapped, &ae) {
		t.Fatal("errors.As should recover *apperr.Error")
	}
	if ae.Code != "WAILS_NOT_READY" {
		t.Fatalf("Code = %q", ae.Code)
	}
}

func TestError_MarshalsLowercaseFields(t *testing.T) {
	b, _ := json.Marshal(New("FOO", map[string]any{"k": 1}))
	got := string(b)
	want := `{"code":"FOO","params":{"k":1}}`
	if got != want {
		t.Fatalf("marshal = %s, want %s", got, want)
	}
}

func TestMarshalUsesStableEnvelopeForApplicationError(t *testing.T) {
	var got Envelope
	if err := json.Unmarshal(Marshal(New("WAILS_NOT_READY", map[string]any{"window": "main"})), &got); err != nil {
		t.Fatal(err)
	}
	if got.Code != "WAILS_NOT_READY" || got.Category != CategoryDomain || got.Message != "WAILS_NOT_READY" || got.OperationID == "" || got.Retryable {
		t.Fatalf("unexpected envelope: %#v", got)
	}
	details, ok := got.Details.(map[string]any)
	if !ok || details["window"] != "main" {
		t.Fatalf("details = %#v", got.Details)
	}
}

func TestMarshalNormalizesUnclassifiedAndRetryableErrors(t *testing.T) {
	var got Envelope
	if err := json.Unmarshal(Marshal(context.DeadlineExceeded), &got); err != nil {
		t.Fatal(err)
	}
	if got.Code != CodeUnclassified || got.Category != CategoryInfrastructure || got.OperationID == "" || !got.Retryable {
		t.Fatalf("unexpected envelope: %#v", got)
	}
}

type testEnvelopeError struct{}

func (testEnvelopeError) Error() string { return "wrapped" }
func (testEnvelopeError) RPCErrorEnvelope() Envelope {
	return Envelope{Code: "test.failed", Category: CategoryAdapter, Message: "test failed", RunID: "run-1", Retryable: true}
}

func TestMarshalFindsWrappedEnvelopeProvider(t *testing.T) {
	var got Envelope
	if err := json.Unmarshal(Marshal(errors.Join(errors.New("outer"), testEnvelopeError{})), &got); err != nil {
		t.Fatal(err)
	}
	if got.Code != "test.failed" || got.RunID != "run-1" || !got.Retryable {
		t.Fatalf("unexpected envelope: %#v", got)
	}
}
