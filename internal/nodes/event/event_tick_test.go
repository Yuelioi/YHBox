package event

import (
	"context"
	"errors"
	"testing"

	"github.com/yottaapp/yotta/internal/node"
)

func TestEventTick_StubReturnsNotWiredError(t *testing.T) {
	registry := node.NewRegistry()
	registry.Register(&EventTick{})
	rn, _ := registry.Get("EventTick")

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{etInIntervalMs: 100}, nil, node.StubServices(), false)

	if r.Error == nil {
		t.Fatal("expected not-wired stub error")
	}
	if !errors.Is(r.Error, errEventTickNotWired) {
		t.Errorf("error = %v, want errEventTickNotWired", r.Error)
	}
}

func TestEventTick_SpecNoExecInSingleOutWithDelta(t *testing.T) {
	spec := (EventTick{}).Spec()
	for _, p := range spec.Inputs {
		if p.Type == "Exec" {
			t.Errorf("EventTick should have no Exec-type input, found %q", p.Name)
		}
	}
	if len(spec.Outputs) != 1 || spec.Outputs[0].Name != etOutOut {
		t.Fatalf("Outputs = %+v, want single %q", spec.Outputs, etOutOut)
	}
	data := spec.Outputs[0].Data
	if len(data) != 1 || data[0].Name != etOutDataDeltaMs || data[0].Type != "Number" {
		t.Errorf("Out.Data = %+v, want single %q Number", data, etOutDataDeltaMs)
	}
	if spec.Category != "Event" {
		t.Errorf("Category = %q, want Event", spec.Category)
	}
}
