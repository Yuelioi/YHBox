package runclassify

import (
	"errors"
	"testing"

	"yotta/internal/apperr"
	"yotta/internal/services/container"
)

func TestRunError_ValidationFailure(t *testing.T) {
	vf := &container.ValidationFailure{Errors: []container.ValidationError{
		{Severity: "error", Code: "NO_START", GraphPath: []string{"main"}},
	}}
	re := RunError(vf)
	if re == nil || len(re.Errors) != 1 || re.Errors[0].Code != "NO_START" {
		t.Fatalf("got %+v", re)
	}
}

func TestRunError_AppErr(t *testing.T) {
	re := RunError(apperr.New("WAILS_NOT_READY", nil))
	if re == nil || re.Code != "WAILS_NOT_READY" {
		t.Fatalf("got %+v", re)
	}
}

func TestRunError_Plain(t *testing.T) {
	re := RunError(errors.New("boom"))
	if re == nil || re.Message != "boom" {
		t.Fatalf("got %+v", re)
	}
}

func TestRunError_Nil(t *testing.T) {
	if RunError(nil) != nil {
		t.Fatal("nil err → nil")
	}
}
