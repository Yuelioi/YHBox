package apperr

import (
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
