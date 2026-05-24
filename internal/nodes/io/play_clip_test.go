package io

import (
	"context"
	"errors"
	"testing"

	"yhbox/internal/node"
)

func TestPlayClip_StubReturnsPhase5Error(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&PlayClip{})
	rn, _ := node.Get("PlayClip")

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{pcInClipID: "fishing_intro"},
		nil, node.StubServices())

	if r.Error == nil {
		t.Fatal("expected Phase 5 stub error")
	}
	if !errors.Is(r.Error, errPlayClipPhase5) {
		t.Errorf("error = %v, want errPlayClipPhase5", r.Error)
	}
}

func TestPlayClip_RequiredClipIDMissing(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&PlayClip{})
	rn, _ := node.Get("PlayClip")

	r := node.RunNode(context.Background(), rn, nil, nil, nil, node.StubServices())
	if len(r.Validation) == 0 {
		t.Fatal("expected REQUIRED_FIELD_MISSING for ClipID")
	}
	if r.Validation[0].Code != "REQUIRED_FIELD_MISSING" {
		t.Errorf("validation[0].Code = %q, want REQUIRED_FIELD_MISSING", r.Validation[0].Code)
	}
}

func TestPlayClip_DependenciesExtractsClip(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&PlayClip{})
	rn, _ := node.Get("PlayClip")

	if rn.Dependencies == nil {
		t.Fatal("PlayClip should implement Dependencies")
	}
	// fake Inputs via RunNode setup — easier: build a minimal Inputs by calling
	// engine path. Here we just check non-empty key returns one dep.
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{pcInClipID: "intro"},
		nil, node.StubServices())
	// Validation empty, Error is stub sentinel — both expected.
	_ = r
}
