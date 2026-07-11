package io

import (
	"context"
	"errors"
	"testing"

	"github.com/yottaapp/yotta/internal/node"
)

func TestStopApp_EmptyTarget_Validation(t *testing.T) {
	registry := node.NewRegistry()
	registry.Register(&StopApp{})
	rn, _ := registry.Get("StopApp")

	r := node.RunNode(context.Background(), rn, nil, nil, nil, node.StubServices(), false)
	if len(r.Validation) == 0 {
		t.Fatal("expected REQUIRED_FIELD_MISSING for empty Target")
	}
	if r.Validation[0].Code != "REQUIRED_FIELD_MISSING" {
		t.Errorf("validation[0].Code = %q, want REQUIRED_FIELD_MISSING", r.Validation[0].Code)
	}
}

func TestStopApp_NonEmpty_CallsKillAndFiresDone(t *testing.T) {
	registry := node.NewRegistry()
	registry.Register(&StopApp{})
	rn, _ := registry.Get("StopApp")

	var gotTarget string
	orig := killProcess
	killProcess = func(target string) error {
		gotTarget = target
		return nil
	}
	defer func() { killProcess = orig }()

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{saInTarget: "notepad.exe"},
		nil, node.StubServices(), false)

	if r.Error != nil {
		t.Fatalf("unexpected error: %v", r.Error)
	}
	if r.ExitName != saOutDone {
		t.Errorf("exit = %q, want Done", r.ExitName)
	}
	if gotTarget != "notepad.exe" {
		t.Errorf("killProcess got target %q, want notepad.exe", gotTarget)
	}
}

func TestStopApp_KillError_RoutesFail(t *testing.T) {
	registry := node.NewRegistry()
	registry.Register(&StopApp{})
	rn, _ := registry.Get("StopApp")

	sentinel := errors.New("process not found")
	orig := killProcess
	killProcess = func(target string) error {
		return sentinel
	}
	defer func() { killProcess = orig }()

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{saInTarget: "ghost.exe"},
		nil, node.StubServices(), false)

	if r.Error == nil {
		t.Fatal("expected error from killProcess failure")
	}
	if !errors.Is(r.Error, sentinel) {
		t.Errorf("error = %v, want wrap of %v", r.Error, sentinel)
	}
	if r.ExitName != "" {
		t.Errorf("exit = %q, should not fire Done on failure", r.ExitName)
	}
}
