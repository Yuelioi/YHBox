package apperr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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
	if ae.ID != "WAILS_NOT_READY" {
		t.Fatalf("ID = %q", ae.ID)
	}
}

func TestError_MarshalsLowercaseFields(t *testing.T) {
	b, _ := json.Marshal(New("FOO", map[string]any{"k": 1}))
	got := string(b)
	want := `{"id":"FOO","params":{"k":1}}`
	if got != want {
		t.Fatalf("marshal = %s, want %s", got, want)
	}
}

func TestMarshalUsesStableEnvelopeForApplicationError(t *testing.T) {
	var got Envelope
	if err := json.Unmarshal(Marshal(New("WAILS_NOT_READY", map[string]any{"window": "main"})), &got); err != nil {
		t.Fatal(err)
	}
	if got.ID != "WAILS_NOT_READY" || got.Category != CategoryDomain || got.OperationID == "" || got.Retryable {
		t.Fatalf("unexpected envelope: %#v", got)
	}
	details, ok := got.Params.(map[string]any)
	if !ok || details["window"] != "main" {
		t.Fatalf("params = %#v", got.Params)
	}
}

func TestMarshalNormalizesUnclassifiedAndRetryableErrors(t *testing.T) {
	var got Envelope
	if err := json.Unmarshal(Marshal(context.DeadlineExceeded), &got); err != nil {
		t.Fatal(err)
	}
	if got.ID != IDUnexpected || got.Category != CategoryInfrastructure || got.OperationID == "" || !got.Retryable {
		t.Fatalf("unexpected envelope: %#v", got)
	}
}

type testEnvelopeError struct{}

func (testEnvelopeError) Error() string { return "wrapped" }
func (testEnvelopeError) RPCErrorEnvelope() Envelope {
	return Envelope{ID: "test.failed", Category: CategoryAdapter, RunID: "run-1", Retryable: true}
}

func TestMarshalFindsWrappedEnvelopeProvider(t *testing.T) {
	var got Envelope
	if err := json.Unmarshal(Marshal(errors.Join(errors.New("outer"), testEnvelopeError{})), &got); err != nil {
		t.Fatal(err)
	}
	if got.ID != "test.failed" || got.RunID != "run-1" || !got.Retryable {
		t.Fatalf("unexpected envelope: %#v", got)
	}
}

func TestMarshalDoesNotExposeUnclassifiedCause(t *testing.T) {
	const privateCause = `adapter win32 failed at C:\\Users\\private\\target.exe`
	encoded := string(Marshal(errors.New(privateCause)))
	if strings.Contains(encoded, "adapter") || strings.Contains(encoded, "private") || strings.Contains(encoded, "target.exe") {
		t.Fatalf("unclassified cause crossed the Wails seam: %s", encoded)
	}
}

func TestMarshalCorrelatesUnclassifiedCauseWithObserver(t *testing.T) {
	cause := errors.New("private implementation detail")
	var observed Envelope
	var observedCause error
	restore := SetObserver(func(envelope Envelope, err error) { observed, observedCause = envelope, err })
	t.Cleanup(restore)
	_ = Marshal(cause)
	if observed.ID != IDUnexpected || observed.OperationID == "" || !errors.Is(observedCause, cause) {
		t.Fatalf("observer = %#v / %v", observed, observedCause)
	}
}

func TestMarshalCorrelatesTypedCauseWithObserver(t *testing.T) {
	cause := fmt.Errorf("%w: private persistence detail", New("recording.finalize.failed", map[string]any{"destination": "workflow-resource"}))
	var observed Envelope
	var observedCause error
	restore := SetObserver(func(envelope Envelope, err error) { observed, observedCause = envelope, err })
	t.Cleanup(restore)
	_ = Marshal(cause)
	if observed.ID != "recording.finalize.failed" || observed.OperationID == "" || !errors.Is(observedCause, cause) {
		t.Fatalf("observer = %#v / %v", observed, observedCause)
	}
}
