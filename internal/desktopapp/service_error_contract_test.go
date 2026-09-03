package desktopapp

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/yottaapp/yotta/internal/apperr"
)

type errorProbeService struct{}

func (*errorProbeService) Fail() error {
	return apperr.New("error.contract.probe", map[string]any{"stage": "binding"})
}

func TestEveryWailsServiceUsesCanonicalErrorMarshaler(t *testing.T) {
	raw, err := os.ReadFile("desktop.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(raw)
	if strings.Contains(source, "application.NewService(") {
		t.Fatal("a Wails service bypasses the canonical service-level error marshaler")
	}
	if got := strings.Count(source, "application.NewServiceWithOptions("); got != 16 {
		t.Fatalf("canonical Wails service registrations = %d, want 16", got)
	}
	if !strings.Contains(source, "application.ServiceOptions{MarshalError: apperr.Marshal}") {
		t.Fatal("canonical Wails service error options are missing")
	}
}

func TestWailsServiceBindingReturnsCanonicalProblem(t *testing.T) {
	_ = application.New(application.Options{})
	bindings := application.NewBindings(nil, nil)
	service := application.NewServiceWithOptions(&errorProbeService{}, application.ServiceOptions{MarshalError: apperr.Marshal})
	if err := bindings.Add(service); err != nil {
		t.Fatal(err)
	}
	method := bindings.Get(&application.CallOptions{MethodName: "github.com/yottaapp/yotta/internal/desktopapp.errorProbeService.Fail"})
	if method == nil {
		t.Fatal("error probe binding is missing")
	}
	_, err := method.Call(context.Background(), nil)
	var callErr *application.CallError
	if !errors.As(err, &callErr) {
		t.Fatalf("binding error = %T %v", err, err)
	}
	raw, ok := callErr.Cause.(json.RawMessage)
	if !ok {
		t.Fatalf("binding cause = %T %#v", callErr.Cause, callErr.Cause)
	}
	var problem apperr.Envelope
	if err := json.Unmarshal(raw, &problem); err != nil {
		t.Fatalf("decode binding cause %s: %v", raw, err)
	}
	if problem.ID != "error.contract.probe" || problem.Category != apperr.CategoryDomain ||
		problem.OperationID == "" {
		t.Fatalf("binding problem = %#v", problem)
	}
}
